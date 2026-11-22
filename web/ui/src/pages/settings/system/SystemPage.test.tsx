import { describe, it, expect, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/handlers";
import { systemStatusFixture, healthyReport, degradedReport } from "@/test/fixtures";
import { renderWithProviders } from "@/test/helpers";
import { createElement } from "react";
import SystemPage from "./SystemPage";

function renderPage() {
  return renderWithProviders(createElement(SystemPage));
}

describe("StatusSection", () => {
  it("shows skeletons while loading", () => {
    // Never resolve — sections stay in loading state
    server.use(
      http.get("/api/v1/system/status", () => new Promise(() => {})),
      http.get("/api/v1/system/health", () => new Promise(() => {})),
      http.get("/api/v1/tasks", () => new Promise(() => {})),
      http.get("/api/v1/system/logs", () => new Promise(() => {})),
      http.get("/api/v1/movies", () => new Promise(() => {})),
      http.get("/api/v1/queue", () => new Promise(() => {}))
    );

    const { container } = renderPage();
    const skeletons = container.querySelectorAll(".skeleton");
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it("renders system info when loaded", async () => {
    server.use(
      http.get("/api/v1/system/status", () =>
        HttpResponse.json(systemStatusFixture)
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("0.0.0-test")).toBeInTheDocument()
    );

    expect(screen.getByText("go1.23.0")).toBeInTheDocument();
    expect(screen.getByText("sqlite3")).toBeInTheDocument();
    expect(screen.getByText(":memory:")).toBeInTheDocument();
    expect(screen.getByText("1h 0m")).toBeInTheDocument();
  });

  it("shows 'Check for updates' button", async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Check for updates")).toBeInTheDocument()
    );
  });

  it("shows 'Up to date' when no update available", async () => {
    server.use(
      http.get("/api/v1/system/updates", () =>
        HttpResponse.json({
          update_available: false,
          current_version: "0.0.0-test",
          latest_version: "0.0.0-test",
        })
      )
    );

    renderPage();
    const user = userEvent.setup();

    await waitFor(() =>
      expect(screen.getByText("Check for updates")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Check for updates"));

    await waitFor(() =>
      expect(screen.getByText(/Up to date/)).toBeInTheDocument()
    );
  });

  it("opens the update window with the newer commits and Prism's commands", async () => {
    server.use(
      http.get("/api/v1/system/updates", () =>
        HttpResponse.json({
          update_available: true,
          current_version: "1111111",
          latest_version: "2222222",
          behind: 2,
          release_url: "https://github.com/beacon-stack/prism/compare/1111111...2222222",
          published_at: "2026-11-21T00:00:00Z",
          commits: [
            { sha: "2222222", subject: "feat: newer", url: "https://github.com/beacon-stack/prism/commit/2222222" },
            { sha: "aaaaaaa", subject: "fix: older", url: "https://github.com/beacon-stack/prism/commit/aaaaaaa" },
          ],
        })
      )
    );

    renderPage();
    const user = userEvent.setup();

    await waitFor(() =>
      expect(screen.getByText("Check for updates")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Check for updates"));

    await waitFor(() =>
      expect(screen.getByText("Prism update available")).toBeInTheDocument()
    );
    expect(screen.getByText("feat: newer")).toBeInTheDocument();
    expect(screen.getByText("fix: older")).toBeInTheDocument();
    expect(screen.getByText(/docker compose pull prism/)).toBeInTheDocument();
    expect(screen.getByText("View on GitHub →")).toHaveAttribute(
      "href",
      "https://github.com/beacon-stack/prism/compare/1111111...2222222"
    );
    await user.click(screen.getByText("Docker"));
    expect(screen.getByText(/docker pull ghcr.io\/beacon-stack\/prism:main/)).toBeInTheDocument();
  });

  it("explains when it can't tell", async () => {
    server.use(
      http.get("/api/v1/system/updates", () =>
        HttpResponse.json({
          update_available: false,
          current_version: "dev",
          latest_version: "",
          message: "This build has no version to compare (a local or development build), so Prism can't tell whether it's out of date.",
        })
      )
    );

    renderPage();
    const user = userEvent.setup();
    await waitFor(() =>
      expect(screen.getByText("Check for updates")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Check for updates"));
    await waitFor(() =>
      expect(screen.getByText(/can't tell whether it's out of date/)).toBeInTheDocument()
    );
    expect(screen.queryByText(/Up to date/)).not.toBeInTheDocument();
  });

  it("closes the update window", async () => {
    server.use(
      http.get("/api/v1/system/updates", () =>
        HttpResponse.json({
          update_available: true,
          current_version: "1111111",
          latest_version: "2222222",
        })
      )
    );

    renderPage();
    const user = userEvent.setup();

    await waitFor(() =>
      expect(screen.getByText("Check for updates")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Check for updates"));

    await waitFor(() =>
      expect(screen.getByText("Prism update available")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Close"));

    await waitFor(() =>
      expect(screen.queryByText("Prism update available")).not.toBeInTheDocument()
    );
  });
});

// ── HealthSection ────────────────────────────────────────────────────────────

describe("HealthSection", () => {
  it("renders 'All systems healthy' for healthy status", async () => {
    server.use(
      http.get("/api/v1/system/health", () => HttpResponse.json(healthyReport))
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("All systems healthy")).toBeInTheDocument()
    );
  });

  it("renders degraded status with check details", async () => {
    server.use(
      http.get("/api/v1/system/health", () =>
        HttpResponse.json(degradedReport)
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Degraded")).toBeInTheDocument()
    );
    expect(screen.getByText("indexer")).toBeInTheDocument();
    expect(screen.getByText("1 indexer offline")).toBeInTheDocument();
  });

  it("shows error message on fetch failure", async () => {
    server.use(
      http.get("/api/v1/system/health", () =>
        HttpResponse.json({ error: "Internal error" }, { status: 500 })
      )
    );

    renderPage();
    await waitFor(() =>
      expect(
        screen.getByText("Failed to load health data.")
      ).toBeInTheDocument()
    );
  });
});

// ── TasksSection ─────────────────────────────────────────────────────────────

describe("TasksSection", () => {
  it("shows 'No tasks registered' for empty list", async () => {
    server.use(
      http.get("/api/v1/tasks", () => HttpResponse.json([]))
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("No tasks registered.")).toBeInTheDocument()
    );
  });

  it("renders task table with names and intervals", async () => {
    server.use(
      http.get("/api/v1/tasks", () =>
        HttpResponse.json([
          { name: "refresh_metadata", interval: "24h" },
          { name: "check_downloads", interval: "5m" },
        ])
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Refresh metadata")).toBeInTheDocument()
    );
    expect(screen.getByText("every 24h")).toBeInTheDocument();
    expect(screen.getByText("Check downloads")).toBeInTheDocument();
    expect(screen.getByText("every 5m")).toBeInTheDocument();
  });

  it("shows Run now buttons for each task", async () => {
    server.use(
      http.get("/api/v1/tasks", () =>
        HttpResponse.json([
          { name: "refresh_metadata", interval: "24h" },
        ])
      )
    );

    renderPage();
    await waitFor(() =>
      expect(screen.getByText("Run now")).toBeInTheDocument()
    );
  });

  it("triggers task and shows 'Running' confirmation", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    server.use(
      http.get("/api/v1/tasks", () =>
        HttpResponse.json([
          { name: "refresh_metadata", interval: "24h" },
        ])
      ),
      http.post("/api/v1/tasks/refresh_metadata/run", () =>
        new HttpResponse(null, { status: 202 })
      )
    );

    renderPage();
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });

    await waitFor(() =>
      expect(screen.getByText("Run now")).toBeInTheDocument()
    );
    await user.click(screen.getByText("Run now"));

    await waitFor(() =>
      expect(screen.getByText(/Running ✓/)).toBeInTheDocument()
    );

    vi.useRealTimers();
  });
});

describe("SystemPage", () => {
  it("renders heading and description", async () => {
    renderPage();
    expect(screen.getByText("System")).toBeInTheDocument();
    expect(
      screen.getByText("What's running and how it's doing.")
    ).toBeInTheDocument();
  });

  it("renders the shared sections: About, Health, Tasks", async () => {
    renderPage();
    expect(await screen.findByRole("region", { name: "About" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Health" })).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Tasks" })).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Logs" })).not.toBeInTheDocument();
  });
});
