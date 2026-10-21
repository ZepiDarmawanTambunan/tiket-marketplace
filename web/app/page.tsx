"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { EventItem } from "@/lib/types";
import { Button, ErrorBox, EventGridSkeleton } from "@/components/ui";
import EventCard from "@/components/EventCard";

export default function HomePage() {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    apiFetch<{ events: EventItem[] }>("/api/events?per_page=8")
      .then((d) => setEvents(d.events))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  }, []);

  const retry = () => {
    setLoading(true);
    setError("");
    load();
  };

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div>
      {/* Hero */}
      <section className="bg-gradient-to-br from-orange-600 via-orange-500 to-purple-700 text-white">
        <div className="mx-auto max-w-7xl px-4 py-16 sm:py-24">
          <h1 className="max-w-2xl text-4xl font-bold leading-tight sm:text-5xl">
            Temukan event seru, beli tiket dalam hitungan detik.
          </h1>
          <p className="mt-4 max-w-xl text-orange-50">
            Konser, seminar, workshop, dan festival — semua tiket resmi dari
            penyelenggara terverifikasi. Bayar aman dengan Midtrans.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Link
              href="/events"
              className="rounded-lg bg-white px-6 py-3 text-sm font-semibold text-orange-700 hover:bg-orange-50"
            >
              Jelajah Event
            </Link>
            <Link
              href="/register"
              className="rounded-lg border border-white/40 px-6 py-3 text-sm font-semibold text-white hover:bg-white/10"
            >
              Jual Tiket Eventmu
            </Link>
          </div>
        </div>
      </section>

      {/* Event pilihan */}
      <section className="mx-auto max-w-7xl px-4 py-12">
        <div className="mb-6 flex items-center justify-between">
          <h2 className="text-2xl font-bold text-zinc-900">Event Pilihan</h2>
          <Link
            href="/events"
            className="text-sm font-medium text-orange-600 hover:underline"
          >
            Lihat semua &rarr;
          </Link>
        </div>
        {loading ? (
          <EventGridSkeleton />
        ) : error ? (
          <div>
            <ErrorBox message={`Gagal memuat event: ${error}`} />
            <Button variant="secondary" onClick={retry}>
              Coba Lagi
            </Button>
          </div>
        ) : events.length === 0 ? (
          <p className="text-sm text-zinc-500">
            Belum ada event yang tayang.
          </p>
        ) : (
          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            {events.map((e) => (
              <EventCard key={e.id} e={e} />
            ))}
          </div>
        )}
      </section>

      {/* Keunggulan */}
      <section className="border-t border-zinc-200 bg-white">
        <div className="mx-auto grid max-w-7xl gap-6 px-4 py-12 sm:grid-cols-3">
          {[
            {
              t: "Tiket resmi & aman",
              d: "Tiket diterbitkan sistem dengan kode unik dan QR terverifikasi anti-palsu.",
            },
            {
              t: "Bayar transparan",
              d: "Biaya layanan tampil jelas saat checkout — tanpa biaya tersembunyi.",
            },
            {
              t: "Untuk penyelenggara",
              d: "Buat event, kelola tiket, scan kehadiran, dan terima payout — semua dalam satu dasbor.",
            },
          ].map((f) => (
            <div key={f.t}>
              <h3 className="font-semibold text-zinc-900">{f.t}</h3>
              <p className="mt-1 text-sm text-zinc-500">{f.d}</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
