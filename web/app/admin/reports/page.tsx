"use client";

import { useEffect, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { ReportSummary } from "@/lib/types";
import { formatRupiah, toDateInput } from "@/lib/format";
import {
  Button,
  Card,
  ErrorBox,
  Field,
  Input,
  PageHeader,
  TableSkeleton,
} from "@/components/ui";

export default function AdminReportsPage() {
  const [report, setReport] = useState<ReportSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  const load = async (f: string, t: string) => {
    try {
      const q = new URLSearchParams();
      if (f) q.set("from", f);
      if (t) q.set("to", t);
      const d = await apiFetch<ReportSummary>(
        `/api/admin/reports/summary?${q.toString()}`
      );
      setReport(d);
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void Promise.resolve().then(() => load("", ""));
  }, []);

  const cards = report
    ? [
        { label: "GMV", value: formatRupiah(report.gmv_cents), hint: "Total nilai order lunas" },
        { label: "Komisi platform", value: formatRupiah(report.commission_cents), hint: "Pendapatan platform" },
        { label: "Fee gateway terkumpul", value: formatRupiah(report.gateway_fee_collected_cents), hint: "Biaya layanan dari pembeli" },
        { label: "Order lunas", value: String(report.orders_paid), hint: "Jumlah transaksi berhasil" },
        { label: "Tiket terjual", value: String(report.tickets_sold), hint: "Unit tiket" },
        { label: "Dana refund", value: formatRupiah(report.refunded_cents), hint: "Total dikembalikan" },
      ]
    : [];

  return (
    <div>
      <PageHeader
        title="Laporan"
        subtitle={report ? `Periode ${report.from} s/d ${report.to}` : "Ringkasan angka platform"}
      />

      <div className="mb-6 grid max-w-xl gap-3 rounded-xl border border-zinc-200 bg-white p-4 sm:grid-cols-3">
        <Field label="Dari">
          <Input type="date" value={from} onChange={(e) => setFrom(toDateInput(e.target.value))} />
        </Field>
        <Field label="Sampai">
          <Input type="date" value={to} onChange={(e) => setTo(toDateInput(e.target.value))} />
        </Field>
        <div className="flex items-end">
          <Button
            onClick={() => {
              setLoading(true);
              load(from, to);
            }}
          >
            Tampilkan
          </Button>
        </div>
      </div>

      <ErrorBox message={error} />

      {loading ? (
        <TableSkeleton />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {cards.map((c) => (
            <Card key={c.label}>
              <p className="text-xs text-zinc-500">{c.label}</p>
              <p className="mt-1 text-2xl font-bold text-zinc-900">{c.value}</p>
              <p className="mt-1 text-xs text-zinc-400">{c.hint}</p>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
