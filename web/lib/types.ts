// Tipe data sesuai kontrak backend/API.md

export type Role = "buyer" | "seller" | "super_admin";

export interface User {
  id: number;
  name: string;
  email: string;
  phone?: string | null;
  role: Role;
  seller_verified: boolean;
  created_at: string;
}

export type EventStatus =
  | "draft"
  | "pending_review"
  | "published"
  | "rejected"
  | "finished";

export interface TicketType {
  id: number;
  event_id?: number;
  name: string;
  price_cents: number;
  quota: number;
  sold: number;
  sale_starts_at?: string | null;
  sale_ends_at?: string | null;
}

export interface EventItem {
  id: number;
  seller_id: number;
  seller_name?: string;
  title: string;
  description: string;
  category: string;
  venue: string;
  address?: string;
  maps_url?: string;
  starts_at: string;
  ends_at: string;
  banner_url?: string;
  status: EventStatus;
  refund_policy?: string;
  ticket_types: TicketType[];
  rating_avg?: number | null; // Fase 2: rata-rata rating ulasan
  rating_count?: number | null; // Fase 2: jumlah ulasan
  created_at: string;
}

export type OrderStatus =
  | "pending"
  | "paid"
  | "cancelled"
  | "failed"
  | "expired"
  | "refund_requested"
  | "refunded";

export interface OrderItem {
  ticket_type_id: number;
  ticket_type_name?: string;
  qty: number;
  price_cents: number;
}

export interface Order {
  id: number;
  order_code: string;
  event_id: number;
  event_title?: string;
  subtotal_cents: number;
  discount_cents?: number; // Fase 2: diskon kode promo (server-computed)
  promo_code?: string | null; // Fase 2: kode promo terpakai
  commission_cents: number;
  gateway_fee_cents: number;
  gross_cents: number;
  gateway_fee_bearer: string;
  status: OrderStatus;
  items: OrderItem[];
  snap_token?: string;
  client_key?: string;
  redirect_url?: string | null;
  payment_note?: string;
  payment_type?: string;
  transaction_id?: string;
  refund_via?: string | null; // Fase 2: "midtrans" | "manual" — sumber refund
  refund_midtrans_status?: string | null; // Fase 2: status refund di Midtrans
  refund_note?: string | null; // Fase 2: detail hasil refund
  created_at: string;
}

export interface Ticket {
  ticket_code: string;
  event_title: string;
  event_starts_at: string;
  ticket_type_name: string;
  price_cents: number;
  qr_payload: string;
  status: "unused" | "used";
  used_at?: string | null;
  order_code: string;
}

export interface CheckinResult {
  ticket_code: string;
  event_title: string;
  ticket_type_name: string;
  holder_name: string;
  checked_in_at: string;
}

export interface MidtransSettings {
  environment: "sandbox" | "production";
  server_key_masked: string;
  client_key_masked: string;
  merchant_id: string;
  commission_default: number;
  gateway_fee_bearer: "buyer" | "seller" | "platform";
  gateway_fee_percent: number;
  gateway_fee_flat_cents: number;
  require_seller_verification: boolean;
  iris_api_key_masked?: string; // Fase 2
  iris_from_account?: string | null; // Fase 2: rekening sumber Iris
  updated_at: string;
}

/** Fase 2: Kode promo */
export type PromoType = "percent" | "flat";
export type PromoScope = "global" | "event";

export interface Promo {
  id: number;
  code: string;
  type: PromoType;
  value: number;
  min_spend_cents: number;
  quota: number;
  used_count?: number;
  per_user_limit: number;
  starts_at: string;
  ends_at: string;
  scope: PromoScope;
  event_id?: number | null;
  event_title?: string;
  active: boolean;
  created_at: string;
}

export interface PromoValidateResult {
  valid: boolean;
  discount_cents: number;
  message: string;
}

/** Fase 2: Ulasan event */
export interface Review {
  id: number;
  user_id: number;
  user_name?: string;
  rating: number; // 1-5
  text?: string | null;
  created_at: string;
  updated_at?: string;
}

/** Fase 2: Riwayat transfer tiket */
export interface TicketTransfer {
  id: number;
  ticket_code: string;
  from_user_id: number;
  from_user_name?: string;
  to_user_id: number;
  to_user_name?: string;
  created_at: string;
}

/** Fase 2: Pengaturan notifikasi WhatsApp */
export interface NotificationSettings {
  wa_provider: string; // "fonnte"
  wa_enabled: boolean;
  wa_fonnte_token_masked?: string;
  updated_at?: string;
}

export interface AuditLog {
  id: number;
  actor_id?: number | null;
  actor_name?: string;
  action: string;
  detail?: string;
  created_at: string;
}

export interface Payout {
  id: number;
  seller_id: number;
  seller_name?: string;
  event_id?: number | null;
  event_title?: string;
  amount_cents: number;
  method?: "manual" | "iris"; // Fase 2: metode pencairan
  iris_status?: string | null; // Fase 2: status pengiriman Iris
  iris_reference_no?: string | null; // Fase 2: reference/ID transaksi Iris
  status: "pending" | "paid" | "cancelled";
  proof_url?: string | null;
  created_at: string;
}

export interface ReportSummary {
  from: string;
  to: string;
  gmv_cents: number;
  commission_cents: number;
  gateway_fee_collected_cents: number;
  orders_paid: number;
  tickets_sold: number;
  refunded_cents: number;
}

export interface Paginated<T> {
  items: T[];
  page: number;
  per_page: number;
  total: number;
}

export interface CheckoutItem {
  ticketTypeId: number;
  name: string;
  price: number; // rupiah per tiket
  qty: number;
}

export const EVENT_STATUS_LABEL: Record<EventStatus, string> = {
  draft: "Draft",
  pending_review: "Menunggu Review",
  published: "Tayang",
  rejected: "Ditolak",
  finished: "Selesai",
};

export const ORDER_STATUS_LABEL: Record<OrderStatus, string> = {
  pending: "Menunggu Pembayaran",
  paid: "Lunas",
  cancelled: "Dibatalkan",
  failed: "Gagal",
  expired: "Kedaluarsa",
  refund_requested: "Refund Diajukan",
  refunded: "Dana Dikembalikan",
};

/** Fase 2: Pengajuan refund */
export type RefundStatus = "requested" | "approved" | "rejected";
export interface Refund {
  id: number;
  order_id: number;
  order_code?: string;
  amount_cents: number;
  reason: string;
  status: RefundStatus;
  handled_by?: number | null;
  midtrans_refund_id?: string;
  raw_response?: string;
  note?: string;
  created_at: string;
}
export const REFUND_STATUS_LABEL: Record<RefundStatus, string> = {
  requested: "Menunggu",
  approved: "Disetujui",
  rejected: "Ditolak",
};
