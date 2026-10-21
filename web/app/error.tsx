"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui";

export default function ErrorPage({
  error,
  retry,
}: {
  error: Error & { digest?: string };
  retry: () => void;
}) {
  const router = useRouter();
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="mx-auto flex max-w-xl flex-col items-center px-4 py-24 text-center">
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-red-50 text-3xl">
        ⚠️
      </div>
      <h1 className="mt-4 text-2xl font-bold text-zinc-900">
        Terjadi kesalahan
      </h1>
      <p className="mt-2 text-sm text-zinc-500">
        Halaman ini gagal dimuat. Coba lagi — bila masih bermasalah, muat ulang
        halaman atau kembali beberapa saat lagi.
      </p>
      {error.digest && (
        <p className="mt-2 font-mono text-xs text-zinc-400">
          Kode error: {error.digest}
        </p>
      )}
      <div className="mt-6 flex gap-2">
        <Button onClick={() => retry()}>Coba Lagi</Button>
        <Button variant="secondary" onClick={() => router.push("/")}>
          Ke Beranda
        </Button>
      </div>
    </div>
  );
}
