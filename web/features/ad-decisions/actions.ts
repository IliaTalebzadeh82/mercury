"use server";

import { MercuryAPIError } from "@/lib/api/mercury";
import type { Decision, Diagnostic, OpportunityInput } from "./api";
import { requestDecision, requestExplanation } from "./api";

type Failure = { ok: false; message: string; fields: Record<string, string>; unavailable: boolean; diagnosticsDisabled?: boolean };
export type DecisionResult = { ok: true; decision: Decision } | Failure;
export type ExplanationResult = { ok: true; diagnostic: Diagnostic } | Failure;

function failure(error: unknown, diagnostic = false): Failure {
  if (error instanceof MercuryAPIError) {
    return {
      ok: false,
      message: diagnostic && error.status === 404 ? "Diagnostic explanations are disabled on the backend." : error.message,
      fields: error.fields,
      unavailable: error.status === 503,
      diagnosticsDisabled: diagnostic && error.status === 404,
    };
  }
  return { ok: false, message: "The decision could not be completed", fields: {}, unavailable: false };
}

export async function createAdDecision(input: OpportunityInput): Promise<DecisionResult> {
  try {
    return { ok: true, decision: await requestDecision(input) };
  } catch (error) {
    return failure(error);
  }
}

export async function explainAdDecision(input: OpportunityInput): Promise<ExplanationResult> {
  try {
    return { ok: true, diagnostic: await requestExplanation(input) };
  } catch (error) {
    return failure(error, true);
  }
}
