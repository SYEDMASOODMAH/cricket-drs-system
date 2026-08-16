import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

// Redirects to /login if there's no authenticated session — every route
// under this one assumes auth.token is present.
export default function ProtectedRoute() {
  const { auth } = useAuth();
  if (!auth) {
    return <Navigate to="/login" replace />;
  }
  return <Outlet />;
}
