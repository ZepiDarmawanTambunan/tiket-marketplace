"use client";

import dynamic from "next/dynamic";
import type { FormEvent } from "react";
import { useCallback, useRef, useState } from "react";
import { apiFetch, ApiError, errMsg } from "@/lib/api";
import type { CheckinResult } from "@/lib/types";
import { formatDateTime } from "@/lib/format";
import {
  Button,
  Card,
  ErrorBox,
  Field,
  Input,
  PageHeader,
  Toggle,
} from "@/components/ui";

const QrScanner = dynamic(() => import("@/components/QrScanner"), {
  ssr: false,
  loading: () => <p className="py-8 text-center text-sm text-zinc-500">Memuat kamera...</p>,
});

export default function ScannerPage() {
  const [useCamera, setUseCamera] = useState(true);
  const [manual, setManual] = useState("");
  const [result, setResult] = useState<CheckinResult | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const lastScan = useRef<string>("");
  const lastTime = useRef<number>(0);

  const checkin = useCallback(async (raw: string) => {
    const code = raw.trim();
    if (!code || busy) return;
    // Hindari double-scan dari frame kamera yang sama
    const now = Date.now();
    if (code === lastScan.current && now - lastTime.current < 5000) return;
    lastScan.current = code;
    lastTime.current = now;

    setBusy(true);
    setError("");
    setResult(null);
    try {
      const body = code.includes(".")
        ? { qr_payload: code }
        : { ticket_code: code };
      const { ...data } = await apiFetch<CheckinResult>("/api/checkin", {
        method: "POST",
        body: JSON.stringify(body),
      });
      setResult(data);
    } catch (e) {
      if (e instanceof ApiError && e.code === "ALREADY_USED") {
        setError("Tiket ini SUDAH dipakai sebelumnya (double-scan ditolak).");
      } else if (e instanceof ApiError && e.status === 403) {
        setError("Tiket ini bukan dari event milik Anda.");
      } else if (e instanceof ApiError && e.status === 422) {
        setError("Order tiket ini belum lunas.");
      } else {
        setError(errMsg(e));
      }
    } finally {
      setBusy(false);
    }
  }, [busy]);

  const submitManual = (e: FormEvent) => {
    e.preventDefault();
    checkin(manual);
  };

  return (
    <div className="max-w-2xl">
      <PageHeader
        title="Scanner Tiket"
        subtitle="Scan QR tiket pengunjung untuk check-in"
      />

      <div className="mb-4">
        <Toggle
          checked={useCamera}
          onChange={setUseCamera}
          label="Gunakan kamera"
        />
      </div>

      {useCamera ? (
        <Card>
          <QrScanner onScan={(t) => checkin(t)} onError={(m) => setError(m)} />
          <p className="mt-2 text-center text-xs text-zinc-500">
            Arahkan kamera ke QR tiket pengunjung
          </p>
        </Card>
      ) : (
        <Card>
          <form onSubmit={submitManual} className="flex gap-2">
            <div className="flex-1">
              <Field label="Kode tiket / isi QR">
                <Input
                  value={manual}
                  onChange={(e) => setManual(e.target.value)}
                  placeholder="TIX-XXXXXXXX"
                />
              </Field>
            </div>
            <div className="flex items-end">
              <Button disabled={busy || !manual.trim()}>
                {busy ? "..." : "Check-in"}
              </Button>
            </div>
          </form>
        </Card>
      )}

      <div className="mt-4">
        <ErrorBox message={error} />
        {result && (
          <Card className="border-green-300 bg-green-50">
            <p className="text-lg font-bold text-green-800">✅ Check-in berhasil</p>
            <dl className="mt-3 space-y-1 text-sm">
              <div className="flex justify-between">
                <dt className="text-green-700">Kode tiket</dt>
                <dd className="font-mono font-medium text-green-900">{result.ticket_code}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-green-700">Pemegang</dt>
                <dd className="font-medium text-green-900">{result.holder_name}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-green-700">Event</dt>
                <dd className="text-green-900">{result.event_title}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-green-700">Tipe</dt>
                <dd className="text-green-900">{result.ticket_type_name}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-green-700">Waktu check-in</dt>
                <dd className="text-green-900">{formatDateTime(result.checked_in_at)}</dd>
              </div>
            </dl>
            <Button variant="secondary" className="mt-4" onClick={() => setResult(null)}>
              Scan Berikutnya
            </Button>
          </Card>
        )}
        {busy && !result && (
          <p className="py-4 text-center text-sm text-zinc-500">Memverifikasi tiket...</p>
        )}
      </div>

    </div>
  );
}
