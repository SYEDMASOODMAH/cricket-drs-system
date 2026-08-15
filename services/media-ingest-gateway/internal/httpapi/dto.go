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
	// Sync fields are omitted (nil/zero-valued omitted via omitempty)
	// until a sync offset has been submitted for this clip.
	SyncOffsetMs         *int64     `json:"sync_offset_ms,omitempty"`
	SyncReferenceClipID  *string    `json:"sync_reference_clip_id,omitempty"`
	SyncCorrelationScore *float64   `json:"sync_correlation_score,omitempty"`
	SyncConfident        bool       `json:"sync_confident"`
	SyncedAt             *time.Time `json:"synced_at,omitempty"`
}

func toClipResponse(c domain.Clip) clipResponse {
	resp := clipResponse{
		ID:                   string(c.ID),
		OrganizationID:       string(c.OrganizationID),
		MatchID:              string(c.MatchID),
		CameraID:             string(c.CameraID),
		ContentHash:          c.ContentHash,
		SizeBytes:            c.SizeBytes,
		UploadedAt:           c.UploadedAt,
		UploadedBy:           string(c.UploadedBy),
		SyncOffsetMs:         c.SyncOffsetMs,
		SyncCorrelationScore: c.SyncCorrelationScore,
		SyncConfident:        c.SyncConfident(),
		SyncedAt:             c.SyncedAt,
	}
	if c.SyncReferenceClipID != nil {
		id := string(*c.SyncReferenceClipID)
		resp.SyncReferenceClipID = &id
	}
	return resp
}

type submitSyncOffsetRequest struct {
	ReferenceClipID  string  `json:"reference_clip_id"`
	OffsetMs         int64   `json:"offset_ms"`
	CorrelationScore float64 `json:"correlation_score"`
}
