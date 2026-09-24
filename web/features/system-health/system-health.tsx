import { Badge } from "@/components/ui/badge";
import type { SystemHealthState } from "./api";

const labels = { healthy: "Healthy", degraded: "Degraded", misconfigured: "Misconfigured", unavailable: "Unavailable" } as const;

export function SystemHealth({ health }: { health: SystemHealthState }) {
  return (
    <section aria-labelledby="system-health-heading" className="overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
      <div className="flex flex-col justify-between gap-4 border-b border-slate-200 px-6 py-5 sm:flex-row sm:items-center">
        <div>
          <h2 id="system-health-heading" className="font-semibold text-slate-950">System health</h2>
          <p className="mt-1 text-sm text-slate-500">Live operational dependency status</p>
        </div>
        <Badge tone={health.kind === "unavailable" || health.kind === "misconfigured" ? "unavailable" : health.kind}>{labels[health.kind]}</Badge>
      </div>
      <dl className="grid gap-px bg-slate-200 sm:grid-cols-2">
        <div className="bg-white px-6 py-5">
          <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">Backend</dt>
          <dd className="mt-2 text-sm font-medium text-slate-900">{health.kind === "unavailable" || health.kind === "misconfigured" ? "Unknown" : "Reachable"}</dd>
        </div>
        <div className="bg-white px-6 py-5">
          <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">Readiness</dt>
          <dd className="mt-2 text-sm font-medium text-slate-900">{health.detail}</dd>
        </div>
      </dl>
    </section>
  );
}

export function SystemHealthLoading() {
  return <div aria-label="Loading system health" className="h-48 animate-pulse rounded-lg border border-slate-200 bg-white" />;
}
