import "server-only";

import { mercuryFetch } from "@/lib/api/mercury";

export type OpportunityInput = {
  opportunityId: string;
  placement: string;
  country: string;
};

export type Decision = {
  decision_id: string;
  opportunity_id: string;
  placement: string;
  country: string;
  outcome: "FILL" | "NO_FILL";
  selection: { campaign_id: string; campaign_version: number } | null;
};

export type Explanation = {
  campaign_id: string;
  campaign_version: number;
  eligible: boolean;
  reasons: Array<"campaign_not_active" | "placement_mismatch" | "country_not_targeted">;
};

export type Diagnostic = {
  decision: Decision;
  explanations: Explanation[];
  truncated: boolean;
};

function payload(input: OpportunityInput) {
  return JSON.stringify({
    opportunity_id: input.opportunityId,
    placement: input.placement,
    country: input.country,
  });
}

export function requestDecision(input: OpportunityInput): Promise<Decision> {
  return mercuryFetch("/ad-decisions", { method: "POST", body: payload(input) });
}

export function requestExplanation(input: OpportunityInput): Promise<Diagnostic> {
  return mercuryFetch("/ad-decisions/explain", { method: "POST", body: payload(input) });
}
