package gitimpact

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func mustJSON(v any) string {
	body, _ := json.Marshal(v)
	return string(body)
}

func resolveOutputPath(cwd string, target string) (string, error) {
	if strings.TrimSpace(target) == "" {
		return "", nil
	}
	cleanCWD, err := canonicalPathWithMissingTail(cwd)
	if err != nil {
		return "", err
	}
	absolute := target
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(cleanCWD, target)
	}
	cleanTarget, err := canonicalPathWithMissingTail(absolute)
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

func canonicalPathWithMissingTail(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)

	missingParts := []string{}
	current := absolute
	for {
		_, statErr := os.Lstat(current)
		if statErr == nil {
			break
		}
		if !os.IsNotExist(statErr) {
			return "", statErr
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		missingParts = append([]string{filepath.Base(current)}, missingParts...)
		current = parent
	}

	resolvedRoot, err := filepath.EvalSymlinks(current)
	if err != nil {
		resolvedRoot = current
	}

	resolved := resolvedRoot
	for _, part := range missingParts {
		resolved = filepath.Join(resolved, part)
	}
	return filepath.Clean(resolved), nil
}
