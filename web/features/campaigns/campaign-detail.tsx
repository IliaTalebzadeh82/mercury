"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import type { ActionResult } from "@/lib/action-result";
import type { Campaign, Placement } from "./api";
import { transitionCampaign, updateBudget, updateName, updatePlacement, updateTargeting } from "./actions";

export function CampaignDetail({ campaign, placements }: { campaign: Campaign; placements: Placement[] }) {
  const router = useRouter();
  const [result, setResult] = useState<ActionResult>();
  const [pending, setPending] = useState(false);
  const [awaitingVersion, setAwaitingVersion] = useState<number>();

  async function execute(command: () => Promise<ActionResult>) {
    setPending(true); setResult(undefined);
    const next = await command();
    if (next.ok && next.resourceVersion !== undefined && next.resourceVersion > campaign.version) {
      setAwaitingVersion(next.resourceVersion);
    }
    setPending(false); setResult(next);
    if (next.ok) router.refresh();
  }

  function form(handler: (data: FormData) => Promise<ActionResult>) {
    return async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      const data = new FormData(event.currentTarget);
      await execute(() => handler(data));
    };
  }

  const mutable = campaign.state !== "ENDED";
  const pausedOrDraft = campaign.state === "PAUSED" || campaign.state === "DRAFT";
  const busy = pending || (awaitingVersion !== undefined && campaign.version < awaitingVersion);

  return (
    <div className="space-y-6">
      {result?.stale && <div role="alert" className="rounded-md border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950">This campaign changed elsewhere. Reload before submitting another command. <Button className="ml-3" onClick={() => router.refresh()}>Reload current campaign</Button></div>}
      {result && !result.ok && !result.stale && <p role="alert" className="rounded-md bg-red-50 p-4 text-sm text-red-800">{result.message}</p>}

      <section className="grid gap-px overflow-hidden rounded-lg border border-slate-200 bg-slate-200 sm:grid-cols-4">
        <Stat label="State" value={campaign.state} />
        <Stat label="Version" value={String(campaign.version)} />
        <Stat label="Placement" value={campaign.placement_code} />
        <Stat label="Configured budget" value={`${campaign.budget.configured_amount_minor} ${campaign.budget.currency} minor units`} />
      </section>

      <div className="grid gap-5 lg:grid-cols-2">
        <form className="panel" onSubmit={form((data) => updateName(campaign.id, campaign.version, String(data.get("name"))))}>
          <h2 className="font-semibold">Name</h2><input aria-label="Edit campaign name" name="name" className="field" defaultValue={campaign.name} disabled={!mutable || busy} required />
          <Button type="submit" disabled={!mutable || busy}>Save name</Button>
        </form>

        <form className="panel" onSubmit={form((data) => updateBudget(campaign.id, campaign.version, String(data.get("budget"))))}>
          <h2 className="font-semibold">Lifetime configured budget</h2><input aria-label="Edit configured budget" name="budget" className="field" defaultValue={campaign.budget.configured_amount_minor} disabled={!mutable || busy} required />
          <p className="text-xs text-slate-500">Currency is immutable: {campaign.budget.currency}. Active campaigns may only keep or increase this amount.</p>
          <Button type="submit" disabled={!mutable || busy}>Save budget</Button>
        </form>

        <form className="panel" onSubmit={form((data) => updatePlacement(campaign.id, campaign.version, String(data.get("placement"))))}>
          <h2 className="font-semibold">Placement</h2><select aria-label="Edit placement" name="placement" className="field" defaultValue={campaign.placement_code} disabled={!pausedOrDraft || busy}>{placements.map((placement) => <option key={placement.code} value={placement.code}>{placement.display_name}</option>)}</select>
          {!pausedOrDraft && campaign.state !== "ENDED" && <p className="text-xs text-slate-500">Pause the campaign before changing placement.</p>}
          <Button type="submit" disabled={!pausedOrDraft || busy}>Save placement</Button>
        </form>

        <form className="panel" onSubmit={form((data) => updateTargeting(campaign.id, campaign.version, String(data.get("countries")).split(",").map((value) => value.trim()).filter(Boolean)))}>
          <h2 className="font-semibold">Country targeting</h2><input aria-label="Edit country targeting" name="countries" className="field" defaultValue={campaign.targeting.countries.join(", ")} disabled={!pausedOrDraft || busy} required />
          {!pausedOrDraft && campaign.state !== "ENDED" && <p className="text-xs text-slate-500">Pause the campaign before changing targeting.</p>}
          <Button type="submit" disabled={!pausedOrDraft || busy}>Save targeting</Button>
        </form>
      </div>

      <section className="panel">
        <h2 className="font-semibold">Lifecycle</h2>
        <div className="flex flex-wrap gap-3">
          {campaign.state === "DRAFT" && <Button disabled={busy} onClick={() => execute(() => transitionCampaign(campaign.id, campaign.version, "activate"))}>Activate</Button>}
          {campaign.state === "ACTIVE" && <Button disabled={busy} onClick={() => execute(() => transitionCampaign(campaign.id, campaign.version, "pause"))}>Pause</Button>}
          {campaign.state === "PAUSED" && <Button disabled={busy} onClick={() => execute(() => transitionCampaign(campaign.id, campaign.version, "resume"))}>Resume</Button>}
          {campaign.state !== "ENDED" && <Button disabled={busy} className="bg-red-700 hover:bg-red-600" onClick={() => execute(() => transitionCampaign(campaign.id, campaign.version, "end"))}>End campaign</Button>}
          {campaign.state === "ENDED" && <p className="text-sm text-slate-600">This campaign is terminal and cannot be edited or reactivated.</p>}
        </div>
      </section>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) { return <div className="bg-white p-5"><p className="text-xs uppercase tracking-wide text-slate-500">{label}</p><p className="mt-2 font-medium text-slate-950">{value}</p></div>; }
