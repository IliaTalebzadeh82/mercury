import { FeatureUnavailable } from "@/components/feature-unavailable";
import { CampaignDetail } from "@/features/campaigns/campaign-detail";
import { getBudget, getCampaign, listPlacements } from "@/features/campaigns/api";

export const dynamic = "force-dynamic";

export default async function CampaignPage({ params }: { params: Promise<{ campaignId: string }> }) {
  const { campaignId } = await params;
  let campaign;
  let placements;
  let budget;
  try {
    [campaign, placements, budget] = await Promise.all([getCampaign(campaignId), listPlacements(), getBudget(campaignId)]);
  } catch {
    return <FeatureUnavailable message="The campaign could not be loaded. It may not exist, or the backend may be unavailable." />;
  }
  return <main className="space-y-7"><header><p className="eyebrow">Campaign detail</p><h1 className="mt-2 text-3xl font-semibold">{campaign.name}</h1><p className="mt-2 text-sm text-slate-500">{campaign.id}</p></header><CampaignDetail campaign={campaign} placements={placements} budget={budget} /></main>;
}
