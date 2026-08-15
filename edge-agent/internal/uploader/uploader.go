// Package uploader pushes an encoded clip (internal/clipformat's output)
// to Media Ingest Gateway's existing upload endpoint — the same
// authenticated-HTTP contract that service already implements and tests
// (services/media-ingest-gateway/openapi.yaml); this package adds no new
// contract, it's just the client side of one that already exists.
package uploader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Client uploads clips to a single Media Ingest Gateway instance.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL string) *Client {
	return &Client{httpClient: http.DefaultClient, baseURL: baseURL}
}

type uploadResponse struct {
	ID string `json:"id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Upload POSTs clipBytes to
// {baseURL}/v1/organizations/{orgID}/matches/{matchID}/clips?camera_id={cameraID},
// matching media-ingest-gateway's handleUploadClip exactly, and returns
// the clip ID it assigns.
func (c *Client) Upload(ctx context.Context, token string, orgID, matchID, cameraID string, clipBytes []byte) (string, error) {
	endpoint := fmt.Sprintf("%s/v1/organizations/%s/matches/%s/clips?%s",
		c.baseURL,
		url.PathEscape(orgID),
		url.PathEscape(matchID),
		url.Values{"camera_id": {cameraID}}.Encode(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(clipBytes))
	if err != nil {
		return "", fmt.Errorf("uploader: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("uploader: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("uploader: read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errBody errorResponse
		if jsonErr := json.Unmarshal(body, &errBody); jsonErr == nil && errBody.Error != "" {
			return "", fmt.Errorf("uploader: gateway rejected upload (%d): %s", resp.StatusCode, errBody.Error)
		}
		return "", fmt.Errorf("uploader: gateway rejected upload (%d): %s", resp.StatusCode, string(body))
	}

	var uploaded uploadResponse
	if err := json.Unmarshal(body, &uploaded); err != nil {
		return "", fmt.Errorf("uploader: decode response: %w", err)
	}
	if uploaded.ID == "" {
		return "", fmt.Errorf("uploader: gateway response missing clip id")
	}
	return uploaded.ID, nil
}
