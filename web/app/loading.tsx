import { SystemHealthLoading } from "@/features/system-health/system-health";

export default function Loading() {
  return (
    <main aria-busy="true" aria-label="Loading platform status" className="space-y-8">
      <div className="space-y-3">
        <div className="h-3 w-32 animate-pulse rounded bg-slate-200" />
        <div className="h-10 w-72 animate-pulse rounded bg-slate-200" />
      </div>
      <SystemHealthLoading />
    </main>
  );
}
