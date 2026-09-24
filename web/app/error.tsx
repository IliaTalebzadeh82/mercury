"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";

export default function ErrorPage({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    console.error("Mercury frontend render failed", error.digest ?? "no-digest");
  }, [error]);

  return (
    <main className="rounded-lg border border-red-200 bg-red-50 p-6" role="alert">
      <p className="eyebrow text-red-700">Interface error</p>
      <h1 className="mt-2 text-2xl font-semibold text-red-950">The operations view could not render</h1>
      <p className="mt-2 max-w-2xl text-sm leading-6 text-red-800">
        This is an interface failure, distinct from a backend readiness degradation. Retry the view; if
        the problem persists, inspect the frontend logs.
      </p>
      <Button className="mt-5" onClick={reset}>Retry</Button>
    </main>
  );
}
