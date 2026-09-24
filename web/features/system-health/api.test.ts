import { afterEach, describe, expect, it, vi } from "vitest";
import { getSystemHealth } from "./api";

describe("getSystemHealth", () => {
  afterEach(() => vi.unstubAllEnvs());

  it("reports healthy when liveness and readiness succeed", async () => {
    const fetcher = vi.fn()
      .mockResolvedValueOnce(new Response('{"status":"ok"}', { status: 200 }))
      .mockResolvedValueOnce(new Response('{"status":"ready"}', { status: 200 }));

    await expect(getSystemHealth(fetcher)).resolves.toEqual({ kind: "healthy", detail: "Backend and PostgreSQL are ready" });
  });

  it("reports readiness failure as degraded state", async () => {
    const fetcher = vi.fn()
      .mockResolvedValueOnce(new Response('{"status":"ok"}', { status: 200 }))
      .mockResolvedValueOnce(new Response('{"status":"database_unavailable"}', { status: 503 }));

    await expect(getSystemHealth(fetcher)).resolves.toEqual({ kind: "degraded", detail: "PostgreSQL is unavailable" });
  });

  it("reports transport failure without throwing", async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error("connection refused"));

    await expect(getSystemHealth(fetcher)).resolves.toEqual({ kind: "unavailable", detail: "Backend could not be reached" });
  });

  it("distinguishes malformed API configuration from backend failure", async () => {
    vi.stubEnv("MERCURY_API_URL", "not-a-url");

    await expect(getSystemHealth(vi.fn())).resolves.toEqual({ kind: "misconfigured", detail: "MERCURY_API_URL is invalid" });
  });
});
