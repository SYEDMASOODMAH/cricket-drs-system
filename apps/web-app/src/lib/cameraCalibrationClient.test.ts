import { describe, expect, it, vi, afterEach } from "vitest";
import { listCameras, registerCamera } from "./cameraCalibrationClient";

describe("registerCamera", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("posts venue_id and model with the bearer token, returns the camera", async () => {
    const camera = {
      id: "camera_1",
      organization_id: "org-1",
      venue_id: "venue-1",
      model: "DJI Action 5 Pro",
      registered_at: "2026-08-01T00:00:00Z",
      registered_by: "user-1",
    };
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => camera,
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await registerCamera(
      "test-token",
      "org-1",
      "venue-1",
      "DJI Action 5 Pro",
    );

    expect(result).toEqual(camera);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toContain("/v1/organizations/org-1/cameras");
    expect(init.headers.Authorization).toBe("Bearer test-token");
    expect(JSON.parse(init.body)).toEqual({
      venue_id: "venue-1",
      model: "DJI Action 5 Pro",
    });
  });

  it("throws on a non-2xx response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 403 }),
    );

    await expect(
      registerCamera("test-token", "org-1", "venue-1", "DJI Action 5 Pro"),
    ).rejects.toThrow(/Failed to register camera \(403\)/);
  });
});

describe("listCameras", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("fetches with the bearer token and returns the camera list", async () => {
    const cameras = [
      {
        id: "camera_1",
        organization_id: "org-1",
        venue_id: "venue-1",
        model: "DJI Action 5 Pro",
        registered_at: "2026-08-01T00:00:00Z",
        registered_by: "user-1",
      },
    ];
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => cameras,
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await listCameras("test-token", "org-1");

    expect(result).toEqual(cameras);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toContain("/v1/organizations/org-1/cameras");
    expect(init.headers.Authorization).toBe("Bearer test-token");
  });
});
