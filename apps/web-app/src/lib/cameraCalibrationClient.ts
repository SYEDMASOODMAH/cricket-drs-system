import { CAMERA_CALIBRATION_URL } from "./config";

export interface Camera {
  id: string;
  organization_id: string;
  venue_id: string;
  model: string;
  registered_at: string;
  registered_by: string;
}

// registerCamera calls camera-calibration's POST
// /v1/organizations/{orgID}/cameras (openapi.yaml) — the "kit
// provisioning" action itself: ties a physical camera unit to a venue.
export async function registerCamera(
  token: string,
  organizationId: string,
  venueId: string,
  model: string,
): Promise<Camera> {
  const res = await fetch(
    `${CAMERA_CALIBRATION_URL}/v1/organizations/${encodeURIComponent(organizationId)}/cameras`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ venue_id: venueId, model }),
    },
  );
  if (!res.ok) {
    throw new Error(`Failed to register camera (${res.status})`);
  }
  return (await res.json()) as Camera;
}

// listCameras calls camera-calibration's GET
// /v1/organizations/{orgID}/cameras — every camera currently registered to
// this org, across all venues.
export async function listCameras(
  token: string,
  organizationId: string,
): Promise<Camera[]> {
  const res = await fetch(
    `${CAMERA_CALIBRATION_URL}/v1/organizations/${encodeURIComponent(organizationId)}/cameras`,
    {
      headers: { Authorization: `Bearer ${token}` },
    },
  );
  if (!res.ok) {
    throw new Error(`Failed to list cameras (${res.status})`);
  }
  return (await res.json()) as Camera[];
}
