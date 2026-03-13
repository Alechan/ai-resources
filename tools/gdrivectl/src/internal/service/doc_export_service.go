package service

import (
	"context"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/googleapi"
)

type DocExportRequest struct {
	ID      string `json:"id"`
	MIME    string `json:"mime"`
	OutPath string `json:"out_path,omitempty"`
}

type DocExportService struct {
	tokens auth.TokenProvider
	drive  googleapi.DriveClient
}

func NewDocExportService(tokens auth.TokenProvider, drive googleapi.DriveClient) *DocExportService {
	return &DocExportService{tokens: tokens, drive: drive}
}

func (s *DocExportService) Run(ctx context.Context, req DocExportRequest) ([]byte, error) {
	token, err := s.tokens.AccessToken(ctx)
	if err != nil {
		return nil, err
	}
	return s.drive.ExportDoc(ctx, token, googleapi.ExportRequest{ID: req.ID, MIME: req.MIME})
}
