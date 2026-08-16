import { useEffect, useState, type FormEvent } from "react";
import { useAuth } from "../auth/AuthContext";
import {
  listCameras,
  registerCamera,
  type Camera,
} from "../lib/cameraCalibrationClient";

// The actual "kit provisioning" workflow (prd.md Section 5.4): register a
// physical camera unit to a venue, and see everything already registered
// to this org. No calibration-status or live health-check here — those
// are separate, later features (see the implementation plan).
export default function CamerasPage() {
  const { auth, logout } = useAuth();
  const [cameras, setCameras] = useState<Camera[]>([]);
  const [venueId, setVenueId] = useState("");
  const [model, setModel] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  async function refresh() {
    if (!auth) return;
    setLoading(true);
    try {
      setCameras(await listCameras(auth.token, auth.organizationId));
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load cameras");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refresh();
  }, [auth]);

  async function handleRegister(e: FormEvent) {
    e.preventDefault();
    if (!auth) return;
    setSubmitting(true);
    setError(null);
    try {
      await registerCamera(auth.token, auth.organizationId, venueId, model);
      setVenueId("");
      setModel("");
      await refresh();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to register camera",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="page">
      <header className="page-header">
        <h1>Camera kit provisioning</h1>
        <button type="button" onClick={logout}>
          Log out
        </button>
      </header>

      <form className="card" onSubmit={handleRegister} aria-label="Register a camera">
        <h2>Register a camera</h2>
        <label htmlFor="venueId">Venue ID</label>
        <input
          id="venueId"
          value={venueId}
          onChange={(e) => setVenueId(e.target.value)}
          required
        />
        <label htmlFor="model">Model</label>
        <input
          id="model"
          value={model}
          onChange={(e) => setModel(e.target.value)}
          placeholder="e.g. DJI Action 5 Pro"
          required
        />
        <button type="submit" disabled={submitting}>
          {submitting ? "Registering…" : "Register camera"}
        </button>
      </form>

      {error && (
        <p role="alert" className="error">
          {error}
        </p>
      )}

      <section className="card">
        <h2>Registered cameras</h2>
        {loading ? (
          <p>Loading…</p>
        ) : cameras.length === 0 ? (
          <p>No cameras registered yet.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Venue</th>
                <th>Model</th>
                <th>Registered</th>
              </tr>
            </thead>
            <tbody>
              {cameras.map((c) => (
                <tr key={c.id}>
                  <td>{c.id}</td>
                  <td>{c.venue_id}</td>
                  <td>{c.model}</td>
                  <td>{new Date(c.registered_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  );
}
