import Link from "next/link";
import { FeatureUnavailable } from "@/components/feature-unavailable";
import { AdvertiserForm } from "@/features/advertisers/advertiser-form";
import { listAdvertisers } from "@/features/advertisers/api";

export const dynamic = "force-dynamic";

export default async function AdvertisersPage() {
  let advertisers;
  try {
    advertisers = await listAdvertisers();
  } catch {
    return <FeatureUnavailable message="Advertisers could not be loaded. Platform health remains available from the main navigation." />;
  }
  return (
    <main className="space-y-8">
      <header><p className="eyebrow">Advertiser console</p><h1 className="mt-2 text-3xl font-semibold">Advertisers</h1><p className="mt-2 text-slate-600">Select an advertiser or create one for the campaign workflow.</p></header>
      <AdvertiserForm />
      <section className="overflow-hidden rounded-lg border border-slate-200 bg-white">
        <h2 className="border-b border-slate-200 px-5 py-4 font-semibold">Available advertisers</h2>
        {advertisers.length === 0 ? <p className="p-5 text-sm text-slate-600">No advertisers exist yet.</p> : <ul className="divide-y divide-slate-200">{advertisers.map((advertiser) => <li key={advertiser.id}><Link className="block px-5 py-4 hover:bg-slate-50" href={`/advertisers/${advertiser.id}/campaigns`}><span className="font-medium">{advertiser.name}</span><span className="ml-3 text-xs text-slate-500">{advertiser.id}</span></Link></li>)}</ul>}
      </section>
    </main>
  );
}
