import "server-only";

import { mercuryFetch } from "@/lib/api/mercury";

export type CampaignState = "DRAFT" | "ACTIVE" | "PAUSED" | "ENDED";
export type Campaign = {
  id: string;
  advertiser_id: string;
  name: string;
  state: CampaignState;
  placement_code: string;
  budget: { configured_amount_minor: string; currency: "EUR" | "GBP" | "USD" };
  targeting: { countries: string[] };
  version: number;
  created_at: string;
  updated_at: string;
};
export type Placement = { code: string; display_name: string };
export type BudgetAccount = {
  campaign_id: string;
  configured_amount_minor: string;
  committed_spend_minor: string;
  remaining_amount_minor: string;
  currency: "EUR" | "GBP" | "USD";
};
export type ConsumptionOutcome = "APPROVED" | "INSUFFICIENT_BUDGET" | "CAMPAIGN_NOT_ACTIVE";
export type BudgetConsumption = {
  consumption_id: string;
  campaign_id: string;
  outcome: ConsumptionOutcome;
  amount: { amount_minor: string; currency: BudgetAccount["currency"] };
  budget: BudgetAccount;
};

export async function listCampaigns(advertiserId: string): Promise<Campaign[]> {
  const response = await mercuryFetch<{ campaigns: Campaign[] }>(`/advertisers/${encodeURIComponent(advertiserId)}/campaigns`);
  return response.campaigns;
}
export function getCampaign(id: string): Promise<Campaign> { return mercuryFetch(`/campaigns/${encodeURIComponent(id)}`); }
export function getBudget(id: string): Promise<BudgetAccount> { return mercuryFetch(`/campaigns/${encodeURIComponent(id)}/budget`); }
export async function listPlacements(): Promise<Placement[]> {
  const response = await mercuryFetch<{ placements: Placement[] }>("/placements");
  return response.placements;
}
