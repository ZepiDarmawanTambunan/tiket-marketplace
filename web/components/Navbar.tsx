"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";

function linkCls(pathname: string, href: string) {
  const active =
    href === "/" ? pathname === "/" : pathname.startsWith(href);
  return `rounded-lg px-3 py-2 text-sm font-medium transition ${
    active
      ? "bg-orange-100 text-orange-700"
      : "text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900"
  }`;
}

export default function Navbar() {
  const { user, logout } = useAuthStore();
  const pathname = usePathname();
  const router = useRouter();

  const handleLogout = () => {
    logout();
    router.push("/");
  };

  return (
    <header className="sticky top-0 z-40 border-b border-zinc-200 bg-white/90 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-4">
        <Link href="/" className="flex items-center gap-2">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-orange-600 text-lg font-bold text-white">
            T
          </span>
          <span className="text-lg font-bold text-zinc-900">TiketHub</span>
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          <Link href="/" className={linkCls(pathname, "/")}>
            Beranda
          </Link>
          <Link href="/events" className={linkCls(pathname, "/events")}>
            Jelajah Event
          </Link>
          {user && (
            <>
              <Link href="/tickets" className={linkCls(pathname, "/tickets")}>
                Tiket Saya
              </Link>
              <Link href="/orders" className={linkCls(pathname, "/orders")}>
                Riwayat
              </Link>
            </>
          )}
          {user?.role === "seller" && (
            <Link href="/seller" className={linkCls(pathname, "/seller")}>
              Dashboard Seller
            </Link>
          )}
          {user?.role === "super_admin" && (
            <Link href="/admin" className={linkCls(pathname, "/admin")}>
              Admin
            </Link>
          )}
        </nav>

        <div className="flex items-center gap-2">
          {user ? (
            <>
              <Link
                href="/profile"
                className="hidden rounded-lg px-3 py-2 text-sm font-medium text-zinc-700 hover:bg-zinc-100 sm:block"
              >
                {user.name}
                {user.role === "seller" && !user.seller_verified && (
                  <span className="ml-1 text-xs text-amber-600">(belum verif)</span>
                )}
              </Link>
              <button
                onClick={handleLogout}
                className="rounded-lg bg-zinc-100 px-3 py-2 text-sm font-medium text-zinc-700 hover:bg-zinc-200"
              >
                Keluar
              </button>
            </>
          ) : (
            <>
              <Link
                href="/login"
                className="rounded-lg px-3 py-2 text-sm font-medium text-zinc-700 hover:bg-zinc-100"
              >
                Masuk
              </Link>
              <Link
                href="/register"
                className="rounded-lg bg-orange-600 px-3 py-2 text-sm font-medium text-white hover:bg-orange-700"
              >
                Daftar
              </Link>
            </>
          )}
        </div>
      </div>

      {/* Nav mobile */}
      <nav className="flex gap-1 no-scrollbar overflow-x-auto border-t border-zinc-100 px-4 py-2 md:hidden">
        <Link href="/" className={linkCls(pathname, "/")}>
          Beranda
        </Link>
        <Link href="/events" className={linkCls(pathname, "/events")}>
          Jelajah
        </Link>
        {user && (
          <>
            <Link href="/tickets" className={linkCls(pathname, "/tickets")}>
              Tiket
            </Link>
            <Link href="/orders" className={linkCls(pathname, "/orders")}>
              Riwayat
            </Link>
          </>
        )}
        {user?.role === "seller" && (
          <Link href="/seller" className={linkCls(pathname, "/seller")}>
            Seller
          </Link>
        )}
        {user?.role === "super_admin" && (
          <Link href="/admin" className={linkCls(pathname, "/admin")}>
            Admin
          </Link>
        )}
      </nav>
    </header>
  );
}
