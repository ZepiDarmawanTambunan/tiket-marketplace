"use client";
import type { ReactNode } from "react";

import Link from "next/link";
import { usePathname } from "next/navigation";
import ProtectedRoute from "@/components/ProtectedRoute";

const LINKS = [
  { href: "/admin", label: "Dashboard" },
  { href: "/admin/events", label: "Moderasi Event" },
  { href: "/admin/transactions", label: "Transaksi" },
  { href: "/admin/refunds", label: "Refund" },
  { href: "/admin/promos", label: "Promo" },
  { href: "/admin/payouts", label: "Payout" },
  { href: "/admin/users", label: "Users" },
  { href: "/admin/sellers", label: "Seller" },
  { href: "/admin/reports", label: "Laporan" },
  { href: "/admin/audit-logs", label: "Audit Log" },
  { href: "/admin/settings", label: "Pengaturan" },
];

export default function AdminLayout({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  return (
    <ProtectedRoute roles={["super_admin"]}>
      <div className="mx-auto max-w-7xl px-4 py-6">
        <nav className="mb-6 flex gap-1 no-scrollbar overflow-x-auto border-b border-zinc-200 pb-2">
          {LINKS.map((l) => {
            const active =
              l.href === "/admin" ? pathname === "/admin" : pathname.startsWith(l.href);
            return (
              <Link
                key={l.href}
                href={l.href}
                className={`whitespace-nowrap rounded-lg px-3 py-2 text-sm font-medium ${
                  active
                    ? "bg-zinc-900 text-white"
                    : "text-zinc-600 hover:bg-zinc-100"
                }`}
              >
                {l.label}
              </Link>
            );
          })}
        </nav>
        {children}
      </div>
    </ProtectedRoute>
  );
}
