import type { ReactNode } from "react";
import { Activity } from "lucide-react";

export function ApplicationShell({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl items-center gap-3 px-6 py-4">
          <span className="flex size-9 items-center justify-center rounded-md bg-slate-950 text-white">
            <Activity aria-hidden="true" className="size-5" />
          </span>
          <div>
            <p className="font-semibold tracking-tight text-slate-950">Mercury</p>
            <p className="text-xs text-slate-500">Operations console</p>
          </div>
        </div>
      </header>
      <div className="mx-auto max-w-6xl px-6 py-10">{children}</div>
    </div>
  );
}
