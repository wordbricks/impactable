package gitimpact

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestConfig(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, DefaultConfigFile)
	content := `onequery:
  org: my-company
  sources:
    github: github-main
    analytics: amplitude-prod
feature_grouping:
  strategies:
    - label_prefix
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}
	return path
}
