"use client";

import { useEffect, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { EventItem, EventStatus } from "@/lib/types";
import { EVENT_STATUS_LABEL } from "@/lib/types";
import { formatDateTime } from "@/lib/format";
import {
  Badge,
  Button,
  EmptyState,
  ErrorBox,
  Field,
  PageHeader,
  Select,
  TableSkeleton,
  SuccessBox,
  Table,
  Td,
  TextArea,
} from "@/components/ui";

export default function AdminEventsPage() {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [statusFilter, setStatusFilter] = useState<EventStatus | "">("pending_review");
  const [rejectFor, setRejectFor] = useState<number | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [busy, setBusy] = useState<number | null>(null);

  const load = (status: EventStatus | "") => {
    setLoading(true);
    setError("");
    const qs = status ? `?status=${status}&per_page=100` : "?per_page=100";
    apiFetch<{ events: EventItem[] }>(`/api/admin/events${qs}`)
      .then((d) => setEvents(d.events ?? []))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    // Muat awal; pemuatan ulang saat filter berubah dipicu dari onChange Select.
    void Promise.resolve().then(() => load("pending_review"));
  }, []);

  const filtered = events; // sudah difilter server-side via ?status=

  const moderate = async (id: number, action: "approve" | "reject") => {
    setBusy(id);
    setError("");
    setSuccess("");
    try {
      if (action === "approve") {
        if (!confirm("Setujui event ini untuk tayang?")) return;
        await apiFetch(`/api/admin/events/${id}/approve`, { method: "POST" });
        setSuccess("Event disetujui dan tayang.");
      } else {
        await apiFetch(`/api/admin/events/${id}/reject`, {
          method: "POST",
          body: JSON.stringify({ reason: rejectReason }),
        });
        setSuccess("Event ditolak.");
        setRejectFor(null);
        setRejectReason("");
      }
      setLoading(true);
      load(statusFilter);
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setBusy(null);
    }
  };

  return (
    <div>
      <PageHeader
        title="Moderasi Event"
        subtitle="Event wajib disetujui admin sebelum tayang"
      />
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      <div className="mb-4 max-w-xs">
        <Field label="Filter status">
          <Select
            value={statusFilter}
            onChange={(e) => {
              const v = e.target.value as EventStatus | "";
              setStatusFilter(v);
              load(v);
            }}
          >
            <option value="">Semua</option>
            <option value="pending_review">Menunggu review</option>
            <option value="published">Tayang</option>
            <option value="rejected">Ditolak</option>
            <option value="draft">Draft</option>
            <option value="finished">Selesai</option>
          </Select>
        </Field>
      </div>

      {loading ? (
        <TableSkeleton />
      ) : filtered.length === 0 ? (
        <EmptyState message="Tidak ada event pada filter ini." />
      ) : (
        <Table head={["Event", "Seller", "Jadwal", "Status", "Aksi"]}>
          {filtered.map((e) => (
            <tr key={e.id}>
              <Td>
                <p className="font-medium">{e.title}</p>
                <p className="text-xs text-zinc-500">
                  {e.category} &middot; {e.venue} &middot; {e.ticket_types.length} tipe tiket
                </p>
              </Td>
              <Td className="text-xs">{e.seller_name ?? `#${e.seller_id}`}</Td>
              <Td className="whitespace-nowrap text-xs">{formatDateTime(e.starts_at)}</Td>
              <Td>
                <Badge status={e.status} label={EVENT_STATUS_LABEL[e.status]} />
              </Td>
              <Td>
                {e.status === "pending_review" && rejectFor !== e.id && (
                  <div className="flex gap-1">
                    <Button
                      disabled={busy === e.id}
                      onClick={() => moderate(e.id, "approve")}
                    >
                      Setujui
                    </Button>
                    <Button
                      variant="danger"
                      disabled={busy === e.id}
                      onClick={() => setRejectFor(e.id)}
                    >
                      Tolak
                    </Button>
                  </div>
                )}
                {rejectFor === e.id && (
                  <div className="min-w-[220px] space-y-2">
                    <TextArea
                      value={rejectReason}
                      onChange={(ev) => setRejectReason(ev.target.value)}
                      placeholder="Alasan penolakan..."
                    />
                    <div className="flex gap-1">
                      <Button
                        variant="danger"
                        disabled={busy === e.id || !rejectReason.trim()}
                        onClick={() => moderate(e.id, "reject")}
                      >
                        Tolak Event
                      </Button>
                      <Button variant="ghost" onClick={() => setRejectFor(null)}>
                        Batal
                      </Button>
                    </div>
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
