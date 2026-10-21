"use client";

import { useEffect, useRef } from "react";
import { Html5Qrcode } from "html5-qrcode";

/**
 * Wrapper html5-qrcode. Dipakai via next/dynamic dengan ssr:false.
 */
export default function QrScanner({
  onScan,
  onError,
}: {
  onScan: (text: string) => void;
  onError?: (msg: string) => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const onScanRef = useRef(onScan);
  useEffect(() => {
    onScanRef.current = onScan;
  });

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const scanner = new Html5Qrcode("tiket-qr-reader");
    let stopped = false;

    scanner
      .start(
        { facingMode: "environment" },
        { fps: 10, qrbox: { width: 250, height: 250 } },
        (decodedText) => {
          if (!stopped) onScanRef.current(decodedText);
        },
        () => {
          /* abaikan frame tanpa QR */
        }
      )
      .catch((err) => {
        onError?.(
          "Tidak bisa mengakses kamera: " +
            (err instanceof Error ? err.message : String(err))
        );
      });

    return () => {
      stopped = true;
      scanner.stop().catch(() => {});
      scanner.clear();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return <div id="tiket-qr-reader" ref={ref} className="w-full overflow-hidden rounded-xl" />;
}
