"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import type { Placement } from "./api";
import { createCampaign } from "./actions";

const schema = z.object({
  name: z.string().trim().min(1).max(200),
  placementCode: z.string().min(1),
  amountMinor: z.string().regex(/^\+?[0-9]+$/).refine((value) => BigInt(value) > BigInt(0)),
  currency: z.enum(["EUR", "GBP", "USD"]),
  countries: z.string().refine((value) => value.split(",").some((country) => /^[A-Za-z]{2}$/.test(country.trim()))),
});
type FormValues = z.infer<typeof schema>;

export function CampaignForm({ advertiserId, placements }: { advertiserId: string; placements: Placement[] }) {
  const router = useRouter();
  const [message, setMessage] = useState<string>();
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "", placementCode: placements[0]?.code ?? "", amountMinor: "", currency: "EUR", countries: "DE" },
  });

  async function submit(values: FormValues) {
    setMessage(undefined);
    const countries = values.countries.split(",").map((country) => country.trim()).filter(Boolean);
    const result = await createCampaign({ advertiserId, ...values, countries });
    if (!result.ok) { setMessage(result.message); return; }
    router.push(`/campaigns/${result.resourceId}`);
  }

  return (
    <form className="grid gap-5 rounded-lg border border-slate-200 bg-white p-6 sm:grid-cols-2" onSubmit={form.handleSubmit(submit)}>
      <Field label="Campaign name" error={form.formState.errors.name?.message}><input aria-label="Campaign name" className="field" {...form.register("name")} /></Field>
      <Field label="Placement" error={form.formState.errors.placementCode?.message}>
        <select aria-label="Placement" className="field" {...form.register("placementCode")}>{placements.map((placement) => <option key={placement.code} value={placement.code}>{placement.display_name}</option>)}</select>
      </Field>
      <Field label="Lifetime budget (minor units)" error={form.formState.errors.amountMinor?.message}><input aria-label="Lifetime budget (minor units)" inputMode="numeric" className="field" {...form.register("amountMinor")} /></Field>
      <Field label="Currency" error={form.formState.errors.currency?.message}><select aria-label="Currency" className="field" {...form.register("currency")}><option>EUR</option><option>GBP</option><option>USD</option></select></Field>
      <Field label="Country codes" error={form.formState.errors.countries?.message}><input aria-label="Country codes" className="field" placeholder="DE, FR" {...form.register("countries")} /></Field>
      <div className="sm:col-span-2">
        {message && <p role="alert" className="mb-3 text-sm text-red-700">{message}</p>}
        <Button type="submit" disabled={form.formState.isSubmitting}>{form.formState.isSubmitting ? "Creating…" : "Create complete draft"}</Button>
      </div>
    </form>
  );
}

function Field({ label, error, children }: { label: string; error?: string; children: React.ReactNode }) {
  return <label className="text-sm font-medium text-slate-800">{label}{children}{error && <span className="mt-1 block text-red-700">Invalid value</span>}</label>;
}
