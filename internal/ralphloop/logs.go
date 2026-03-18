package ralphloop

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type loopLogger struct {
	path string
	mu   sync.Mutex
}

func newLoopLogger(path string) *loopLogger {
	return &loopLogger{path: path}
}

func (l *loopLogger) append(channel string, payload string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(l.path), 0o755)
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = fmt.Fprintf(file, "%s %s: %s\n", nowUTC().Format(timeRFC3339()), channel, strings.TrimSpace(payload))
}

func findLogs(repoRoot string, selector string) ([]string, error) {
	paths := []string{}
	for _, root := range []string{filepath.Join(repoRoot, ".worktree"), filepath.Join(repoRoot, ".worktrees")} {
		if !fileExists(root) {
			continue
		}
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if filepath.Base(path) != "ralph-loop.log" {
				return nil
			}
			if strings.TrimSpace(selector) != "" && !strings.Contains(path, selector) {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(paths, func(i, j int) bool {
		left, _ := os.Stat(paths[i])
		right, _ := os.Stat(paths[j])
		if left == nil || right == nil {
			return paths[i] < paths[j]
		}
		return left.ModTime().After(right.ModTime())
	})
	return paths, nil
}

func readTail(path string, lines int, raw bool) ([]logRecord, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimRight(string(body), "\n")
	if trimmed == "" {
		return []logRecord{}, nil
	}
	parts := strings.Split(trimmed, "\n")
	if lines > 0 && len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	records := make([]logRecord, 0, len(parts))
	for i, line := range parts {
		record := logRecord{Line: line, Raw: raw, LineNumber: i + 1}
		if raw {
			record.Rendered = line
		} else {
			record.Rendered = renderLogLine(line)
		}
		records = append(records, record)
	}
	return records, nil
}

func followLog(ctx context.Context, path string, lines int, raw bool, stdout io.Writer) error {
	cmd := exec.CommandContext(ctx, "tail", "-n", fmt.Sprintf("%d", lines), "-f", path)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		record := logRecord{Line: line, Raw: raw, Rendered: line}
		if !raw {
			record.Rendered = renderLogLine(line)
		}
		body, _ := json.Marshal(record)
		_, _ = fmt.Fprintln(stdout, string(body))
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return cmd.Wait()
}

func renderLogLine(line string) string {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) != 3 {
		return line
	}
	payload := parts[2]
	record := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &record); err != nil {
		return line
	}
	if text, ok := record["message"].(string); ok && strings.TrimSpace(text) != "" {
		return text
	}
	if text, ok := record["text"].(string); ok && strings.TrimSpace(text) != "" {
		return text
	}
	return line
}
