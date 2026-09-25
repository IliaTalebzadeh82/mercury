"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { createAdvertiser } from "./actions";

const schema = z.object({ name: z.string().trim().min(1).max(200) });
type FormValues = z.infer<typeof schema>;

export function AdvertiserForm() {
  const router = useRouter();
  const [message, setMessage] = useState<string>();
  const form = useForm<FormValues>({ resolver: zodResolver(schema), defaultValues: { name: "" } });

  async function submit(values: FormValues) {
    setMessage(undefined);
    const result = await createAdvertiser(values);
    if (!result.ok) { setMessage(result.message); return; }
    router.push(`/advertisers/${result.resourceId}/campaigns`);
  }

  return (
    <form className="space-y-4 rounded-lg border border-slate-200 bg-white p-5" onSubmit={form.handleSubmit(submit)}>
      <div>
        <label className="text-sm font-medium" htmlFor="advertiser-name">Advertiser name</label>
        <input id="advertiser-name" className="mt-1 block w-full rounded-md border border-slate-300 px-3 py-2" {...form.register("name")} />
        {form.formState.errors.name && <p className="mt-1 text-sm text-red-700">Enter a name of at most 200 characters.</p>}
      </div>
      {message && <p role="alert" className="text-sm text-red-700">{message}</p>}
      <Button type="submit" disabled={form.formState.isSubmitting}>{form.formState.isSubmitting ? "Creating…" : "Create advertiser"}</Button>
    </form>
  );
}
