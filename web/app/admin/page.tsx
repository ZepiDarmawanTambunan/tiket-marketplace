"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { ReportSummary } from "@/lib/types";
import { formatRupiah } from "@/lib/format";
import { Button, Card, ErrorBox, PageHeader, StatCardsSkeleton } from "@/components/ui";

export default function AdminDashboard() {
  const [report, setReport] = useState<ReportSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<ReportSummary>("/api/admin/reports/summary")
      .then(setReport)
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  }, []);

  const cards = report
    ? [
        { label: "GMV (30 hari)", value: formatRupiah(report.gmv_cents) },
        { label: "Komisi platform", value: formatRupiah(report.commission_cents) },
        { label: "Fee gateway terkumpul", value: formatRupiah(report.gateway_fee_collected_cents) },
        { label: "Order lunas", value: String(report.orders_paid) },
        { label: "Tiket terjual", value: String(report.tickets_sold) },
        { label: "Dana refund", value: formatRupiah(report.refunded_cents) },
      ]
    : [];

  return (
    <div>
      <PageHeader
        title="Dashboard Admin"
        subtitle={report ? `Periode ${report.from} s/d ${report.to}` : "Ringkasan platform"}
        actions={
          <Link href="/admin/settings">
            <Button variant="secondary">Pengaturan</Button>
          </Link>
        }
      />
      <ErrorBox message={error} />
      {loading ? (
        <StatCardsSkeleton />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {cards.map((c) => (
            <Card key={c.label}>
              <p className="text-xs text-zinc-500">{c.label}</p>
              <p className="mt-1 text-2xl font-bold text-zinc-900">{c.value}</p>
            </Card>
          ))}
        </div>
      )}

      <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {[
          { href: "/admin/events", t: "Moderasi Event", d: "Setujui atau tolak event yang diajukan seller." },
          { href: "/admin/refunds", t: "Refund", d: "Tinjau pengajuan refund dari pembeli." },
          { href: "/admin/payouts", t: "Payout", d: "Catat pencairan dana ke seller." },
        ].map((l) => (
          <Link key={l.href} href={l.href}>
            <Card className="transition hover:shadow-md">
              <p className="font-semibold text-zinc-900">{l.t}</p>
              <p className="mt-1 text-sm text-zinc-500">{l.d}</p>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  );
}
