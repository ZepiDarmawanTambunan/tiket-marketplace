"use client";

import { useCallback, useEffect, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { Order } from "@/lib/types";
import { ORDER_STATUS_LABEL } from "@/lib/types";
import { formatDateTime, formatRupiah, toDateInput } from "@/lib/format";
import {
  Badge,
  Button,
  EmptyState,
  ErrorBox,
  Field,
  Input,
  PageHeader,
  Pagination,
  Select,
  TableSkeleton,
  Table,
  Td,
} from "@/components/ui";

export default function AdminTransactionsPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [status, setStatus] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  const load = useCallback(async (p: number, s: string, f: string, t: string) => {
    try {
      const q = new URLSearchParams({ page: String(p), per_page: "20" });
      if (s) q.set("status", s);
      if (f) q.set("from", f);
      if (t) q.set("to", t);
      const d = await apiFetch<{
        orders: Order[];
        page: number;
        per_page: number;
        total: number;
      }>(`/api/admin/transactions?${q.toString()}`);
      setOrders(d.orders);
      setPage(d.page);
      setTotal(d.total);
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void Promise.resolve().then(() => load(1, "", "", ""));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const apply = () => {
    setLoading(true);
    load(1, status, from, to);
  };

  return (
    <div>
      <PageHeader title="Transaksi" subtitle="Semua order & pembayaran Midtrans" />

      <div className="mb-4 grid gap-3 rounded-xl border border-zinc-200 bg-white p-4 sm:grid-cols-4">
        <Field label="Status">
          <Select value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="">Semua</option>
            <option value="pending">Pending</option>
            <option value="paid">Lunas</option>
            <option value="refund_requested">Refund diajukan</option>
            <option value="refunded">Refunded</option>
            <option value="cancelled">Dibatalkan</option>
            <option value="failed">Gagal</option>
            <option value="expired">Kedaluarsa</option>
          </Select>
        </Field>
        <Field label="Dari">
          <Input type="date" value={from} onChange={(e) => setFrom(toDateInput(e.target.value))} />
        </Field>
        <Field label="Sampai">
          <Input type="date" value={to} onChange={(e) => setTo(toDateInput(e.target.value))} />
        </Field>
        <div className="flex items-end">
          <Button onClick={apply}>Filter</Button>
        </div>
      </div>

      <ErrorBox message={error} />

      {loading ? (
        <TableSkeleton />
      ) : orders.length === 0 ? (
        <EmptyState message="Tidak ada transaksi." />
      ) : (
        <>
          <Table head={["Order", "Event", "Gross", "Komisi", "Fee", "Midtrans", "Status", "Waktu"]}>
            {orders.map((o) => (
              <tr key={o.id}>
                <Td className="font-mono text-xs">{o.order_code}</Td>
                <Td className="text-xs">{o.event_title ?? `#${o.event_id}`}</Td>
                <Td className="font-medium">{formatRupiah(o.gross_cents)}</Td>
                <Td className="text-xs">{formatRupiah(o.commission_cents)}</Td>
                <Td className="text-xs">{formatRupiah(o.gateway_fee_cents)}</Td>
                <Td className="text-xs text-zinc-500">
                  {o.payment_type ?? "-"}
                  {o.transaction_id && <span className="block font-mono">{o.transaction_id}</span>}
                </Td>
                <Td>
                  <Badge status={o.status} label={ORDER_STATUS_LABEL[o.status]} />
                </Td>
                <Td className="whitespace-nowrap text-xs">{formatDateTime(o.created_at)}</Td>
              </tr>
            ))}
          </Table>
          <Pagination
            page={page}
            perPage={20}
            total={total}
            onPage={(p) => {
              setLoading(true);
              load(p, status, from, to);
            }}
          />
        </>
      )}
    </div>
  );
}
