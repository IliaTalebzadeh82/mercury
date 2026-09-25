import "server-only";

import { mercuryFetch } from "@/lib/api/mercury";

export type Advertiser = { id: string; name: string; created_at: string };

export async function listAdvertisers(): Promise<Advertiser[]> {
  const response = await mercuryFetch<{ advertisers: Advertiser[] }>("/advertisers");
  return response.advertisers;
}
