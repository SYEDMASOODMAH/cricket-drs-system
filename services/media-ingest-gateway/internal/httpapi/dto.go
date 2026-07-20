package httpapi

import (
	"time"

	"github.com/cricketdrs/services/media-ingest-gateway/internal/domain"
)

type clipResponse struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	MatchID        string    `json:"match_id"`
	CameraID       string    `json:"camera_id"`
	ContentHash    string    `json:"content_hash"`
	SizeBytes      int64     `json:"size_bytes"`
	UploadedAt     time.Time `json:"uploaded_at"`
	UploadedBy     string    `json:"uploaded_by"`
}

func toClipResponse(c domain.Clip) clipResponse {
	return clipResponse{
		ID:             string(c.ID),
		OrganizationID: string(c.OrganizationID),
		MatchID:        string(c.MatchID),
		CameraID:       string(c.CameraID),
		ContentHash:    c.ContentHash,
		SizeBytes:      c.SizeBytes,
		UploadedAt:     c.UploadedAt,
		UploadedBy:     string(c.UploadedBy),
	}
}
