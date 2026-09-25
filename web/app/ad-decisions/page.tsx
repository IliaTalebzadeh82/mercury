import { DecisionLab } from "@/features/ad-decisions/decision-lab";
import { listPlacements } from "@/features/campaigns/api";

export const dynamic = "force-dynamic";

export default async function AdDecisionsPage() {
  let placements;
  try {
    placements = await listPlacements();
  } catch {
    return <section role="alert" className="rounded-lg border border-amber-200 bg-amber-50 p-6"><h1 className="text-xl font-semibold text-amber-950">Decision engine unavailable</h1><p className="mt-2 text-sm text-amber-900">Placements could not be loaded from the Mercury backend. Try again after backend readiness is restored.</p></section>;
  }
  return <main className="space-y-8"><header className="max-w-3xl"><p className="eyebrow">Developer tooling</p><h1 className="mt-2 text-3xl font-semibold">Decision Lab</h1><p className="mt-2 text-slate-600">Submit a transient advertising opportunity and inspect the normalized deterministic decision.</p></header><DecisionLab placements={placements} /></main>;
}
