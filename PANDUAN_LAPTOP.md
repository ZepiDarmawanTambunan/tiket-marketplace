# Menjalankan Tiket Marketplace di Laptop + Cloudflare Tunnel (Testing)

Server tempat saya berjalan **tidak bisa** menjalankan Cloudflare Tunnel —
dikonfirmasi 2026-10-09: egress proxy-nya memblokir `cloudflared` secara
spesifik, dan koneksi langsung mati total. Jadi tunnel gratisan tetap bisa
dipakai, tapi **harus dijalankan dari laptop/PC kamu sendiri**, yang jaringannya
tidak diblokir.

Tujuannya: dapat URL publik `https://xxx.trycloudflare.com` untuk backend API,
supaya **webhook Midtrans yang asli** bisa masuk — tidak perlu lagi simulasi
webhook manual.

## 1. Install (sekali saja)

1. **Docker Desktop** (sudah termasuk `docker compose`):
   https://www.docker.com/products/docker-desktop/
2. **cloudflared** (klien tunnel resmi Cloudflare, gratis):
   - Windows: `winget install cloudflare.cloudflared`
     (atau unduh exe dari https://github.com/cloudflare/cloudflared/releases)
   - macOS: `brew install cloudflare/cloudflare/cloudflared`
   - Linux: unduh paket `.deb`/biner dari releases di atas.
3. Verifikasi: `cloudflared --version`

## 2. Salin kode ke laptop

Folder proyek ini (`tiket-marketplace/`, berisi `backend/`, `web/`, `deploy/`)
belum ada di GitHub — salin manual ke laptop, misal lewat arsip zip +
Google Drive/USB. Kalau mau, saya bisa push ke repo GitHub baru
(`ZepiDarmawanTambunan/tiket-marketplace`) — tinggal bilang.

## 3. Konfigurasi (di laptop)

```bash
cd tiket-marketplace/deploy
cp .env.example .env
```

Isi nilai rahasia di `.env` dengan string acak (jangan pakai contohnya):

```bash
openssl rand -hex 32   # jalankan 4x untuk DB_PASSWORD, JWT_SECRET, SETTINGS_MASTER_KEY, QR_SECRET
```

Isi ke `DB_PASSWORD`, `JWT_SECRET`, `SETTINGS_MASTER_KEY`, `QR_SECRET`.
Biarkan `MIDTRANS_*_KEY` tetap `DUMMY` — kunci sandbox asli diisi nanti
lewat panel admin (bagian 6), bukan di file.

> `NEXT_PUBLIC_API_URL` di-bake ke dalam image web saat build. Kalau nanti
> diubah, build ulang: `docker compose -f docker-compose.yml -f docker-compose.laptop.yml build web`

## 4. Jalankan stack

```bash
cd tiket-marketplace/deploy
docker compose -f docker-compose.yml -f docker-compose.laptop.yml up --build -d mysql api web
```

Tunggu ~1 menit (MySQL perlu sehat dulu), lalu migrasi + seed **sekali saja**:

```bash
docker compose -f docker-compose.yml -f docker-compose.laptop.yml exec api /app/migrate up
docker compose -f docker-compose.yml -f docker-compose.laptop.yml exec api /app/seed
```

Cek:

```bash
curl http://localhost:8080/health   # harus menjawab {"data":{"status":"ok"}}
```

Buka http://localhost:3000 di browser — web sudah jalan.

## 5. Nyalakan Cloudflare Tunnel

Di terminal **baru** (biarkan jalan selama testing):

```bash
cloudflared tunnel --url http://localhost:8080
```

Akan muncul baris seperti:

```
Your quick Tunnel has been created! Visit it at (it may take some time to be online):
https://acak-kata-disini.trycloudflare.com
```

Catat URL `https://....trycloudflare.com` itu. **URL ini berubah setiap
tunnel di-restart** — jadi langkah 6 perlu diulang tiap nyalakan tunnel baru.

## 6. Daftarkan webhook di dashboard Midtrans

1. Buka https://dashboard.midtrans.com → pastikan mode **Sandbox**.
2. **Settings → Configuration**:
   - **Payment Notification URL** isi:
     `https://<url-tunnel-kamu>/api/webhooks/midtrans`
     (contoh: `https://acak-kata-disini.trycloudflare.com/api/webhooks/midtrans`)
   - Finish/Unfinish/Error Redirect URL: boleh dikosongkan atau isi
     `http://localhost:3000` untuk testing lokal.
3. Simpan.

## 7. Login & isi kunci Midtrans

1. Buka http://localhost:3000 → login sebagai super admin:
   **admin@tiket.local** / **admin123** — **langsung ganti passwordnya**
   setelah masuk.
2. Panel admin → **Pengaturan → Midtrans**: isi **Server Key** + **Client Key**
   sandbox (yang sudah kamu punya), lalu klik **Tes Koneksi**.
   > Kunci disimpan terenkripsi di database — jangan taruh di file `.env`.
3. Di halaman yang sama ada **webhook URL siap-copy** — cocokkan dengan yang
   didaftarkan di langkah 6.

## 8. Uji bayar end-to-end (webhook asli)

1. Register akun seller → buat event → tambah tipe tiket → ajukan.
2. Login admin → approve event.
3. Register akun buyer → checkout 1–2 tiket → bayar lewat Snap Midtrans.
   Kartu tes sandbox resmi: **4811 1111 1111 1114**, expiry bebas (masa depan),
   CVV bebas.
4. Setelah pembayaran sukses, Midtrans memanggil webhook lewat tunnel →
   tiket otomatis terbit di halaman `/tickets` buyer. **Tidak perlu simulasi
   webhook manual lagi.**

## 9. Perintah berguna

```bash
# (dari folder deploy/, selalu sertakan kedua -f)
docker compose -f docker-compose.yml -f docker-compose.laptop.yml logs -f api   # log API
docker compose -f docker-compose.yml -f docker-compose.laptop.yml down          # hentikan
docker compose -f docker-compose.yml -f docker-compose.laptop.yml down -v      # hentikan + HAPUS database
```

## Catatan

- Quick tunnel (`--url`) **tidak untuk produksi**: URL acak, berubah tiap
  restart, dan koneksinya lewat infrastruktur Cloudflare. Untuk produksi
  tetap butuh VPS + domain + HTTPS (lihat `PANDUAN_PRODUKSI.md`).
- Database di laptop **terpisah** dari database di server saya — data uji,
  user, dan event harus dibuat ulang di laptop.
- Fitur yang butuh aktivasi terpisah tetap sama: Iris (payout otomatis) dan
  token Fonnte (notifikasi WA) diisi via panel admin bila sudah ada.
- Jangan expose port laptop ke internet di luar tunnel tanpa paham risikonya.
