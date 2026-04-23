package gitimpact

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AgentPhasePayload is the required JSON object returned by each app-server
// phase turn.
type AgentPhasePayload struct {
	Directive      Directive       `json:"directive"`
	WaitMessage    string          `json:"wait_message"`
	Output         string          `json:"output"`
	CollectedData  *CollectedData  `json:"collected_data"`
	LinkedData     *LinkedData     `json:"linked_data"`
	ScoredData     *ScoredData     `json:"scored_data"`
	AnalysisResult *AnalysisResult `json:"analysis_result"`
	Error          string          `json:"error"`
}

func (p AgentPhasePayload) err() error {
	if strings.TrimSpace(p.Error) == "" {
		return nil
	}
	return fmt.Errorf("%s", strings.TrimSpace(p.Error))
}

// ParseAgentPhasePayload extracts and decodes the JSON object from an agent
// response. The parser accepts plain JSON or JSON wrapped in a fenced block.
func ParseAgentPhasePayload(response string) (AgentPhasePayload, error) {
	payload, err := extractAgentJSONObject(response)
	if err != nil {
		return AgentPhasePayload{}, err
	}

	var decoded AgentPhasePayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return AgentPhasePayload{}, fmt.Errorf("decode agent phase payload: %w", err)
	}
	if decoded.Directive == "" {
		return AgentPhasePayload{}, fmt.Errorf("agent phase payload missing directive")
	}
	if !isSupportedAgentDirective(decoded.Directive) {
		return AgentPhasePayload{}, fmt.Errorf("unsupported agent directive %q", decoded.Directive)
	}
	return decoded, nil
}

func isSupportedAgentDirective(value Directive) bool {
	switch value {
	case DirectiveAdvancePhase, DirectiveComplete, DirectiveContinue, DirectiveRetry, DirectiveWait:
		return true
	default:
		return false
	}
}

func extractAgentJSONObject(response string) ([]byte, error) {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return nil, fmt.Errorf("agent response is empty")
	}
	if strings.HasPrefix(trimmed, "{") {
		if object, ok := balancedJSONObject(trimmed, 0); ok {
			return []byte(object), nil
		}
	}

	if fenced, ok := extractJSONFence(trimmed); ok {
		if object, ok := balancedJSONObject(fenced, 0); ok {
			return []byte(object), nil
		}
	}

	for idx, ch := range trimmed {
		if ch != '{' {
			continue
		}
		if object, ok := balancedJSONObject(trimmed, idx); ok {
			return []byte(object), nil
		}
	}

	return nil, fmt.Errorf("agent response did not contain a JSON object")
}

func extractJSONFence(text string) (string, bool) {
	start := strings.Index(text, "```")
	if start < 0 {
		return "", false
	}
	afterStart := text[start+3:]
	newline := strings.IndexByte(afterStart, '\n')
	if newline < 0 {
		return "", false
	}
	bodyStart := start + 3 + newline + 1
	end := strings.Index(text[bodyStart:], "```")
	if end < 0 {
		return "", false
	}
	return strings.TrimSpace(text[bodyStart : bodyStart+end]), true
}

func balancedJSONObject(text string, start int) (string, bool) {
	if start < 0 || start >= len(text) || text[start] != '{' {
		return "", false
	}

	depth := 0
	inString := false
	escaped := false
	for idx := start; idx < len(text); idx++ {
		ch := text[idx]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : idx+1], true
			}
		}
	}
	return "", false
}
