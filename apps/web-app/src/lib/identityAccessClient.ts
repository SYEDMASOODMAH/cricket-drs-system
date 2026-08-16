import { IDENTITY_ACCESS_URL } from "./config";

export interface LoginResult {
  token: string;
}

// login authenticates against identity-access's POST /v1/auth/login
// (openapi.yaml: LoginRequest -> LoginResponse) and returns the bearer
// token to attach to every subsequent authenticated request.
export async function login(
  organizationId: string,
  email: string,
  password: string,
): Promise<LoginResult> {
  const res = await fetch(`${IDENTITY_ACCESS_URL}/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      organization_id: organizationId,
      email,
      password,
    }),
  });
  if (!res.ok) {
    throw new Error(
      res.status === 401
        ? "Invalid organization, email, or password"
        : `Login failed (${res.status})`,
    );
  }
  return (await res.json()) as LoginResult;
}
