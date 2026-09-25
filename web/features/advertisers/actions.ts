"use server";

import { randomUUID } from "node:crypto";
import { revalidatePath } from "next/cache";
import type { ActionResult } from "@/lib/action-result";
import { actionError } from "@/lib/api/action-result";
import { mercuryFetch } from "@/lib/api/mercury";
import type { Advertiser } from "./api";

export async function createAdvertiser(input: { name: string }): Promise<ActionResult> {
  try {
    const advertiser = await mercuryFetch<Advertiser>("/advertisers", {
      method: "POST",
      headers: { "Idempotency-Key": randomUUID() },
      body: JSON.stringify({ name: input.name }),
    });
    revalidatePath("/advertisers");
    return { ok: true, resourceId: advertiser.id };
  } catch (error) {
    return actionError(error);
  }
}
