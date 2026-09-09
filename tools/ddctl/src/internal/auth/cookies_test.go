package auth

import (
	"testing"
)

func TestParseCookieString_RoundTrip(t *testing.T) {
	t.Parallel()

	// Given
	raw := JoinCookiePairs([]string{"dogweb=abc", "dd_csrf_token=xyz", "_dd_s_v2=ok"})

	// When
	cookies := ParseCookieString(raw)

	// Then
	if len(cookies) != 3 {
		t.Fatalf("len(cookies) = %d, want 3", len(cookies))
	}
	if cookies[0].Name != "dogweb" || cookies[0].Value != "abc" {
		t.Fatalf("first cookie = %+v", cookies[0])
	}
}

func TestParseCookieString_SemicolonWithoutSpace(t *testing.T) {
	t.Parallel()

	cookies := ParseCookieString("dogweb=abc;dd_csrf_token=xyz")
	if len(cookies) != 2 {
		t.Fatalf("len(cookies) = %d, want 2", len(cookies))
	}
}
