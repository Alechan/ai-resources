package safety

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGchatctlResourcesContainNoEnvironmentOrCredentialMaterial(t *testing.T) {
	root := repositoryRoot(t)
	googleCookiePattern := regexp.MustCompile(`(?i)(SID|HSID|SSID|APISID|SAPISID|OSID|__Secure-1PSID)=[A-Za-z0-9_-]{8,}`)
	headerPattern := regexp.MustCompile(`(?i)(cookie|authorization):[ \t]+[a-z0-9]`)
	formPattern := regexp.MustCompile(`(?i)(token|cookie)=[a-z0-9]{8,}`)
	denylist := loadDenylist(t)
	for _, relative := range []string{"tools/gchatctl", "skills/gchatctl-conversation-ops"} {
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
			if googleCookiePattern.MatchString(text) {
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

func TestDocumentationCoversSupportedCommands(t *testing.T) {
	root := repositoryRoot(t)
	for _, relative := range []string{"tools/gchatctl/README.md", "skills/gchatctl-conversation-ops/SKILL.md"} {
		data, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("%s: %v", relative, err)
		}
		text := string(data)
		for _, command := range []string{"gchatctl init", "gchatctl doctor", "gchatctl learn", "gchatctl fixture", "gchatctl topics list", "gchatctl spaces list", "gchatctl spaces get", "gchatctl members list", "gchatctl search messages", "gchatctl conversation export"} {
			if !strings.Contains(text, command) {
				t.Errorf("%s does not document %q", relative, command)
			}
		}
	}
}

func loadDenylist(t *testing.T) []string {
	t.Helper()
	path := os.Getenv("GCHATCTL_SAFETY_DENYLIST")
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

func repositoryRoot(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(workingDirectory, "..", "..", "..", "..", ".."))
}
