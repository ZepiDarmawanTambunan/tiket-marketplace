"use client";

import { useEffect, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { AuditLog } from "@/lib/types";
import { formatDateTime } from "@/lib/format";
import {
  Button,
  EmptyState,
  ErrorBox,
  Field,
  Input,
  PageHeader,
  Pagination,
  TableSkeleton,
  Table,
  Td,
} from "@/components/ui";

export default function AdminAuditLogsPage() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [action, setAction] = useState("");
  const [actorId, setActorId] = useState("");

  const load = async (p: number, a: string, actor: string) => {
    try {
      const q = new URLSearchParams({ page: String(p), per_page: "20" });
      if (a) q.set("action", a);
      if (actor) q.set("actor_id", actor);
      const d = await apiFetch<{
        audit_logs: AuditLog[];
        page: number;
        per_page: number;
        total: number;
      }>(`/api/admin/audit-logs?${q.toString()}`);
      setLogs(d.audit_logs);
      setPage(d.page);
      setTotal(d.total);
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void Promise.resolve().then(() => load(1, "", ""));
  }, []);

  return (
    <div>
      <PageHeader title="Audit Log" subtitle="Jejak aktivitas penting di platform" />

      <div className="mb-4 grid max-w-2xl gap-3 rounded-xl border border-zinc-200 bg-white p-4 sm:grid-cols-3">
        <Field label="Aksi">
          <Input
            value={action}
            onChange={(e) => setAction(e.target.value)}
            placeholder="cth: settings.update"
          />
        </Field>
        <Field label="ID Pelaku">
          <Input
            type="number"
            value={actorId}
            onChange={(e) => setActorId(e.target.value)}
            placeholder="cth: 1"
          />
        </Field>
        <div className="flex items-end">
          <Button
            onClick={() => {
              setLoading(true);
              load(1, action, actorId);
            }}
          >
            Filter
          </Button>
        </div>
      </div>

      <ErrorBox message={error} />

      {loading ? (
        <TableSkeleton />
      ) : logs.length === 0 ? (
        <EmptyState message="Tidak ada log." />
      ) : (
        <>
          <Table head={["Waktu", "Pelaku", "Aksi", "Detail"]}>
            {logs.map((l) => (
              <tr key={l.id}>
                <Td className="whitespace-nowrap text-xs">{formatDateTime(l.created_at)}</Td>
                <Td className="text-xs">
                  {l.actor_name ?? "-"}
                  {l.actor_id != null && <span className="text-zinc-400"> (#{l.actor_id})</span>}
                </Td>
                <Td>
                  <code className="rounded bg-zinc-100 px-1.5 py-0.5 text-xs">{l.action}</code>
                </Td>
                <Td className="max-w-md text-xs text-zinc-600">{l.detail ?? "-"}</Td>
              </tr>
            ))}
          </Table>
          <Pagination
            page={page}
            perPage={20}
            total={total}
            onPage={(p) => {
              setLoading(true);
              load(p, action, actorId);
            }}
          />
        </>
      )}
    </div>
  );
}
