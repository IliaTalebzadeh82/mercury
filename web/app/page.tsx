import { SystemHealth } from "@/features/system-health/system-health";
import { getSystemHealth } from "@/features/system-health/api";

export const dynamic = "force-dynamic";

export default async function HomePage() {
  const health = await getSystemHealth();

  return (
    <main className="space-y-8">
      <header className="max-w-3xl space-y-3">
        <p className="eyebrow">Engineering foundation</p>
        <h1 className="text-3xl font-semibold tracking-tight text-slate-950 sm:text-4xl">
          Platform overview
        </h1>
        <p className="text-base leading-7 text-slate-600">
          Mercury exposes dependency health explicitly. A degraded backend remains visible here rather
          than taking down the operator interface.
        </p>
      </header>
      <SystemHealth health={health} />
    </main>
  );
}
