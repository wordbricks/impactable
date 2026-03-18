package ralphloop

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type codexClient interface {
	Initialize(ctx context.Context) error
	StartThread(ctx context.Context, model string, cwd string, approval string, sandbox any) (string, error)
	RunTurn(ctx context.Context, threadID string, prompt string, timeout time.Duration) (string, string, error)
	Close() error
	SetNotificationHandler(func(jsonRPCNotification))
}

type rpcEnvelope struct {
	ID     *int64          `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type appServerClient struct {
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	pending       map[int64]chan rpcEnvelope
	pendingMu     sync.Mutex
	nextID        int64
	notifications chan jsonRPCNotification
	readErr       chan error
	waitErr       chan error
	handler       func(jsonRPCNotification)
	logger        *loopLogger
	closeOnce     sync.Once
}

func newAppServerClient(logger *loopLogger) (codexClient, error) {
	commandText := strings.TrimSpace(os.Getenv("RALPH_LOOP_CODEX_COMMAND"))
	parts := strings.Fields(commandText)
	if len(parts) == 0 {
		parts = []string{"codex", "app-server"}
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	client := &appServerClient{
		cmd:           cmd,
		stdin:         stdin,
		pending:       map[int64]chan rpcEnvelope{},
		notifications: make(chan jsonRPCNotification, 256),
		readErr:       make(chan error, 1),
		waitErr:       make(chan error, 1),
		logger:        logger,
	}
	go client.readLoop(stdout, "stdout")
	go client.readLoop(stderr, "stderr")
	go func() {
		client.waitErr <- cmd.Wait()
	}()
	return client, nil
}

func (c *appServerClient) SetNotificationHandler(handler func(jsonRPCNotification)) {
	c.handler = handler
}

func (c *appServerClient) Initialize(ctx context.Context) error {
	if _, err := c.request(ctx, "initialize", map[string]any{
		"clientInfo": map[string]any{"name": "ralph-loop", "version": "0.1.0"},
	}); err != nil {
		return err
	}
	return c.notify("initialized", map[string]any{})
}

func (c *appServerClient) StartThread(ctx context.Context, model string, cwd string, approval string, sandbox any) (string, error) {
	result, err := c.request(ctx, "thread/start", map[string]any{
		"model":          model,
		"cwd":            cwd,
		"approvalPolicy": approval,
		"sandbox":        sandbox,
	})
	if err != nil {
		return "", err
	}
	record := map[string]any{}
	if err := json.Unmarshal(result, &record); err != nil {
		return "", err
	}
	thread, _ := record["thread"].(map[string]any)
	id, _ := thread["id"].(string)
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("thread/start returned no thread id")
	}
	return id, nil
}

func (c *appServerClient) RunTurn(ctx context.Context, threadID string, prompt string, timeout time.Duration) (string, string, error) {
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if _, err := c.request(requestCtx, "turn/start", map[string]any{
		"threadId": threadID,
		"input": []map[string]any{{
			"type": "text",
			"text": prompt,
		}},
	}); err != nil {
		return "", "", err
	}

	textParts := []string{}
	for {
		select {
		case err := <-c.waitErr:
			if err == nil {
				return "", "", fmt.Errorf("codex app-server exited unexpectedly")
			}
			return "", "", err
		case err := <-c.readErr:
			if err == nil {
				return "", "", fmt.Errorf("codex app-server stream closed unexpectedly")
			}
			return "", "", err
		case <-requestCtx.Done():
			return "", "", fmt.Errorf("turn timed out after %s", timeout)
		case notification := <-c.notifications:
			if c.handler != nil {
				c.handler(notification)
			}
			switch strings.TrimSpace(notification.Method) {
			case "item/completed":
				if text := extractAgentText(notification.Params["item"]); strings.TrimSpace(text) != "" {
					textParts = append(textParts, text)
				}
			case "turn/completed":
				turn, _ := notification.Params["turn"].(map[string]any)
				status, _ := turn["status"].(string)
				return strings.TrimSpace(status), strings.TrimSpace(strings.Join(textParts, "\n")), nil
			}
		}
	}
}

func (c *appServerClient) Close() error {
	var err error
	c.closeOnce.Do(func() {
		_ = c.stdin.Close()
		if c.cmd.Process != nil {
			err = c.cmd.Process.Kill()
		}
	})
	return err
}

func (c *appServerClient) request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := atomic.AddInt64(&c.nextID, 1)
	responseCh := make(chan rpcEnvelope, 1)
	c.pendingMu.Lock()
	c.pending[id] = responseCh
	c.pendingMu.Unlock()

	payload, err := json.Marshal(map[string]any{
		"id":     id,
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}
	c.logger.append("stdin", string(payload))
	if _, err := c.stdin.Write(append(payload, '\n')); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case envelope := <-responseCh:
		if envelope.Error != nil {
			return nil, fmt.Errorf("%s failed: %s", method, envelope.Error.Message)
		}
		return envelope.Result, nil
	}
}

func (c *appServerClient) notify(method string, params any) error {
	payload, err := json.Marshal(map[string]any{
		"method": method,
		"params": params,
	})
	if err != nil {
		return err
	}
	c.logger.append("stdin", string(payload))
	_, err = c.stdin.Write(append(payload, '\n'))
	return err
}

func (c *appServerClient) readLoop(reader io.Reader, channel string) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		c.logger.append(channel, line)
		if channel == "stderr" {
			continue
		}
		envelope := rpcEnvelope{}
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			continue
		}
		if envelope.ID != nil {
			c.pendingMu.Lock()
			ch := c.pending[*envelope.ID]
			delete(c.pending, *envelope.ID)
			c.pendingMu.Unlock()
			if ch != nil {
				ch <- envelope
			}
			continue
		}
		if strings.TrimSpace(envelope.Method) == "" {
			continue
		}
		params := map[string]any{}
		_ = json.Unmarshal(envelope.Params, &params)
		c.notifications <- jsonRPCNotification{Method: envelope.Method, Params: params}
	}
	c.readErr <- scanner.Err()
}

func extractAgentText(raw any) string {
	record, _ := raw.(map[string]any)
	if direct, ok := record["text"].(string); ok && strings.TrimSpace(direct) != "" {
		return stripCompletionSignal(direct)
	}
	content, _ := record["content"].([]any)
	parts := make([]string, 0, len(content))
	for _, rawPart := range content {
		part, _ := rawPart.(map[string]any)
		if text, ok := part["text"].(string); ok && strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}
	return stripCompletionSignal(strings.Join(parts, "\n"))
}
