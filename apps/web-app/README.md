# web-app

Coach / Organizer / Board web application. React + TypeScript per `docs/architecture.md` Section 6.

**Status:** first real feature — camera kit provisioning (`docs/phases.md` Phase 2 deliverable). Login
against Identity & Access, then register/list an organization's cameras against Camera Calibration
Service (`prd.md` Section 5.4). The full dashboard layout (`docs/design.md` Section 8) is still Phase 7/9.

## What's here

```
src/
  lib/            fetch-based API clients for identity-access (login) and camera-calibration
                  (register/list cameras)
  auth/           AuthContext — holds {token, organizationId}, persisted to localStorage
  routes/         LoginPage, CamerasPage (the provisioning UI), ProtectedRoute
```

No calibration-status UI, no live pre-match camera health-check, no signup/create-org UI — see the
implementation plan's "Explicitly deferred" section for why.

## Run locally

Requires `identity-access` and `camera-calibration` both running (on different ports locally, since
both default to 8080 — see each service's README).

```bash
npm install
npm run dev
```

### Configuration (environment variables, `VITE_*` — see Vite's env docs)

| Variable | Default | Notes |
|---|---|---|
| `VITE_IDENTITY_ACCESS_URL` | `http://localhost:8080` | |
| `VITE_CAMERA_CALIBRATION_URL` | `http://localhost:8080` | |

### Bootstrapping a test org (no signup UI exists — this is a one-time platform-onboarding action)

```bash
curl -s localhost:8081/v1/organizations -d '{
  "name": "Montreal Cricket Association",
  "admin_email": "admin@mca.example",
  "admin_password": "correct-horse-battery-staple"
}'
```

Then log in through the app's `/login` page with that organization ID, email, and password.

## Test

```
npm test
```

`lib/`'s API clients are unit-tested against a mocked `fetch`; `LoginPage` has a render/submit/error
smoke test via `@testing-library/react`. Both are wired into CI (`.github/workflows/ci.yml`).
