"use client";

import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { Payout, User, MidtransSettings } from "@/lib/types";
import { formatDateTime, formatRupiah } from "@/lib/format";
import {
  Badge,
  Button,
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

function irisStatusLabel(p: Payout): string {
  const s = (p.iris_status ?? "").toLowerCase();
  if (!s) return "-";
  if (s.includes("success") || s.includes("sent") || s.includes("completed"))
    return "Terkirim";
  if (s.includes("fail") || s.includes("error") || s.includes("reject"))
    return "Gagal";
  if (s.includes("pending")) return "Proses";
  return p.iris_status ?? "-";
}

function PayoutRow({
  payout,
  onChanged,
  onMessage,
}: {
  payout: Payout;
  onChanged: () => void;
  onMessage: (ok: string, err: string) => void;
}) {
  const [payFor, setPayFor] = useState(false);
  const [proofUrl, setProofUrl] = useState("");
  const [busy, setBusy] = useState(false);
  const p = payout;

  const refreshIrisStatus = async () => {
    setBusy(true);
    onMessage("", "");
    try {
      const d = await apiFetch<{ status: string; reference?: string; message?: string }>(
        `/api/admin/payouts/${p.id}/iris-status`
      );
      onMessage(
        `Status Iris payout #${p.id}: ${d.status}${d.message ? ` — ${d.message}` : ""}`,
        ""
      );
      onChanged();
    } catch (e) {
      onMessage("", errMsg(e));
    } finally {
      setBusy(false);
    }
  };

  const sendIris = async () => {
    if (
      !confirm(
        `Kirim payout ${formatRupiah(p.amount_cents)} ke seller ${
          p.seller_name ?? `#${p.seller_id}`
        } via Midtrans Iris?`
      )
    )
      return;
    setBusy(true);
    onMessage("", "");
    try {
      const d = await apiFetch<{ message?: string }>(
        `/api/admin/payouts/${p.id}/send-iris`,
        { method: "POST" }
      );
      onMessage(d.message ?? "Permintaan kirim via Iris dikirim.", "");
      onChanged();
    } catch (e) {
      onMessage("", errMsg(e));
    } finally {
      setBusy(false);
    }
  };

  const markPaid = async () => {
    setBusy(true);
    onMessage("", "");
    try {
      await apiFetch(`/api/admin/payouts/${p.id}/pay`, {
        method: "POST",
        body: JSON.stringify({ proof_url: proofUrl || undefined }),
      });
      onMessage("Payout ditandai sudah dibayar.", "");
      setPayFor(false);
      setProofUrl("");
      onChanged();
    } catch (e) {
      onMessage("", errMsg(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <tr>
      <Td className="whitespace-nowrap text-xs">{formatDateTime(p.created_at)}</Td>
      <Td className="text-xs">
        {p.seller_name ?? `#${p.seller_id}`}
        {p.event_title && <span className="block text-zinc-500">{p.event_title}</span>}
      </Td>
      <Td className="font-medium">{formatRupiah(p.amount_cents)}</Td>
      <Td className="text-xs">{p.method === "iris" ? "Iris" : "Manual"}</Td>
      <Td>
        <Badge
          status={p.status}
          label={p.status === "paid" ? "Sudah dibayar" : p.status === "pending" ? "Pending" : "Dibatalkan"}
        />
        {p.method === "iris" && (p.iris_status || p.iris_reference_no) && (
          <div className="mt-1 text-xs text-zinc-500">
            <Badge status="pending" label={irisStatusLabel(p)} />
            {p.iris_reference_no && (
              <span className="ml-1 block font-mono text-[11px]">{p.iris_reference_no}</span>
            )}
          </div>
        )}
      </Td>
      <Td>
        {p.status === "pending" && !payFor && (
          <div className="flex flex-wrap gap-1">
            {p.method === "iris" && (
              <>
                <Button variant="secondary" disabled={busy} onClick={sendIris}>
                  Kirim via Iris
                </Button>
                <Button variant="ghost" disabled={busy} onClick={refreshIrisStatus}>
                  Cek Status
                </Button>
              </>
            )}
            <Button variant="secondary" onClick={() => setPayFor(true)}>
              Tandai Manual
            </Button>
          </div>
        )}
        {p.status === "pending" && payFor && (
          <div className="min-w-[200px] space-y-2">
            <Field label="URL bukti transfer">
              <Input value={proofUrl} onChange={(e) => setProofUrl(e.target.value)} placeholder="https://..." />
            </Field>
            <div className="flex gap-1">
              <Button disabled={busy} onClick={markPaid}>
                Simpan
              </Button>
              <Button variant="ghost" onClick={() => setPayFor(false)}>
                Batal
              </Button>
            </div>
          </div>
        )}
        {p.proof_url && (
          <a href={p.proof_url} target="_blank" rel="noreferrer" className="ml-2 text-xs text-orange-600 hover:underline">
            Bukti
          </a>
        )}
      </Td>
    </tr>
  );
}

export default function AdminPayoutsPage() {
  const [payouts, setPayouts] = useState<Payout[]>([]);
  const [sellers, setSellers] = useState<User[]>([]);
  const [irisConfigured, setIrisConfigured] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [statusFilter, setStatusFilter] = useState("");

  // form payout baru
  const [sellerId, setSellerId] = useState("");
  const [eventId, setEventId] = useState("");
  const [amount, setAmount] = useState("");
  const [method, setMethod] = useState<"manual" | "iris">("manual");
  const [busy, setBusy] = useState(false);

  const load = () => {
    const q = new URLSearchParams({ per_page: "50" });
    if (statusFilter) q.set("status", statusFilter);
    apiFetch<{ payouts: Payout[] }>(`/api/admin/payouts?${q.toString()}`)
      .then((d) => setPayouts(d.payouts))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    void Promise.resolve().then(load);
    apiFetch<{ users: User[] }>("/api/admin/users?role=seller&per_page=100")
      .then((d) => setSellers(d.users))
      .catch(() => {});
    apiFetch<MidtransSettings>("/api/admin/settings/midtrans")
      .then((s) => setIrisConfigured(Boolean(s.iris_api_key_masked)))
      .catch(() => setIrisConfigured(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusFilter]);

  const setMessages = (ok: string, err: string) => {
    setSuccess(ok);
    setError(err);
  };

  const createPayout = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    setSuccess("");
    try {
      await apiFetch("/api/admin/payouts", {
        method: "POST",
        body: JSON.stringify({
          seller_id: Number(sellerId),
          event_id: eventId ? Number(eventId) : undefined,
          amount_cents: Number(amount),
          method,
        }),
      });
      setSuccess(`Payout dibuat (pending, metode ${method === "iris" ? "Iris" : "manual"}).`);
      setSellerId("");
      setEventId("");
      setAmount("");
      setMethod("manual");
      setLoading(true);
      load();
    } catch (err) {
      setError(errMsg(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div>
      <PageHeader title="Payout ke Seller" subtitle="Catat pencairan dana hasil penjualan" />
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      {irisConfigured === false && (
        <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
          ⚠️ Midtrans Iris belum dikonfigurasi (API key / rekening sumber kosong).
          Payout via Iris tidak bisa dikirim — lengkapi di{" "}
          <a href="/admin/settings" className="font-medium underline">
            Pengaturan
          </a>
          , atau gunakan metode manual.
        </div>
      )}

      <div className="mb-6 grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-1 rounded-xl border border-zinc-200 bg-white p-5">
          <h2 className="mb-3 font-semibold text-zinc-900">Buat Payout</h2>
          <form onSubmit={createPayout} className="space-y-3">
            <Field label="Seller *">
              <Select required value={sellerId} onChange={(e) => setSellerId(e.target.value)}>
                <option value="">Pilih seller</option>
                {sellers.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name} ({s.email})
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Event ID *">
              <Input required type="number" value={eventId} onChange={(e) => setEventId(e.target.value)} placeholder="3" />
            </Field>
            <Field label="Jumlah (Rp) *">
              <Input required type="number" min={1} value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="436500" />
            </Field>
            <Field label="Metode *">
              <Select value={method} onChange={(e) => setMethod(e.target.value as "manual" | "iris")}>
                <option value="manual">Manual (transfer bank + bukti)</option>
                <option value="iris" disabled={irisConfigured === false}>
                  Midtrans Iris{irisConfigured === false ? " (belum dikonfigurasi)" : ""}
                </option>
              </Select>
            </Field>
            <Button disabled={busy || !sellerId || !amount}>
              {busy ? "..." : "Buat Payout"}
            </Button>
          </form>
        </div>

        <div className="lg:col-span-2">
          <div className="mb-3 max-w-xs">
            <Field label="Filter status">
              <Select
                value={statusFilter}
                onChange={(e) => {
                  setLoading(true);
                  setStatusFilter(e.target.value);
                }}
              >
                <option value="">Semua</option>
                <option value="pending">Pending</option>
                <option value="paid">Paid</option>
                <option value="cancelled">Cancelled</option>
              </Select>
            </Field>
          </div>
          {loading ? (
            <TableSkeleton />
          ) : payouts.length === 0 ? (
            <EmptyState message="Belum ada payout." />
          ) : (
            <Table head={["Tanggal", "Seller", "Jumlah", "Metode", "Status", "Aksi"]}>
              {payouts.map((p) => (
                <PayoutRow
                  key={p.id}
                  payout={p}
                  onChanged={load}
                  onMessage={setMessages}
                />
              ))}
            </Table>
          )}
        </div>
      </div>
    </div>
  );
}
