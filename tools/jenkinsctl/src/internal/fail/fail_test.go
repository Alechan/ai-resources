package fail

import (
	"testing"
)

func TestNewError(t *testing.T) {
	err := New(ErrConfig, "missing env")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expected := "[CONFIG_ERROR] missing env"
	if err.Error() != expected {
		t.Fatalf("expected %s, got %s", expected, err.Error())
	}
}
