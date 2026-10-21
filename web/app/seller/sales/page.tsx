"use client";

import { useEffect, useMemo, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { EventItem, Order } from "@/lib/types";
import { ORDER_STATUS_LABEL } from "@/lib/types";
import { formatDateTime, formatRupiah } from "@/lib/format";
import {
  Badge,
  EmptyState,
  ErrorBox,
  Field,
  PageHeader,
  Select,
  TableSkeleton,
  Table,
  Td,
  Pagination,
} from "@/components/ui";

/**
 * ASUMSI (perlu konfirmasi worker backend): GET /api/orders dengan token
 * seller mengembalikan order untuk event milik seller tersebut
 * (difilter server-side berdasarkan kepemilikan event).
 */
export default function SellerSalesPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [eventFilter, setEventFilter] = useState("");

  useEffect(() => {
    apiFetch<{ events: EventItem[] }>("/api/events?mine=true&per_page=100")
      .then((d) => setEvents(d.events))
      .catch(() => {});
  }, []);

  const load = (p: number, status: string) => {
    const q = new URLSearchParams({ page: String(p), per_page: "20" });
    if (status) q.set("status", status);
    apiFetch<{ orders: Order[]; page: number; per_page: number; total: number }>(
      `/api/orders?${q.toString()}`
    )
      .then((d) => {
        setOrders(d.orders);
        setPage(d.page);
        setTotal(d.total);
      })
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    void Promise.resolve().then(() => load(1, statusFilter));
  }, [statusFilter]);

  const filtered = useMemo(
    () =>
      eventFilter
        ? orders.filter((o) => o.event_id === Number(eventFilter))
        : orders,
    [orders, eventFilter]
  );

  const paidRevenue = useMemo(
    () =>
      filtered
        .filter((o) => o.status === "paid")
        .reduce((s, o) => s + (o.subtotal_cents - o.commission_cents), 0),
    [filtered]
  );

  return (
    <div>
      <PageHeader
        title="Penjualan"
        subtitle="Order tiket untuk event Anda"
      />

      <div className="mb-4 grid gap-3 sm:grid-cols-3">
        <Field label="Filter status">
          <Select
            value={statusFilter}
            onChange={(e) => {
              setLoading(true);
              setStatusFilter(e.target.value);
            }}
          >
            <option value="">Semua status</option>
            <option value="paid">Lunas</option>
            <option value="pending">Menunggu pembayaran</option>
            <option value="refund_requested">Refund diajukan</option>
            <option value="refunded">Dana dikembalikan</option>
            <option value="cancelled">Dibatalkan</option>
          </Select>
        </Field>
        <Field label="Filter event">
          <Select value={eventFilter} onChange={(e) => setEventFilter(e.target.value)}>
            <option value="">Semua event</option>
            {events.map((e) => (
              <option key={e.id} value={e.id}>
                {e.title}
              </option>
            ))}
          </Select>
        </Field>
        <div className="flex items-end">
          <div className="rounded-lg bg-green-50 px-4 py-2 text-sm">
            <span className="text-green-700">Pendapatan bersih (lunas): </span>
            <strong className="text-green-800">{formatRupiah(paidRevenue)}</strong>
          </div>
        </div>
      </div>

      <ErrorBox message={error} />

      {loading ? (
        <TableSkeleton />
      ) : filtered.length === 0 ? (
        <EmptyState message="Belum ada order." />
      ) : (
        <>
          <Table head={["Order", "Event", "Item", "Total", "Status", "Waktu"]}>
            {filtered.map((o) => (
              <tr key={o.id}>
                <Td className="font-mono text-xs">{o.order_code}</Td>
                <Td>{o.event_title ?? `#${o.event_id}`}</Td>
                <Td>
                  {(o.items ?? []).map((it, i) => (
                    <span key={i} className="block text-xs">
                      {it.ticket_type_name ?? `#${it.ticket_type_id}`} × {it.qty}
                    </span>
                  ))}
                </Td>
                <Td className="font-medium">{formatRupiah(o.gross_cents)}</Td>
                <Td>
                  <Badge status={o.status} label={ORDER_STATUS_LABEL[o.status]} />
                </Td>
                <Td className="whitespace-nowrap text-xs">
                  {formatDateTime(o.created_at)}
                </Td>
              </tr>
            ))}
          </Table>
          <Pagination
            page={page}
            perPage={20}
            total={total}
            onPage={(p) => {
              setLoading(true);
              load(p, statusFilter);
            }}
          />
        </>
      )}
    </div>
  );
}
