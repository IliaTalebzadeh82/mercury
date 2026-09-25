import Link from "next/link";
import { FeatureUnavailable } from "@/components/feature-unavailable";
import { listCampaigns } from "@/features/campaigns/api";

export const dynamic = "force-dynamic";

export default async function CampaignsPage({ params }: { params: Promise<{ advertiserId: string }> }) {
  const { advertiserId } = await params;
  let campaigns;
  try {
    campaigns = await listCampaigns(advertiserId);
  } catch {
    return <FeatureUnavailable message="Campaigns could not be loaded from the Mercury backend." />;
  }
  return <main className="space-y-7">
    <header className="flex items-end justify-between gap-4"><div><p className="eyebrow">Advertiser console</p><h1 className="mt-2 text-3xl font-semibold">Campaigns</h1><p className="mt-2 text-sm text-slate-500">Advertiser {advertiserId}</p></div><Link className="rounded-md bg-slate-950 px-4 py-2 text-sm font-medium text-white" href={`/advertisers/${advertiserId}/campaigns/new`}>Create campaign</Link></header>
    {campaigns.length === 0 ? <section className="rounded-lg border border-dashed border-slate-300 bg-white p-8 text-center text-slate-600">No campaigns exist for this advertiser.</section> : <div className="overflow-hidden rounded-lg border border-slate-200 bg-white"><table className="w-full text-left text-sm"><thead className="bg-slate-50 text-slate-600"><tr><th className="px-5 py-3">Name</th><th>State</th><th>Placement</th><th>Configured budget</th></tr></thead><tbody className="divide-y divide-slate-200">{campaigns.map((campaign) => <tr key={campaign.id}><td className="px-5 py-4"><Link className="font-medium text-slate-950 underline" href={`/campaigns/${campaign.id}`}>{campaign.name}</Link></td><td>{campaign.state}</td><td>{campaign.placement_code}</td><td>{campaign.budget.configured_amount_minor} {campaign.budget.currency}</td></tr>)}</tbody></table></div>}
  </main>;
}
