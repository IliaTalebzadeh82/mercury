import type { Metadata } from "next";
import type { ReactNode } from "react";
import { ApplicationShell } from "@/components/layout/application-shell";
import "./globals.css";

export const metadata: Metadata = {
  title: "Mercury Operations",
  description: "Operational view of the Mercury advertising platform",
};

export default function RootLayout({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <html lang="en">
      <body>
        <ApplicationShell>{children}</ApplicationShell>
      </body>
    </html>
  );
}
