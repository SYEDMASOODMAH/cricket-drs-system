import { Navigate, Route, Routes } from "react-router-dom";
import ProtectedRoute from "./routes/ProtectedRoute";
import LoginPage from "./routes/LoginPage";
import CamerasPage from "./routes/CamerasPage";

// Real dashboard views (design.md Section 8's persistent left-nav + KPI
// card grid) land in Phase 7/9 per phases.md — this is web-app's first
// real feature, kit provisioning, not the full app shell.
export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<ProtectedRoute />}>
        <Route path="/cameras" element={<CamerasPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/cameras" replace />} />
    </Routes>
  );
}
