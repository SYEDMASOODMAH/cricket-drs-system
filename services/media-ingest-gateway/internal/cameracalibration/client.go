// Package cameracalibration is an HTTP adapter implementing
// service.CameraRegistry by calling Camera Calibration Service's camera
// registry endpoint. This is media-ingest-gateway's one piece of real
// service-to-service communication — mirrors match-tournament's
// internal/identityaccess/client.go, the only other synchronous
// cross-service call in this codebase.
package cameracalibration

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a client against baseURL (Camera Calibration Service's
// address — e.g. http://localhost:8080 locally). A 5s timeout keeps a
// slow/hung Camera Calibration Service from blocking uploads indefinitely.
// transport is nil in tests (net/http.DefaultTransport is fine there);
// cmd/main.go passes observability.HTTPClientTransport(nil) so the
// registration check shows up as a child span of the upload request that
// triggered it.
func NewClient(baseURL string, transport http.RoundTripper) *Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second, Transport: transport},
	}
}

// IsRegistered implements service.CameraRegistry. It forwards token
// unchanged, so Camera Calibration Service's own org/role authorization on
// this endpoint decides whether the caller may even see this record — this
// adapter does not widen or bypass that.
func (c *Client) IsRegistered(ctx context.Context, token string, orgID domain.OrganizationID, cameraID domain.CameraID) (bool, error) {
	url := fmt.Sprintf("%s/v1/organizations/%s/cameras/%s", c.baseURL, orgID, cameraID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("cameracalibration: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("cameracalibration: registration check request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		// Not registered — a normal "not eligible", not an error.
		return false, nil
	default:
		return false, fmt.Errorf("cameracalibration: unexpected status %d checking camera registration", resp.StatusCode)
	}
}
