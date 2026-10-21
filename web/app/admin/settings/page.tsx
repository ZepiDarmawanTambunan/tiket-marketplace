"use client";

import { useEffect, useState } from "react";
import type { ChangeEvent } from "react";
import { apiFetch, errMsg, API_URL } from "@/lib/api";
import type { MidtransSettings, NotificationSettings } from "@/lib/types";
import { useAdminStore } from "@/store/admin";
import {
  Button,
  Card,
  ErrorBox,
  Field,
  Input,
  PageHeader,
  Select,
  FormSkeleton,
  SuccessBox,
  Toggle,
} from "@/components/ui";

interface FormState {
  merchant_id: string;
  server_key: string;
  client_key: string;
  environment: string;
  commission_default: string;
  gateway_fee_bearer: string;
  gateway_fee_percent: string;
  gateway_fee_flat_cents: string;
  require_seller_verification: boolean;
  iris_api_key: string;
  iris_from_account: string;
}

function toForm(s: MidtransSettings): FormState {
  return {
    merchant_id: s.merchant_id ?? "",
    server_key: "",
    client_key: "",
    environment: s.environment,
    commission_default: String(s.commission_default),
    gateway_fee_bearer: s.gateway_fee_bearer,
    gateway_fee_percent: String(s.gateway_fee_percent),
    gateway_fee_flat_cents: String(s.gateway_fee_flat_cents),
    require_seller_verification: s.require_seller_verification,
    iris_api_key: "",
    iris_from_account: s.iris_from_account ?? "",
  };
}

function SettingsForm({ initial }: { initial: MidtransSettings }) {
  const { saveSettings } = useAdminStore();
  // Inisialisasi dari props (bukan via effect) agar lolos
  // react-hooks/set-state-in-effect; parent me-remount form via key
  // setiap settings berubah.
  const [form, setForm] = useState<FormState>(() => toForm(initial));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [testResult, setTestResult] = useState("");
  const [testing, setTesting] = useState(false);
  const [copied, setCopied] = useState(false);

  const set =
    (k: keyof FormState) =>
    (e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
      setForm((f) => ({ ...f, [k]: e.target.value }));

  const webhookUrl = `${API_URL}/api/webhooks/midtrans`;

  const save = async () => {
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      // Hanya kirim secret bila diisi (field kosong = tidak diubah)
      const patch: Record<string, unknown> = {
        merchant_id: form.merchant_id,
        environment: form.environment,
        commission_default: Number(form.commission_default),
        gateway_fee_bearer: form.gateway_fee_bearer,
        gateway_fee_percent: Number(form.gateway_fee_percent),
        gateway_fee_flat_cents: Number(form.gateway_fee_flat_cents),
        require_seller_verification: form.require_seller_verification,
      };
      if (form.server_key.trim()) patch.server_key = form.server_key.trim();
      if (form.client_key.trim()) patch.client_key = form.client_key.trim();
      if (form.iris_api_key.trim()) patch.iris_api_key = form.iris_api_key.trim();
      patch.iris_from_account = form.iris_from_account.trim() || null;
      await saveSettings(patch);
      setSuccess("Pengaturan tersimpan.");
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setSaving(false);
    }
  };

  const testConnection = async () => {
    setTesting(true);
    setTestResult("");
    try {
      const d = await apiFetch<{
        ok: boolean;
        http_status?: number;
        message?: string;
        note?: string;
      }>("/api/admin/settings/midtrans/test", { method: "POST" });
      setTestResult(
        d.ok
          ? `✅ Koneksi OK (HTTP ${d.http_status ?? "?"}). ${d.message ?? ""}`
          : `⚠️ ${d.message ?? "Koneksi bermasalah."} ${d.note ?? ""} (HTTP ${d.http_status ?? "?"})`
      );
    } catch (e) {
      setTestResult(`❌ ${errMsg(e)}`);
    } finally {
      setTesting(false);
    }
  };

  const copyWebhook = async () => {
    try {
      await navigator.clipboard.writeText(webhookUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setError("Gagal menyalin — salin manual URL di bawah.");
    }
  };

  return (
    <div className="space-y-6">
      <ErrorBox message={error} />
      <SuccessBox message={success} />

      {/* Midtrans */}
      <Card>
        <h2 className="mb-4 font-semibold text-zinc-900">Midtrans</h2>
        <div className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Merchant ID">
              <Input
                value={form.merchant_id}
                onChange={set("merchant_id")}
                placeholder="G123456789"
              />
            </Field>
            <Field label="Environment">
              <Select value={form.environment} onChange={set("environment")}>
                <option value="sandbox">Sandbox</option>
                <option value="production">Production</option>
              </Select>
            </Field>
          </div>
          <Field
            label="Server Key"
            hint={`Tersimpan: ${initial.server_key_masked} • kosongkan bila tidak ingin mengubah`}
          >
            <Input
              type="password"
              value={form.server_key}
              onChange={set("server_key")}
              placeholder="SB-Mid-server-..."
              autoComplete="off"
            />
          </Field>
          <Field
            label="Client Key"
            hint={`Tersimpan: ${initial.client_key_masked} • kosongkan bila tidak ingin mengubah`}
          >
            <Input
              type="password"
              value={form.client_key}
              onChange={set("client_key")}
              placeholder="SB-Mid-client-..."
              autoComplete="off"
            />
          </Field>
          <div>
            <Button
              variant="secondary"
              onClick={testConnection}
              disabled={testing}
            >
              {testing ? "Menguji..." : "Tes Koneksi"}
            </Button>
            {testResult && (
              <p className="mt-2 text-sm text-zinc-700">{testResult}</p>
            )}
          </div>

          <div className="border-t border-zinc-100 pt-4">
            <h3 className="mb-1 text-sm font-semibold text-zinc-900">
              Midtrans Iris (payout otomatis)
            </h3>
            <p className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800">
              ⚠️ Fitur Iris membutuhkan <b>aktivasi terpisah di dashboard Midtrans</b>{" "}
              (MAP → Settings → Iris). Tanpa aktivasi, pengiriman via Iris akan gagal
              walau kredensial sudah diisi.
            </p>
            <div className="space-y-4">
              <Field
                label="Iris API Key"
                hint={`Tersimpan: ${initial.iris_api_key_masked || "belum diisi"} • kosongkan bila tidak ingin mengubah`}
              >
                <Input
                  type="password"
                  value={form.iris_api_key}
                  onChange={set("iris_api_key")}
                  placeholder="IRIS-..."
                  autoComplete="off"
                />
              </Field>
              <Field
                label="Rekening sumber Iris"
                hint="Nomor rekening yang terdaftar di Iris sebagai sumber dana"
              >
                <Input
                  value={form.iris_from_account}
                  onChange={set("iris_from_account")}
                  placeholder="1234567890"
                />
              </Field>
            </div>
          </div>
          <Field label="Webhook URL (pasang di dashboard Midtrans)">
            <div className="flex gap-2">
              <Input
                readOnly
                value={webhookUrl}
                className="font-mono text-xs"
              />
              <Button variant="secondary" onClick={copyWebhook}>
                {copied ? "Tersalin!" : "Salin"}
              </Button>
            </div>
          </Field>
        </div>
      </Card>

      {/* Komisi & fee */}
      <Card>
        <h2 className="mb-4 font-semibold text-zinc-900">Komisi & Biaya</h2>
        <div className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              label="Komisi platform default (%)"
              hint="Ditampilkan transparan saat checkout"
            >
              <Input
                type="number"
                min={0}
                max={100}
                step="0.1"
                value={form.commission_default}
                onChange={set("commission_default")}
              />
            </Field>
            <Field label="Fee gateway ditanggung oleh">
              <Select
                value={form.gateway_fee_bearer}
                onChange={set("gateway_fee_bearer")}
              >
                <option value="buyer">Pembeli (transparan di checkout)</option>
                <option value="seller">Seller</option>
                <option value="platform">Platform</option>
              </Select>
            </Field>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Fee gateway (%)">
              <Input
                type="number"
                min={0}
                step="0.1"
                value={form.gateway_fee_percent}
                onChange={set("gateway_fee_percent")}
              />
            </Field>
            <Field label="Fee gateway flat (Rp)">
              <Input
                type="number"
                min={0}
                value={form.gateway_fee_flat_cents}
                onChange={set("gateway_fee_flat_cents")}
              />
            </Field>
          </div>
        </div>
      </Card>

      {/* Kebijakan */}
      <Card>
        <h2 className="mb-4 font-semibold text-zinc-900">Kebijakan</h2>
        <Toggle
          checked={form.require_seller_verification}
          onChange={(v) =>
            setForm((f) => ({ ...f, require_seller_verification: v }))
          }
          label="Wajibkan verifikasi seller sebelum jualan"
        />
        <p className="mt-1 text-xs text-zinc-500">
          Bila aktif, seller harus terverifikasi admin sebelum bisa membuat
          event. Bila nonaktif (default), verifikasi hanya sebagai lencana
          kepercayaan.
        </p>
      </Card>

      <Button onClick={save} disabled={saving} className="px-8">
        {saving ? "Menyimpan..." : "Simpan Pengaturan"}
      </Button>
    </div>
  );
}

/** Fase 2: Notifikasi WhatsApp */
function NotificationForm() {
  const [provider, setProvider] = useState("fonnte");
  const [token, setToken] = useState("");
  const [enabled, setEnabled] = useState(false);
  const [tokenMasked, setTokenMasked] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testTo, setTestTo] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [testResult, setTestResult] = useState("");

  useEffect(() => {
    let cancelled = false;
    apiFetch<NotificationSettings>("/api/admin/settings/notifications")
      .then((s) => {
        if (cancelled) return;
        setProvider(s.wa_provider || "fonnte");
        setEnabled(s.wa_enabled);
        setTokenMasked(s.wa_fonnte_token_masked ?? "");
      })
      .catch((e) => {
        if (!cancelled) setError(errMsg(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const save = async () => {
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      const patch: Record<string, unknown> = {
        wa_provider: provider,
        wa_enabled: enabled,
      };
      if (token.trim()) patch.wa_fonnte_token = token.trim();
      const d = await apiFetch<NotificationSettings>(
        "/api/admin/settings/notifications",
        { method: "PUT", body: JSON.stringify(patch) }
      );
      setTokenMasked(d.wa_fonnte_token_masked ?? "");
      setToken("");
      setSuccess("Pengaturan notifikasi tersimpan.");
    } catch (e) {
      setError(errMsg(e));
    } finally {
      setSaving(false);
    }
  };

  const sendTest = async () => {
    setTesting(true);
    setTestResult("");
    try {
      const d = await apiFetch<{ ok: boolean; message?: string }>(
        "/api/admin/settings/notifications/test",
        { method: "POST", body: JSON.stringify({ to: testTo.trim() }) }
      );
      setTestResult(
        d.ok
          ? `✅ ${d.message ?? "Pesan tes terkirim."}`
          : `⚠️ ${d.message ?? "Gagal mengirim pesan tes."}`
      );
    } catch (e) {
      setTestResult(`❌ ${errMsg(e)}`);
    } finally {
      setTesting(false);
    }
  };

  if (loading) return <FormSkeleton fields={3} />;

  return (
    <div className="space-y-4">
      <ErrorBox message={error} />
      <SuccessBox message={success} />
      <Field label="Provider WhatsApp">
        <Select value={provider} onChange={(e) => setProvider(e.target.value)}>
          <option value="fonnte">Fonnte</option>
        </Select>
      </Field>
      <Field
        label="Token Fonnte"
        hint={`Tersimpan: ${tokenMasked || "belum diisi"} • kosongkan bila tidak ingin mengubah`}
      >
        <Input
          type="password"
          value={token}
          onChange={(e) => setToken(e.target.value)}
          placeholder="token fonnte..."
          autoComplete="off"
        />
      </Field>
      <Toggle
        checked={enabled}
        onChange={setEnabled}
        label="Aktifkan notifikasi WhatsApp"
      />
      <p className="text-xs text-zinc-500">
        Bila aktif, platform mengirim notifikasi WA (tiket diterbitkan, status
        order, dsb.) melalui provider di atas.
      </p>
      <Button variant="secondary" onClick={save} disabled={saving}>
        {saving ? "Menyimpan..." : "Simpan Notifikasi"}
      </Button>

      <div className="border-t border-zinc-100 pt-4">
        <h3 className="mb-2 text-sm font-semibold text-zinc-900">Kirim Tes</h3>
        <div className="flex gap-2">
          <Input
            value={testTo}
            onChange={(e) => setTestTo(e.target.value)}
            placeholder="Nomor WA, mis. 6281234567890"
            className="max-w-xs"
          />
          <Button
            variant="secondary"
            onClick={sendTest}
            disabled={testing || !testTo.trim()}
          >
            {testing ? "Mengirim..." : "Kirim Tes"}
          </Button>
        </div>
        {testResult && <p className="mt-2 text-sm text-zinc-700">{testResult}</p>}
      </div>
    </div>
  );
}

export default function AdminSettingsPage() {
  const { settings, loading, fetchSettings } = useAdminStore();
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    void Promise.resolve().then(async () => {
      try {
        await fetchSettings();
      } catch (e) {
        if (!cancelled) setError(errMsg(e));
      }
    });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="max-w-3xl">
      <PageHeader
        title="Pengaturan"
        subtitle="Midtrans, komisi, dan kebijakan platform"
      />
      <ErrorBox message={error} />
      {loading || !settings ? (
        <FormSkeleton />
      ) : (
        <SettingsForm key={settings.updated_at} initial={settings} />
      )}
      <Card>
        <h2 className="mb-4 font-semibold text-zinc-900">Notifikasi WhatsApp</h2>
        <NotificationForm />
      </Card>
    </div>
  );
}
