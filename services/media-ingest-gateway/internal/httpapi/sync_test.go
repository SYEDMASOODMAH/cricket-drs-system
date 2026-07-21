package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func submitSync(t *testing.T, h http.Handler, token, orgID, matchID, clipID, referenceClipID string, offsetMs int64, correlationScore float64) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"reference_clip_id": referenceClipID,
		"offset_ms":         offsetMs,
		"correlation_score": correlationScore,
	})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	path := "/v1/organizations/" + orgID + "/matches/" + matchID + "/clips/" + clipID + "/sync"
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func uploadClipID(t *testing.T, h http.Handler, token, orgID, matchID, cameraID string, content []byte) string {
	t.Helper()
	rec := uploadClip(t, h, token, orgID, matchID, cameraID, content)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 uploading clip, got %d, body %s", rec.Code, rec.Body.String())
	}
	var uploaded struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &uploaded); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	return uploaded.ID
}

func TestSubmitSync_EndToEnd_Confident(t *testing.T) {
	h := newTestAPI(t)
	referenceID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", []byte("reference"))
	targetID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-2", []byte("target"))

	rec := submitSync(t, h, orgAAdminToken, "org-a", "match-1", targetID, referenceID, 250, 0.9)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var clip struct {
		SyncOffsetMs        *int64 `json:"sync_offset_ms"`
		SyncReferenceClipID string `json:"sync_reference_clip_id"`
		SyncConfident       bool   `json:"sync_confident"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &clip); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if clip.SyncOffsetMs == nil || *clip.SyncOffsetMs != 250 {
		t.Fatalf("unexpected sync offset: %+v", clip.SyncOffsetMs)
	}
	if clip.SyncReferenceClipID != referenceID {
		t.Fatalf("unexpected reference clip id: %q, want %q", clip.SyncReferenceClipID, referenceID)
	}
	if !clip.SyncConfident {
		t.Error("expected a 0.9 correlation score to be sync-confident")
	}
}

func TestSubmitSync_LowConfidence(t *testing.T) {
	h := newTestAPI(t)
	referenceID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", []byte("reference"))
	targetID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-2", []byte("target"))

	rec := submitSync(t, h, orgAAdminToken, "org-a", "match-1", targetID, referenceID, 250, 0.1)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body %s", rec.Code, rec.Body.String())
	}
	var clip struct {
		SyncConfident bool `json:"sync_confident"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &clip); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if clip.SyncConfident {
		t.Error("expected a 0.1 correlation score to not be sync-confident")
	}
}

func TestSubmitSync_PermissionDeniedReturns403(t *testing.T) {
	h := newTestAPI(t)
	referenceID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", []byte("reference"))
	targetID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-2", []byte("target"))

	rec := submitSync(t, h, orgAPlayerToken, "org-a", "match-1", targetID, referenceID, 250, 0.9)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitSync_CrossTenantReturns403(t *testing.T) {
	h := newTestAPI(t)
	referenceID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", []byte("reference"))
	targetID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-2", []byte("target"))

	rec := submitSync(t, h, orgBAdminToken, "org-a", "match-1", targetID, referenceID, 250, 0.9)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitSync_SelfReferenceReturns400(t *testing.T) {
	h := newTestAPI(t)
	targetID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-2", []byte("target"))

	rec := submitSync(t, h, orgAAdminToken, "org-a", "match-1", targetID, targetID, 0, 1.0)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitSync_InvalidCorrelationScoreReturns400(t *testing.T) {
	h := newTestAPI(t)
	referenceID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", []byte("reference"))
	targetID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-2", []byte("target"))

	rec := submitSync(t, h, orgAAdminToken, "org-a", "match-1", targetID, referenceID, 250, 1.5)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitSync_TargetClipNotFoundReturns404(t *testing.T) {
	h := newTestAPI(t)
	referenceID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-1", []byte("reference"))

	rec := submitSync(t, h, orgAAdminToken, "org-a", "match-1", "does-not-exist", referenceID, 250, 0.9)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitSync_ReferenceClipNotFoundReturns404(t *testing.T) {
	h := newTestAPI(t)
	targetID := uploadClipID(t, h, orgAAdminToken, "org-a", "match-1", "cam-2", []byte("target"))

	rec := submitSync(t, h, orgAAdminToken, "org-a", "match-1", targetID, "does-not-exist", 250, 0.9)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body %s", rec.Code, rec.Body.String())
	}
}
