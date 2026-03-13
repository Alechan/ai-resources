package service

import (
	"context"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/googleapi"
)

type DoctorReport struct {
	GcloudBin    string `json:"gcloud_bin"`
	GcloudExists bool   `json:"gcloud_exists"`
	TokenOK      bool   `json:"token_ok"`
	DriveOK      bool   `json:"drive_ok"`
	DocsOK       bool   `json:"docs_ok"`
	Note         string `json:"note,omitempty"`
}

type DoctorService struct {
	gcloudBin string
	hasGcloud bool
	tokens    auth.TokenProvider
	drive     googleapi.DriveClient
	docs      googleapi.DocsClient
}

func NewDoctorService(gcloudBin string, hasGcloud bool, tokens auth.TokenProvider, drive googleapi.DriveClient, docs googleapi.DocsClient) *DoctorService {
	return &DoctorService{gcloudBin: gcloudBin, hasGcloud: hasGcloud, tokens: tokens, drive: drive, docs: docs}
}

func (s *DoctorService) Run(ctx context.Context) (DoctorReport, error) {
	r := DoctorReport{GcloudBin: s.gcloudBin, GcloudExists: s.hasGcloud}
	token, err := s.tokens.AccessToken(ctx)
	if err != nil {
		return r, err
	}
	r.TokenOK = true
	if err := s.drive.Probe(ctx, token); err != nil {
		return r, err
	}
	r.DriveOK = true
	if err := s.docs.Probe(ctx, token); err != nil {
		return r, err
	}
	r.DocsOK = true
	r.Note = "all checks passed"
	return r, nil
}
