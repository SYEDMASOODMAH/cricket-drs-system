import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { login as loginRequest } from "../lib/identityAccessClient";

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [organizationId, setOrganizationId] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const { token } = await loginRequest(organizationId, email, password);
      login(token, organizationId);
      navigate("/cameras");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="page page-centered">
      <form className="card" onSubmit={handleSubmit} aria-label="Log in">
        <h1>Cricket DRS</h1>
        <p className="subtitle">Coach / Organizer / Board</p>

        <label htmlFor="organizationId">Organization ID</label>
        <input
          id="organizationId"
          value={organizationId}
          onChange={(e) => setOrganizationId(e.target.value)}
          required
        />

        <label htmlFor="email">Email</label>
        <input
          id="email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />

        <label htmlFor="password">Password</label>
        <input
          id="password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />

        {error && (
          <p role="alert" className="error">
            {error}
          </p>
        )}

        <button type="submit" disabled={submitting}>
          {submitting ? "Logging in…" : "Log in"}
        </button>
      </form>
    </div>
  );
}
