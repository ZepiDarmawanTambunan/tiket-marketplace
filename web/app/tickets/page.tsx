"use client";

import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { QRCodeSVG } from "qrcode.react";
import { apiFetch, errMsg } from "@/lib/api";
import type { Ticket, TicketTransfer } from "@/lib/types";
import { formatDateTime, formatRupiah } from "@/lib/format";
import ProtectedRoute from "@/components/ProtectedRoute";
import {
  Badge,
  Button,
  Card,
  EmptyState,
  ErrorBox,
  Field,
  Input,
  PageHeader,
  CardListSkeleton,
  SuccessBox,
  Table,
  Td,
} from "@/components/ui";

function TransferModal({
  ticket,
  onDone,
  onClose,
}: {
  ticket: Ticket;
  onDone: () => void;
  onClose: () => void;
}) {
  const [email, setEmail] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (!confirming) {
      setConfirming(true);
      return;
    }
    setBusy(true);
    setError("");
    try {
      await apiFetch("/api/tickets/transfer", {
        method: "POST",
        body: JSON.stringify({
          ticket_code: ticket.ticket_code,
          to_email: email.trim(),
        }),
      });
      setSuccess(`Tiket ${ticket.ticket_code} berhasil ditransfer ke ${email.trim()}.`);
      onDone();
    } catch (err) {
      setError(errMsg(err));
      setConfirming(false);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-bold text-zinc-900">Transfer Tiket</h2>
        <p className="mt-1 text-sm text-zinc-500">
          {ticket.event_title} • {ticket.ticket_type_name} •{" "}
          <span className="font-mono text-xs">{ticket.ticket_code}</span>
        </p>
        <form onSubmit={submit} className="mt-4 space-y-3">
          <ErrorBox message={error} />
          <SuccessBox message={success} />
          {success ? (
            <Button className="w-full" type="button" onClick={onClose}>
              Tutup
            </Button>
          ) : confirming ? (
            <>
              <p className="rounded-lg bg-amber-50 border border-amber-200 px-3 py-2 text-sm text-amber-800">
                Transfer tiket <b>{ticket.ticket_code}</b> ke <b>{email.trim()}</b>?
                Setelah ditransfer, tiket tidak lagi menjadi milikmu.
              </p>
              <div className="flex gap-2">
                <Button disabled={busy} className="flex-1">
                  {busy ? "Memproses..." : "Ya, Transfer"}
                </Button>
                <Button
                  variant="ghost"
                  type="button"
                  onClick={() => setConfirming(false)}
                >
                  Batal
                </Button>
              </div>
            </>
          ) : (
            <>
              <Field
                label="Email penerima *"
                hint="Tiket berpindah ke akun dengan email ini"
              >
                <Input
                  required
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="penerima@mail.com"
                />
              </Field>
              <div className="flex gap-2">
                <Button disabled={!email.trim()} className="flex-1">
                  Lanjut
                </Button>
                <Button variant="ghost" type="button" onClick={onClose}>
                  Batal
                </Button>
              </div>
            </>
          )}
        </form>
      </div>
    </div>
  );
}

function TicketCard({
  ticket,
  onTransfer,
}: {
  ticket: Ticket;
  onTransfer: (t: Ticket) => void;
}) {
  return (
    <Card className="text-center">
      <div className="mb-3 flex items-center justify-between text-left">
        <Badge
          status={ticket.status}
          label={ticket.status === "unused" ? "Belum dipakai" : "Sudah dipakai"}
        />
        <span className="font-mono text-xs text-zinc-500">
          {ticket.ticket_code}
        </span>
      </div>
      <div className="mx-auto w-fit rounded-lg border border-zinc-200 bg-white p-3">
        <QRCodeSVG value={ticket.qr_payload} size={160} />
      </div>
      <h3 className="mt-3 font-semibold text-zinc-900">{ticket.event_title}</h3>
      <p className="text-xs text-zinc-500">
        {ticket.ticket_type_name} &middot; {formatDateTime(ticket.event_starts_at)}
      </p>
      <p className="mt-1 text-sm font-bold text-zinc-900">
        {formatRupiah(ticket.price_cents)}
      </p>
      <p className="mt-2 text-xs text-zinc-400">
        Order {ticket.order_code}
        {ticket.used_at && ` • dipakai ${formatDateTime(ticket.used_at)}`}
      </p>
      {ticket.status === "unused" && (
        <Button
          variant="secondary"
          className="mt-3 w-full"
          onClick={() => onTransfer(ticket)}
        >
          Transfer Tiket
        </Button>
      )}
    </Card>
  );
}

function TicketsContent() {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [transfers, setTransfers] = useState<TicketTransfer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [tab, setTab] = useState<"tickets" | "history">("tickets");
  const [transferFor, setTransferFor] = useState<Ticket | null>(null);

  const loadTickets = () => {
    setLoading(true);
    apiFetch<{ tickets: Ticket[] }>("/api/tickets")
      .then((d) => setTickets(d.tickets))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  };

  const loadTransfers = () => {
    apiFetch<{ transfers: TicketTransfer[] }>("/api/tickets/transfers")
      .then((d) => setTransfers(d.transfers))
      .catch((e) => setError(errMsg(e)));
  };

  useEffect(() => {
    void Promise.resolve().then(() => {
      loadTickets();
      loadTransfers();
    });
  }, []);

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <PageHeader
        title="Tiket Saya"
        subtitle="Tunjukkan QR ini ke panitia saat check-in"
      />
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      <div className="mb-6 flex gap-1 border-b border-zinc-200 pb-2">
        {(
          [
            ["tickets", "Tiket Saya"],
            ["history", "Riwayat Transfer"],
          ] as const
        ).map(([key, label]) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`rounded-lg px-3 py-2 text-sm font-medium ${
              tab === key
                ? "bg-orange-100 text-orange-700"
                : "text-zinc-600 hover:bg-zinc-100"
            }`}
          >
            {label}
            {key === "history" && transfers.length > 0 && (
              <span className="ml-1.5 rounded-full bg-zinc-200 px-1.5 text-xs">
                {transfers.length}
              </span>
            )}
          </button>
        ))}
      </div>

      {tab === "tickets" ? (
        loading ? (
          <CardListSkeleton />
        ) : tickets.length === 0 ? (
          <EmptyState message="Belum ada tiket. Tiket muncul di sini setelah pembayaran lunas." />
        ) : (
          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {tickets.map((t) => (
              <TicketCard key={t.ticket_code} ticket={t} onTransfer={setTransferFor} />
            ))}
          </div>
        )
      ) : transfers.length === 0 ? (
        <EmptyState message="Belum ada riwayat transfer tiket." />
      ) : (
        <Table head={["Tanggal", "Kode Tiket", "Penerima", "Status"]}>
          {transfers.map((tr) => (
            <tr key={tr.id}>
              <Td className="whitespace-nowrap text-xs">
                {formatDateTime(tr.created_at)}
              </Td>
              <Td className="font-mono text-xs">{tr.ticket_code}</Td>
              <Td className="text-xs">{tr.to_user_name ?? `#${tr.to_user_id}`}</Td>
              <Td>
                <Badge status="paid" label="Terkirim" />
              </Td>
            </tr>
          ))}
        </Table>
      )}

      {transferFor && (
        <TransferModal
          ticket={transferFor}
          onClose={() => setTransferFor(null)}
          onDone={() => {
            setSuccess("Transfer berhasil. Tiket sudah berpindah pemilik.");
            loadTickets();
            loadTransfers();
          }}
        />
      )}
    </div>
  );
}

export default function TicketsPage() {
  return (
    <ProtectedRoute>
      <TicketsContent />
    </ProtectedRoute>
  );
}
