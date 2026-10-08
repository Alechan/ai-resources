package topics

import (
	"strings"
	"testing"
)

func TestSlotShapeReportsOccupiedIndexes(t *testing.T) {
	// Given
	raw := []byte(`[null,32,null,[null,null,null,null,[111]],[3,1,4],1000,20,[["spaceidxxx1"]],[222],[333],2]`)

	// When
	got := SlotShape(raw)

	// Then
	if strings.Contains(got, "spaceidxxx1") {
		t.Fatalf("leaked string: %s", got)
	}
	for _, expected := range []string{"len=11", "1:num", "3:", "7:[[str(11,id)]]", "8:[num]", "10:num"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("shape %q missing %q", got, expected)
		}
	}
}

func TestShapeRedactsLiveEmptyListTopicsEnvelope(t *testing.T) {
	// Given
	raw := []byte(")]}'\n[[\"dfe.t.lt\",null,null,[\"1789000000000001\"],false,false,[true]]]")

	// When
	got := Shape(raw)

	// Then
	if strings.Contains(got, "dfe.t.lt") || strings.Contains(got, "1789000000000001") {
		t.Fatalf("leaked string: %s", got)
	}
	if !strings.Contains(got, "bytes=") || !strings.Contains(got, "prefix=xssi") || !strings.Contains(got, "str(8,str)") {
		t.Fatalf("shape = %s", got)
	}
}
