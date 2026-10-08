package launchd

import (
	"strings"
	"testing"
	"time"
)

func TestParseIntervalSixHours(t *testing.T) {
	got, err := ParseInterval("6h")
	if err != nil {
		t.Fatalf("ParseInterval() error = %v", err)
	}
	if got != 6*time.Hour {
		t.Fatalf("ParseInterval() = %v, want 6h", got)
	}
}

func TestParseIntervalRejectsNonCalendarInterval(t *testing.T) {
	for _, input := range []string{"0", "90m", "25h", "garbage"} {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseInterval(input); err == nil {
				t.Fatalf("ParseInterval(%q) error = nil, want error", input)
			}
		})
	}
}

func TestGeneratePlist(t *testing.T) {
	plist, err := GeneratePlist(Options{
		Label:           "com.example.gh-copilot-credits",
		BinaryPath:      "/Users/example/bin/gh-copilot-credits",
		CSVPath:         "/Users/example/Library/Application Support/gh-copilot-credits/usage.csv",
		Interval:        6 * time.Hour,
		StdoutPath:      "/Users/example/Library/Logs/gh-copilot-credits/stdout.log",
		StderrPath:      "/Users/example/Library/Logs/gh-copilot-credits/stderr.log",
		EnvironmentPath: "/opt/homebrew/bin:/usr/bin:/bin",
	})
	if err != nil {
		t.Fatalf("GeneratePlist() error = %v", err)
	}
	for _, want := range []string{
		"com.example.gh-copilot-credits",
		"/Users/example/bin/gh-copilot-credits",
		"current",
		"append",
		"--csv",
		"Library/Application Support/gh-copilot-credits/usage.csv",
		"<key>Hour</key><integer>0</integer>",
		"<key>Hour</key><integer>6</integer>",
		"<key>Hour</key><integer>12</integer>",
		"<key>Hour</key><integer>18</integer>",
		"<key>EnvironmentVariables</key>",
		"<key>PATH</key><string>/opt/homebrew/bin:/usr/bin:/bin</string>",
	} {
		if !strings.Contains(string(plist), want) {
			t.Errorf("plist does not contain %q:\n%s", want, plist)
		}
	}
}

func TestGeneratePlistRejectsUnsafeOptions(t *testing.T) {
	_, err := GeneratePlist(Options{Label: "", BinaryPath: "/bin/tool", CSVPath: "/tmp/usage.csv", Interval: 6 * time.Hour})
	if err == nil {
		t.Fatal("GeneratePlist() error = nil, want error")
	}
}
