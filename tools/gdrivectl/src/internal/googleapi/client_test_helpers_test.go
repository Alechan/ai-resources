package googleapi

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type rewriteTransport struct {
	base   *url.URL
	inner  http.RoundTripper
	errOut error
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.errOut != nil {
		return nil, t.errOut
	}
	clone := req.Clone(req.Context())
	clone.URL.Scheme = t.base.Scheme
	clone.URL.Host = t.base.Host
	clone.Host = t.base.Host
	return t.inner.RoundTrip(clone)
}

func testHTTPClient(baseURL string) *http.Client {
	u, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}
	return &http.Client{
		Transport: &rewriteTransport{
			base:  u,
			inner: http.DefaultTransport,
		},
	}
}

func testHTTPClientWithErr(err error) *http.Client {
	return &http.Client{
		Transport: &rewriteTransport{
			base:   &url.URL{Scheme: "http", Host: "example.invalid"},
			inner:  http.DefaultTransport,
			errOut: err,
		},
	}
}

func readFixture(t *testing.T, rel string) []byte {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	p := filepath.Join(repoRoot, "testdata", rel)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", rel, err)
	}
	return b
}
