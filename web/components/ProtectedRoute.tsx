"use client";
import type { ReactNode } from "react";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { Skeleton, SkeletonText } from "@/components/ui";
import type { Role } from "@/lib/types";

export default function ProtectedRoute({
  roles,
  children,
}: {
  roles?: Role[];
  children: ReactNode;
}) {
  const { user, token } = useAuthStore();
  const router = useRouter();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    // Ditandai async (microtask) agar tidak memicu
    // react-hooks/set-state-in-effect. Mencegah flash konten sebelum
    // token dari localStorage (zustand persist) terbaca.
    let on = true;
    queueMicrotask(() => {
      if (on) setReady(true);
    });
    return () => {
      on = false;
    };
  }, []);

  useEffect(() => {
    if (!ready) return;
    if (!token) {
      router.replace("/login");
    } else if (roles && user && !roles.includes(user.role)) {
      router.replace("/");
    }
  }, [ready, token, user, roles, router]);

  if (!ready || !token) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-8">
        <Skeleton className="h-8 w-56" />
        <div className="mt-6">
          <SkeletonText lines={4} />
        </div>
      </div>
    );
  }

  if (roles && user && !roles.includes(user.role)) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-16 text-center">
        <h1 className="text-xl font-bold text-zinc-900">Akses Ditolak</h1>
        <p className="mt-2 text-sm text-zinc-500">
          Halaman ini membutuhkan peran: {roles.join(", ")}.
        </p>
      </div>
    );
  }

  return <>{children}</>;
}
