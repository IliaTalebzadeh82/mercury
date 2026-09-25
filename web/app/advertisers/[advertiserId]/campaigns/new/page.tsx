import { FeatureUnavailable } from "@/components/feature-unavailable";
import { CampaignForm } from "@/features/campaigns/campaign-form";
import { listPlacements } from "@/features/campaigns/api";

export const dynamic = "force-dynamic";

export default async function NewCampaignPage({ params }: { params: Promise<{ advertiserId: string }> }) {
  const { advertiserId } = await params;
  let placements;
  try {
    placements = await listPlacements();
  } catch {
    return <FeatureUnavailable message="Campaign creation options could not be loaded." />;
  }
  return <main className="space-y-7"><header><p className="eyebrow">Campaign creation</p><h1 className="mt-2 text-3xl font-semibold">Create complete draft</h1><p className="mt-2 text-slate-600">A draft is complete and valid but not enabled.</p></header><CampaignForm advertiserId={advertiserId} placements={placements} /></main>;
}
