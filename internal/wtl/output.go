package wtl

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type textObserver struct {
	writer         io.Writer
	lastDeltaEnded bool
	sawDelta       bool
}

func (o *textObserver) Observe(event runEvent) {
	switch event.Event {
	case eventTurnStarted:
		_, _ = fmt.Fprintf(o.writer, "[turn %d] running...\n", event.Iteration)
		o.lastDeltaEnded = true
		o.sawDelta = false
	case eventTurnDelta:
		if event.Text == "" {
			return
		}
		_, _ = io.WriteString(o.writer, event.Text)
		o.lastDeltaEnded = strings.HasSuffix(event.Text, "\n")
		o.sawDelta = true
	case eventTurnFinished:
		if event.Response == "" {
			o.lastDeltaEnded = true
			return
		}
		if !o.sawDelta {
			_, _ = io.WriteString(o.writer, event.Response)
			o.lastDeltaEnded = strings.HasSuffix(event.Response, "\n")
		}
		if o.lastDeltaEnded {
			return
		}
		_, _ = io.WriteString(o.writer, "\n")
		o.lastDeltaEnded = true
	case eventRunCompleted:
		_, _ = fmt.Fprintln(o.writer, "Done: your request was completed successfully.")
	case eventRunExhausted:
		message := "Stopped: maximum iterations reached."
		if event.Reason == "max_retry" {
			message = "Stopped: maximum retries reached."
		}
		_, _ = fmt.Fprintln(o.writer, message)
	case eventRunInterrupted:
		_, _ = fmt.Fprintln(o.writer, "Stopped: user interrupt.")
	}
}

type ndjsonObserver struct {
	writer io.Writer
}

func (o *ndjsonObserver) Observe(event runEvent) {
	body, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(o.writer, "%s\n", body)
}

type jsonCollector struct {
	events []runEvent
}

func (o *jsonCollector) Observe(event runEvent) {
	o.events = append(o.events, event)
}

func resolveOutput(requested string, stdout io.Writer) string {
	if strings.TrimSpace(requested) != "" {
		return requested
	}
	if isTerminalWriter(stdout) {
		return "text"
	}
	return "json"
}

func isTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func isTerminalReader(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
