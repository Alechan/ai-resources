package service

import (
	"context"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/googleapi"
)

type SearchRequest struct {
	Query    string `json:"query"`
	Corpora  string `json:"corpora,omitempty"`
	DriveID  string `json:"drive_id,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
	Fields   string `json:"fields,omitempty"`
}

type SearchService struct {
	tokens auth.TokenProvider
	drive  googleapi.DriveClient
}

func NewSearchService(tokens auth.TokenProvider, drive googleapi.DriveClient) *SearchService {
	return &SearchService{tokens: tokens, drive: drive}
}

func (s *SearchService) Run(ctx context.Context, req SearchRequest) (map[string]any, error) {
	token, err := s.tokens.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	return s.drive.Search(ctx, token, googleapi.SearchRequest(req))
}
