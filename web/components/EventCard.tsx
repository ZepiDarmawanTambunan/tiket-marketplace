"use client";

import Link from "next/link";
import type { EventItem } from "@/lib/types";
import { formatDate, formatRupiah } from "@/lib/format";
import { Card } from "@/components/ui";

export default function EventCard({ e }: { e: EventItem }) {
  const types = e.ticket_types ?? [];
  const cheapest = types.length
    ? Math.min(...types.map((t) => t.price_cents))
    : null;
  return (
    <Link href={`/events/${e.id}`} className="group block h-full">
      <Card className="h-full overflow-hidden p-0 transition group-hover:shadow-md">
        {e.banner_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={e.banner_url}
            alt={e.title}
            className="h-40 w-full object-cover"
          />
        ) : (
          <div className="flex h-40 w-full items-center justify-center bg-gradient-to-br from-orange-100 to-purple-100 text-4xl">
            🎟️
          </div>
        )}
        <div className="p-4">
          <span className="text-xs font-medium uppercase tracking-wide text-orange-600">
            {e.category}
          </span>
          <h3 className="mt-1 font-semibold text-zinc-900 group-hover:text-orange-700">
            {e.title}
          </h3>
          <p className="mt-1 text-xs text-zinc-500">
            📍 {e.venue} &middot; {formatDate(e.starts_at)}
          </p>
          <p className="mt-3 text-sm font-bold text-zinc-900">
            {cheapest != null ? `Mulai ${formatRupiah(cheapest)}` : "Lihat detail"}
          </p>
        </div>
      </Card>
    </Link>
  );
}
