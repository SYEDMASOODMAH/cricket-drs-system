import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from "react";

interface AuthState {
  token: string;
  organizationId: string;
}

interface AuthContextValue {
  auth: AuthState | null;
  login: (token: string, organizationId: string) => void;
  logout: () => void;
}

const STORAGE_KEY = "cricket-drs-auth";

// Token storage: localStorage. Simplest option and not XSS-hardened — a
// known, dev-appropriate simplification (this app has no other persisted
// state or third-party scripts yet to make that a live risk), same
// transparency norm as the backend's InsecureDevSigningKey.
function readStoredAuth(): AuthState | null {
  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as AuthState;
  } catch {
    return null;
  }
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [auth, setAuth] = useState<AuthState | null>(readStoredAuth);

  const login = useCallback((token: string, organizationId: string) => {
    const next = { token, organizationId };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    setAuth(next);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY);
    setAuth(null);
  }, []);

  return (
    <AuthContext.Provider value={{ auth, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
