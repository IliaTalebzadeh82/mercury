import { mercuryAPIURL } from "@/lib/config/env";

export type SystemHealthState =
  | { kind: "healthy"; detail: "Backend and PostgreSQL are ready" }
  | { kind: "degraded"; detail: string }
  | { kind: "misconfigured"; detail: string }
  | { kind: "unavailable"; detail: string };

type StatusPayload = { status?: unknown };

export async function getSystemHealth(fetcher: typeof fetch = fetch): Promise<SystemHealthState> {
  let baseURL: string;
  try {
    baseURL = mercuryAPIURL();
  } catch {
    return { kind: "misconfigured", detail: "MERCURY_API_URL is invalid" };
  }

  try {
    const liveness = await fetcher(`${baseURL}/healthz`, { cache: "no-store", signal: AbortSignal.timeout(3000) });
    if (!liveness.ok) {
      return { kind: "unavailable", detail: `Backend liveness returned HTTP ${liveness.status}` };
    }

    const readiness = await fetcher(`${baseURL}/readyz`, { cache: "no-store", signal: AbortSignal.timeout(3000) });
    const payload = await safeStatus(readiness);
    if (readiness.ok) {
      return { kind: "healthy", detail: "Backend and PostgreSQL are ready" };
    }
    if (readiness.status === 503) {
      const reason = payload.status === "database_unavailable" ? "PostgreSQL is unavailable" : "Backend is not ready";
      return { kind: "degraded", detail: reason };
    }
    return { kind: "unavailable", detail: `Backend readiness returned HTTP ${readiness.status}` };
  } catch {
    return { kind: "unavailable", detail: "Backend could not be reached" };
  }
}

async function safeStatus(response: Response): Promise<StatusPayload> {
  try {
    return (await response.json()) as StatusPayload;
  } catch {
    return {};
  }
}
