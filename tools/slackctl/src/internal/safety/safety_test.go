package safety

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSlackctlResourcesContainNoEnvironmentOrCredentialMaterial(t *testing.T) {
	root := repositoryRoot(t)
	credentialPattern := regexp.MustCompile(`(?i)xox[a-z]-[a-z0-9-]{8,}`)
	headerPattern := regexp.MustCompile(`(?i)(cookie|authorization):[ \t]+[a-z0-9]`)
	formPattern := regexp.MustCompile(`(?i)(token|cookie)=[a-z0-9]{8,}`)
	denylist := loadDenylist(t)
	for _, relative := range []string{"tools/slackctl", "skills/slackctl-conversation-ops"} {
		path := filepath.Join(root, relative)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required resource missing: %s", relative)
		}
		err := filepath.WalkDir(path, func(name string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || strings.HasSuffix(name, ".test") {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				t.Errorf("%s is a symlink; safety scan refuses external content", name)
				return nil
			}
			data, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			text := string(data)
			for _, forbidden := range []string{"/" + "Users/", "/" + "home/"} {
				if strings.Contains(text, forbidden) {
					t.Errorf("%s contains absolute home path %q", name, forbidden)
				}
			}
			if credentialPattern.MatchString(text) {
				t.Errorf("%s contains credential-shaped material", name)
			}
			if (filepath.Ext(name) != ".go" && headerPattern.MatchString(text)) || formPattern.MatchString(text) {
				t.Errorf("%s contains a statically embedded authentication value", name)
			}
			for _, forbidden := range denylist {
				if strings.Contains(text, forbidden) {
					t.Errorf("%s contains a locally denied value", name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

	}
}

func loadDenylist(t *testing.T) []string {
	t.Helper()
	path := os.Getenv("SLACKCTL_SAFETY_DENYLIST")
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read safety denylist: %v", err)
	}
	var values []string
	for _, line := range strings.Split(string(data), "\n") {
		if value := strings.TrimSpace(line); value != "" && !strings.HasPrefix(value, "#") {
			values = append(values, value)
		}
	}
	return values
}

func TestDocumentationCoversSupportedCommands(t *testing.T) {
	root := repositoryRoot(t)
	for _, relative := range []string{"tools/slackctl/README.md", "skills/slackctl-conversation-ops/SKILL.md"} {
		data, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("%s: %v", relative, err)
		}
		text := string(data)
		for _, command := range []string{"slackctl init", "slackctl doctor", "slackctl conversation export"} {
			if !strings.Contains(text, command) {
				t.Errorf("%s does not document %q", relative, command)
			}
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(workingDirectory, "..", "..", "..", "..", ".."))
}
