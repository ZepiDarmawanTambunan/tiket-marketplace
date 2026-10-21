import { create } from "zustand";
import type { CheckoutItem } from "@/lib/types";

interface CheckoutState {
  eventId: number | null;
  eventTitle: string;
  items: CheckoutItem[];
  /** Siapkan item checkout dari halaman detail event */
  start: (eventId: number, eventTitle: string, items: CheckoutItem[]) => void;
  clear: () => void;
  subtotal: () => number;
}

export const useCheckoutStore = create<CheckoutState>()((set, get) => ({
  eventId: null,
  eventTitle: "",
  items: [],
  start: (eventId, eventTitle, items) =>
    set({ eventId, eventTitle, items }),
  clear: () => set({ eventId: null, eventTitle: "", items: [] }),
  subtotal: () =>
    get().items.reduce((sum, i) => sum + i.price * i.qty, 0),
}));
