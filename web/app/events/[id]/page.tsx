"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { apiFetch, errMsg } from "@/lib/api";
import type { EventItem, TicketType } from "@/lib/types";
import { formatDateTime, formatRupiah } from "@/lib/format";
import { useAuthStore } from "@/store/auth";
import { useCheckoutStore } from "@/store/checkout";
import ReviewSection from "@/components/ReviewSection";
import {
  Button,
  Card,
  DetailSkeleton,
  EmptyState,
  ErrorBox,
  PageHeader,
} from "@/components/ui";

export default function EventDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const { token } = useAuthStore();
  const startCheckout = useCheckoutStore((s) => s.start);

  const [event, setEvent] = useState<EventItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [qty, setQty] = useState<Record<number, number>>({});

  useEffect(() => {
    apiFetch<{ event: EventItem }>(`/api/events/${id}`)
      .then((d) => setEvent(d.event))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  }, [id]);

  const setTicketQty = (t: TicketType, v: number) => {
    const remaining = Math.max(0, t.quota - t.sold);
    const clamped = Math.max(0, Math.min(remaining, v));
    setQty((q) => ({ ...q, [t.id]: clamped }));
  };

  const totalQty = Object.values(qty).reduce((a, b) => a + b, 0);
  const subtotal =
    event?.ticket_types?.reduce(
      (s, t) => s + t.price_cents * (qty[t.id] ?? 0),
      0
    ) ?? 0;

  const handleBuy = () => {
    if (!event || totalQty === 0) return;
    if (!token) {
      router.push(`/login?next=/events/${id}`);
      return;
    }
    startCheckout(
      event.id,
      event.title,
      (event.ticket_types ?? [])
        .filter((t) => (qty[t.id] ?? 0) > 0)
        .map((t) => ({
          ticketTypeId: t.id,
          name: t.name,
          price: t.price_cents,
          qty: qty[t.id],
        }))
    );
    router.push("/checkout");
  };

  if (loading) return <DetailSkeleton />;
  if (error || !event)
    return (
      <div className="mx-auto max-w-5xl px-4 py-8">
        <ErrorBox message={error || "Event tidak ditemukan"} />
        <Link href="/events" className="text-sm text-orange-600 hover:underline">
          &larr; Kembali ke daftar event
        </Link>
      </div>
    );

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <PageHeader title={event.title} subtitle={`${event.category} • ${event.venue}`} />

      {event.banner_url && (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={event.banner_url}
          alt={event.title}
          className="mb-6 h-64 w-full rounded-xl object-cover"
        />
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          <Card>
            <h2 className="mb-2 font-semibold text-zinc-900">Tentang Event</h2>
            <p className="whitespace-pre-line text-sm text-zinc-600">
              {event.description || "Tidak ada deskripsi."}
            </p>
            <dl className="mt-4 space-y-2 text-sm">
              <div className="flex gap-2">
                <dt className="w-28 shrink-0 text-zinc-500">Lokasi</dt>
                <dd className="text-zinc-900">
                  {event.venue}
                  {event.address && ` — ${event.address}`}
                  {event.maps_url && (
                    <a
                      href={event.maps_url}
                      target="_blank"
                      rel="noreferrer"
                      className="ml-2 text-orange-600 hover:underline"
                    >
                      Lihat peta
                    </a>
                  )}
                </dd>
              </div>
              <div className="flex gap-2">
                <dt className="w-28 shrink-0 text-zinc-500">Waktu</dt>
                <dd className="text-zinc-900">
                  {formatDateTime(event.starts_at)} — {formatDateTime(event.ends_at)}
                </dd>
              </div>
              {event.refund_policy && (
                <div className="flex gap-2">
                  <dt className="w-28 shrink-0 text-zinc-500">Kebijakan refund</dt>
                  <dd className="text-zinc-900">{event.refund_policy}</dd>
                </div>
              )}
            </dl>
          </Card>

          <ReviewSection
            eventId={event.id}
            ratingAvg={event.rating_avg}
            ratingCount={event.rating_count}
          />
        </div>

        <div>
          <Card>
            <h2 className="mb-3 font-semibold text-zinc-900">Pilih Tiket</h2>
            {(event.ticket_types ?? []).length === 0 ? (
              <EmptyState message="Belum ada tipe tiket untuk event ini." />
            ) : (
              <div className="space-y-3">
                {(event.ticket_types ?? []).map((t) => {
                  const remaining = Math.max(0, t.quota - t.sold);
                  const soldOut = remaining === 0;
                  return (
                    <div
                      key={t.id}
                      className="rounded-lg border border-zinc-200 p-3"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div>
                          <p className="font-medium text-zinc-900">{t.name}</p>
                          <p className="text-sm font-bold text-orange-700">
                            {formatRupiah(t.price_cents)}
                          </p>
                          <p className="text-xs text-zinc-500">
                            {soldOut ? "Habis terjual" : `Sisa ${remaining} tiket`}
                          </p>
                        </div>
                        {!soldOut && (
                          <div className="flex items-center gap-2">
                            <button
                              className="flex h-8 w-8 items-center justify-center rounded-lg bg-zinc-100 text-lg hover:bg-zinc-200"
                              onClick={() => setTicketQty(t, (qty[t.id] ?? 0) - 1)}
                              aria-label="kurangi"
                            >
                              −
                            </button>
                            <span className="w-6 text-center text-sm font-semibold">
                              {qty[t.id] ?? 0}
                            </span>
                            <button
                              className="flex h-8 w-8 items-center justify-center rounded-lg bg-zinc-100 text-lg hover:bg-zinc-200"
                              onClick={() => setTicketQty(t, (qty[t.id] ?? 0) + 1)}
                              aria-label="tambah"
                            >
                              +
                            </button>
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            <div className="mt-4 border-t border-zinc-200 pt-4">
              <div className="mb-3 flex items-center justify-between text-sm">
                <span className="text-zinc-500">
                  Subtotal ({totalQty} tiket)
                </span>
                <span className="font-bold text-zinc-900">
                  {formatRupiah(subtotal)}
                </span>
              </div>
              <Button
                className="w-full"
                disabled={totalQty === 0}
                onClick={handleBuy}
              >
                Beli Sekarang
              </Button>
              <p className="mt-2 text-center text-xs text-zinc-500">
                Biaya layanan & komisi dihitung transparan saat checkout.
              </p>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}
