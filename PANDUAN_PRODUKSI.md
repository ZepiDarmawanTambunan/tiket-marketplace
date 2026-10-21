# Tiket Marketplace — Panduan Go-Live Production

Status platform (2026-10-07): review + testing akhir lulus 11/11, data uji
dibersihkan (tinggal akun super admin), Midtrans **sandbox** keys terkonfigurasi
& tervalidasi, token Fonnte terkonfigurasi & tes terkirim. Siap menuju production.

> Webhook Midtrans **asli** baru bisa jalan setelah website punya domain publik
> + HTTPS. Sampai saat itu, pembayaran production belum bisa diterima.

## 1. Checklist aksi Zepi (pemilik keputusan)

| # | Aksi | Keterangan |
|---|------|-----------|
| 1 | **Ganti password admin default** | WAJIB sebelum go-live. Login panel admin → ubah password. |
| 2 | Beli VPS + domain | Lihat rekomendasi §2. Estimasi: VPS ±Rp200rb/bln + domain .id ±Rp219rb/thn. |
| 3 | Aktifkan **mode production** Midtrans | dashboard.midtrans.com → ambil **Server Key + Client Key production** → isi via panel admin → Pengaturan → Midtrans (tersimpan terenkripsi). |
| 4 | Daftarkan **webhook URL** di Midtrans | Dashboard → Settings → Configuration → Payment Notification URL: `https://DOMAIN_KAMU/api/webhooks/midtrans`. Tanpa ini, status bayar tidak otomatis masuk. |
| 5 | (Opsional) Aktivasi **Iris** untuk payout otomatis | Menu Iris/Payouts di dashboard → ambil **Iris API key** → isi via panel admin. Tanpa ini, payout tetap bisa manual (`/admin/payouts`). |
| 6 | Putuskan: **`FinishExpired()`** | Event yang lewat tanggal belum otomatis jadi `finished`. Pilih: (a) cron internal di API, atau (b) tombol manual di panel admin. |
| 7 | Putuskan: **rate limiting** | Endpoint publik (auth, webhook) belum ada rate limit — disarankan sebelum go-live karena webhook terekspos ke internet. |

## 2. Rekomendasi VPS + domain (cek harga 7 Okt 2026)

Kebutuhan satu box: Go API + Next.js + MySQL → minimum nyaman **2 vCPU / 4 GB RAM**,
datacenter Jakarta/Singapura, Ubuntu 22.04/24.04 LTS.

| Provider | Paket | Spek | Harga |
|----------|-------|------|-------|
| Jagoan Hosting | GALAXY | 4 vCPU / 4 GB / 100 GB, unmetered | Rp200.000/bln |
| Exabytes | NVMe M1 | 1 vCPU / 4 GB / 50 GB | Rp194.000/bln |
| Exabytes | NVMe C1 | 2 vCPU / 4 GB / 100 GB | Rp261.000/bln |

Domain: **.id ±Rp219.000/thn** (tanpa dokumen, cocok untuk brand Indonesia);
.com ±Rp137–150rb tahun pertama, renewal ±Rp201rb/thn.

Rekomendasi: VPS 2 vCPU/4 GB + domain **.id**.

## 3. Arsitektur deploy

```
Internet ──HTTPS──▶ Caddy (:80/:443, auto TLS Let's Encrypt)
                        ├── /api/* ──▶ api:8080   (Go)
                        └── /*     ──▶ web:3000   (Next.js)
api ──▶ mysql:3306 (tiket_db)
```

File deploy ada di `deploy/`: `docker-compose.yml`, `Dockerfile.api`,
`Dockerfile.web`, `Caddyfile`, `.env.production.example`.

## 4. Langkah deploy di VPS

```bash
# 1. Di VPS (Ubuntu): install Docker
curl -fsSL https://get.docker.com | sh

# 2. Clone repo, masuk ke deploy/
git clone <repo> && cd tiket-marketplace/deploy

# 3. Isi environment production
cp .env.production.example .env.production
nano .env.production   # DOMAIN, secrets acak (JWT_SECRET, SETTINGS_MASTER_KEY,
                       # QR_SECRET), SEED_ADMIN_PASSWORD, DB password kuat

# 4. Jalankan
docker compose --env-file .env.production up -d --build

# 5. Migrasi + seed (sekali saja)
docker compose --env-file .env.production exec api /app/migrate up
docker compose --env-file .env.production exec api /app/seed

# 6. Arahkan DNS domain (A record) ke IP publik VPS, tunggu propagasi,
#    lalu Caddy otomatis menerbitkan sertifikat HTTPS.
```

**Penting:**
- `SETTINGS_MASTER_KEY`, `JWT_SECRET`, `QR_SECRET` wajib acak & **sama** di
  semua proses; kalau diganti setelah seed, isi ulang Midtrans keys via panel admin.
- `SEED_ADMIN_PASSWORD` hanya dipakai saat seed pertama; setelah login, ganti
  password via panel admin (checklist #1).
- `NEXT_PUBLIC_API_URL` di web harus `https://DOMAIN_KAMU/api` (bukan localhost).
- CORS: isi `CORS_ALLOWED_ORIGINS=https://DOMAIN_KAMU`.

## 5. Verifikasi pasca-deploy

1. `https://DOMAIN_KAMU/health` → 200 OK (via Caddy → api).
2. Buka web, login sebagai admin, **ganti password default**.
3. Isi Midtrans production keys via panel admin → Pengaturan → Midtrans.
4. Daftarkan webhook URL di dashboard Midtrans (checklist #4).
5. Uji beli sungguhan nominal kecil (mis. Rp10.000) → bayar via QRIS →
   pastikan tiket terbit otomatis (webhook asli bekerja).
6. Uji tombol "Kirim Tes" WhatsApp ke nomor sendiri.

## 6. Setelah go-live

- Pantau log: `docker compose logs -f api web`.
- Backup MySQL berkala (cron `mysqldump` → object storage / VPS backup).
- Saat siap scale: pisahkan MySQL ke managed database, tambah rate limiting
  (checklist #7), dan aktifkan Iris (checklist #5).
