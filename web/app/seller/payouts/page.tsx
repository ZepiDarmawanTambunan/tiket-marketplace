"use client";

import { useEffect, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { Payout } from "@/lib/types";
import { formatDateTime, formatRupiah } from "@/lib/format";
import { useAuthStore } from "@/store/auth";
import {
  Badge,
  EmptyState,
  ErrorBox,
  PageHeader,
  TableSkeleton,
  Table,
  Td,
} from "@/components/ui";

/**
 * ASUMSI (perlu konfirmasi worker backend): seller boleh membaca payout
 * miliknya via GET /api/seller/payouts.
 * Bila backend menolak (403), halaman menampilkan pesan yang jelas.
 */
export default function SellerPayoutsPage() {
  const user = useAuthStore((s) => s.user);
  const [payouts, setPayouts] = useState<Payout[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [forbidden, setForbidden] = useState(false);

  useEffect(() => {
    if (!user) return;
    apiFetch<{ payouts: Payout[] }>("/api/seller/payouts?per_page=50")
      .then((d) => setPayouts(d.payouts))
      .catch((e) => {
        if (e instanceof Error && "status" in e && (e as { status: number }).status === 403) {
          setForbidden(true);
        } else {
          setError(errMsg(e));
        }
      })
      .finally(() => setLoading(false));
  }, [user]);

  const pendingTotal = payouts
    .filter((p) => p.status === "pending")
    .reduce((s, p) => s + p.amount_cents, 0);

  return (
    <div>
      <PageHeader
        title="Payout"
        subtitle="Dana hasil penjualan yang ditransfer ke Anda"
      />
      <ErrorBox message={error} />

      {loading ? (
        <TableSkeleton />
      ) : forbidden ? (
        <EmptyState message="Data payout dikelola oleh admin. Hubungi admin untuk informasi pencairan dana Anda." />
      ) : payouts.length === 0 ? (
        <EmptyState message="Belum ada payout." />
      ) : (
        <>
          <div className="mb-4 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-800">
            Menunggu pencairan: <strong>{formatRupiah(pendingTotal)}</strong>
          </div>
          <Table head={["Tanggal", "Event", "Jumlah", "Status", "Bukti"]}>
            {payouts.map((p) => (
              <tr key={p.id}>
                <Td className="whitespace-nowrap text-xs">
                  {formatDateTime(p.created_at)}
                </Td>
                <Td>{p.event_title ?? (p.event_id ? `Event #${p.event_id}` : "-")}</Td>
                <Td className="font-medium">{formatRupiah(p.amount_cents)}</Td>
                <Td>
                  <Badge status={p.status} label={p.status === "paid" ? "Sudah dibayar" : p.status === "pending" ? "Menunggu" : "Dibatalkan"} />
                </Td>
                <Td>
                  {p.proof_url ? (
                    <a href={p.proof_url} target="_blank" rel="noreferrer" className="text-orange-600 hover:underline">
                      Lihat bukti
                    </a>
                  ) : (
                    <span className="text-zinc-400">-</span>
                  )}
                </Td>
              </tr>
            ))}
          </Table>
        </>
      )}
    </div>
  );
}
