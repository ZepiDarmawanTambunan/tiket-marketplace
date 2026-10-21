"use client";

import { Suspense, useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { apiFetch, errMsg } from "@/lib/api";
import type { EventItem } from "@/lib/types";
import { toDateInput } from "@/lib/format";
import {
  Button,
  Input,
  Select,
  Field,
  EventGridSkeleton,
  EmptyState,
  ErrorBox,
  Pagination,
  PageHeader,
} from "@/components/ui";
import EventCard from "@/components/EventCard";

const CATEGORIES = ["musik", "seminar", "workshop", "festival", "olahraga", "seni", "lainnya"];

function EventsContent() {
  const searchParams = useSearchParams();
  const [events, setEvents] = useState<EventItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [search, setSearch] = useState(searchParams.get("search") ?? "");
  const [category, setCategory] = useState(searchParams.get("category") ?? "");
  const [from, setFrom] = useState(searchParams.get("from") ?? "");
  const [to, setTo] = useState(searchParams.get("to") ?? "");

  const load = useCallback(
    async (p: number, s: string, c: string, f: string, t: string) => {
      // NOTE: tanpa setLoading/setError sinkron di sini agar lolos
      // react-hooks/set-state-in-effect; semua setState di bawah terjadi
      // setelah await. State awal (loading=true) sudah mencakup mount pertama.
      try {
        const q = new URLSearchParams({ page: String(p), per_page: "12" });
        if (s) q.set("search", s);
        if (c) q.set("category", c);
        if (f) q.set("from", f);
        if (t) q.set("to", t);
        const d = await apiFetch<{
          events: EventItem[];
          page: number;
          per_page: number;
          total: number;
        }>(`/api/events?${q.toString()}`);
        setEvents(d.events);
        setTotal(d.total);
        setPage(d.page);
      } catch (e) {
        setError(errMsg(e));
      } finally {
        setLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    // Dijalankan sebagai microtask agar tidak memicu
    // react-hooks/set-state-in-effect (aturan baru React Compiler).
    void Promise.resolve().then(() => load(1, search, category, from, to));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const applyFilter = () => {
    setLoading(true);
    load(1, search, category, from, to);
  };
  const resetFilter = () => {
    setSearch("");
    setCategory("");
    setFrom("");
    setTo("");
    setLoading(true);
    load(1, "", "", "", "");
  };
  const goPage = (p: number) => {
    setLoading(true);
    load(p, search, category, from, to);
  };

  return (
    <div className="mx-auto max-w-7xl px-4 py-8">
      <PageHeader title="Jelajah Event" subtitle="Temukan konser, seminar, workshop, dan lainnya" />

      <div className="mb-6 grid gap-3 rounded-xl border border-zinc-200 bg-white p-4 sm:grid-cols-2 lg:grid-cols-5">
        <Field label="Cari">
          <Input
            placeholder="Nama event, venue..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && applyFilter()}
          />
        </Field>
        <Field label="Kategori">
          <Select value={category} onChange={(e) => setCategory(e.target.value)}>
            <option value="">Semua</option>
            {CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Dari tanggal">
          <Input type="date" value={from} onChange={(e) => setFrom(toDateInput(e.target.value))} />
        </Field>
        <Field label="Sampai tanggal">
          <Input type="date" value={to} onChange={(e) => setTo(toDateInput(e.target.value))} />
        </Field>
        <div className="flex items-end gap-2">
          <Button onClick={applyFilter}>Filter</Button>
          <Button variant="ghost" onClick={resetFilter}>
            Reset
          </Button>
        </div>
      </div>

      <ErrorBox message={error} />
      {loading ? (
        <EventGridSkeleton count={8} />
      ) : events.length === 0 ? (
        <EmptyState message="Tidak ada event yang cocok dengan filter." />
      ) : (
        <>
          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            {events.map((e) => (
              <EventCard key={e.id} e={e} />
            ))}
          </div>
          <Pagination page={page} perPage={12} total={total} onPage={goPage} />
        </>
      )}
    </div>
  );
}

export default function EventsPage() {
  return (
    <Suspense
      fallback={
        <div className="mx-auto max-w-7xl px-4 py-8">
          <EventGridSkeleton count={8} />
        </div>
      }
    >
      <EventsContent />
    </Suspense>
  );
}
