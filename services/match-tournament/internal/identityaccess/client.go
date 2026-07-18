// Package identityaccess is an HTTP adapter implementing
// service.ConsentChecker by calling Identity & Access's consent endpoint.
// This is match-tournament's one piece of real service-to-service
// communication, per architecture.md Section 7 ("Synchronous: gRPC or
// REST+JSON for request/response... e.g., 'get match details'").
package identityaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cricketdrs/services/match-tournament/internal/domain"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient builds a client against baseURL (Identity & Access's address —
// e.g. http://localhost:8080 locally). A 5s timeout keeps a slow/hung
// Identity & Access from blocking roster-management requests indefinitely.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type consentResponse struct {
	Grants map[string]bool `json:"grants"`
}

// IsEligibleForDRS implements service.ConsentChecker. It forwards token
// unchanged, so Identity & Access's own GetConsent authorization
// (self/guardian/organizer_admin/board_admin) decides whether the caller
// may even see this record — this adapter does not widen or bypass that.
func (c *Client) IsEligibleForDRS(ctx context.Context, token string, userID domain.UserID) (bool, error) {
	url := fmt.Sprintf("%s/v1/users/%s/consent", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("identityaccess: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("identityaccess: consent check request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var body consentResponse
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return false, fmt.Errorf("identityaccess: decode consent response: %w", err)
		}
		// Mirrors identity-access's domain.ConsentRecord.EligibleForDRS
		// exactly: video capture + AI analysis are the baseline; footage
		// reuse is a separate opt-in and irrelevant here.
		return body.Grants["video_capture"] && body.Grants["ai_analysis"], nil
	case http.StatusNotFound:
		// No consent record yet — a normal "not eligible", not an error.
		return false, nil
	default:
		return false, fmt.Errorf("identityaccess: unexpected status %d checking consent", resp.StatusCode)
	}
}
