"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import type { Placement } from "@/features/campaigns/api";
import { createAdDecision, explainAdDecision } from "./actions";
import type { Decision, Diagnostic, OpportunityInput } from "./api";

const canonicalUUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
const schema = z.object({
  opportunityId: z.string().regex(canonicalUUID, "Use a canonical lowercase UUID").refine((value) => value !== "00000000-0000-0000-0000-000000000000", "Use a non-nil UUID"),
  placement: z.string().min(1, "Select a placement"),
  country: z.string().trim().regex(/^[A-Za-z]{2}$/, "Use a two-letter country code"),
});
type FormValues = z.infer<typeof schema>;

export function DecisionLab({ placements }: { placements: Placement[] }) {
  const [decision, setDecision] = useState<Decision>();
  const [submitted, setSubmitted] = useState<OpportunityInput>();
  const [diagnostic, setDiagnostic] = useState<Diagnostic>();
  const [message, setMessage] = useState<string>();
  const [unavailable, setUnavailable] = useState(false);
  const [explaining, setExplaining] = useState(false);
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { opportunityId: "", placement: placements[0]?.code ?? "", country: "US" },
  });

  function generateOpportunity() {
    form.setValue("opportunityId", crypto.randomUUID(), { shouldValidate: true });
  }

  async function submit(values: FormValues) {
    setMessage(undefined);
    setUnavailable(false);
    setDiagnostic(undefined);
    const input = { ...values, country: values.country.trim() };
    const result = await createAdDecision(input);
    if (!result.ok) {
      setDecision(undefined);
      setMessage(result.message);
      setUnavailable(result.unavailable);
      for (const [field, text] of Object.entries(result.fields)) {
        if (field === "opportunity_id") form.setError("opportunityId", { message: text });
        if (field === "placement") form.setError("placement", { message: text });
        if (field === "country") form.setError("country", { message: text });
      }
      return;
    }
    setSubmitted(input);
    setDecision(result.decision);
  }

  async function explain() {
    if (!submitted) return;
    setExplaining(true);
    setMessage(undefined);
    const result = await explainAdDecision(submitted);
    setExplaining(false);
    if (!result.ok) {
      setMessage(result.message);
      setUnavailable(result.unavailable);
      return;
    }
    setDiagnostic(result.diagnostic);
  }

  return (
    <div className="space-y-6">
      <form className="grid gap-5 rounded-lg border border-slate-200 bg-white p-6 sm:grid-cols-2" onSubmit={form.handleSubmit(submit)}>
        <Field label="Opportunity ID" error={form.formState.errors.opportunityId?.message} wide>
          <div className="flex gap-2">
            <input aria-label="Opportunity ID" className="field" {...form.register("opportunityId")} />
            <Button type="button" className="mt-[0.35rem] bg-slate-600 hover:bg-slate-500" onClick={generateOpportunity}>Generate</Button>
          </div>
        </Field>
        <Field label="Placement" error={form.formState.errors.placement?.message}>
          <select aria-label="Placement" className="field" {...form.register("placement")}>
            {placements.map((placement) => <option key={placement.code} value={placement.code}>{placement.display_name}</option>)}
          </select>
        </Field>
        <Field label="Country" error={form.formState.errors.country?.message}>
          <input aria-label="Country" className="field" maxLength={2} {...form.register("country")} />
        </Field>
        <div className="sm:col-span-2">
          {message && <p role="alert" className={`mb-3 text-sm ${unavailable ? "text-amber-800" : "text-red-700"}`}>{message}</p>}
          <Button type="submit" disabled={form.formState.isSubmitting}>{form.formState.isSubmitting ? "Deciding…" : "Request decision"}</Button>
        </div>
      </form>

      {decision && (
        <section aria-live="polite" className="rounded-lg border border-slate-200 bg-white p-6">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div><p className="eyebrow">Decision result</p><h2 className="mt-1 text-2xl font-semibold">{decision.outcome}</h2></div>
            <Button type="button" className="bg-slate-600 hover:bg-slate-500" disabled={explaining} onClick={explain}>{explaining ? "Explaining…" : "Explain decision"}</Button>
          </div>
          <dl className="mt-5 grid gap-4 text-sm sm:grid-cols-2">
            <Value label="Decision ID" value={decision.decision_id} />
            <Value label="Opportunity ID" value={decision.opportunity_id} />
            <Value label="Normalized placement" value={decision.placement} />
            <Value label="Normalized country" value={decision.country} />
            {decision.selection ? <><Value label="Campaign ID" value={decision.selection.campaign_id} /><Value label="Campaign version" value={String(decision.selection.campaign_version)} /></> : <Value label="Selection" value="No eligible campaign" />}
          </dl>
        </section>
      )}

      {diagnostic && (
        <section className="rounded-lg border border-slate-200 bg-white p-6">
          <h2 className="font-semibold">Diagnostic explanation</h2>
          {diagnostic.truncated && <p className="mt-2 text-sm text-amber-800">Showing a bounded sample; additional campaigns were omitted.</p>}
          <ul className="mt-4 divide-y divide-slate-200">
            {diagnostic.explanations.map((item) => <li className="py-3 text-sm" key={item.campaign_id}><span className="font-mono">{item.campaign_id}</span><span className="ml-3 font-medium">{item.eligible ? "eligible" : item.reasons.join(", ")}</span></li>)}
          </ul>
        </section>
      )}
    </div>
  );
}

function Field({ label, error, wide, children }: { label: string; error?: string; wide?: boolean; children: React.ReactNode }) {
  return <label className={`text-sm font-medium text-slate-800 ${wide ? "sm:col-span-2" : ""}`}>{label}{children}{error && <span className="mt-1 block text-red-700">{error}</span>}</label>;
}

function Value({ label, value }: { label: string; value: string }) {
  return <div><dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt><dd className="mt-1 break-all font-mono text-slate-900">{value}</dd></div>;
}
