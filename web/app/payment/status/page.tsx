"use client";

import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { apiFetch, errMsg } from "@/lib/api";
import type { Order } from "@/lib/types";
import { ORDER_STATUS_LABEL } from "@/lib/types";
import { formatRupiah, formatDateTime } from "@/lib/format";
import ProtectedRoute from "@/components/ProtectedRoute";
import {
  Button,
  Card,
  Badge,
  ErrorBox,
  Spinner,
  PageHeader,
} from "@/components/ui";

const TERMINAL: Order["status"][] = ["paid", "cancelled", "failed", "expired", "refunded"];

function StatusContent() {
  const searchParams = useSearchParams();
  const orderId = searchParams.get("order_id");
  const isDev = searchParams.get("dev") === "1";

  const [order, setOrder] = useState<Order | null>(null);
  const [error, setError] = useState("");
  const timer = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchOrder = useCallback(async () => {
    if (!orderId) return;
    try {
      const { order: o } = await apiFetch<{ order: Order }>(
        `/api/orders/${orderId}`
      );
      setOrder(o);
      if (TERMINAL.includes(o.status) && timer.current) {
        clearInterval(timer.current);
        timer.current = null;
      }
    } catch (e) {
      setError(errMsg(e));
      if (timer.current) {
        clearInterval(timer.current);
        timer.current = null;
      }
    }
  }, [orderId]);

  useEffect(() => {
    if (!orderId) return;
    void Promise.resolve().then(() => fetchOrder());
    timer.current = setInterval(() => {
      void fetchOrder();
    }, 3000);
    return () => {
      if (timer.current) {
        clearInterval(timer.current);
        timer.current = null;
      }
    };
  }, [orderId, fetchOrder]);

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <PageHeader title="Status Pembayaran" />

      {isDev && (
        <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
          Mode development: Midtrans belum terhubung (snap token dummy). Order
          dibuat dengan status <strong>pending</strong> — status akan berubah
          menjadi <strong>paid</strong> setelah webhook Midtrans diterima.
        </div>
      )}

      <ErrorBox message={error} />

      {!orderId ? (
        <ErrorBox message="Parameter order_id tidak ditemukan di URL." />
      ) : !order && !error ? (
        <Spinner />
      ) : order ? (
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-zinc-500">Kode Order</p>
              <p className="font-mono font-semibold text-zinc-900">{order.order_code}</p>
            </div>
            <Badge status={order.status} label={ORDER_STATUS_LABEL[order.status]} />
          </div>

          <div className="mt-4 space-y-1 text-sm">
            <div className="flex justify-between">
              <span className="text-zinc-500">Total bayar</span>
              <span className="font-bold text-zinc-900">{formatRupiah(order.gross_cents)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-zinc-500">Waktu order</span>
              <span className="text-zinc-800">{formatDateTime(order.created_at)}</span>
            </div>
          </div>

          {order.status === "pending" && (
            <p className="mt-4 text-center text-xs text-zinc-500">
              Menunggu konfirmasi pembayaran... halaman ini diperbarui otomatis.
            </p>
          )}

          {order.status === "paid" && (
            <div className="mt-6 text-center">
              <p className="text-green-700 font-medium">🎉 Pembayaran berhasil!</p>
              <Link href="/tickets" className="mt-3 inline-block">
                <Button>Lihat Tiket Saya</Button>
              </Link>
            </div>
          )}

          {(order.status === "cancelled" ||
            order.status === "failed" ||
            order.status === "expired") && (
            <div className="mt-6 text-center">
              <Link href="/events">
                <Button variant="secondary">Cari Event Lain</Button>
              </Link>
            </div>
          )}
        </Card>
      ) : null}

      <div className="mt-6 text-center">
        <Link href="/orders" className="text-sm text-orange-600 hover:underline">
          Lihat semua riwayat transaksi
        </Link>
      </div>
    </div>
  );
}

export default function PaymentStatusPage() {
  return (
    <ProtectedRoute>
      <Suspense fallback={<Spinner />}>
        <StatusContent />
      </Suspense>
    </ProtectedRoute>
  );
}
