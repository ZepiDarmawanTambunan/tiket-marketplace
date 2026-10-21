"use client";

import { use, useEffect, useState } from "react";
import type { FormEvent } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { EventItem, TicketType } from "@/lib/types";
import { EVENT_STATUS_LABEL } from "@/lib/types";
import {
  fromDatetimeLocal,
  toDatetimeLocal,
  formatRupiah,
} from "@/lib/format";
import {
  Badge,
  Button,
  Card,
  EmptyState,
  ErrorBox,
  Field,
  Input,
  PageHeader,
  FormSkeleton,
  SuccessBox,
  Table,
  Td,
} from "@/components/ui";
import EventForm, {
  eventToForm,
  type EventFormValues,
} from "@/components/EventForm";

/* ---------- Form tambah/edit tipe tiket ---------- */
function TicketTypeForm({
  eventId,
  existing,
  onDone,
}: {
  eventId: number;
  existing?: TicketType | null;
  onDone: () => void;
}) {
  const [name, setName] = useState(existing?.name ?? "");
  const [price, setPrice] = useState(existing ? String(existing.price_cents) : "");
  const [quota, setQuota] = useState(existing ? String(existing.quota) : "");
  const [saleStart, setSaleStart] = useState(toDatetimeLocal(existing?.sale_starts_at));
  const [saleEnd, setSaleEnd] = useState(toDatetimeLocal(existing?.sale_ends_at));
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      const body = {
        name,
        price_cents: Number(price),
        quota: Number(quota),
        sale_starts_at: saleStart ? fromDatetimeLocal(saleStart) : undefined,
        sale_ends_at: saleEnd ? fromDatetimeLocal(saleEnd) : undefined,
      };
      if (existing) {
        await apiFetch(`/api/ticket-types/${existing.id}`, {
          method: "PUT",
          body: JSON.stringify(body),
        });
      } else {
        await apiFetch(`/api/events/${eventId}/ticket-types`, {
          method: "POST",
          body: JSON.stringify(body),
        });
      }
      onDone();
    } catch (err) {
      setError(errMsg(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={submit} className="space-y-3 rounded-lg bg-zinc-50 p-4">
      <ErrorBox message={error} />
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="Nama tipe *">
          <Input required value={name} onChange={(e) => setName(e.target.value)} placeholder="Reguler / VIP" />
        </Field>
        <Field label="Harga (Rp) *">
          <Input required type="number" min={0} value={price} onChange={(e) => setPrice(e.target.value)} placeholder="150000" />
        </Field>
        <Field label="Kuota *">
          <Input required type="number" min={1} value={quota} onChange={(e) => setQuota(e.target.value)} placeholder="500" />
        </Field>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Mulai dijual">
            <Input type="datetime-local" value={saleStart} onChange={(e) => setSaleStart(e.target.value)} />
          </Field>
          <Field label="Berakhir">
            <Input type="datetime-local" value={saleEnd} onChange={(e) => setSaleEnd(e.target.value)} />
          </Field>
        </div>
      </div>
      <div className="flex gap-2">
        <Button disabled={saving}>{saving ? "Menyimpan..." : existing ? "Simpan Perubahan" : "Tambah Tipe Tiket"}</Button>
        <Button variant="ghost" type="button" onClick={onDone}>
          Batal
        </Button>
      </div>
    </form>
  );
}

/* ---------- Halaman edit ---------- */
export default function EditEventPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const [event, setEvent] = useState<EventItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [editing, setEditing] = useState<TicketType | null>(null);

  const load = () => {
    apiFetch<{ event: EventItem }>(`/api/events/${id}`)
      .then((d) => setEvent(d.event))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  };

  useEffect(load, [id]);

  const saveEvent = async (v: EventFormValues) => {
    try {
      const { event: e } = await apiFetch<{ event: EventItem }>(`/api/events/${id}`, {
        method: "PUT",
        body: JSON.stringify({
          title: v.title,
          description: v.description,
          category: v.category,
          venue: v.venue,
          address: v.address || undefined,
          maps_url: v.maps_url || undefined,
          starts_at: fromDatetimeLocal(v.starts_at),
          ends_at: fromDatetimeLocal(v.ends_at),
          banner_url: v.banner_url || undefined,
          refund_policy: v.refund_policy || undefined,
        }),
      });
      setEvent(e);
      setSuccess("Event tersimpan.");
    } catch (e) {
      throw new Error(errMsg(e));
    }
  };

  const deleteTicketType = async (tt: TicketType) => {
    if (!confirm(`Hapus tipe tiket "${tt.name}"?`)) return;
    setError("");
    try {
      await apiFetch(`/api/ticket-types/${tt.id}`, { method: "DELETE" });
      setSuccess("Tipe tiket dihapus.");
      setLoading(true);
      load();
    } catch (e) {
      setError(errMsg(e));
    }
  };

  const submitReview = async () => {
    setError("");
    try {
      const { event: e } = await apiFetch<{ event: EventItem }>(
        `/api/events/${id}/submit`,
        { method: "POST" }
      );
      setEvent(e);
      setSuccess("Event diajukan untuk review admin.");
    } catch (e) {
      setError(errMsg(e));
    }
  };

  if (loading)
    return <FormSkeleton />;
  if (!event)
    return <ErrorBox message={error || "Event tidak ditemukan"} />;

  const editable = event.status === "draft" || event.status === "rejected";

  return (
    <div className="max-w-4xl space-y-6">
      <PageHeader
        title={`Edit: ${event.title}`}
        subtitle="Kelola detail event dan tipe tiket"
        actions={<Badge status={event.status} label={EVENT_STATUS_LABEL[event.status]} />}
      />
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      <Card>
        <h2 className="mb-4 font-semibold text-zinc-900">Detail Event</h2>
        {editable ? (
          <EventForm initial={eventToForm(event)} onSubmit={saveEvent} submitLabel="Simpan Perubahan" />
        ) : (
          <p className="text-sm text-zinc-500">
            Event berstatus <strong>{EVENT_STATUS_LABEL[event.status]}</strong> tidak bisa
            diubah lagi.
          </p>
        )}
      </Card>

      <Card>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="font-semibold text-zinc-900">Tipe Tiket</h2>
          {editable && !showAdd && (
            <Button variant="secondary" onClick={() => { setEditing(null); setShowAdd(true); }}>
              + Tambah Tipe
            </Button>
          )}
        </div>

        {showAdd && (
          <div className="mb-4">
            <TicketTypeForm
              eventId={event.id}
              existing={editing}
              onDone={() => { setShowAdd(false); setEditing(null); setLoading(true); load(); }}
            />
          </div>
        )}

        {(event.ticket_types ?? []).length === 0 ? (
          <EmptyState message="Belum ada tipe tiket. Tambahkan minimal satu tipe tiket." />
        ) : (
          <Table head={["Nama", "Harga", "Kuota", "Terjual", "Aksi"]}>
            {event.ticket_types.map((t) => (
              <tr key={t.id}>
                <Td className="font-medium">{t.name}</Td>
                <Td>{formatRupiah(t.price_cents)}</Td>
                <Td>{t.quota}</Td>
                <Td>{t.sold}</Td>
                <Td>
                  {editable && (
                    <div className="flex gap-1">
                      <Button
                        variant="secondary"
                        onClick={() => { setEditing(t); setShowAdd(true); }}
                      >
                        Edit
                      </Button>
                      <Button variant="danger" onClick={() => deleteTicketType(t)}>
                        Hapus
                      </Button>
                    </div>
                  )}
                </Td>
              </tr>
            ))}
          </Table>
        )}
        <p className="mt-2 text-xs text-zinc-500">
          Tipe tiket yang sudah terjual tidak bisa diubah harga/kuotanya atau dihapus.
        </p>
      </Card>

      {(event.status === "draft" || event.status === "rejected") && (
        <Card>
          <h2 className="mb-2 font-semibold text-zinc-900">Ajukan untuk Review</h2>
          <p className="mb-3 text-sm text-zinc-500">
            Setelah diajukan, event menunggu persetujuan admin sebelum tayang.
          </p>
          <Button
            onClick={submitReview}
            disabled={event.ticket_types.length === 0}
          >
            Ajukan Review ke Admin
          </Button>
        </Card>
      )}
    </div>
  );
}
