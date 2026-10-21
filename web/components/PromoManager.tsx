"use client";

import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { EventItem, Promo, PromoScope, PromoType } from "@/lib/types";
import {
  fromDatetimeLocal,
  toDatetimeLocal,
  formatDate,
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
  Select,
  TableSkeleton,
  SuccessBox,
  Table,
  Td,
} from "@/components/ui";

interface FormState {
  code: string;
  type: PromoType;
  value: string;
  min_spend: string;
  quota: string;
  per_user_limit: string;
  starts_at: string;
  ends_at: string;
  scope: PromoScope;
  event_id: string;
}

const EMPTY: FormState = {
  code: "",
  type: "percent",
  value: "",
  min_spend: "0",
  quota: "",
  per_user_limit: "1",
  starts_at: "",
  ends_at: "",
  scope: "event",
  event_id: "",
};

function promoValueLabel(p: Promo): string {
  return p.type === "percent" ? `${p.value}%` : formatRupiah(p.value);
}

function PromoForm({
  initial,
  events,
  isAdmin,
  onSaved,
  onCancel,
}: {
  initial?: Promo | null;
  events: EventItem[];
  isAdmin: boolean;
  onSaved: () => void;
  onCancel: () => void;
}) {
  const [form, setForm] = useState<FormState>(() => {
    if (!initial) return EMPTY;
    return {
      code: initial.code,
      type: initial.type,
      value: String(initial.value),
      min_spend: String(initial.min_spend_cents),
      quota: String(initial.quota),
      per_user_limit: String(initial.per_user_limit),
      starts_at: toDatetimeLocal(initial.starts_at),
      ends_at: toDatetimeLocal(initial.ends_at),
      scope: initial.scope,
      event_id: initial.event_id ? String(initial.event_id) : "",
    };
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const set = (k: keyof FormState) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      const body: Record<string, unknown> = {
        code: form.code.trim().toUpperCase(),
        type: form.type,
        value: Number(form.value),
        min_spend_cents: Number(form.min_spend || 0),
        quota: Number(form.quota),
        per_user_limit: Number(form.per_user_limit || 1),
        starts_at: form.starts_at ? fromDatetimeLocal(form.starts_at) : undefined,
        ends_at: form.ends_at ? fromDatetimeLocal(form.ends_at) : undefined,
        scope: isAdmin ? form.scope : "event",
      };
      const scope = isAdmin ? form.scope : "event";
      if (scope === "event") body.event_id = Number(form.event_id);
      if (initial) {
        await apiFetch(`/api/promos/${initial.id}`, {
          method: "PUT",
          body: JSON.stringify(body),
        });
      } else {
        await apiFetch("/api/promos", {
          method: "POST",
          body: JSON.stringify(body),
        });
      }
      onSaved();
    } catch (err) {
      setError(errMsg(err));
    } finally {
      setSaving(false);
    }
  };

  const scope = isAdmin ? form.scope : "event";

  return (
    <Card>
      <h2 className="mb-4 font-semibold text-zinc-900">
        {initial ? "Edit Kode Promo" : "Buat Kode Promo"}
      </h2>
      <form onSubmit={submit} className="space-y-3">
        <ErrorBox message={error} />
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Kode *">
            <Input
              required
              value={form.code}
              onChange={set("code")}
              placeholder="DISKON20"
              className="font-mono uppercase"
            />
          </Field>
          {isAdmin && (
            <Field label="Cakupan">
              <Select value={form.scope} onChange={set("scope")}>
                <option value="event">Per event</option>
                <option value="global">Global (semua event)</option>
              </Select>
            </Field>
          )}
        </div>
        {scope === "event" && (
          <Field label="Event *">
            <Select
              required
              value={form.event_id}
              onChange={set("event_id")}
            >
              <option value="">
                {isAdmin ? "Pilih event" : "Pilih event milikmu"}
              </option>
              {events.map((ev) => (
                <option key={ev.id} value={ev.id}>
                  {ev.title}
                </option>
              ))}
            </Select>
          </Field>
        )}
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Tipe *">
            <Select value={form.type} onChange={set("type")}>
              <option value="percent">Persen (%)</option>
              <option value="flat">Nominal (Rp)</option>
            </Select>
          </Field>
          <Field
            label={form.type === "percent" ? "Nilai (%) *" : "Nilai (Rp) *"}
          >
            <Input
              required
              type="number"
              min={1}
              max={form.type === "percent" ? 100 : undefined}
              value={form.value}
              onChange={set("value")}
              placeholder={form.type === "percent" ? "20" : "50000"}
            />
          </Field>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Minimal belanja (Rp)">
            <Input
              type="number"
              min={0}
              value={form.min_spend}
              onChange={set("min_spend")}
              placeholder="0"
            />
          </Field>
          <Field label="Kuota penggunaan *">
            <Input
              required
              type="number"
              min={1}
              value={form.quota}
              onChange={set("quota")}
              placeholder="100"
            />
          </Field>
        </div>
        <div className="grid gap-3 sm:grid-cols-3">
          <Field label="Limit per user">
            <Input
              type="number"
              min={1}
              value={form.per_user_limit}
              onChange={set("per_user_limit")}
            />
          </Field>
          <Field label="Mulai berlaku">
            <Input
              type="datetime-local"
              value={form.starts_at}
              onChange={set("starts_at")}
            />
          </Field>
          <Field label="Berakhir">
            <Input
              type="datetime-local"
              value={form.ends_at}
              onChange={set("ends_at")}
            />
          </Field>
        </div>
        <div className="flex gap-2 pt-1">
          <Button disabled={saving} className="flex-1">
            {saving ? "Menyimpan..." : initial ? "Simpan Perubahan" : "Buat Promo"}
          </Button>
          <Button variant="ghost" type="button" onClick={onCancel}>
            Batal
          </Button>
        </div>
      </form>
    </Card>
  );
}

/**
 * Manager kode promo — dipakai halaman seller (/seller/promos, hanya event
 * milik sendiri) dan admin (/admin/promos, termasuk scope global).
 */
export default function PromoManager({ isAdmin }: { isAdmin: boolean }) {
  const [promos, setPromos] = useState<Promo[]>([]);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Promo | null>(null);
  const [busy, setBusy] = useState<number | null>(null);

  const load = () => {
    const q = new URLSearchParams({ per_page: "100" });
    setLoading(true);
    apiFetch<{ promos: Promo[] }>(`/api/promos?${q.toString()}`)
      .then((d) => setPromos(d.promos))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
    apiFetch<{ events: EventItem[] }>("/api/events?mine=true&per_page=100")
      .then((d) => setEvents(d.events))
      .catch(() => {});
  };

  useEffect(() => {
    void Promise.resolve().then(load);
  }, []);

  const startCreate = () => {
    setEditing(null);
    setShowForm(true);
  };

  const startEdit = (p: Promo) => {
    setEditing(p);
    setShowForm(true);
  };

  const onSaved = () => {
    setShowForm(false);
    setEditing(null);
    setSuccess(editing ? "Kode promo diperbarui." : "Kode promo dibuat.");
    load();
  };

  const deactivate = async (p: Promo) => {
    if (!confirm(`Nonaktifkan kode promo ${p.code}?`)) return;
    setBusy(p.id);
    setError("");
    setSuccess("");
    try {
      await apiFetch(`/api/promos/${p.id}`, { method: "DELETE" });
      setSuccess(`Kode promo ${p.code} dinonaktifkan.`);
      load();
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setBusy(null);
    }
  };

  return (
    <div>
      <PageHeader
        title="Kode Promo"
        subtitle={
          isAdmin
            ? "Buat promo global atau per event"
            : "Buat kode promo khusus untuk event milikmu"
        }
        actions={
          !showForm && (
            <Button onClick={startCreate}>+ Buat Promo</Button>
          )
        }
      />
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      {showForm ? (
        <div className="max-w-2xl">
          <PromoForm
            initial={editing}
            events={events}
            isAdmin={isAdmin}
            onSaved={onSaved}
            onCancel={() => {
              setShowForm(false);
              setEditing(null);
            }}
          />
        </div>
      ) : loading ? (
        <TableSkeleton />
      ) : promos.length === 0 ? (
        <EmptyState message="Belum ada kode promo." />
      ) : (
        <Table
          head={[
            "Kode",
            "Nilai",
            "Min. Belanja",
            "Kuota",
            "Periode",
            "Cakupan",
            "Status",
            "Aksi",
          ]}
        >
          {promos.map((p) => (
            <tr key={p.id}>
              <Td className="font-mono font-medium">{p.code}</Td>
              <Td>{promoValueLabel(p)}</Td>
              <Td>{p.min_spend_cents > 0 ? formatRupiah(p.min_spend_cents) : "-"}</Td>
              <Td>
                {p.used_count ?? 0}/{p.quota}
              </Td>
              <Td className="whitespace-nowrap text-xs">
                {formatDate(p.starts_at)} — {formatDate(p.ends_at)}
              </Td>
              <Td className="text-xs">
                {p.scope === "global" ? "Global" : p.event_title ?? `Event #${p.event_id}`}
              </Td>
              <Td>
                <Badge
                  status={p.active ? "approved" : "cancelled"}
                  label={p.active ? "Aktif" : "Nonaktif"}
                />
              </Td>
              <Td>
                <div className="flex gap-1">
                  <Button variant="secondary" onClick={() => startEdit(p)}>
                    Edit
                  </Button>
                  {p.active && (
                    <Button
                      variant="danger"
                      disabled={busy === p.id}
                      onClick={() => deactivate(p)}
                    >
                      Nonaktifkan
                    </Button>
                  )}
                </div>
              </Td>
            </tr>
          ))}
        </Table>
      )}
    </div>
  );
}
