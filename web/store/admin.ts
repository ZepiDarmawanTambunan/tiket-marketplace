import { create } from "zustand";
import type { MidtransSettings } from "@/lib/types";
import { apiFetch } from "@/lib/api";

interface AdminState {
  settings: MidtransSettings | null;
  loading: boolean;
  fetchSettings: () => Promise<void>;
  saveSettings: (patch: Record<string, unknown>) => Promise<void>;
}

export const useAdminStore = create<AdminState>()((set) => ({
  settings: null,
  loading: false,
  fetchSettings: async () => {
    set({ loading: true });
    try {
      const settings = await apiFetch<MidtransSettings>(
        "/api/admin/settings/midtrans"
      );
      set({ settings });
    } finally {
      set({ loading: false });
    }
  },
  saveSettings: async (patch) => {
    const settings = await apiFetch<MidtransSettings>(
      "/api/admin/settings/midtrans",
      { method: "PUT", body: JSON.stringify(patch) }
    );
    set({ settings });
  },
}));
