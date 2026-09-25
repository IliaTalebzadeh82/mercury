import "server-only";

import { mercuryAPIURL } from "@/lib/config/env";

export class MercuryAPIError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly fields: Record<string, string> = {},
  ) {
    super(message);
  }
}

type ErrorPayload = {
  error?: { code?: string; message?: string; fields?: Record<string, string> };
};

export async function mercuryFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${mercuryAPIURL()}/v1${path}`, {
      ...init,
      cache: "no-store",
      headers: { "Content-Type": "application/json", ...init.headers },
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    throw new MercuryAPIError(503, "backend_unavailable", "Mercury backend could not be reached");
  }

  if (!response.ok) {
    let payload: ErrorPayload = {};
    try {
      payload = (await response.json()) as ErrorPayload;
    } catch {
      // The boundary deliberately replaces an invalid upstream error body.
    }
    throw new MercuryAPIError(
      response.status,
      payload.error?.code ?? "unexpected_response",
      payload.error?.message ?? `Mercury backend returned HTTP ${response.status}`,
      payload.error?.fields,
    );
  }

  return (await response.json()) as T;
}
