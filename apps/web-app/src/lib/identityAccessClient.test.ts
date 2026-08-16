import { describe, expect, it, vi, afterEach } from "vitest";
import { login } from "./identityAccessClient";

describe("login", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("posts credentials and returns the token on success", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ token: "abc.def.ghi" }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await login("org-1", "admin@example.com", "password123");

    expect(result).toEqual({ token: "abc.def.ghi" });
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toContain("/v1/auth/login");
    expect(JSON.parse(init.body)).toEqual({
      organization_id: "org-1",
      email: "admin@example.com",
      password: "password123",
    });
  });

  it("throws a specific message on 401", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 401 }),
    );

    await expect(login("org-1", "bad@example.com", "wrong")).rejects.toThrow(
      /Invalid organization, email, or password/,
    );
  });

  it("throws a generic message on other failures", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 500 }),
    );

    await expect(login("org-1", "a@example.com", "x")).rejects.toThrow(
      /Login failed \(500\)/,
    );
  });
});
