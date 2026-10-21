"use client";

import { useState } from "react";
import type { FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { apiFetch, errMsg } from "@/lib/api";
import type { User } from "@/lib/types";
import { useAuthStore } from "@/store/auth";
import { Button, Card, ErrorBox, Field, Input, Toggle } from "@/components/ui";

export default function RegisterPage() {
  const router = useRouter();
  const login = useAuthStore((s) => s.login);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [asSeller, setAsSeller] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const d = await apiFetch<{ user: User; token: string }>(
        "/api/auth/register",
        {
          method: "POST",
          body: JSON.stringify({
            name,
            email,
            phone: phone || undefined,
            password,
            role: asSeller ? "seller" : "buyer",
          }),
        }
      );
      login(d.user, d.token);
      router.push(asSeller ? "/seller" : "/");
    } catch (err) {
      setError(errMsg(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto max-w-md px-4 py-12">
      <Card>
        <h1 className="mb-1 text-2xl font-bold text-zinc-900">Daftar</h1>
        <p className="mb-6 text-sm text-zinc-500">
          Buat akun untuk membeli tiket — atau jualan sebagai seller.
        </p>
        <ErrorBox message={error} />
        <form onSubmit={submit} className="space-y-4">
          <Field label="Nama lengkap">
            <Input
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Nama Anda"
              autoComplete="name"
            />
          </Field>
          <Field label="Email">
            <Input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="nama@email.com"
              autoComplete="email"
            />
          </Field>
          <Field label="No. HP (opsional)">
            <Input
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="08xxxxxxxxxx"
              autoComplete="tel"
            />
          </Field>
          <Field label="Kata sandi">
            <Input
              type="password"
              required
              minLength={6}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Minimal 6 karakter"
              autoComplete="new-password"
            />
          </Field>
          <div className="rounded-lg border border-zinc-200 bg-zinc-50 p-3">
            <Toggle
              checked={asSeller}
              onChange={setAsSeller}
              label="Daftar sebagai seller (jual tiket event)"
            />
            <p className="mt-1 text-xs text-zinc-500">
              Verifikasi seller bersifat opsional — Anda bisa langsung membuat
              event setelah daftar.
            </p>
          </div>
          <Button className="w-full" disabled={loading}>
            {loading ? "Memproses..." : "Buat Akun"}
          </Button>
        </form>
        <p className="mt-4 text-center text-sm text-zinc-500">
          Sudah punya akun?{" "}
          <Link href="/login" className="font-medium text-orange-600 hover:underline">
            Masuk
          </Link>
        </p>
      </Card>
    </div>
  );
}
