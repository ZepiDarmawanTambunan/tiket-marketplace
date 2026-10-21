import Link from "next/link";

export default function NotFound() {
  return (
    <div className="mx-auto flex max-w-xl flex-col items-center px-4 py-24 text-center">
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-orange-50 text-3xl">
        🎟️
      </div>
      <h1 className="mt-4 text-2xl font-bold text-zinc-900">
        Halaman tidak ditemukan
      </h1>
      <p className="mt-2 text-sm text-zinc-500">
        Alamat yang kamu tuju tidak ada atau sudah dipindahkan.
      </p>
      <div className="mt-6 flex gap-2">
        <Link
          href="/events"
          className="rounded-lg bg-orange-600 px-4 py-2 text-sm font-medium text-white hover:bg-orange-700"
        >
          Jelajah Event
        </Link>
        <Link
          href="/"
          className="rounded-lg bg-zinc-100 px-4 py-2 text-sm font-medium text-zinc-900 hover:bg-zinc-200"
        >
          Ke Beranda
        </Link>
      </div>
    </div>
  );
}
