"use client";

import { useState } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { toDatetimeLocal } from "@/lib/format";
import { Button, ErrorBox, Field, Input, Select, TextArea } from "@/components/ui";

export interface EventFormValues {
  title: string;
  description: string;
  category: string;
  venue: string;
  address: string;
  maps_url: string;
  starts_at: string;
  ends_at: string;
  banner_url: string;
  refund_policy: string;
}

const CATEGORIES = ["musik", "seminar", "workshop", "festival", "olahraga", "seni", "lainnya"];

export const emptyEventForm: EventFormValues = {
  title: "",
  description: "",
  category: "musik",
  venue: "",
  address: "",
  maps_url: "",
  starts_at: "",
  ends_at: "",
  banner_url: "",
  refund_policy: "",
};

export function eventToForm(e: {
  title: string;
  description: string;
  category: string;
  venue: string;
  address?: string;
  maps_url?: string;
  starts_at: string;
  ends_at: string;
  banner_url?: string;
  refund_policy?: string;
}): EventFormValues {
  return {
    title: e.title,
    description: e.description,
    category: e.category,
    venue: e.venue,
    address: e.address ?? "",
    maps_url: e.maps_url ?? "",
    starts_at: toDatetimeLocal(e.starts_at),
    ends_at: toDatetimeLocal(e.ends_at),
    banner_url: e.banner_url ?? "",
    refund_policy: e.refund_policy ?? "",
  };
}

export default function EventForm({
  initial,
  onSubmit,
  submitLabel,
}: {
  initial: EventFormValues;
  onSubmit: (values: EventFormValues) => Promise<void>;
  submitLabel: string;
}) {
  const [v, setV] = useState<EventFormValues>(initial);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const set = (k: keyof EventFormValues) => (
    e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) => setV((prev) => ({ ...prev, [k]: e.target.value }));

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setSaving(true);
    try {
      await onSubmit(v);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menyimpan");
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={submit} className="space-y-4">
      <ErrorBox message={error} />
      <Field label="Judul event *">
        <Input required value={v.title} onChange={set("title")} placeholder="Konser Jazz Malam Minggu" />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Kategori *">
          <Select value={v.category} onChange={set("category")}>
            {CATEGORIES.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
          </Select>
        </Field>
        <Field label="Venue *">
          <Input required value={v.venue} onChange={set("venue")} placeholder="GOR Jambi" />
        </Field>
      </div>
      <Field label="Deskripsi">
        <TextArea value={v.description} onChange={set("description")} placeholder="Ceritakan event Anda..." />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Mulai *">
          <Input type="datetime-local" required value={v.starts_at} onChange={set("starts_at")} />
        </Field>
        <Field label="Selesai *">
          <Input type="datetime-local" required value={v.ends_at} onChange={set("ends_at")} />
        </Field>
      </div>
      <Field label="Alamat">
        <Input value={v.address} onChange={set("address")} placeholder="Jl. ..." />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Link Google Maps">
          <Input value={v.maps_url} onChange={set("maps_url")} placeholder="https://maps.google.com/..." />
        </Field>
        <Field label="URL Banner">
          <Input value={v.banner_url} onChange={set("banner_url")} placeholder="https://..." />
        </Field>
      </div>
      <Field
        label="Kebijakan refund"
        hint="Contoh: Refund maksimal H-7 sebelum acara."
      >
        <TextArea value={v.refund_policy} onChange={set("refund_policy")} />
      </Field>
      <Button disabled={saving}>{saving ? "Menyimpan..." : submitLabel}</Button>
    </form>
  );
}
