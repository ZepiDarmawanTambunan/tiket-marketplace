# API Kontrak — Backend Marketplace Tiket (Fase 1 + Fase 2, Development)

Base URL: `http://localhost:8080` — semua path di bawah berawalan `/api`.
Semua request/response memakai JSON. Format respons sukses standar:

```json
{ "data": { ... } }
```

Format respons error standar:

```json
{ "error": { "code": "SOME_CODE", "message": "penjelasan manusiawi" } }
```

Kode HTTP umum: `200` sukses, `201` dibuat, `400` validasi, `401` tidak/belum login,
`403` terlarang (role tidak cukup), `404` tidak ditemukan, `409` konflik (mis. tiket
sudah dipakai), `422` bisnis rule gagal, `500` server error.

Semua nominal uang dalam **rupiah (integer)**, kolom `*_cents` di DB.

## Autentikasi

Header: `Authorization: Bearer <jwt_token>`

Token JWT berisi `user_id` dan `role` (`buyer` | `seller` | `super_admin`),
berlaku 24 jam (configurable via `JWT_ACCESS_TTL`).

Catatan register: role default `buyer`; opsional `role: "seller"` untuk langsung jualan.
Seller tidak wajib terverifikasi dulu (lihat settings `require_seller_verification`,
default `false`) — verifikasi seller hanya lencana/kepercayaan, bukan gerbang jualan.

---

## Auth

### POST /api/auth/register
Publik. Mendaftarkan user baru.

Request:
```json
{ "name": "Budi", "email": "budi@mail.com", "phone": "08123456789",
  "password": "rahasia123", "role": "seller" }
```
`role` opsional (`buyer`|`seller`), default `buyer`. `phone` opsional.

Response `201`:
```json
{ "data": { "user": { "id": 1, "name": "Budi", "email": "budi@mail.com",
  "phone": "08123456789", "role": "seller", "seller_verified": false,
  "created_at": "2026-10-05T10:00:00Z" },
  "token": "eyJhbGciOi..." } }
```

### POST /api/auth/login
Publik.

Request: `{ "email": "budi@mail.com", "password": "rahasia123" }`

Response `200`: `{ "data": { "user": {...}, "token": "eyJhbGciOi..." } }`
Sama bentuknya dengan register. `401` bila kredensial salah.

### GET /api/auth/me
Auth: Bearer. Mengembalikan profil user yang sedang login.

Response `200`: `{ "data": { "user": {...} } }`

---

## Events (publik)

### GET /api/events
Publik. Hanya event `published` (kecuali pemanggil seller/admin melihat miliknya —
gunakan filter `mine=true` dengan token).

Query params: `category`, `search` (cocok title/venue/deskripsi), `from`/`to`
(tanggal mulai, format `YYYY-MM-DD`), `page` (default 1), `per_page` (default 20, max 100),
`mine=true` (auth: hanya event milik seller yang login).

Response `200`:
```json
{ "data": { "events": [
  { "id": 3, "seller_id": 2, "title": "Konser Jazz", "description": "...",
    "category": "musik", "venue": "GOR Jambi", "address": "...",
    "maps_url": "https://maps.google.com/...", "starts_at": "2026-12-01T19:00:00Z",
    "ends_at": "2026-12-01T22:00:00Z", "banner_url": "https://...",
    "status": "published", "refund_policy": "...",
    "ticket_types": [ { "id": 5, "name": "Reguler", "price_cents": 150000,
      "quota": 500, "sold": 12, "sale_starts_at": "...", "sale_ends_at": "..." } ],
    "created_at": "..." }
], "page": 1, "per_page": 20, "total": 42 } }
```

### GET /api/events/:id
Publik. Detail satu event + ticket types-nya. `404` bila belum published dan
pemanggil bukan pemilik/admin. Response event kini memuat
`rating_avg` (rata-rata rating, 0 bila belum ada) dan `rating_count`
(Fase 2 — lihat Rating & Ulasan).

## Events (seller)

Auth: Bearer, role `seller` (atau `super_admin`).

### POST /api/events
Membuat event baru — selalu berstatus `draft` (tidak langsung tayang).

Request:
```json
{ "title": "Konser Jazz", "description": "...", "category": "musik",
  "venue": "GOR Jambi", "address": "Jl. ...", "maps_url": "https://...",
  "starts_at": "2026-12-01T19:00:00Z", "ends_at": "2026-12-01T22:00:00Z",
  "banner_url": "https://...", "refund_policy": "Refund H-7 ..." }
```
Response `201`: `{ "data": { "event": {...} } }` dengan `status: "draft"`.

### PUT /api/events/:id
Edit event milik sendiri. Hanya boleh bila status `draft` atau `rejected`
(setelah published tidak bisa diubah — keamanan harga/kuota).

### DELETE /api/events/:id
Hapus event milik sendiri. Hanya bila `draft`/`rejected` dan belum ada order.

### POST /api/events/:id/submit
Seller mengajukan event `draft`/`rejected` untuk direview admin → status
`pending_review`. Response: event dengan status baru.

## Events (admin)

Auth: Bearer, role `super_admin`.

### POST /api/admin/events/:id/approve
Menyetujui event `pending_review` → `published`.

### POST /api/admin/events/:id/reject
Request: `{ "reason": "Banner tidak jelas" }` → status `rejected`.

### GET /api/admin/events
List SEMUA event untuk moderasi. Query: `status` (`pending_review`/`published`/`draft`/`rejected`/`finished`), `page`, `per_page`.
Response `200`: `{ "data": { "events": [ ... ], "page": 1, "per_page": 20, "total": 3 } }`

---

## Ticket Types

### GET /api/events/:id/ticket-types
Publik (bagian dari detail event, endpoint mandiri tersedia). List tipe tiket satu event.

Response `200`: `{ "data": { "ticket_types": [ {...} ] } }`

### POST /api/events/:id/ticket-types
Auth seller pemilik event. Tambah tipe tiket (event harus `draft`/`rejected`).

Request:
```json
{ "name": "VIP", "price_cents": 500000, "quota": 100,
  "sale_starts_at": "2026-10-06T00:00:00Z", "sale_ends_at": "2026-11-30T23:59:59Z" }
```
Response `201`: `{ "data": { "ticket_type": {...} } }`

### PUT /api/ticket-types/:id
Auth seller pemilik. Edit tipe tiket (tidak boleh bila sudah ada yang terjual —
`409` bila `sold > 0` dan perubahan menyentuh harga/kuota).

### DELETE /api/ticket-types/:id
Auth seller pemilik. Hapus bila belum ada penjualan (`409` bila `sold > 0`).

---

## Orders

### POST /api/orders
Auth: Bearer (buyer/seller/admin bisa membeli).

Request:
```json
{ "event_id": 3, "promo_code": "HEMAT10",
  "items": [ { "ticket_type_id": 5, "qty": 2 },
             { "ticket_type_id": 6, "qty": 1 } ] }
```
`promo_code` opsional (Fase 2): divalidasi server-side; bila tidak valid → `422`
dan order TIDAK dibuat. Diskon memotong subtotal SEBELUM komisi & fee dihitung.

Perhitungan (server-side, jangan percaya angka dari client):
- `subtotal` = Σ (price × qty)
- `discount` = promo: `percent` → subtotal × value/100 (dibulatkan, max 100%);
  `flat` → value rupiah (dibatasi ≤ subtotal)
- `net` = subtotal − discount
- `commission` = `commission_default`% × net (default 3%, configurable)
- `gateway_fee`: dihitung dari net bila `gateway_fee_bearer=buyer` → ditambahkan
  ke total bayar pembeli sebagai "biaya layanan" transparan; bila
  `seller`/`platform` → dicatat terpisah, tidak menambah total bayar.
- `gross` (total bayar) = net + commission + (gateway_fee bila bearer=buyer)

Fee gateway dihitung dari setting: `gateway_fee_percent` (default 2.9) +
`gateway_fee_flat` (default Rp 2.000), dibulatkan ke atas ke rupiah.

Response `201`:
```json
{ "data": { "order": {
    "id": 10, "order_code": "TIX-20261005-A8K2", "event_id": 3,
    "subtotal_cents": 450000, "discount_cents": 45000, "promo_code": "HEMAT10",
    "commission_cents": 12150,
    "gateway_fee_cents": 13745, "gross_cents": 430895,
    "gateway_fee_bearer": "buyer", "status": "pending",
    "items": [ { "ticket_type_id": 5, "qty": 2, "price_cents": 150000 } ],
    "created_at": "..." },
  "snap_token": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "client_key": "SB-Mid-client-xxxx",
  "redirect_url": "https://app.sandbox.midtrans.com/snap/v3/redirection/..." } }
```

Bila Midtrans tidak terjangkau / key masih dummy (mode development),
order TETAP dibuat dan response berisi `"snap_token": "DUMMY"`,
`"redirect_url": null`, plus `"payment_note": "mode development: snap dummy"`.
Frontend tetap bisa lanjut ke halaman "menunggu pembayaran".

Kode order unik format `TIX-YYYYMMDD-XXXX` (XXXX = 4 karakter alfanumerik acak).

### GET /api/orders/:id
Auth: pemilik order atau admin. Detail order + status pembayaran terkini.

### GET /api/orders
Riwayat order — perilakunya tergantung role:
- `buyer`: order milik sendiri.
- `seller`: order untuk event miliknya (semua pembeli).
- `super_admin`: semua order.

Query: `status`, `page`, `per_page`.

Response `200`: `{ "data": { "orders": [ ... ], "page": 1, "per_page": 20, "total": 5 } }`

### POST /api/orders/:id/refund
Auth: buyer pemilik. Request: `{ "reason": "..." }`.
Hanya untuk order `paid` dan sesuai kebijakan refund event → status
`refund_requested`. Response: order dengan status baru. `422` bila tidak memenuhi syarat.

---

## Webhook Midtrans

### POST /api/webhooks/midtrans
Publik (dipanggil server Midtrans), TAPI diverifikasi ketat:

1. `signature_key` dari body harus sama dengan
   `SHA512(order_id + status_code + gross_amount + ServerKey)`.
   Tidak cocok → `401`, notifikasi diabaikan.
2. **Idempotency**: kombinasi `(order_id, transaction_id, transaction_status)`
   yang sudah pernah diproses diabaikan (response `200` dengan
   `"status": "duplicate_ignored"`).

Mapping status:
| transaction_status Midtrans | status order |
|---|---|
| `settlement`, `capture` (fraud_status=accept) | `paid` |
| `pending` | `pending` |
| `expire`, `cancel` | `cancelled` |
| `deny` | `failed` |

Saat transisi ke `paid` (atomik, dalam satu transaksi DB):
- generate `order_items`: satu baris per tiket dengan `ticket_code` unik
  (format `TIX-XXXXXXXX`) dan `qr_payload` = `HMAC-SHA256(ticket_code, QR_SECRET)`.
- `ticket_types.sold` bertambah sesuai qty.

Response `200`: `{ "data": { "status": "ok" } }`.

---

## Tiket Saya

### GET /api/tickets
Auth: Bearer. Tiket milik user (hanya dari order `paid`).

Response `200`:
```json
{ "data": { "tickets": [
  { "ticket_code": "TIX-A8K2Q7M1", "event_title": "Konser Jazz",
    "event_starts_at": "2026-12-01T19:00:00Z", "ticket_type_name": "Reguler",
    "price_cents": 150000, "qr_payload": "a3f9c1...", "status": "unused",
    "used_at": null, "order_code": "TIX-20261005-A8K2",
    "holder_user_id": 7, "holder_name": "Budi" }
] } }
```
`status`: `unused` | `used`. Tiket yang ditransfer (Fase 2) tampil di akun
pemegang terkini (`holder_user_id`/`holder_name`), bukan lagi pembeli awal.

---

## Check-in

### POST /api/checkin
Auth: role `seller` (harus pemilik event tiket tsb) atau `super_admin` — panitia.

Request — salah satu:
```json
{ "ticket_code": "TIX-A8K2Q7M1" }
```
atau (scan QR):
```json
{ "qr_payload": "TIX-A8K2Q7M1.a3f9c1..." }
```
Bila `qr_payload` dipakai, server memverifikasi HMAC-nya sebelum lookup
(anti QR palsu).

Response `200`:
```json
{ "data": { "ticket_code": "TIX-A8K2Q7M1", "event_title": "Konser Jazz",
  "ticket_type_name": "Reguler", "holder_name": "Budi",
  "checked_in_at": "2026-12-01T18:32:10Z" } }
```

- Tiket sudah dipakai → `409` `{ "error": { "code": "ALREADY_USED", ... } }`
- Tiket bukan dari event milik seller tsb → `403`
- Order belum paid → `422`

---

## Admin — Settings Midtrans

Auth: Bearer, role `super_admin` untuk semua endpoint di bawah.

### GET /api/admin/settings/midtrans
Response `200`:
```json
{ "data": {
  "environment": "sandbox",
  "server_key_masked": "****9f2c",
  "client_key_masked": "****7ab1",
  "merchant_id": "G123456789",
  "commission_default": 3,
  "gateway_fee_bearer": "buyer",
  "gateway_fee_percent": 2.9,
  "gateway_fee_flat_cents": 2000,
  "require_seller_verification": false,
  "iris_api_key_masked": "****3f2a",
  "iris_from_account": "Tiket Marketplace",
  "updated_at": "2026-10-05T10:00:00Z" } }
```
⚠️ Key server TIDAK PERNAH dikembalikan plaintext — hanya 4 karakter terakhir
untuk identifikasi. Nilai sensitif disimpan terenkripsi AES-256-GCM di DB
(master key dari env `SETTINGS_MASTER_KEY`).

### PUT /api/admin/settings/midtrans
Simpan/patch settings. Request (semua field opsional):
```json
{ "server_key": "SB-Mid-server-xxx", "client_key": "SB-Mid-client-xxx",
  "merchant_id": "G123456789", "environment": "sandbox",
  "commission_default": 3, "gateway_fee_bearer": "buyer",
  "gateway_fee_percent": 2.9, "gateway_fee_flat_cents": 2000,
  "require_seller_verification": false,
  "iris_api_key": "IRIS-xxx", "iris_from_account": "Tiket Marketplace" }
```
`iris_api_key` terenkripsi saat disimpan (tidak pernah dikembalikan plaintext);
butuh aktivasi Iris terpisah via tim support/sales Midtrans (bukan dari dashboard
Midtrans/Snap — lihat bagian Payout via Midtrans Iris).
Setiap perubahan dicatat di audit log. Response: sama seperti GET (tetap masked).

### POST /api/admin/settings/midtrans/test
Tes koneksi ke Midtrans API memakai server key tersimpan: memanggil
`GET https://api.sandbox.midtrans.com/v2/{dummy_order_id}/status`
dengan order id dummy. Mengembalikan hasil apa adanya:
```json
{ "data": { "ok": false, "http_status": 404,
  "message": "Transaction doesn't exist.", "note": "key dummy?" } }
```
`ok: true` hanya bila Midtrans merespons 2xx/404-yang-sah (koneksi + auth valid).

## Config publik

### GET /api/config/midtrans-client-key
Publik (tanpa auth) — dibutuhkan frontend untuk load `snap.js`.
```json
{ "data": { "client_key": "SB-Mid-client-xxxx", "environment": "sandbox" } }
```
Client key memang publik secara desain (dipakai di browser), jadi plaintext di sini OK.

---

## Admin — Users

### GET /api/admin/users
Query: `role`, `search`, `page`, `per_page`.
Response: list user (tanpa password_hash).

### PUT /api/admin/users/:id/role
Request: `{ "role": "seller" }` (`buyer`|`seller`|`super_admin`).

### PUT /api/admin/users/:id/verify-seller
Request: `{ "verified": true, "bank_name": "BCA", "bank_account": "1234567890" }`
Menandai seller terverifikasi (opsional per kebijakan; jualan tetap bisa tanpa ini).

---

## Admin — Transaksi, Refund, Payout

### GET /api/admin/transactions
Query: `status` (order status), `from`/`to` (`YYYY-MM-DD`), `page`, `per_page`.
List order + info pembayaran Midtrans (payment_type, transaction_id).

### GET /api/admin/refunds
Auth: `super_admin`. List pengajuan refund untuk moderasi. Query: `status`
(`requested`|`approved`|`rejected`), `page`, `per_page`.
Response `200`: `{ "data": { "refunds": [ { "id": 1, "order_id": 6,
"order_code": "TIX-20261007-R2K4", "amount_cents": 319700, "reason": "...",
"status": "requested", "midtrans_refund_id": null, "note": null, ... } ],
"page": 1, "per_page": 20, "total": 2 } }`

### POST /api/admin/refunds/:id/approve
Menyetujui refund → order `refunded`. Body opsional (Fase 2):
```json
{ "amount_cents": 100000, "reason": "refund parsial" }
```
Tanpa body = refund penuh sebesar gross order. Saat approve, backend mencoba
refund langsung via Midtrans Core API
(`POST {apiBase}/v2/{order_id}/refund` dengan `refund_key` unik, Basic auth
server key, timeout 60 dtk). Bila Midtrans tidak terjangkau/gagal, refund TETAP
di-approve dengan `note: "manual"` (perilaku Fase 1) — request tidak pernah
gagal karena ini. Response refund memuat `midtrans_refund_id`, `raw_response`,
dan `note`.

### POST /api/admin/refunds/:id/reject
Request: `{ "reason": "..." }` → refund `rejected`, order kembali `paid`.

### GET /api/admin/payouts
List payout ke seller. Query: `status`, `seller_id`, `page`, `per_page`.

### POST /api/admin/payouts
Request: `{ "seller_id": 2, "event_id": 3, "amount_cents": 436500, "method": "iris" }`
`method` opsional: `manual` (default) | `iris`. Membuat record payout `pending`.

### POST /api/admin/payouts/:id/pay
Request: `{ "proof_url": "https://..." }` → payout `paid` + bukti transfer.

---

## Admin — Laporan & Audit

### GET /api/admin/reports/summary
Query: `from`, `to` (`YYYY-MM-DD`; default 30 hari terakhir).
Response:
```json
{ "data": { "from": "2026-09-05", "to": "2026-10-05",
  "gmv_cents": 12500000, "commission_cents": 375000,
  "gateway_fee_collected_cents": 410000,
  "orders_paid": 84, "tickets_sold": 132, "refunded_cents": 0 } }
```

### GET /api/admin/audit-logs
Query: `action`, `actor_id`, `page`, `per_page`.
Response: list `{ id, actor_id, actor_name, action, detail, created_at }`.

---

## Health

### GET /health
Publik. `{ "data": { "status": "ok" } }`.

---

## Fase 2

### Kode Promo

Auth: Bearer, role `seller` (promo miliknya; seller hanya boleh `scope=event`
untuk event miliknya) atau `super_admin` (semua scope).

#### POST /api/promos
Request:
```json
{ "code": "HEMAT10", "type": "percent", "value": 10,
  "min_spend_cents": 0, "quota": 100, "per_user_limit": 2,
  "starts_at": "2026-10-01T00:00:00Z", "ends_at": "2026-12-31T23:59:59Z",
  "scope": "event", "event_id": 3 }
```
- `type`: `percent` (value = %, 0–100) | `flat` (value = rupiah).
- `quota`/`per_user_limit`: null = tanpa batas.
- `scope`: `global` | `event` (`event` wajib `event_id`).

Code dinormalisasi ke UPPERCASE dan unik global. Response `201`:
`{ "data": { "promo": { "id": 1, "code": "HEMAT10", ..., "active": true } } }`.
`422` bila validasi gagal, `409` bila code sudah dipakai.

#### GET /api/promos
Query: `active` (`true`/`false`), `event_id`, `page`, `per_page`.
Seller melihat promo miliknya; admin melihat semua.
Response: `{ "data": { "promos": [...], "page": 1, "per_page": 20, "total": 2 } }`.

#### PUT /api/promos/:id
Edit promo (full-replace; `code` immutable → `422` bila diubah).
Field yang dihilangkan memakai nilai saat ini, kecuali `quota`/`per_user_limit`/
`starts_at`/`ends_at` yang bisa dikosongkan (null = tanpa batas).

#### DELETE /api/promos/:id
Soft delete: `active=false`. Response: `{ "data": { "deleted": true } }`.

#### POST /api/promos/validate
Auth: user mana pun. Cek kode terhadap keranjang tanpa membuat order.

Request:
```json
{ "code": "HEMAT10", "event_id": 3,
  "items": [ { "ticket_type_id": 5, "qty": 2 } ] }
```
Response `200` (selalu 200; invalid = `valid: false`):
```json
{ "data": { "valid": true, "discount_cents": 20000,
  "message": "kode HEMAT10 valid: hemat Rp 20000", "code": "HEMAT10" } }
```
Aturan: kode ada & aktif, dalam jendela waktu, scope cocok dengan event,
subtotal ≥ `min_spend_cents`, pemakaian total < `quota`, pemakaian user <
`per_user_limit`.

#### POST /api/orders (promo)
Tambahan field opsional `promo_code` (lihat bagian Orders di atas): diskon
memotong subtotal SEBELUM komisi & fee; satu kode per order; pemakaian dicatat
di `promo_usages`. Response order memuat `discount_cents` dan `promo_code`.

---

### Rating & Ulasan

#### POST /api/events/:id/reviews
Auth: buyer. Syarat: punya order `paid` untuk event tsb DAN `ends_at` event
sudah lewat. Satu ulasan per user per event.

Request: `{ "rating": 5, "text": "Mantap!" }` (`text` opsional, rating 1–5).
`422` bila syarat tidak terpenuhi, `409` bila sudah pernah mengulas.
Response `201`: `{ "data": { "review": {...} } }`.

#### PUT /api/events/:id/reviews
Auth: edit ulasan milik sendiri. Request sama seperti create.

#### GET /api/events/:id/reviews
Publik. List ulasan + ringkasan rating.

Response `200`:
```json
{ "data": { "reviews": [
    { "id": 1, "event_id": 3, "user_id": 7, "user_name": "Budi",
      "rating": 5, "text": "Mantap!", "created_at": "...", "updated_at": "..." }
  ],
  "summary": { "avg": 4.5, "count": 2 },
  "page": 1, "per_page": 20, "total": 2 } }
```

#### DELETE /api/admin/reviews/:id
Auth: `super_admin`. Moderasi: hapus ulasan. Response: `{ "data": { "deleted": true } }`.

`GET /api/events/:id` kini memuat `rating_avg` dan `rating_count`.

---

### Transfer Tiket

#### POST /api/tickets/transfer
Auth: Bearer. Memindahkan tiket ke user lain yang sudah terdaftar.

Request: `{ "ticket_code": "TIX-A8K2Q7M1", "to_email": "teman@mail.com" }`

Syarat: pemanggil adalah pemegang tiket saat ini, tiket `unused`, order `paid`
(tiket dari order `refunded` ditolak `422`), event belum dimulai, penerima
terdaftar, bukan diri sendiri. Efek: `holder_user_id` tiket → penerima,
transfer dicatat, penerima mendapat notifikasi WhatsApp (bila diaktifkan).
Response `200`:
```json
{ "data": { "ticket_code": "TIX-A8K2Q7M1", "holder_name": "Teman",
  "transferred_to_user_id": 9 } }
```

#### GET /api/tickets
Tiket milik user = tiket yang dipegang (`holder_user_id`, atau pemilik order
bila belum pernah ditransfer) dari order `paid`. Response memuat
`holder_user_id` dan `holder_name` pemegang terkini.

#### GET /api/tickets/transfers
Auth: Bearer. Riwayat transfer yang melibatkan user login (sebagai pengirim
maupun penerima).
Response: `{ "data": { "transfers": [ { "id": 1, "ticket_code": "...",
"from_user_id": 7, "from_user_name": "Budi", "to_user_id": 9,
"to_user_name": "Teman", "created_at": "..." } ], "page": 1, "per_page": 20, "total": 1 } }`.

Check-in tetap memakai kode yang sama; `holder_name` pada response check-in
adalah nama pemegang terkini.

---

### Refund via Midtrans (Fase 2)

`POST /api/admin/refunds/:id/approve` kini mencoba refund dana langsung via
Midtrans Core API (lihat perubahan di bagian Admin — Transaksi, Refund, Payout).
Alur: panggil `POST {apiBase}/v2/{order_id}/refund` (`refund_key` unik, Basic
auth server key, timeout 60 dtk) → catat `midtrans_refund_id` + `raw_response`.
Gagal/tidak terjangkau → fallback: approve dengan `note: "manual"`.
Tetap hanya `super_admin` yang bisa approve (seller hanya bisa melihat refund
eventnya via `GET /api/orders?status=refund_requested` dengan role seller).

---

### Payout via Midtrans Iris

Settings baru (terenkripsi, via tabel settings): `iris_api_key` (creator API key
dari dashboard Iris — beda dengan server key payment gateway),
`iris_from_account` (label opsional).

Auth Iris: HTTP Basic dengan Iris API key sebagai username (sesuai
https://iris-docs.midtrans.com). Base URL: sandbox
`https://app.sandbox.midtrans.com/iris/api/v1`, production
`https://app.midtrans.com/iris/api/v1`. Timeout 60 dtk.

`POST /api/admin/payouts` menerima `method`: `manual` (default) | `iris`.

#### POST /api/admin/payouts/:id/send-iris
Auth: `super_admin`. Mengirim payout `pending` via Iris. Beneficiary diambil
dari profil seller (`bank_name`, `bank_account`, nama) — `422` bila data bank
belum lengkap. Request create payout:
`{ "payouts": [ { "beneficiary_name", "beneficiary_account",
"beneficiary_bank", "beneficiary_email", "amount": "174600.00",
"notes": "Payout tiket event #3" } ] }`.
Sukses → payout `processing` + `iris_reference_no` + `iris_status` tersimpan;
seller mendapat notifikasi WhatsApp.
`422` `"Iris belum dikonfigurasi"` bila `iris_api_key` kosong.

#### GET /api/admin/payouts/:id/iris-status
Auth: `super_admin`. Refresh status dari Iris (`GET /payouts/{reference_no}`) →
update `iris_status`; bila `completed` → payout `paid`, bila
`failed`/`rejected` → payout kembali `pending` (bisa dicoba ulang/manual).

`POST /api/admin/payouts/:id/pay` (manual + `proof_url`) tetap tersedia sebagai
fallback dan kini juga bisa melunasi payout berstatus `processing`.

---

### Notifikasi WhatsApp

Package baru `internal/notify`: interface `Sender { Send(to, message string) error }`
dengan implementasi `fonnte` (`POST https://api.fonnte.com/send`, header
`Authorization: <token>`, form `target` + `message`, timeout 60 dtk).

Settings (terenkripsi bila sensitif): `wa_provider` (default `"fonnte"`),
`wa_fonnte_token`, `wa_enabled` (default `false`).

Trigger (fire-and-forget — gagal kirim hanya di-log, tidak menggagalkan alur):
tiket terbit saat webhook `paid`, refund di-approve, payout `paid`/terkirim via
Iris, tiket diterima via transfer. Nomor tujuan = `phone` user.

#### GET /api/admin/settings/notifications
Auth: `super_admin`. Response (masked):
```json
{ "data": { "wa_provider": "fonnte", "wa_fonnte_token_masked": "****ab12",
  "wa_enabled": false } }
```

#### PUT /api/admin/settings/notifications
Request (semua opsional):
`{ "wa_provider": "fonnte", "wa_fonnte_token": "xxx", "wa_enabled": true }`.
Token disimpan terenkripsi; perubahan di-audit-log.

#### POST /api/admin/settings/notifications/test
Request: `{ "to": "081234567890" }`. Mengirim pesan tes secara sinkron.
Response: `{ "data": { "ok": true, "message": "pesan tes terkirim ke ..." } }`
atau `ok: false` dengan alasan (mis. belum diaktifkan / token kosong).

---

## Ringkasan status

- Event: `draft` → `pending_review` → `published` | `rejected`; `finished`
  (di-set sistem/admin setelah `ends_at` lewat).
- Order: `pending` → `paid` | `cancelled` | `failed` | `expired`;
  `paid` → `refund_requested` → `refunded` (approve) / kembali `paid` (reject).
- Refund: `requested` → `approved` | `rejected`.
- Payout: `pending` → `processing` (terkirim via Iris) → `paid` | `cancelled`.
- Tiket: `unused` → `used` (sekali pakai; double-scan ditolak 409).
