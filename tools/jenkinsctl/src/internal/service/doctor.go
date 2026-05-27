package service

import (
	"github.com/alejandro-danos/jenkinsctl/internal/jenkinsapi"
	"net/http"
)

type DoctorService struct {
	client *jenkinsapi.Client
}

func NewDoctorService(client *jenkinsapi.Client) *DoctorService {
	return &DoctorService{client: client}
}

func (s *DoctorService) CheckConnectivity() error {
	resp, err := s.client.Get("api/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return classifyHTTPError(resp, s.client.BuildURL("api/json"))
	}
	return nil
}
