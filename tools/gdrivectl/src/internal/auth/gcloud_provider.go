package auth

import (
	"context"
	"os/exec"
	"strings"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
)

type GcloudTokenProvider struct {
	bin string
}

func NewGcloudTokenProvider(bin string) *GcloudTokenProvider {
	return &GcloudTokenProvider{bin: bin}
}

func (p *GcloudTokenProvider) AccessToken(ctx context.Context) (string, error) {
	if strings.TrimSpace(p.bin) == "" {
		return "", fail.NewConfig("gcloud binary path is empty", "set --gcloud-bin or GDRIVECTL_GCLOUD_BIN")
	}
	cmd := exec.CommandContext(ctx, p.bin, "auth", "print-access-token")
	out, err := cmd.CombinedOutput()
	if err != nil {
		s := strings.TrimSpace(string(out))
		combined := strings.ToLower(strings.TrimSpace(s + " " + err.Error()))
		if strings.Contains(combined, "scope") || strings.Contains(combined, "insufficient") {
			return "", fail.NewScope("insufficient auth scope", "run: gcloud auth login --enable-gdrive-access --update-adc")
		}
		if strings.Contains(combined, "not found") ||
			strings.Contains(combined, "executable file not found") ||
			strings.Contains(combined, "no such file or directory") {
			return "", fail.NewConfig("gcloud binary not found", "set --gcloud-bin to your gcloud executable")
		}
		if strings.Contains(combined, "permission denied") ||
			strings.Contains(combined, "credentials.db") ||
			strings.Contains(combined, "unable to create private file") ||
			strings.Contains(combined, ".config/gcloud") {
			return "", fail.NewConfig("gcloud auth config store unavailable", "run in a shell with writable gcloud config, or set CLOUDSDK_CONFIG to a writable directory")
		}
		return "", fail.NewAuth("unable to get access token", "run: gcloud auth login --enable-gdrive-access --update-adc")
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fail.NewAuth("received empty access token", "run: gcloud auth login --enable-gdrive-access --update-adc")
	}
	return token, nil
}
