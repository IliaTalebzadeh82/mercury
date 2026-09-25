"use server";

import { randomUUID } from "node:crypto";
import { revalidatePath } from "next/cache";
import type { ActionResult } from "@/lib/action-result";
import { actionError } from "@/lib/api/action-result";
import { mercuryFetch } from "@/lib/api/mercury";
import type { Campaign } from "./api";

type CreateInput = {
  advertiserId: string; name: string; placementCode: string; amountMinor: string;
  currency: string; countries: string[];
};

export async function createCampaign(input: CreateInput): Promise<ActionResult> {
  try {
    const campaign = await mercuryFetch<Campaign>(`/advertisers/${encodeURIComponent(input.advertiserId)}/campaigns`, {
      method: "POST", headers: { "Idempotency-Key": randomUUID() },
      body: JSON.stringify({ name: input.name, placement_code: input.placementCode, budget: { configured_amount_minor: input.amountMinor, currency: input.currency }, targeting: { countries: input.countries } }),
    });
    revalidatePath(`/advertisers/${input.advertiserId}/campaigns`);
    return { ok: true, resourceId: campaign.id };
  } catch (error) { return actionError(error); }
}

async function mutate(id: string, version: number, suffix: string, method: "PUT" | "POST", body?: unknown): Promise<ActionResult> {
  try {
    const campaign = await mercuryFetch<Campaign>(`/campaigns/${encodeURIComponent(id)}/${suffix}`, {
      method, headers: { "If-Match": `"${version}"` }, body: body === undefined ? undefined : JSON.stringify(body),
    });
    revalidatePath(`/campaigns/${id}`);
    return { ok: true, resourceVersion: campaign.version };
  } catch (error) { return actionError(error); }
}

export async function updateName(id: string, version: number, name: string) { return mutate(id, version, "name", "PUT", { name }); }
export async function updateBudget(id: string, version: number, configuredAmountMinor: string) { return mutate(id, version, "budget", "PUT", { configured_amount_minor: configuredAmountMinor }); }
export async function updatePlacement(id: string, version: number, placementCode: string) { return mutate(id, version, "placement", "PUT", { placement_code: placementCode }); }
export async function updateTargeting(id: string, version: number, countries: string[]) { return mutate(id, version, "targeting", "PUT", { countries }); }
export async function transitionCampaign(id: string, version: number, transition: "activate" | "pause" | "resume" | "end") { return mutate(id, version, transition, "POST"); }
