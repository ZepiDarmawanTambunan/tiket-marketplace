"use client";

import { useEffect, useState } from "react";
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
  TableSkeleton,
  SuccessBox,
  Table,
  Td,
} from "@/components/ui";

export default function AdminSellersPage() {
  const [sellers, setSellers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [verifying, setVerifying] = useState<number | null>(null);
  const [bankName, setBankName] = useState("");
  const [bankAccount, setBankAccount] = useState("");

  const load = () => {
    apiFetch<{ users: User[] }>("/api/admin/users?role=seller&per_page=100")
      .then((d) => setSellers(d.users))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const verify = async (u: User, verified: boolean) => {
    setError("");
    setSuccess("");
    try {
      await apiFetch(`/api/admin/users/${u.id}/verify-seller`, {
        method: "PUT",
        body: JSON.stringify({
          verified,
          bank_name: bankName || undefined,
          bank_account: bankAccount || undefined,
        }),
      });
      setSuccess(
        verified
          ? `${u.name} ditandai terverifikasi.`
          : `Verifikasi ${u.name} dicabut.`
      );
      setVerifying(null);
      setBankName("");
      setBankAccount("");
      setLoading(true);
      load();
    } catch (e) {
      setError(errMsg(e));
    }
  };

  return (
    <div>
      <PageHeader
        title="Verifikasi Seller"
        subtitle="Verifikasi bersifat opsional — seller bisa jualan tanpa ini kecuali kebijakan diubah di Pengaturan"
      />
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      {loading ? (
        <TableSkeleton />
      ) : sellers.length === 0 ? (
        <EmptyState message="Belum ada seller terdaftar." />
      ) : (
        <Table head={["Seller", "Status Verifikasi", "Terdaftar", "Aksi"]}>
          {sellers.map((u) => (
            <tr key={u.id}>
              <Td>
                <p className="font-medium">{u.name}</p>
                <p className="text-xs text-zinc-500">{u.email}</p>
              </Td>
              <Td>
                <Badge
                  status={u.seller_verified ? "paid" : "pending"}
                  label={u.seller_verified ? "Terverifikasi" : "Belum terverifikasi"}
                />
              </Td>
              <Td className="whitespace-nowrap text-xs">{formatDateTime(u.created_at)}</Td>
              <Td>
                {verifying === u.id ? (
                  <div className="space-y-2">
                    <Field label="Nama bank">
                      <Input value={bankName} onChange={(e) => setBankName(e.target.value)} placeholder="BCA" />
                    </Field>
                    <Field label="No. rekening">
                      <Input value={bankAccount} onChange={(e) => setBankAccount(e.target.value)} placeholder="1234567890" />
                    </Field>
                    <div className="flex gap-1">
                      <Button onClick={() => verify(u, true)}>Verifikasi</Button>
                      <Button variant="ghost" onClick={() => setVerifying(null)}>
                        Batal
                      </Button>
                    </div>
                  </div>
                ) : u.seller_verified ? (
                  <Button variant="secondary" onClick={() => verify(u, false)}>
                    Cabut Verifikasi
                  </Button>
                ) : (
                  <Button variant="secondary" onClick={() => setVerifying(u.id)}>
                    Verifikasi
                  </Button>
                )}
              </Td>
            </tr>
          ))}
        </Table>
      )}
    </div>
  );
}
