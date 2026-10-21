"use client";

import { useCallback, useEffect, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { User } from "@/lib/types";
import { formatDateTime } from "@/lib/format";
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
  SuccessBox,
  Table,
  Td,
} from "@/components/ui";

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [roleFilter, setRoleFilter] = useState("");
  const [search, setSearch] = useState("");
  const [busy, setBusy] = useState<number | null>(null);

  const load = useCallback(async (p: number, role: string, s: string) => {
    try {
      const q = new URLSearchParams({ page: String(p), per_page: "20" });
      if (role) q.set("role", role);
      if (s) q.set("search", s);
      const d = await apiFetch<{ users: User[]; page: number; per_page: number; total: number }>(
        `/api/admin/users?${q.toString()}`
      );
      setUsers(d.users);
      setPage(d.page);
      setTotal(d.total);
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void Promise.resolve().then(() => load(1, roleFilter, search));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [roleFilter]);

  const changeRole = async (u: User, role: string) => {
    if (!confirm(`Ubah peran ${u.name} menjadi ${role}?`)) return;
    setBusy(u.id);
    setError("");
    setSuccess("");
    try {
      await apiFetch(`/api/admin/users/${u.id}/role`, {
        method: "PUT",
        body: JSON.stringify({ role }),
      });
      setSuccess(`Peran ${u.name} diubah menjadi ${role}.`);
      setLoading(true);
      load(page, roleFilter, search);
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setBusy(null);
    }
  };

  return (
    <div>
      <PageHeader title="Kelola Users" subtitle="Ubah peran dan kelola akun pengguna" />
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      <div className="mb-4 grid gap-3 sm:grid-cols-3">
        <Field label="Filter peran">
          <Select
            value={roleFilter}
            onChange={(e) => {
              setLoading(true);
              setRoleFilter(e.target.value);
            }}
          >
            <option value="">Semua</option>
            <option value="buyer">Buyer</option>
            <option value="seller">Seller</option>
            <option value="super_admin">Super Admin</option>
          </Select>
        </Field>
        <Field label="Cari nama/email">
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                setLoading(true);
                load(1, roleFilter, search);
              }
            }}
            placeholder="ketik lalu Enter"
          />
        </Field>
        <div className="flex items-end">
          <Button
            variant="secondary"
            onClick={() => {
              setLoading(true);
              load(1, roleFilter, search);
            }}
          >
            Cari
          </Button>
        </div>
      </div>

      {loading ? (
        <TableSkeleton />
      ) : users.length === 0 ? (
        <EmptyState message="Tidak ada user." />
      ) : (
        <>
          <Table head={["User", "Peran", "Seller Terverifikasi", "Terdaftar", "Aksi"]}>
            {users.map((u) => (
              <tr key={u.id}>
                <Td>
                  <p className="font-medium">{u.name}</p>
                  <p className="text-xs text-zinc-500">{u.email}</p>
                </Td>
                <Td>
                  <Badge
                    status={u.role === "super_admin" ? "pending_review" : u.role === "seller" ? "pending" : "published"}
                    label={u.role}
                  />
                </Td>
                <Td>{u.role === "seller" ? (u.seller_verified ? "Ya" : "Belum") : "-"}</Td>
                <Td className="whitespace-nowrap text-xs">{formatDateTime(u.created_at)}</Td>
                <Td>
                  <Select
                    value={u.role}
                    disabled={busy === u.id}
                    onChange={(e) => changeRole(u, e.target.value)}
                    aria-label={`Ubah peran ${u.name}`}
                  >
                    <option value="buyer">buyer</option>
                    <option value="seller">seller</option>
                    <option value="super_admin">super_admin</option>
                  </Select>
                </Td>
              </tr>
            ))}
          </Table>
          <Pagination page={page} perPage={20} total={total} onPage={(p) => load(p, roleFilter, search)} />
        </>
      )}
    </div>
  );
}
