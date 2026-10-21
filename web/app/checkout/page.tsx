"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { apiFetch, errMsg } from "@/lib/api";
import type { Order, PromoValidateResult } from "@/lib/types";
import { formatRupiah } from "@/lib/format";
import { useCheckoutStore } from "@/store/checkout";
import { loadSnap } from "@/lib/snap";
import ProtectedRoute from "@/components/ProtectedRoute";
import {
  Button,
  Card,
  ErrorBox,
  PageHeader,
  Spinner,
} from "@/components/ui";

const COMMISSION_PCT_INFO = 3;

function Row({
  label,
  value,
  bold,
  info,
}: {
  label: string;
  value: string;
  bold?: boolean;
  info?: string;
}) {
  return (
    <div className="flex items-start justify-between gap-4 py-1.5 text-sm">
      <span className="text-zinc-500">
        {label}
        {info && <span className="block text-xs text-zinc-400">{info}</span>}
      </span>
      <span className={bold ? "font-bold text-zinc-900" : "text-zinc-800"}>
        {value}
      </span>
    </div>
  );
}

function CheckoutContent() {
  const router = useRouter();
  const { eventId, eventTitle, items, subtotal, clear } = useCheckoutStore();
  const [order, setOrder] = useState<Order | null>(null);
  const [processing, setProcessing] = useState(false);
  const [paying, setPaying] = useState(false);
  const [error, setError] = useState("");

  // Fase 2: kode promo
  const [promoInput, setPromoInput] = useState("");
  const [promoCode, setPromoCode] = useState<string | null>(null);
  const [promoDiscount, setPromoDiscount] = useState(0);
  const [promoChecking, setPromoChecking] = useState(false);
  const [promoError, setPromoError] = useState("");

  const preSubtotal = subtotal();
  // Estimasi info sebelum order dibuat (angka pasti dari server setelah order).
  // Komisi dihitung dari subtotal setelah dikurangi diskon.
  const estCommission = Math.round(
    ((preSubtotal - promoDiscount) * COMMISSION_PCT_INFO) / 100
  );

  const applyPromo = async () => {
    const code = promoInput.trim().toUpperCase();
    if (!code || !eventId) return;
    setPromoChecking(true);
    setPromoError("");
    try {
      const res = await apiFetch<PromoValidateResult>("/api/promos/validate", {
        method: "POST",
        body: JSON.stringify({
          code,
          event_id: eventId,
          items: items.map((i) => ({ ticket_type_id: i.ticketTypeId, qty: i.qty })),
        }),
      });
      if (res.valid) {
        setPromoCode(code);
        setPromoDiscount(res.discount_cents);
      } else {
        setPromoCode(null);
        setPromoDiscount(0);
        setPromoError(res.message || "Kode promo tidak valid.");
      }
    } catch (e) {
      setPromoCode(null);
      setPromoDiscount(0);
      setPromoError(errMsg(e));
    } finally {
      setPromoChecking(false);
    }
  };

  const removePromo = () => {
    setPromoCode(null);
    setPromoDiscount(0);
    setPromoInput("");
    setPromoError("");
  };

  const handlePay = async () => {
    if (!eventId || items.length === 0) return;
    setProcessing(true);
    setError("");
    try {
      // 1. Buat order -> dapat snap token + rincian biaya pasti dari server
      const res = await apiFetch<{
        order: Order;
        snap_token: string;
        client_key: string;
        redirect_url?: string | null;
        payment_note?: string;
      }>("/api/orders", {
        method: "POST",
        body: JSON.stringify({
          event_id: eventId,
          items: items.map((i) => ({ ticket_type_id: i.ticketTypeId, qty: i.qty })),
          ...(promoCode ? { promo_code: promoCode } : {}),
        }),
      });
      setOrder(res.order);

      // Mode development: snap token dummy -> langsung ke halaman status
      if (res.snap_token === "DUMMY") {
        clear();
        router.push(`/payment/status?order_id=${res.order.id}&dev=1`);
        return;
      }

      // 2. Ambil client key + environment, load snap.js
      setPaying(true);
      const cfg = await apiFetch<{ client_key: string; environment: string }>(
        "/api/config/midtrans-client-key"
      );
      await loadSnap(cfg.client_key, cfg.environment);

      // 3. Buka popup pembayaran Midtrans
      window.snap?.pay(res.snap_token, {
        onSuccess: () => {
          clear();
          router.push(`/payment/status?order_id=${res.order.id}`);
        },
        onPending: () => {
          clear();
          router.push(`/payment/status?order_id=${res.order.id}`);
        },
        onError: () => {
          setPaying(false);
          setError("Pembayaran gagal. Silakan coba lagi.");
        },
        onClose: () => {
          setPaying(false);
          setError(
            "Jendela pembayaran ditutup. Order tetap tersimpan — Anda bisa membayar lagi dari halaman Riwayat."
          );
        },
      });
    } catch (e) {
      setError(errMsg(e));
      // Wajib reset paying: bila snap.js gagal dimuat, spinner tidak boleh
      // menggantung selamanya — tombol bayar harus kembali agar bisa retry.
      setPaying(false);
    } finally {
      setProcessing(false);
    }
  };

  if (!eventId || items.length === 0) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-16 text-center">
        <h1 className="text-xl font-bold text-zinc-900">Checkout kosong</h1>
        <p className="mt-2 text-sm text-zinc-500">
          Pilih tiket dulu dari halaman event.
        </p>
        <Link
          href="/events"
          className="mt-4 inline-block rounded-lg bg-orange-600 px-4 py-2 text-sm font-medium text-white"
        >
          Jelajah Event
        </Link>
      </div>
    );
  }

  const shownSubtotal = order ? order.subtotal_cents : preSubtotal;
  const shownCommission = order ? order.commission_cents : estCommission;
  const shownFee = order ? order.gateway_fee_cents : null;
  const shownGross = order ? order.gross_cents : null;

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <PageHeader title="Checkout" subtitle={eventTitle} />

      <Card>
        <h2 className="mb-3 font-semibold text-zinc-900">Ringkasan Pesanan</h2>
        <div className="divide-y divide-zinc-100">
          {items.map((i) => (
            <div key={i.ticketTypeId} className="flex justify-between py-2 text-sm">
              <span className="text-zinc-700">
                {i.name} <span className="text-zinc-400">× {i.qty}</span>
              </span>
              <span className="text-zinc-900">{formatRupiah(i.price * i.qty)}</span>
            </div>
          ))}
        </div>

        <div className="mt-4 border-t border-zinc-200 pt-3">
          {/* Kode promo (fase 2) */}
          <div className="mb-3 border-b border-zinc-200 pb-3">
            {promoCode ? (
              <div className="flex items-center justify-between rounded-lg border border-green-200 bg-green-50 px-3 py-2">
                <span className="text-sm">
                  <span className="font-mono font-bold text-green-800">
                    {promoCode}
                  </span>{" "}
                  <span className="text-green-700">
                    aktif — hemat {formatRupiah(order?.discount_cents ?? promoDiscount)}
                  </span>
                </span>
                <button
                  type="button"
                  onClick={removePromo}
                  disabled={processing || paying}
                  className="text-xs font-medium text-zinc-500 hover:text-red-600"
                >
                  Hapus
                </button>
              </div>
            ) : (
              <div>
                <div className="flex gap-2">
                  <input
                    value={promoInput}
                    onChange={(e) => setPromoInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        e.preventDefault();
                        applyPromo();
                      }
                    }}
                    placeholder="Kode promo (mis. DISKON20)"
                    disabled={processing || paying || promoChecking}
                    className="w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm uppercase placeholder:normal-case outline-none focus:border-orange-500 focus:ring-2 focus:ring-orange-100"
                  />
                  <Button
                    variant="secondary"
                    onClick={applyPromo}
                    disabled={!promoInput.trim() || processing || paying || promoChecking}
                  >
                    {promoChecking ? "..." : "Pakai"}
                  </Button>
                </div>
                {promoError && (
                  <p className="mt-1.5 text-xs text-red-600">{promoError}</p>
                )}
              </div>
            )}
          </div>

          <Row label="Subtotal" value={formatRupiah(shownSubtotal)} />
          {(order?.discount_cents ?? promoDiscount) > 0 && (
            <Row
              label={`Diskon promo (${order?.promo_code ?? promoCode ?? ""})`}
              value={`− ${formatRupiah(order?.discount_cents ?? promoDiscount)}`}
            />
          )}
          <Row
            label="Komisi platform"
            value={order ? formatRupiah(shownCommission) : `± ${formatRupiah(shownCommission)}`}
            info={`Sudah termasuk dalam harga • default ${COMMISSION_PCT_INFO}% dari subtotal`}
          />
          <Row
            label="Biaya layanan"
            value={shownFee != null ? formatRupiah(shownFee) : "dihitung saat bayar"}
            info="Fee gateway Midtrans, ditanggung pembeli — tampil transparan"
          />
          <div className="mt-2 border-t border-zinc-200 pt-2">
            <Row
              label="Total bayar"
              value={
                shownGross != null
                  ? formatRupiah(shownGross)
                  : formatRupiah(preSubtotal - promoDiscount)
              }
              bold
            />
          </div>
        </div>

        <ErrorBox message={error} />

        {order && error && (
          <p className="mt-3 text-center text-sm">
            Order <span className="font-mono font-medium">{order.order_code}</span>{" "}
            sudah dibuat.{" "}
            <Link
              href={`/payment/status?order_id=${order.id}`}
              className="font-medium text-orange-700 underline"
            >
              Lihat Status Pembayaran
            </Link>
          </p>
        )}

        <div className="mt-4">
          {processing || paying ? (
            <Spinner label={paying ? "Membuka pembayaran..." : "Membuat pesanan..."} />
          ) : (
            <Button className="w-full py-3 text-base" onClick={handlePay}>
              Bayar {order ? formatRupiah(order.gross_cents) : ""}
            </Button>
          )}
          <p className="mt-3 text-center text-xs text-zinc-500">
            Pembayaran aman via Midtrans (sandbox). Dengan membayar Anda menyetujui
            kebijakan refund penyelenggara event.
          </p>
        </div>
      </Card>
    </div>
  );
}

export default function CheckoutPage() {
  return (
    <ProtectedRoute>
      <CheckoutContent />
    </ProtectedRoute>
  );
}
