package service

import (
	"context"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/googleapi"
)

type UploadRequest struct {
	Path     string `json:"path"`
	Name     string `json:"name,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	MIME     string `json:"mime,omitempty"`
	Fields   string `json:"fields,omitempty"`
}

type UploadService struct {
	tokens auth.TokenProvider
	drive  googleapi.DriveClient
}

func NewUploadService(tokens auth.TokenProvider, drive googleapi.DriveClient) *UploadService {
	return &UploadService{tokens: tokens, drive: drive}
}

func (s *UploadService) Run(ctx context.Context, req UploadRequest) (map[string]any, error) {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return nil, fail.NewValidation("missing --path", "provide a readable local file path")
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fail.NewValidation("unable to access upload file", "verify --path points to a readable file")
	}
	if info.IsDir() {
		return nil, fail.NewValidation("upload path is a directory", "provide a file path via --path")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fail.NewValidation("unable to read upload file", "verify --path points to a readable file")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = filepath.Base(path)
	}
	mimeType := strings.TrimSpace(req.MIME)
	if mimeType == "" {
		mimeType = detectMIME(path, content)
	}

	token, err := s.tokens.AccessToken(ctx)
	if err != nil {
		return nil, err
	}

	return s.drive.Upload(ctx, token, googleapi.UploadRequest{
		Name:     name,
		ParentID: strings.TrimSpace(req.ParentID),
		MIME:     mimeType,
		Fields:   strings.TrimSpace(req.Fields),
		Content:  content,
	})
}

func detectMIME(path string, content []byte) string {
	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		if byExt := mime.TypeByExtension(ext); byExt != "" {
			return byExt
		}
	}
	if len(content) > 0 {
		n := len(content)
		if n > 512 {
			n = 512
		}
		return http.DetectContentType(content[:n])
	}
	return "application/octet-stream"
}
