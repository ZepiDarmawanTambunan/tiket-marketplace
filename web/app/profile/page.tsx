"use client";

import { useEffect, useState } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { User } from "@/lib/types";
import { formatDateTime } from "@/lib/format";
import { useAuthStore } from "@/store/auth";
import ProtectedRoute from "@/components/ProtectedRoute";
import {
  Badge,
  Button,
  Card,
  ErrorBox,
  PageHeader,
  FormSkeleton,
} from "@/components/ui";

function ProfileContent() {
  const { user, setUser } = useAuthStore();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<{ user: User }>("/api/auth/me")
      .then((d) => setUser(d.user))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  }, [setUser]);

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <PageHeader title="Profil Saya" />
      <ErrorBox message={error} />
      {loading || !user ? (
        <FormSkeleton />
      ) : (
        <Card>
          <dl className="space-y-3 text-sm">
            <div className="flex justify-between border-b border-zinc-100 pb-3">
              <dt className="text-zinc-500">Nama</dt>
              <dd className="font-medium text-zinc-900">{user.name}</dd>
            </div>
            <div className="flex justify-between border-b border-zinc-100 pb-3">
              <dt className="text-zinc-500">Email</dt>
              <dd className="font-medium text-zinc-900">{user.email}</dd>
            </div>
            <div className="flex justify-between border-b border-zinc-100 pb-3">
              <dt className="text-zinc-500">No. HP</dt>
              <dd className="font-medium text-zinc-900">{user.phone || "-"}</dd>
            </div>
            <div className="flex justify-between border-b border-zinc-100 pb-3">
              <dt className="text-zinc-500">Peran</dt>
              <dd>
                <Badge
                  status={user.role === "buyer" ? "published" : "pending_review"}
                  label={
                    user.role === "super_admin"
                      ? "Super Admin"
                      : user.role === "seller"
                        ? "Seller"
                        : "Buyer"
                  }
                />
              </dd>
            </div>
            {user.role === "seller" && (
              <div className="flex justify-between border-b border-zinc-100 pb-3">
                <dt className="text-zinc-500">Verifikasi seller</dt>
                <dd>
                  <Badge
                    status={user.seller_verified ? "paid" : "pending"}
                    label={user.seller_verified ? "Terverifikasi" : "Belum terverifikasi"}
                  />
                </dd>
              </div>
            )}
            <div className="flex justify-between">
              <dt className="text-zinc-500">Terdaftar sejak</dt>
              <dd className="font-medium text-zinc-900">
                {formatDateTime(user.created_at)}
              </dd>
            </div>
          </dl>

          {user.role === "buyer" && (
            <div className="mt-6 rounded-lg bg-orange-50 p-4 text-sm text-orange-900">
              <p className="font-medium">Ingin jualan tiket?</p>
              <p className="mt-1 text-orange-800/80">
                Hubungi admin untuk mengubah peran akun Anda menjadi seller,
                atau daftar akun baru sebagai seller.
              </p>
            </div>
          )}

          <div className="mt-6">
            <Button variant="secondary" onClick={() => window.location.reload()}>
              Muat Ulang Data
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
}

export default function ProfilePage() {
  return (
    <ProtectedRoute>
      <ProfileContent />
    </ProtectedRoute>
  );
}
