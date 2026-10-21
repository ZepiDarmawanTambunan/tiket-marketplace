"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch, errMsg } from "@/lib/api";
import type { EventItem } from "@/lib/types";
import { EVENT_STATUS_LABEL } from "@/lib/types";
import { formatDate, formatRupiah } from "@/lib/format";
import {
  Badge,
  Button,
  Card,
  EmptyState,
  ErrorBox,
  SuccessBox,
  PageHeader,
  TableSkeleton,
  Table,
  Td,
} from "@/components/ui";

export default function SellerEventsPage() {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [busyId, setBusyId] = useState<number | null>(null);

  const load = () => {
    apiFetch<{ events: EventItem[] }>("/api/events?mine=true&per_page=100")
      .then((d) => setEvents(d.events))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const doAction = async (id: number, action: "submit" | "delete") => {
    if (action === "delete" && !confirm("Hapus event ini?")) return;
    setBusyId(id);
    setError("");
    setSuccess("");
    try {
      if (action === "submit") {
        await apiFetch(`/api/events/${id}/submit`, { method: "POST" });
        setSuccess("Event diajukan untuk review admin.");
      } else {
        await apiFetch(`/api/events/${id}`, { method: "DELETE" });
        setSuccess("Event dihapus.");
      }
      setLoading(true);
      load();
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div>
      <PageHeader
        title="Event Saya"
        subtitle="Kelola event dan pantau status moderasi"
        actions={
          <Link href="/seller/events/new">
            <Button>+ Buat Event</Button>
          </Link>
        }
      />
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      {loading ? (
        <TableSkeleton />
      ) : events.length === 0 ? (
        <EmptyState message="Belum ada event. Buat event pertama Anda!" />
      ) : (
        <Table head={["Event", "Tanggal", "Tiket Terjual", "Status", "Aksi"]}>
          {events.map((e) => {
            const sold = (e.ticket_types ?? []).reduce((s, t) => s + t.sold, 0);
            const revenue = (e.ticket_types ?? []).reduce(
              (s, t) => s + t.sold * t.price_cents,
              0
            );
            const editable = e.status === "draft" || e.status === "rejected";
            return (
              <tr key={e.id}>
                <Td>
                  <p className="font-medium">{e.title}</p>
                  <p className="text-xs text-zinc-500">
                    {e.category} &middot; {e.venue}
                  </p>
                </Td>
                <Td className="whitespace-nowrap">{formatDate(e.starts_at)}</Td>
                <Td>
                  {sold} tiket
                  <span className="block text-xs text-zinc-500">
                    {formatRupiah(revenue)}
                  </span>
                </Td>
                <Td>
                  <Badge status={e.status} label={EVENT_STATUS_LABEL[e.status]} />
                </Td>
                <Td>
                  <div className="flex flex-wrap gap-1">
                    {editable && (
                      <Link href={`/seller/events/${e.id}/edit`}>
                        <Button variant="secondary">Edit</Button>
                      </Link>
                    )}
                    {(e.status === "draft" || e.status === "rejected") && (
                      <Button
                        variant="secondary"
                        disabled={busyId === e.id}
                        onClick={() => doAction(e.id, "submit")}
                      >
                        Ajukan Review
                      </Button>
                    )}
                    {e.status === "published" && (
                      <Link href={`/events/${e.id}`} target="_blank">
                        <Button variant="ghost">Lihat</Button>
                      </Link>
                    )}
                    {editable && (
                      <Button
                        variant="danger"
                        disabled={busyId === e.id}
                        onClick={() => doAction(e.id, "delete")}
                      >
                        Hapus
                      </Button>
                    )}
                  </div>
                </Td>
              </tr>
            );
          })}
        </Table>
      )}

      <Card className="mt-6 text-sm text-zinc-600">
        <p className="font-medium text-zinc-900">Status moderasi</p>
        <ul className="mt-2 list-disc space-y-1 pl-5">
          <li><strong>Draft</strong> — masih bisa diedit & dihapus.</li>
          <li><strong>Menunggu Review</strong> — sedang diperiksa admin.</li>
          <li><strong>Tayang</strong> — bisa dibeli publik; tidak bisa diedit.</li>
          <li><strong>Ditolak</strong> — perbaiki lalu ajukan ulang.</li>
        </ul>
      </Card>
    </div>
  );
}
