"use client";

import { useEffect, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { Refund } from "@/lib/types";
import { REFUND_STATUS_LABEL } from "@/lib/types";
import { formatDateTime, formatRupiah } from "@/lib/format";
import {
  Badge,
  Button,
  EmptyState,
  ErrorBox,
  PageHeader,
  TableSkeleton,
  SuccessBox,
  Table,
  Td,
  TextArea,
} from "@/components/ui";

export default function AdminRefundsPage() {
  const [refunds, setRefunds] = useState<Refund[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [rejectFor, setRejectFor] = useState<number | null>(null);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState<number | null>(null);

  const load = () => {
    apiFetch<{ refunds: Refund[] }>("/api/admin/refunds?status=requested&per_page=100")
      .then((d) => setRefunds(d.refunds))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    void Promise.resolve().then(load);
  }, []);

  const decide = async (r: Refund, approve: boolean) => {
    setBusy(r.id);
    setError("");
    setSuccess("");
    try {
      if (approve) {
        if (!confirm(`Setujui refund ${r.order_code ?? `#${r.order_id}`}?`)) return;
        // Approve bisa memicu refund otomatis via Midtrans; respons {refund}
        // memuat midtrans_refund_id bila via Midtrans, atau note "manual".
        const d = await apiFetch<{ refund: Refund }>(
          `/api/admin/refunds/${r.id}/approve`,
          { method: "POST" }
        );
        const ref = d.refund;
        if (ref?.midtrans_refund_id) {
          setSuccess(
            `Refund ${ref.order_code ?? ""} disetujui — dana dikembalikan VIA MIDTRANS (ref: ${ref.midtrans_refund_id}).`
          );
        } else {
          setSuccess(
            `Refund ${r.order_code ?? ""} disetujui — dicatat MANUAL` +
              (ref?.note ? ` — ${ref.note}` : " — kembalikan dana manual ke pembeli.")
          );
        }
      } else {
        await apiFetch(`/api/admin/refunds/${r.id}/reject`, {
          method: "POST",
          body: JSON.stringify({ reason }),
        });
        setSuccess(`Refund ${r.order_code ?? ""} ditolak — order kembali lunas.`);
        setRejectFor(null);
        setReason("");
      }
      setLoading(true);
      load();
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setBusy(null);
    }
  };

  return (
    <div>
      <PageHeader
        title="Pengajuan Refund"
        subtitle="Setujui / tolak refund. Persetujuan bisa memicu refund otomatis via Midtrans, atau dicatat manual."
      />
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      {loading ? (
        <TableSkeleton />
      ) : refunds.length === 0 ? (
        <EmptyState message="Tidak ada pengajuan refund yang menunggu." />
      ) : (
        <Table head={["Order", "Jumlah", "Alasan", "Status", "Diajukan", "Aksi"]}>
          {refunds.map((r) => (
            <tr key={r.id}>
              <Td className="font-mono text-xs">{r.order_code ?? `#${r.order_id}`}</Td>
              <Td className="font-medium">{formatRupiah(r.amount_cents)}</Td>
              <Td className="max-w-[240px] truncate text-xs">{r.reason}</Td>
              <Td>
                <Badge status={r.status === "requested" ? "pending" : r.status} label={REFUND_STATUS_LABEL[r.status]} />
              </Td>
              <Td className="whitespace-nowrap text-xs">{formatDateTime(r.created_at)}</Td>
              <Td>
                {rejectFor === r.id ? (
                  <div className="min-w-[220px] space-y-2">
                    <TextArea
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      placeholder="Alasan penolakan..."
                    />
                    <div className="flex gap-1">
                      <Button
                        variant="danger"
                        disabled={busy === r.id || !reason.trim()}
                        onClick={() => decide(r, false)}
                      >
                        Tolak Refund
                      </Button>
                      <Button variant="ghost" onClick={() => setRejectFor(null)}>
                        Batal
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex gap-1">
                    <Button disabled={busy === r.id} onClick={() => decide(r, true)}>
                      Setujui
                    </Button>
                    <Button variant="danger" disabled={busy === r.id} onClick={() => setRejectFor(r.id)}>
                      Tolak
                    </Button>
                  </div>
                )}
              </Td>
            </tr>
          ))}
        </Table>
      )}
    </div>
  );
}
