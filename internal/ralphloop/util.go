package ralphloop

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	nonSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)
	hyphenPattern  = regexp.MustCompile(`-+`)
	pullURLPattern = regexp.MustCompile(`https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+/pull/\d+`)
)

func slugify(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = nonSlugPattern.ReplaceAllString(slug, "-")
	slug = hyphenPattern.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}

func sanitizeBranch(value string) string {
	slug := slugify(strings.ReplaceAll(value, "/", "-"))
	slug = strings.Trim(slug, "-.")
	return slug
}

func defaultWorkBranch(prompt string) string {
	slug := slugify(prompt)
	if len(slug) > 58 {
		slug = strings.Trim(slug[:58], "-")
	}
	if slug == "" {
		slug = "task"
	}
	return "ralph-" + slug
}

func defaultInitBranch() string {
	return fmt.Sprintf("ralph-%s", strings.ReplaceAll(strings.TrimSpace(nowUTC().Format("20060102-150405")), " ", "-"))
}

func deriveWorktreeID(worktreePath string) (string, error) {
	canonical, err := filepath.EvalSymlinks(worktreePath)
	if err != nil {
		canonical, err = filepath.Abs(worktreePath)
		if err != nil {
			return "", err
		}
	}
	sum := sha1.Sum([]byte(canonical))
	shortHash := hex.EncodeToString(sum[:])[:8]
	base := slugify(filepath.Base(canonical))
	if base == "" {
		base = "worktree"
	}
	return base + "-" + shortHash, nil
}

func runtimeRoot(worktreePath string, worktreeID string) string {
	return filepath.Join(worktreePath, ".worktree", worktreeID)
}

func parseFields(mask string) []string {
	if strings.TrimSpace(mask) == "" {
		return nil
	}
	parts := strings.Split(mask, ",")
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		fields = append(fields, trimmed)
	}
	return fields
}

func mustJSON(v any) string {
	body, _ := json.Marshal(v)
	return string(body)
}

func containsCompletionSignal(text string) bool {
	return strings.Contains(text, completeToken)
}

func stripCompletionSignal(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, completeToken, ""))
}

func extractPullURL(text string) string {
	return pullURLPattern.FindString(text)
}

func resolveOutputPath(cwd string, target string) (string, error) {
	if strings.TrimSpace(target) == "" {
		return "", nil
	}
	absolute := target
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(cwd, target)
	}
	cleanCWD, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	cleanTarget, err := filepath.Abs(absolute)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(cleanCWD, cleanTarget)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("output file must stay under %s", cleanCWD)
	}
	return cleanTarget, nil
}

func applyFieldMaskMap(record map[string]any, fields []string) map[string]any {
	if len(fields) == 0 {
		return record
	}
	filtered := make(map[string]any, len(fields))
	for _, field := range fields {
		if value, ok := record[field]; ok {
			filtered[field] = value
		}
	}
	return filtered
}

func applyFieldMaskSlice(records []map[string]any, fields []string) []map[string]any {
	if len(fields) == 0 {
		return records
	}
	filtered := make([]map[string]any, 0, len(records))
	for _, record := range records {
		filtered = append(filtered, applyFieldMaskMap(record, fields))
	}
	return filtered
}

func writeBytes(stdout *os.File, payload []byte) error {
	_, err := stdout.Write(payload)
	return err
}
