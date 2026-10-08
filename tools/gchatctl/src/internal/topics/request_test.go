package topics

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHeartbeatBodyUsesTrailingClientBlob(t *testing.T) {
	// Given
	raw := `[null,40` + strings.Repeat(",null", 97) + `,[1,2,3,"en"]]`

	// When
	got, ok := HeartbeatBody(raw)

	// Then
	if !ok {
		t.Fatal("expected heartbeat body")
	}
	if strings.Contains(got, `"en"`) == false || strings.Contains(got, "40") {
		t.Fatalf("heartbeat should wrap trailing blob only: %s", got)
	}
}

func TestRewriteBodyConvertsTopicSelectorToTimeCursor(t *testing.T) {
	// Given
	raw := `[null,null,null,[null,null,[[[null,"spaceidxxx1",[["webidxxxx01"]]]]]],[3,1,4],1000,20,[["spaceidxxx1"]],null,null,2` + strings.Repeat(",null", 88) + `,[1,2,3,"en"]]`

	// When
	got, err := RewriteBody(raw, RequestOptions{SpaceID: "AAQAP4TTGjI", Cursor: 1789000000000001})
	if err != nil {
		t.Fatal(err)
	}

	// Then
	var body []any
	if err := json.Unmarshal([]byte(got), &body); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(body[3])) != "[null,null,null,null,[1789000000000001]]" {
		t.Fatalf("slot 3 = %s", mustJSON(body[3]))
	}
	if string(mustJSON(body[8])) != "[1789000000000001]" || string(mustJSON(body[9])) != "[1789000000000001]" {
		t.Fatalf("cursor slots 8=%s 9=%s", mustJSON(body[8]), mustJSON(body[9]))
	}
	if strings.Contains(string(mustJSON(body[3])), "webidxxxx01") {
		t.Fatal("topic selector was not cleared")
	}
	if !strings.Contains(string(mustJSON(body[7])), `"AAQAP4TTGjI"`) {
		t.Fatalf("space slot = %s", mustJSON(body[7]))
	}
	if int(body[1].(float64)) != 32 {
		t.Fatalf("page size = %#v", body[1])
	}
	if !strings.Contains(string(mustJSON(body[99])), `"en"`) {
		t.Fatalf("trailing slot = %s", mustJSON(body[99]))
	}
}

func TestRewriteBodyPreservesPageSizeWhenUnspecified(t *testing.T) {
	// Given
	raw := `[null,40,null,[null,111],[3,1,4],1000,20,[["oldspaceid1"]],[222],[333],2]`

	// When
	got, err := RewriteBody(raw, RequestOptions{SpaceID: "AAQAP4TTGjI", Cursor: 1789000000000001})
	if err != nil {
		t.Fatal(err)
	}

	// Then
	var body []any
	if err := json.Unmarshal([]byte(got), &body); err != nil {
		t.Fatal(err)
	}
	if int(body[1].(float64)) != 40 {
		t.Fatalf("page size = %#v", body[1])
	}
}

func TestRewriteBodyKeepsCompactCursorContainer(t *testing.T) {
	// Given
	raw := `[null,40,null,[null,111],[3,1,4],1000,20,[["oldspaceid1"]],[222],[333],2]`

	// When
	got, err := RewriteBody(raw, RequestOptions{SpaceID: "AAQAP4TTGjI", Cursor: 1789000000000001, PageSize: 32})
	if err != nil {
		t.Fatal(err)
	}

	// Then
	var body []any
	if err := json.Unmarshal([]byte(got), &body); err != nil {
		t.Fatal(err)
	}
	encoded := string(mustJSON(body[3]))
	if encoded != "[null,1789000000000001]" {
		t.Fatalf("compact cursor slot = %s", encoded)
	}
}

func TestRewriteBodyReplacesSpaceAndCursor(t *testing.T) {
	// Given
	raw := `[null,40,null,[null,null,null,null,[111]],[3,4],1000,20,[[null,null,["oldspaceid1"]]],[222],[333],2]`

	// When
	got, err := RewriteBody(raw, RequestOptions{SpaceID: "AAQAP4TTGjI", Cursor: 1789000000000001, PageSize: 32})
	if err != nil {
		t.Fatal(err)
	}

	// Then
	var body []any
	if err := json.Unmarshal([]byte(got), &body); err != nil {
		t.Fatal(err)
	}
	if int(body[1].(float64)) != 32 {
		t.Fatalf("page size = %#v", body[1])
	}
	encoded := string(mustJSON(body[7]))
	if !strings.Contains(encoded, `"AAQAP4TTGjI"`) || strings.Contains(encoded, "oldspaceid1") {
		t.Fatalf("space slot = %s", encoded)
	}
	cursorSlot := string(mustJSON(body[3]))
	if !strings.Contains(cursorSlot, "1789000000000001") {
		t.Fatalf("cursor = %s", cursorSlot)
	}
}

func TestRewriteBodyPreservesTrailingSlots(t *testing.T) {
	// Given
	raw := `[null,40,null,[null,null,null,null,[111]],[3,4],1000,20,[["oldspaceid1"]],[222],[333],2` + strings.Repeat(",null", 88) + `,[1,2,3,"en"]]`

	// When
	got, err := RewriteBody(raw, RequestOptions{SpaceID: "AAQAP4TTGjI", Cursor: 1789000000000001, PageSize: 32})
	if err != nil {
		t.Fatal(err)
	}

	// Then
	var body []any
	if err := json.Unmarshal([]byte(got), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 100 {
		t.Fatalf("len = %d", len(body))
	}
	encoded := string(mustJSON(body[99]))
	if !strings.Contains(encoded, `"en"`) {
		t.Fatalf("trailing slot = %s", encoded)
	}
}

func TestRewriteBodySynthesizesTemplate(t *testing.T) {
	// When
	got, err := RewriteBody("", RequestOptions{SpaceID: "spaceidxxxx1", Cursor: 1789000000000001, PageSize: 32})

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"spaceidxxxx1"`) || !strings.Contains(got, "1789000000000001") {
		t.Fatalf("body = %s", got)
	}
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
