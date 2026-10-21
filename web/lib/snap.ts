/** Loader dinamis Midtrans Snap.js (browser only). */

declare global {
  interface Window {
    snap?: {
      pay: (
        token: string,
        callbacks?: {
          onSuccess?: (result: unknown) => void;
          onPending?: (result: unknown) => void;
          onError?: (result: unknown) => void;
          onClose?: () => void;
        }
      ) => void;
    };
  }
}

let snapPromise: Promise<void> | null = null;

function snapUrl(environment: string): string {
  return environment === "production"
    ? "https://app.midtrans.com/snap/snap.js"
    : "https://app.sandbox.midtrans.com/snap/snap.js";
}

/**
 * Memuat snap.js dengan data-client-key dari backend
 * (GET /api/config/midtrans-client-key).
 */
export function loadSnap(
  clientKey: string,
  environment: string = "sandbox"
): Promise<void> {
  if (typeof window === "undefined") {
    return Promise.reject(new Error("Hanya bisa dijalankan di browser"));
  }
  if (window.snap) return Promise.resolve();
  if (snapPromise) return snapPromise;

  snapPromise = new Promise<void>((resolve, reject) => {
    const script = document.createElement("script");
    script.src = snapUrl(environment);
    script.setAttribute("data-client-key", clientKey);
    script.onload = () => resolve();
    script.onerror = () => {
      snapPromise = null;
      reject(new Error("Gagal memuat script pembayaran Midtrans"));
    };
    document.head.appendChild(script);
  });
  return snapPromise;
}
