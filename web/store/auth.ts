import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { User } from "@/lib/types";
import { apiFetch } from "@/lib/api";

interface AuthState {
  user: User | null;
  token: string | null;
  login: (user: User, token: string) => void;
  logout: () => void;
  setUser: (user: User) => void;
  refreshMe: () => Promise<void>;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      token: null,
      login: (user, token) => set({ user, token }),
      logout: () => set({ user: null, token: null }),
      setUser: (user) => set({ user }),
      refreshMe: async () => {
        const { user } = await apiFetch<{ user: User }>("/api/auth/me");
        set({ user });
      },
    }),
    {
      name: "tikethub-auth",
      partialize: (s) => ({ user: s.user, token: s.token }),
    }
  )
);
