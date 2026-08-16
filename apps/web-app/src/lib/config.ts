// Base URLs for the two backend services this app calls, driven by Vite
// env vars (VITE_* — see https://vitejs.dev/guide/env-and-mode). Defaults
// to http://localhost:8080 for both, matching every Go service's own
// zero-config-local-default pattern (only one can actually run on 8080 at
// once locally — see the services' READMEs for running two side by side on
// different ports during local dev).
export const IDENTITY_ACCESS_URL: string =
  import.meta.env.VITE_IDENTITY_ACCESS_URL ?? "http://localhost:8080";

export const CAMERA_CALIBRATION_URL: string =
  import.meta.env.VITE_CAMERA_CALIBRATION_URL ?? "http://localhost:8080";
