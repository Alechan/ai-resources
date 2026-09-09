package fail

import "testing"

func TestEnvelopeIncludesDetails(t *testing.T) {
	// Given
	err := NewAPI("request failed", "retry later", "HTTP 503")

	// When
	env := err.Envelope()
	errObj, ok := env["error"].(map[string]any)
	if !ok {
		t.Fatalf("error envelope missing error object: %#v", env)
	}

	// Then
	if errObj["details"] != "HTTP 503" {
		t.Fatalf("details = %v, want HTTP 503", errObj["details"])
	}
}
