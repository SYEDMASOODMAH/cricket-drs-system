import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi, afterEach } from "vitest";
import { AuthProvider } from "../auth/AuthContext";
import LoginPage from "./LoginPage";

vi.mock("../lib/identityAccessClient", () => ({
  login: vi.fn(),
}));

import { login as loginRequest } from "../lib/identityAccessClient";

function renderLoginPage() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <LoginPage />
      </AuthProvider>
    </MemoryRouter>,
  );
}

describe("LoginPage", () => {
  afterEach(() => {
    vi.mocked(loginRequest).mockReset();
    localStorage.clear();
  });

  it("renders the login form", () => {
    renderLoginPage();
    expect(screen.getByLabelText(/organization id/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /log in/i }),
    ).toBeInTheDocument();
  });

  it("submits credentials and stores the token on success", async () => {
    vi.mocked(loginRequest).mockResolvedValue({ token: "test-token" });
    renderLoginPage();

    fireEvent.change(screen.getByLabelText(/organization id/i), {
      target: { value: "org-1" },
    });
    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "admin@example.com" },
    });
    fireEvent.change(screen.getByLabelText(/password/i), {
      target: { value: "correct-horse-battery-staple" },
    });
    fireEvent.click(screen.getByRole("button", { name: /log in/i }));

    await waitFor(() => {
      expect(loginRequest).toHaveBeenCalledWith(
        "org-1",
        "admin@example.com",
        "correct-horse-battery-staple",
      );
    });
    await waitFor(() => {
      expect(localStorage.getItem("cricket-drs-auth")).toContain(
        "test-token",
      );
    });
  });

  it("shows an error message when login fails", async () => {
    vi.mocked(loginRequest).mockRejectedValue(new Error("Invalid organization, email, or password"));
    renderLoginPage();

    fireEvent.change(screen.getByLabelText(/organization id/i), {
      target: { value: "org-1" },
    });
    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "bad@example.com" },
    });
    fireEvent.change(screen.getByLabelText(/password/i), {
      target: { value: "wrong" },
    });
    fireEvent.click(screen.getByRole("button", { name: /log in/i }));

    expect(
      await screen.findByText(/invalid organization, email, or password/i),
    ).toBeInTheDocument();
  });
});
