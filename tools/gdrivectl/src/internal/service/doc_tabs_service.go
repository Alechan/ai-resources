package service

import (
	"context"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/googleapi"
)

type DocTabsRequest struct {
	ID string `json:"id"`
}

type DocTabsService struct {
	tokens auth.TokenProvider
	docs   googleapi.DocsClient
}

func NewDocTabsService(tokens auth.TokenProvider, docs googleapi.DocsClient) *DocTabsService {
	return &DocTabsService{tokens: tokens, docs: docs}
}

func (s *DocTabsService) Run(ctx context.Context, req DocTabsRequest) (map[string]any, error) {
	token, err := s.tokens.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	return s.docs.DocTabs(ctx, token, googleapi.DocTabsRequest(req))
}
