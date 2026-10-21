# Tiket Marketplace — Panduan Development (Fase 1)

Monorepo: `backend/` (Go REST API) + `web/` (Next.js). Database MySQL `tiket_db`.

Keputusan user yang sudah diimplementasikan: komisi platform default **3%**,
fee gateway Midtrans default ditanggung **pembeli** (tampil transparan sebagai
"Biaya layanan" saat checkout), verifikasi seller **opsional**
(`require_seller_verification`, default OFF), event **wajib approve admin**
sebelum tayang, Midtrans **sandbox** untuk development.

## Prasyarat

- Go 1.23+: `export PATH=$HOME/go-dist/bin:$PATH` (JANGAN install ke /usr/local)
- Node 24 + npm 10 (`node --version`, `npm --version`)
- MySQL 8 berjalan di `127.0.0.1:3306`. Bila belum jalan:
  `setsid nohup ~/mysql/start-mysqld.sh > ~/mysql-data/mysqld.log 2>&1 < /dev/null &`
- Untuk `npm install` / `npm run build` yang berat:
  `export TMPDIR=$HOME/workspace/.tmp` (jangan pakai /tmp, tmpfs 512MB)

## Database

Database `tiket_db` + user `tiket` **sudah dibuat**. DSN tersimpan di
`.db.env` (chmod 600, terdaftar di `.gitignore` — JANGAN di-commit):

```bash
set -a; source ~/workspace/tiket-marketplace/.db.env; set +a
```

## Backend (port 8080)

```bash
export PATH=$HOME/go-dist/bin:$PATH
cd ~/workspace/tiket-marketplace/backend
cp .env.example .env   # lalu isi JWT_SECRET, SETTINGS_MASTER_KEY, QR_SECRET (acak, min 32 char)

set -a; source ../.db.env; set +a
set -a; source .env; set +a

go run ./cmd/migrate up   # sekali saja (atau tiap ada migrasi baru)
go run ./cmd/seed         # sekali saja; buat super admin + settings default
go run ./cmd/api          # :8080 — cek curl http://localhost:8080/health
```

Verifikasi: `go build ./...`, `go vet ./...`, `gofmt -l .` (harus kosong).
Kontrak endpoint lengkap: `backend/API.md`.

### Kredensial seed

- Super admin: `admin@tiket.local` / `admin123`
  (dari env `SEED_ADMIN_PASSWORD` bila di-set; **WAJIB diganti setelah login pertama**)
- Settings default dari seed: `commission_default=3`,
  `require_seller_verification=false`, `gateway_fee_bearer=buyer`,
  `midtrans_environment=sandbox`, keys = `DUMMY-*`.

### Kunci Midtrans asli (untuk uji bayar nyata)

Daftar di dashboard.midtrans.com → ambil **Server Key** + **Client Key** sandbox,
lalu isi lewat salah satu cara:

1. Panel admin → Pengaturan → Midtrans (UI, terenkripsi AES-256-GCM di DB), atau
2. `PUT /api/admin/settings/midtrans` sebagai super_admin, atau
3. Env saat seed: `MIDTRANS_SERVER_KEY=... MIDTRANS_CLIENT_KEY=... go run ./cmd/seed`

Tanpa key asli, checkout tetap jalan dengan `snap_token: "DUMMY"` (mode
development) — order dibuat, tapi pembayaran Midtrans tidak bisa diuji.

## Frontend (port 3000)

```bash
cd ~/workspace/tiket-marketplace/web
export TMPDIR=$HOME/workspace/.tmp
cp .env.example .env.local   # NEXT_PUBLIC_API_URL=http://localhost:8080 (default)
npm install
npm run dev                  # http://localhost:3000
```

Verifikasi: `npm run lint` (harus 0 error), `npm run build` (harus sukses).

Halaman penting: `/` beranda, `/events` jelajah, `/checkout`,
`/tickets` (QR tiket), `/seller/scanner` (butuh HTTPS/localhost + izin kamera;
ada fallback input manual), `/admin/settings` (form Midtrans + tombol Tes
Koneksi + webhook URL siap-copy), `/admin/events` (moderasi).

## Cara test alur end-to-end (dengan key dummy)

1. Register seller: `POST /api/auth/register`
   `{"name":"S","email":"s@t.local","phone":"081","password":"pass1234","role":"seller"}`
2. Login → buat event `POST /api/events` → tambah tipe tiket
   `POST /api/events/:id/ticket-types` → ajukan `POST /api/events/:id/submit`
3. Login sebagai `admin@tiket.local` → `POST /api/admin/events/:id/approve`
4. Register buyer → `POST /api/orders {"event_id":..,"items":[{"ticket_type_id":..,"qty":2}]}`
   → response berisi rincian subtotal + komisi 3% + biaya layanan + total
5. Simulasi webhook Midtrans (ganti ORDER_CODE, GROSS_CENTS, dan SERVER_KEY):

```bash
SIG=$(python3 -c "import hashlib; print(hashlib.sha512(('ORDER_CODE'+'200'+'GROSS_CENTS'+'SERVER_KEY').encode()).hexdigest())")
curl -X POST http://localhost:8080/api/webhooks/midtrans -H 'Content-Type: application/json' \
  -d "{\"order_id\":\"ORDER_CODE\",\"status_code\":\"200\",\"gross_amount\":\"GROSS_CENTS\",\"signature_key\":\"$SIG\",\"transaction_status\":\"settlement\",\"transaction_id\":\"trx-1\",\"payment_type\":\"qris\"}"
```

6. `GET /api/tickets` (buyer) → tiket + `qr_payload` (format `<kode>.<hmac>`)
7. Check-in: `POST /api/checkin {"ticket_code":"..."}` (token seller) →
   scan kedua untuk kode yang sama harus `409 ALREADY_USED`

## Catatan / gap yang diketahui

- Refund approve di Fase 1 hanya mengubah status; mutasi dana aktual via
  dashboard Midtrans.
- Jendela waktu `refund_policy` belum di-parse otomatis (teks bebas per event).
- Belum ada rate limiting di endpoint publik (auth, webhook).
- `FinishExpired()` (event → finished) ada di service tapi belum ada
  scheduler/endpoint — putuskan: cron internal atau tombol admin.
- Data uji dari smoke test masih ada di `tiket_db`
  (`seller1@test.local`, `buyer1@test.local`, event "Konser Smoke Test").
  Bersihkan bila mengganggu:
  `DELETE FROM tiket_db.users WHERE email LIKE '%@test.local';`
  (order/tiket terkait ikut terhapus via FK — cek dulu relasinya).
- `SETTINGS_MASTER_KEY`, `JWT_SECRET`, `QR_SECRET` wajib SAMA antara proses
  `seed` dan `api`; kalau diganti, isi ulang Midtrans keys via panel admin.

---

# Fase 2 (Development) — 2026-10-07

Fitur baru: payout otomatis via Midtrans Iris (+ fallback manual), refund via
Midtrans Refund API (+ fallback manual), kode promo/diskon, rating & ulasan
event, transfer tiket antar pengguna, notifikasi WhatsApp (provider Fonnte).

## Migrasi

```bash
go run ./cmd/migrate up   # menerapkan 003_phase2 (promo_codes, promo_usages,
                          # reviews, ticket_transfers + kolom baru)
```

## 1. Payout via Midtrans Iris

Iris **tidak ada** di dashboard Midtrans (Snap) — tidak ada menu Iris/Payouts di
sana. Iris adalah produk terpisah dengan portal sendiri: aktivasi harus lewat
**tim support/sales Midtrans** (minta aktivasi Iris + IRIS API Key untuk Creator
& Approver, sandbox + production). Setelah disetujui, key diambil dari dashboard
Iris — **beda total** dengan server key payment gateway.

Isi lewat panel admin → Pengaturan → Midtrans: `iris_api_key` (terenkripsi di
DB, tampil tersamarkan) + `iris_from_account` (label opsional).

Alur: admin buat payout (`POST /api/admin/payouts` dengan `method: "iris"`,
atau pilih metode di halaman `/admin/payouts`) → "Kirim via Iris"
(`POST /api/admin/payouts/:id/send-iris`) — beneficiary diambil dari profil
seller (`bank_name`, `bank_account`; wajib diisi via verifikasi seller) →
"Cek Status" (`GET /api/admin/payouts/:id/iris-status`).

**Tanpa `iris_api_key`**: endpoint Iris balas `422 "Iris belum dikonfigurasi"`;
payout manual (`POST /api/admin/payouts/:id/pay` + `proof_url`) tetap bisa
dipakai sebagai fallback.

## 2. Refund via Midtrans

Approve refund (`POST /api/admin/refunds/:id/approve`, body opsional
`{amount_cents, reason}` untuk partial refund) kini mencoba memanggil
`POST https://api.sandbox.midtrans.com/v2/{order_id}/refund` terlebih dulu.
Hasilnya dicatat di tabel `refunds` (`midtrans_refund_id`, `raw_response`).

**Bila API Midtrans gagal/tak terjangkau** (atau egress lambat): fallback
otomatis — refund tetap di-approve dengan `note: "manual"` (perilaku Fase 1),
dana dikembalikan manual via dashboard Midtrans. Request tidak pernah gagal
karena ini. Tiket dari order yang di-refund otomatis tidak berlaku (check-in
menolak dengan 422).

Daftar pengajuan: `GET /api/admin/refunds?status=requested` (halaman
`/admin/refunds`).

## 3. Kode promo/diskon

Seller buat kode untuk eventnya (`/seller/promos`), admin bisa buat kode
global (`/admin/promos`). Tipe: persen (`percent`, maks 100) atau potongan
tetap (`flat`, rupiah). Aturan: kode unik, min. belanja, kuota pakai total,
limit per user, masa berlaku, scope global/per event.

Checkout: field kode promo → `POST /api/promos/validate` → diskon memotong
**subtotal SEBELUM komisi 3% & biaya layanan dihitung**. Satu kode per order;
pemakaian dicatat di `promo_usages` (idempoten per order).

## 4. Rating & ulasan

Buyer dengan order **lunas** dan event yang **sudah selesai** bisa memberi
1–5 bintang + teks (satu per user per event, bisa diubah). Rata-rata tampil di
detail event (`rating_avg`, `rating_count`); daftar publik di
`GET /api/events/:id/reviews`; admin bisa hapus ulasan
(`DELETE /api/admin/reviews/:id`).

## 5. Transfer tiket

Pemilik tiket (belum dipakai, event belum mulai) bisa transfer ke email user
terdaftar lain (halaman `/tickets` → tombol Transfer). Kode tiket & QR **tetap
sama** (tetap valid untuk check-in); yang berpindah adalah pemiliknya
(`holder_user_id`). Riwayat di `GET /api/tickets/transfers`; penerima dapat
notifikasi WhatsApp bila diaktifkan.

## 6. Notifikasi WhatsApp (Fonnte)

Abstraksi provider di `backend/internal/notify` (interface `Sender`; default
implementasi Fonnte: `POST https://api.fonnte.com/send`).

Pengaturan (panel admin → Pengaturan → Notifikasi WhatsApp, atau
`PUT /api/admin/settings/notifications`): `wa_provider` (`fonnte`),
`wa_fonnte_token` (terenkripsi), `wa_enabled`. Ambil token gratis/berbayar di
fonnte.com → hubungkan nomor WA.

Trigger otomatis (fire-and-forget — gagal kirim hanya di-log, tidak
menggagalkan alur utama): tiket terbit, refund disetujui, payout terkirim,
tiket diterima via transfer. Tombol "Kirim Tes" untuk verifikasi nomor.

**Tanpa token / nonaktif**: notifikasi diam-diam dinonaktifkan.

## Catatan / gap Fase 2

- Panggilan live ke Midtrans/Iris/Fonnte butuh egress yang cukup cepat;
  timeout HTTP client 60 dtk. Bila timeout, yang teruji adalah jalur fallback
  (snap DUMMY, refund manual, payout manual, notifikasi skip) — semuanya
  bekerja sesuai desain.
- Iris & Fonnte belum teruji live (butuh `iris_api_key` dari dashboard Iris
  dan token Fonnte asli dari Zepi).
- Refund partial: `amount_cents` < gross didukung API; kebijakan bisnis
  (kapan boleh partial) mengikuti keputusan admin.
