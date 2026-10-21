"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch, errMsg } from "@/lib/api";
import type { EventItem } from "@/lib/types";
import { formatRupiah } from "@/lib/format";
import { Button, Card, ErrorBox, PageHeader, StatCardsSkeleton } from "@/components/ui";

export default function SellerDashboard() {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<{ events: EventItem[] }>("/api/events?mine=true&per_page=100")
      .then((d) => setEvents(d.events))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  }, []);

  const ticketsSold = events.reduce(
    (s, e) => s + (e.ticket_types ?? []).reduce((a, t) => a + t.sold, 0),
    0
  );
  const grossEst = events.reduce(
    (s, e) => s + (e.ticket_types ?? []).reduce((a, t) => a + t.sold * t.price_cents, 0),
    0
  );
  const published = events.filter((e) => e.status === "published").length;
  const pending = events.filter((e) => e.status === "pending_review").length;

  const stats = [
    { label: "Total Event", value: String(events.length) },
    { label: "Event Tayang", value: String(published) },
    { label: "Menunggu Review", value: String(pending) },
    { label: "Tiket Terjual", value: String(ticketsSold) },
    { label: "Estimasi Omzet Kotor", value: formatRupiah(grossEst) },
  ];

  return (
    <div>
      <PageHeader
        title="Dashboard Seller"
        subtitle="Ringkasan penjualan tiket Anda"
        actions={
          <Link href="/seller/events/new">
            <Button>+ Buat Event</Button>
          </Link>
        }
      />
      <ErrorBox message={error} />
      {loading ? (
        <StatCardsSkeleton />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          {stats.map((s) => (
            <Card key={s.label}>
              <p className="text-xs text-zinc-500">{s.label}</p>
              <p className="mt-1 text-2xl font-bold text-zinc-900">{s.value}</p>
            </Card>
          ))}
        </div>
      )}
      <div className="mt-6 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
        <p className="font-medium">Alur publikasi event</p>
        <p className="mt-1">
          Event baru berstatus <strong>Draft</strong> → ajukan untuk direview →
          admin menyetujui → <strong>Tayang</strong>. Event yang tayang tidak bisa
          diubah lagi.
        </p>
      </div>
    </div>
  );
}
