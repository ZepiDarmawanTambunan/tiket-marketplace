"use client";

import { useRouter } from "next/navigation";
import { apiFetch, errMsg } from "@/lib/api";
import { fromDatetimeLocal } from "@/lib/format";
import type { EventItem } from "@/lib/types";
import { Card, PageHeader, SuccessBox } from "@/components/ui";
import EventForm, {
  emptyEventForm,
  type EventFormValues,
} from "@/components/EventForm";
import { useState } from "react";

export default function NewEventPage() {
  const router = useRouter();
  const [notice, setNotice] = useState("");

  const create = async (v: EventFormValues) => {
    try {
      const { event } = await apiFetch<{ event: EventItem }>("/api/events", {
        method: "POST",
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
      setNotice("Event dibuat sebagai draft. Tambahkan tipe tiket di bawah.");
      router.push(`/seller/events/${event.id}/edit`);
    } catch (e) {
      throw new Error(errMsg(e));
    }
  };

  return (
    <div className="max-w-3xl">
      <PageHeader
        title="Buat Event Baru"
        subtitle="Event tersimpan sebagai draft — wajib direview admin sebelum tayang"
      />
      <SuccessBox message={notice} />
      <Card>
        <EventForm initial={emptyEventForm} onSubmit={create} submitLabel="Buat Event" />
      </Card>
    </div>
  );
}
