"use client";
import type { ReactNode } from "react";

import Link from "next/link";
import { usePathname } from "next/navigation";
import ProtectedRoute from "@/components/ProtectedRoute";

const LINKS = [
  { href: "/seller", label: "Ringkasan" },
  { href: "/seller/events", label: "Event Saya" },
  { href: "/seller/sales", label: "Penjualan" },
  { href: "/seller/promos", label: "Kode Promo" },
  { href: "/seller/payouts", label: "Payout" },
  { href: "/seller/scanner", label: "Scanner Tiket" },
];

export default function SellerLayout({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  return (
    <ProtectedRoute roles={["seller", "super_admin"]}>
      <div className="mx-auto max-w-7xl px-4 py-6">
        <nav className="mb-6 flex gap-1 no-scrollbar overflow-x-auto border-b border-zinc-200 pb-2">
          {LINKS.map((l) => {
            const active =
              l.href === "/seller" ? pathname === "/seller" : pathname.startsWith(l.href);
            return (
              <Link
                key={l.href}
                href={l.href}
                className={`whitespace-nowrap rounded-lg px-3 py-2 text-sm font-medium ${
                  active
                    ? "bg-orange-100 text-orange-700"
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
