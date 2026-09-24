import type { HTMLAttributes } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva("inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold", {
  variants: {
    tone: {
      healthy: "border-emerald-200 bg-emerald-50 text-emerald-800",
      degraded: "border-amber-200 bg-amber-50 text-amber-900",
      unavailable: "border-red-200 bg-red-50 text-red-800",
    },
  },
  defaultVariants: { tone: "healthy" },
});

export function Badge({ className, tone, ...props }: HTMLAttributes<HTMLSpanElement> & VariantProps<typeof badgeVariants>) {
  return <span className={cn(badgeVariants({ tone }), className)} {...props} />;
}
