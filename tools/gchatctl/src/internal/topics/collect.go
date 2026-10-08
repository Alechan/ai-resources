package topics

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
)

type Pager interface {
	ListTopicsPage(context.Context, RequestOptions) (Page, error)
}

type Heartbeater interface {
	Heartbeat(context.Context) error
}

type ExportOptions struct {
	SpaceID      string
	StartWebID   string
	PageSize     int
	MaxPages     int
	AllowPartial bool
	Now          func() time.Time
}

func SinceWebID(page Page, webID string) (Page, bool) {
	webID = strings.TrimSpace(webID)
	if webID == "" {
		return page, true
	}
	startID := ""
	for _, topic := range page.Topics {
		if topicContainsWebID(topic, webID) {
			startID = topic.ID
			break
		}
	}
	if startID == "" {
		return Page{RPC: page.RPC, Complete: page.Complete, Topics: []Topic{}}, false
	}
	filtered := Page{RPC: page.RPC, Complete: page.Complete, Topics: []Topic{}}
	for _, topic := range page.Topics {
		if topic.ID >= startID {
			filtered.Topics = append(filtered.Topics, topic)
		}
	}
	sortTopics(&filtered)
	return filtered, true
}

func Merge(pages ...Page) Page {
	merged := Page{Complete: true, Topics: []Topic{}}
	seen := map[string]int{}
	for _, page := range pages {
		if page.RPC != "" {
			merged.RPC = page.RPC
		}
		merged.Complete = merged.Complete && page.Complete
		for _, topic := range page.Topics {
			if index, ok := seen[topic.ID]; ok {
				merged.Topics[index] = topic
				continue
			}
			seen[topic.ID] = len(merged.Topics)
			merged.Topics = append(merged.Topics, topic)
		}
	}
	sortTopics(&merged)
	return merged
}

type ListOptions struct {
	SpaceID  string
	PageSize int
	MaxPages int
	Now      func() time.Time
}

func ListSpace(ctx context.Context, pager Pager, opts ListOptions) (Page, error) {
	if opts.SpaceID == "" {
		return Page{}, errors.New("space ID is required")
	}
	pages, err := walkPages(ctx, pager, ExportOptions{
		SpaceID:  opts.SpaceID,
		PageSize: opts.PageSize,
		MaxPages: opts.MaxPages,
		Now:      opts.Now,
	})
	if err != nil {
		return Page{}, err
	}
	merged := Merge(pages...)
	if len(pages) > 0 {
		merged.Complete = pages[len(pages)-1].Complete
	}
	return merged, nil
}

func ExportFrom(ctx context.Context, pager Pager, opts ExportOptions) (Page, error) {
	if opts.SpaceID == "" {
		return Page{}, errors.New("space ID is required")
	}
	if opts.StartWebID == "" {
		return Page{}, errors.New("starting message ID is required")
	}
	pages, err := walkPages(ctx, pager, opts)
	if err != nil {
		return Page{}, err
	}
	merged := Merge(pages...)
	filtered, ok := SinceWebID(merged, opts.StartWebID)
	if !ok {
		if opts.AllowPartial {
			merged.Complete = false
			return merged, nil
		}
		return Page{}, errors.New("starting message was not found in accessible list_topics pages")
	}
	filtered.Complete = true
	for _, page := range pages {
		if _, found := SinceWebID(page, opts.StartWebID); found {
			return filtered, nil
		}
	}
	filtered.Complete = false
	return filtered, nil
}

func walkPages(ctx context.Context, pager Pager, opts ExportOptions) ([]Page, error) {
	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = 100
	}
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	cursor := nowFn().UnixMicro()
	var pages []Page
	seen := map[string]bool{}
	for i := 0; i < maxPages; i++ {
		page, err := pager.ListTopicsPage(ctx, RequestOptions{SpaceID: opts.SpaceID, Cursor: cursor, PageSize: opts.PageSize})
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
		newTopics := 0
		for _, topic := range page.Topics {
			if !seen[topic.ID] {
				seen[topic.ID] = true
				newTopics++
			}
		}
		if opts.StartWebID != "" {
			if _, ok := SinceWebID(page, opts.StartWebID); ok {
				break
			}
		}
		oldest := OldestCursor(page)
		if newTopics == 0 || page.Complete || len(page.Topics) == 0 || oldest == 0 || oldest >= cursor {
			break
		}
		if heartbeater, ok := pager.(Heartbeater); ok {
			_ = heartbeater.Heartbeat(ctx)
		}
		cursor = oldest
	}
	return pages, nil
}

func topicContainsWebID(topic Topic, webID string) bool {
	for _, message := range topic.Messages {
		if message.WebID == webID {
			return true
		}
	}
	return false
}

func sortTopics(page *Page) {
	slices.SortFunc(page.Topics, func(a, b Topic) int {
		return strings.Compare(a.ID, b.ID)
	})
}
