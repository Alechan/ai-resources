package topics

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakePager struct {
	pages map[int64]Page
	calls []RequestOptions
}

func (f *fakePager) ListTopicsPage(_ context.Context, opts RequestOptions) (Page, error) {
	f.calls = append(f.calls, opts)
	page, ok := f.pages[opts.Cursor]
	if !ok {
		return Page{Topics: []Topic{}, Complete: true}, nil
	}
	return page, nil
}

func TestListSpacePagesUntilComplete(t *testing.T) {
	// Given
	now := time.UnixMicro(1789000000003000).UTC()
	newer := Page{Complete: false, Topics: []Topic{
		{ID: "1789000000002000", SpaceID: "spaceidxxxx1", Messages: []Message{{WebID: "webidnewer01", Text: "newer"}}},
	}}
	older := Page{Complete: true, Topics: []Topic{
		{ID: "1789000000001000", SpaceID: "spaceidxxxx1", Messages: []Message{{WebID: "webidolder01", Text: "older"}}},
	}}
	pager := &fakePager{pages: map[int64]Page{
		now.UnixMicro():     newer,
		OldestCursor(newer): older,
	}}

	// When
	page, err := ListSpace(t.Context(), pager, ListOptions{
		SpaceID: "spaceidxxxx1",
		Now:     func() time.Time { return now },
	})

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(pager.calls) != 2 || pager.calls[0].SpaceID != "spaceidxxxx1" {
		t.Fatalf("calls = %#v", pager.calls)
	}
	if len(page.Topics) != 2 || !page.Complete {
		t.Fatalf("page = %#v", page)
	}
}

func TestListSpaceRequiresSpaceID(t *testing.T) {
	if _, err := ListSpace(t.Context(), &fakePager{}, ListOptions{}); err == nil || err.Error() != "space ID is required" {
		t.Fatalf("err = %v", err)
	}
}

func TestExportFromHeartbeatsBetweenPages(t *testing.T) {
	now := time.UnixMicro(1789000000003000).UTC()
	newer := Page{Complete: false, Topics: []Topic{
		{ID: "1789000000002000", Messages: []Message{{WebID: "webidnewer01"}}},
	}}
	older := Page{Complete: true, Topics: []Topic{
		{ID: "1789000000001000", Messages: []Message{{WebID: "qMAc4oTc2i8"}}},
	}}
	pager := &fakeHeartPager{fakePager: fakePager{pages: map[int64]Page{
		now.UnixMicro():     newer,
		OldestCursor(newer): older,
	}}}

	// When
	_, err := ExportFrom(t.Context(), pager, ExportOptions{
		SpaceID:    "spaceidxxxx1",
		StartWebID: "qMAc4oTc2i8",
		Now:        func() time.Time { return now },
	})

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if pager.beats != 1 {
		t.Fatalf("heartbeats = %d, want 1", pager.beats)
	}
}

type fakeHeartPager struct {
	fakePager
	beats int
}

func (f *fakeHeartPager) Heartbeat(context.Context) error {
	f.beats++
	return nil
}

func TestExportFromOldestPageInclusive(t *testing.T) {
	// Given
	now := time.UnixMicro(1789000000003000).UTC()
	newer := Page{Complete: false, Topics: []Topic{
		{ID: "1789000000002000", SpaceID: "spaceidxxxx1", Messages: []Message{{WebID: "webidnewer01", Text: "newer"}}},
		{ID: "1789000000003000", SpaceID: "spaceidxxxx1", Messages: []Message{{WebID: "webidnewest1", Text: "newest"}}},
	}}
	older := Page{Complete: true, Topics: []Topic{
		{ID: "1789000000001000", SpaceID: "spaceidxxxx1", Messages: []Message{{WebID: "qMAc4oTc2i8", Text: "start"}}},
		{ID: "1789000000001500", SpaceID: "spaceidxxxx1", Messages: []Message{{WebID: "webidmidxx1", Text: "between"}}},
	}}
	pager := &fakePager{pages: map[int64]Page{
		now.UnixMicro():     newer,
		OldestCursor(newer): older,
	}}

	// When
	page, err := ExportFrom(t.Context(), pager, ExportOptions{
		SpaceID:    "spaceidxxxx1",
		StartWebID: "qMAc4oTc2i8",
		PageSize:   2,
		Now:        func() time.Time { return now },
	})

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(pager.calls) != 2 {
		t.Fatalf("calls = %#v", pager.calls)
	}
	if pager.calls[0].SpaceID != "spaceidxxxx1" || pager.calls[0].Cursor != now.UnixMicro() {
		t.Fatalf("first call = %#v", pager.calls[0])
	}
	if len(page.Topics) != 4 {
		t.Fatalf("topics = %#v", page.Topics)
	}
	if page.Topics[0].Messages[0].WebID != "qMAc4oTc2i8" {
		t.Fatalf("oldest kept = %#v", page.Topics[0])
	}
	if !page.Complete {
		t.Fatal("expected complete export once the start message is included")
	}
}

func TestExportFromStopsWhenPagesDoNotAdvance(t *testing.T) {
	page := Page{Complete: false, Topics: []Topic{
		{ID: "1789000000002000", Messages: []Message{{WebID: "otherwebid1"}}},
	}}
	now := time.UnixMicro(1789000000003000).UTC()
	pager := &fakePager{pages: map[int64]Page{now.UnixMicro(): page, OldestCursor(page): page}}
	_, err := ExportFrom(t.Context(), pager, ExportOptions{
		SpaceID:    "spaceidxxxx1",
		StartWebID: "qMAc4oTc2i8",
		Now:        func() time.Time { return now },
	})
	if err == nil || !strings.Contains(err.Error(), "starting message was not found") {
		t.Fatalf("err = %v", err)
	}
	if len(pager.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(pager.calls))
	}
}

func TestExportFromMissingStart(t *testing.T) {
	pager := &fakePager{pages: map[int64]Page{
		1: {Complete: true, Topics: []Topic{{ID: "2", Messages: []Message{{WebID: "otherwebid1"}}}}},
	}}
	_, err := ExportFrom(t.Context(), pager, ExportOptions{SpaceID: "spaceidxxxx1", StartWebID: "qMAc4oTc2i8", Now: func() time.Time { return time.UnixMicro(1).UTC() }})
	if err == nil || err.Error() != "starting message was not found in accessible list_topics pages" {
		t.Fatalf("err = %v", err)
	}
}

func TestSinceWebIDKeepsCompleteStartThread(t *testing.T) {
	// Given
	page := Page{Topics: []Topic{
		{ID: "1", Messages: []Message{{WebID: "olderwebid1"}}},
		{ID: "2", Messages: []Message{{WebID: "qMAc4oTc2i8"}, {WebID: "replywebid1"}}},
		{ID: "3", Messages: []Message{{WebID: "newerwebid1"}}},
	}}

	// When
	got, ok := SinceWebID(page, "qMAc4oTc2i8")

	// Then
	if !ok || len(got.Topics) != 2 || got.Topics[0].ID != "2" || len(got.Topics[0].Messages) != 2 {
		t.Fatalf("got = %#v ok=%t", got, ok)
	}
}
