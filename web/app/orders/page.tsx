"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch, errMsg } from "@/lib/api";
import type { Order } from "@/lib/types";
import { ORDER_STATUS_LABEL } from "@/lib/types";
import { formatDateTime, formatRupiah } from "@/lib/format";
import ProtectedRoute from "@/components/ProtectedRoute";
import {
  Badge,
  Button,
  Card,
  EmptyState,
  ErrorBox,
  SuccessBox,
  PageHeader,
  CardListSkeleton,
  TextArea,
} from "@/components/ui";

function OrdersContent() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [refundFor, setRefundFor] = useState<number | null>(null);
  const [reason, setReason] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const load = () => {
    apiFetch<{ orders: Order[] }>("/api/orders?per_page=50")
      .then((d) => setOrders(d.orders))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const submitRefund = async (orderId: number) => {
    setSubmitting(true);
    setError("");
    setSuccess("");
    try {
      await apiFetch(`/api/orders/${orderId}/refund`, {
        method: "POST",
        body: JSON.stringify({ reason }),
      });
      setSuccess("Pengajuan refund terkirim. Menunggu persetujuan admin.");
      setRefundFor(null);
      setReason("");
      setLoading(true);
      load();
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <PageHeader title="Riwayat Transaksi" subtitle="Semua pembelian tiket Anda" />
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      {loading ? (
        <CardListSkeleton />
      ) : orders.length === 0 ? (
        <EmptyState message="Belum ada transaksi." />
      ) : (
        <div className="space-y-4">
          {orders.map((o) => (
            <Card key={o.id}>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p className="font-mono text-sm font-semibold text-zinc-900">
                    {o.order_code}
                  </p>
                  <p className="text-xs text-zinc-500">
                    {o.event_title ?? `Event #${o.event_id}`} &middot;{" "}
                    {formatDateTime(o.created_at)}
                  </p>
                </div>
                <Badge status={o.status} label={ORDER_STATUS_LABEL[o.status]} />
              </div>

              <div className="mt-3 space-y-1 text-sm">
                {(o.items ?? []).map((it, idx) => (
                  <div key={idx} className="flex justify-between text-zinc-600">
                    <span>
                      {it.ticket_type_name ?? `Tiket #${it.ticket_type_id}`} × {it.qty}
                    </span>
                    <span>{formatRupiah(it.price_cents * it.qty)}</span>
                  </div>
                ))}
                {(o.discount_cents ?? 0) > 0 && (
                  <div className="flex justify-between text-sm text-green-700">
                    <span>Diskon promo ({o.promo_code})</span>
                    <span>− {formatRupiah(o.discount_cents)}</span>
                  </div>
                )}
                <div className="flex justify-between border-t border-zinc-100 pt-2 font-semibold text-zinc-900">
                  <span>Total bayar</span>
                  <span>{formatRupiah(o.gross_cents)}</span>
                </div>
                <p className="text-xs text-zinc-400">
                  Termasuk komisi platform {formatRupiah(o.commission_cents)} & biaya
                  layanan {formatRupiah(o.gateway_fee_cents)}
                </p>
              </div>

              <div className="mt-3 flex flex-wrap gap-2">
                {o.status === "pending" && (
                  <Link href={`/payment/status?order_id=${o.id}`}>
                    <Button variant="secondary">Cek Status / Bayar</Button>
                  </Link>
                )}
                {o.status === "paid" && refundFor !== o.id && (
                  <Button variant="ghost" onClick={() => setRefundFor(o.id)}>
                    Ajukan Refund
                  </Button>
                )}
              </div>

              {refundFor === o.id && (
                <div className="mt-3 rounded-lg bg-zinc-50 p-3">
                  <p className="mb-2 text-sm font-medium text-zinc-700">
                    Alasan pengajuan refund
                  </p>
                  <TextArea
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                    placeholder="Contoh: event dibatalkan / tidak bisa hadir..."
                  />
                  <div className="mt-2 flex gap-2">
                    <Button
                      disabled={submitting || !reason.trim()}
                      onClick={() => submitRefund(o.id)}
                    >
                      {submitting ? "Mengirim..." : "Kirim Pengajuan"}
                    </Button>
                    <Button variant="ghost" onClick={() => setRefundFor(null)}>
                      Batal
                    </Button>
                  </div>
                </div>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

export default function OrdersPage() {
  return (
    <ProtectedRoute>
      <OrdersContent />
    </ProtectedRoute>
  );
}
