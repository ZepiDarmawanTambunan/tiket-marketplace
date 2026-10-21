"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";

interface PublicConfig {
  commission_percent: number;
  gateway_fee_bearer: string;
}

/** Info footer yang nilainya diambil live dari pengaturan platform (DB). */
export default function FooterInfo() {
  const [cfg, setCfg] = useState<PublicConfig | null>(null);

  useEffect(() => {
    apiFetch<PublicConfig>("/api/config/public")
      .then(setCfg)
      .catch(() => setCfg(null));
  }, []);

  const commission =
    cfg != null ? `Komisi platform ${cfg.commission_percent}%` : "Komisi platform kompetitif";

  return <span>Pembayaran aman via Midtrans &middot; {commission}</span>;
}
