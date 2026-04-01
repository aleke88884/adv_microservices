package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type DoctorClient interface {
	DoctorExists(doctorID string) (bool, error)
}

type HTTPDoctorClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPDoctorClient(baseURL string) *HTTPDoctorClient {
	return &HTTPDoctorClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type DoctorResponse struct {
	ID             string `json:"id"`
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

func (c *HTTPDoctorClient) DoctorExists(doctorID string) (bool, error) {
	url := fmt.Sprintf("%s/doctors/%s", c.baseURL, doctorID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to call doctor service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, errors.New("doctor service returned unexpected status")
	}

	var doctor DoctorResponse
	if err := json.NewDecoder(resp.Body).Decode(&doctor); err != nil {
		return false, fmt.Errorf("failed to decode doctor response: %w", err)
	}

	return true, nil
}
