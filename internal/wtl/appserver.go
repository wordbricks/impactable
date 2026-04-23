package wtl

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
)

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

type notification struct {
	Method string
	Params map[string]any
}

// AppServerConfig configures a reusable Codex app-server client.
type AppServerConfig struct {
	CWD            string
	Model          string
	ServiceName    string
	ClientName     string
	ClientTitle    string
	ClientVersion  string
	ApprovalPolicy string
	Sandbox        string
	CommandEnv     string
}

// AppServerTurnResult is the completed result of one app-server turn.
type AppServerTurnResult struct {
	TurnID       string
	Status       string
	Response     string
	ErrorMessage string
}

// AppServerClient exposes the Codex app-server thread/turn primitive for
// product-specific WTL integrations outside the standalone wtl CLI.
type AppServerClient struct {
	runner *appServerRunner
	cfg    runConfig
}

type appServerRunner struct {
	cfg           runConfig
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	initialized   bool
	pending       map[int64]chan rpcEnvelope
	pendingMu     sync.Mutex
	nextID        int64
	notifications chan notification
	readErr       chan error
	waitErr       chan error
	closeOnce     sync.Once
}

// NewAppServerClient starts a Codex app-server process and returns a reusable
// client. Call Close when the owning run finishes.
func NewAppServerClient(cfg AppServerConfig) (*AppServerClient, error) {
	runCfg := runConfig{
		CWD:            cfg.CWD,
		Model:          cfg.Model,
		ServiceName:    cfg.ServiceName,
		ClientName:     cfg.ClientName,
		ClientTitle:    cfg.ClientTitle,
		ClientVersion:  cfg.ClientVersion,
		ApprovalPolicy: cfg.ApprovalPolicy,
		Sandbox:        cfg.Sandbox,
		CommandEnv:     cfg.CommandEnv,
	}
	runner, err := newAppServerRunner(runCfg)
	if err != nil {
		return nil, err
	}
	return &AppServerClient{runner: runner, cfg: runCfg}, nil
}

// StartThread starts a Codex thread for this client configuration.
func (c *AppServerClient) StartThread(ctx context.Context) (string, error) {
	if c == nil || c.runner == nil {
		return "", fmt.Errorf("app-server client is nil")
	}
	return c.runner.Start(ctx, c.cfg)
}

// RunTurn runs one raw prompt in an existing Codex thread.
func (c *AppServerClient) RunTurn(ctx context.Context, threadID string, prompt string, onDelta func(string)) (AppServerTurnResult, error) {
	if c == nil || c.runner == nil {
		return AppServerTurnResult{}, fmt.Errorf("app-server client is nil")
	}
	result, err := c.runner.runTurn(ctx, threadID, prompt, onDelta)
	return AppServerTurnResult{
		TurnID:       result.TurnID,
		Status:       result.Status,
		Response:     result.Response,
		ErrorMessage: result.ErrorMessage,
	}, err
}

// Close terminates the underlying app-server process.
func (c *AppServerClient) Close() error {
	if c == nil || c.runner == nil {
		return nil
	}
	return c.runner.Close()
}

func newAppServerRunner(cfg runConfig) (*appServerRunner, error) {
	commandEnv := strings.TrimSpace(cfg.CommandEnv)
	if commandEnv == "" {
		commandEnv = "WTL_CODEX_COMMAND"
	}
	commandText := strings.TrimSpace(os.Getenv(commandEnv))
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
	runner := &appServerRunner{
		cfg:           cfg,
		cmd:           cmd,
		stdin:         stdin,
		pending:       map[int64]chan rpcEnvelope{},
		notifications: make(chan notification, 512),
		readErr:       make(chan error, 1),
		waitErr:       make(chan error, 1),
	}
	go runner.readLoop(stdout, false)
	go runner.readLoop(stderr, true)
	go func() {
		runner.waitErr <- cmd.Wait()
	}()
	return runner, nil
}

func (r *appServerRunner) Start(ctx context.Context, cfg runConfig) (string, error) {
	if !r.initialized {
		if _, err := r.request(ctx, "initialize", map[string]any{
			"clientInfo": clientInfoParams(cfg),
		}); err != nil {
			return "", err
		}
		if err := r.notify("initialized", map[string]any{}); err != nil {
			return "", err
		}
		r.initialized = true
	}

	result, err := r.request(ctx, "thread/start", threadStartParams(cfg))
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

func threadStartParams(cfg runConfig) map[string]any {
	approvalPolicy := strings.TrimSpace(cfg.ApprovalPolicy)
	if approvalPolicy == "" {
		approvalPolicy = defaultApprovalPolicy
	}
	sandbox := strings.TrimSpace(cfg.Sandbox)
	if sandbox == "" {
		sandbox = defaultSandbox
	}
	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = "wtl"
	}
	return map[string]any{
		"model":          cfg.Model,
		"cwd":            cfg.CWD,
		"approvalPolicy": approvalPolicy,
		"sandbox":        sandbox,
		"serviceName":    serviceName,
	}
}

func clientInfoParams(cfg runConfig) map[string]any {
	name := strings.TrimSpace(cfg.ClientName)
	if name == "" {
		name = "wtl"
	}
	title := strings.TrimSpace(cfg.ClientTitle)
	if title == "" {
		title = "WhatTheLoop CLI"
	}
	version := strings.TrimSpace(cfg.ClientVersion)
	if version == "" {
		version = "0.1.0"
	}
	return map[string]any{
		"name":    name,
		"title":   title,
		"version": version,
	}
}

func (r *appServerRunner) RunTurn(ctx context.Context, threadID string, prompt string, onDelta func(string)) (turnResult, error) {
	return r.runTurn(ctx, threadID, buildTurnPrompt(prompt), onDelta)
}

func (r *appServerRunner) runTurn(ctx context.Context, threadID string, prompt string, onDelta func(string)) (turnResult, error) {
	turnID, err := r.startTurn(ctx, threadID, prompt)
	if err != nil {
		return turnResult{}, err
	}

	var collected strings.Builder
	finalResponse := ""
	for {
		select {
		case err := <-r.waitErr:
			if err == nil {
				return turnResult{}, fmt.Errorf("codex app-server exited unexpectedly")
			}
			return turnResult{}, err
		case err := <-r.readErr:
			if err == nil {
				return turnResult{}, fmt.Errorf("codex app-server stream closed unexpectedly")
			}
			return turnResult{}, err
		case <-ctx.Done():
			_ = r.interruptTurn(context.Background(), threadID, turnID)
			return turnResult{
				TurnID:   turnID,
				Status:   "interrupted",
				Response: collected.String(),
			}, ctx.Err()
		case note := <-r.notifications:
			switch strings.TrimSpace(note.Method) {
			case "item/agentMessage/delta":
				text := deltaText(note.Params)
				if text == "" {
					continue
				}
				collected.WriteString(text)
				if onDelta != nil {
					onDelta(text)
				}
			case "item/completed":
				item, _ := note.Params["item"].(map[string]any)
				itemType, _ := item["type"].(string)
				if itemType != "agentMessage" {
					continue
				}
				if text := extractAgentText(item); strings.TrimSpace(text) != "" {
					finalResponse = text
				}
			case "turn/completed":
				turn, _ := note.Params["turn"].(map[string]any)
				id, _ := turn["id"].(string)
				if id != "" && id != turnID {
					continue
				}
				status, _ := turn["status"].(string)
				if finalResponse == "" {
					finalResponse = collected.String()
				}
				result := turnResult{
					TurnID:   turnID,
					Status:   status,
					Response: finalResponse,
				}
				if status == "failed" {
					errBody, _ := turn["error"].(map[string]any)
					if message, _ := errBody["message"].(string); strings.TrimSpace(message) != "" {
						result.ErrorMessage = message
						return result, errors.New(message)
					}
					return result, fmt.Errorf("turn failed")
				}
				return result, nil
			}
		}
	}
}

func (r *appServerRunner) Close() error {
	var err error
	r.closeOnce.Do(func() {
		_ = r.stdin.Close()
		if r.cmd.Process != nil {
			err = r.cmd.Process.Kill()
		}
	})
	return err
}

func (r *appServerRunner) startTurn(ctx context.Context, threadID string, prompt string) (string, error) {
	result, err := r.request(ctx, "turn/start", map[string]any{
		"threadId": threadID,
		"input": []map[string]any{{
			"type": "text",
			"text": prompt,
		}},
	})
	if err != nil {
		return "", err
	}
	record := map[string]any{}
	if err := json.Unmarshal(result, &record); err != nil {
		return "", err
	}
	turn, _ := record["turn"].(map[string]any)
	turnID, _ := turn["id"].(string)
	if strings.TrimSpace(turnID) == "" {
		return "", fmt.Errorf("turn/start returned no turn id")
	}
	return turnID, nil
}

func (r *appServerRunner) interruptTurn(ctx context.Context, threadID string, turnID string) error {
	if strings.TrimSpace(threadID) == "" || strings.TrimSpace(turnID) == "" {
		return nil
	}
	_, err := r.request(ctx, "turn/interrupt", map[string]any{
		"threadId": threadID,
		"turnId":   turnID,
	})
	return err
}

func (r *appServerRunner) request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := atomic.AddInt64(&r.nextID, 1)
	responseCh := make(chan rpcEnvelope, 1)
	r.pendingMu.Lock()
	r.pending[id] = responseCh
	r.pendingMu.Unlock()

	payload, err := json.Marshal(map[string]any{
		"id":     id,
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}
	if _, err := r.stdin.Write(append(payload, '\n')); err != nil {
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

func (r *appServerRunner) notify(method string, params any) error {
	payload, err := json.Marshal(map[string]any{
		"method": method,
		"params": params,
	})
	if err != nil {
		return err
	}
	_, err = r.stdin.Write(append(payload, '\n'))
	return err
}

func (r *appServerRunner) readLoop(reader io.Reader, discard bool) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if discard {
			continue
		}
		line := scanner.Text()
		envelope := rpcEnvelope{}
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			continue
		}
		if envelope.ID != nil {
			r.pendingMu.Lock()
			ch := r.pending[*envelope.ID]
			delete(r.pending, *envelope.ID)
			r.pendingMu.Unlock()
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
		r.notifications <- notification{Method: envelope.Method, Params: params}
	}
	r.readErr <- scanner.Err()
}

func deltaText(params map[string]any) string {
	if text, _ := params["text"].(string); text != "" {
		return text
	}
	if delta, _ := params["delta"].(string); delta != "" {
		return delta
	}
	return ""
}

func extractAgentText(raw any) string {
	record, _ := raw.(map[string]any)
	if direct, ok := record["text"].(string); ok && strings.TrimSpace(direct) != "" {
		return direct
	}
	content, _ := record["content"].([]any)
	parts := make([]string, 0, len(content))
	for _, rawPart := range content {
		part, _ := rawPart.(map[string]any)
		if text, ok := part["text"].(string); ok && strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func buildTurnPrompt(prompt string) string {
	return strings.TrimSpace(fmt.Sprintf(
		"User request:\n%s\n\nContinue working on this request. When you determine that the task is fully complete, include %s at the end of your response. Do not include it if the task is still in progress or requires additional steps.",
		prompt,
		completionMarker,
	))
}
