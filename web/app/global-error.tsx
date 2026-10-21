"use client";

export default function GlobalError({
  error,
  retry,
}: {
  error: Error & { digest?: string };
  retry: () => void;
}) {
  return (
    <html lang="id">
      <body
        style={{
          fontFamily: "system-ui, sans-serif",
          display: "flex",
          minHeight: "100vh",
          alignItems: "center",
          justifyContent: "center",
          margin: 0,
          background: "#fafafa",
          color: "#18181b",
        }}
      >
        <div style={{ textAlign: "center", padding: "2rem", maxWidth: 480 }}>
          <h1 style={{ fontSize: 24, fontWeight: 700 }}>Terjadi kesalahan</h1>
          <p style={{ color: "#71717a", fontSize: 14, marginTop: 8 }}>
            Aplikasi gagal dimuat. Silakan coba lagi.
            {error.digest ? ` (Kode: ${error.digest})` : ""}
          </p>
          <button
            onClick={() => retry()}
            style={{
              marginTop: 16,
              background: "#ea580c",
              color: "#fff",
              border: "none",
              borderRadius: 8,
              padding: "8px 16px",
              fontSize: 14,
              fontWeight: 500,
              cursor: "pointer",
            }}
          >
            Coba Lagi
          </button>
        </div>
      </body>
    </html>
  );
}
