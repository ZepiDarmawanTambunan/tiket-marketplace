// Package model defines the domain entities of the ticket marketplace.
package model

import "time"

// Role constants for users.
const (
	RoleBuyer      = "buyer"
	RoleSeller     = "seller"
	RoleSuperAdmin = "super_admin"
)

// Event status constants.
const (
	EventDraft         = "draft"
	EventPendingReview = "pending_review"
	EventPublished     = "published"
	EventRejected      = "rejected"
	EventFinished      = "finished"
)

// Order status constants.
const (
	OrderPending         = "pending"
	OrderPaid            = "paid"
	OrderFailed          = "failed"
	OrderCancelled       = "cancelled"
	OrderExpired         = "expired"
	OrderRefundRequested = "refund_requested"
	OrderRefunded        = "refunded"
)

// Refund status constants.
const (
	RefundRequested = "requested"
	RefundApproved  = "approved"
	RefundRejected  = "rejected"
)

// Payout status constants.
const (
	PayoutPending    = "pending"
	PayoutProcessing = "processing"
	PayoutPaid       = "paid"
	PayoutCancelled  = "cancelled"
)

// Payout method constants.
const (
	PayoutMethodManual = "manual"
	PayoutMethodIris   = "iris"
)

// Promo type constants.
const (
	PromoTypePercent = "percent"
	PromoTypeFlat    = "flat"
)

// Promo scope constants.
const (
	PromoScopeGlobal = "global"
	PromoScopeEvent  = "event"
)

// Gateway fee bearer options.
const (
	FeeBearerBuyer    = "buyer"
	FeeBearerSeller   = "seller"
	FeeBearerPlatform = "platform"
)

// User is a platform account: buyer, seller, or super admin.
type User struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone,omitempty"`
	PasswordHash   string    `json:"-"`
	Role           string    `json:"role"`
	SellerVerified bool      `json:"seller_verified"`
	BankName       string    `json:"bank_name,omitempty"`
	BankAccount    string    `json:"bank_account,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Event is a ticketed event created by a seller, published after admin approval.
type Event struct {
	ID           int64        `json:"id"`
	SellerID     int64        `json:"seller_id"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Category     string       `json:"category"`
	Venue        string       `json:"venue"`
	Address      string       `json:"address"`
	MapsURL      string       `json:"maps_url"`
	StartsAt     time.Time    `json:"starts_at"`
	EndsAt       time.Time    `json:"ends_at"`
	BannerURL    string       `json:"banner_url"`
	Status       string       `json:"status"`
	RefundPolicy string       `json:"refund_policy"`
	CreatedAt    time.Time    `json:"created_at"`
	RatingAvg    float64      `json:"rating_avg"`
	RatingCount  int64        `json:"rating_count"`
	TicketTypes  []TicketType `json:"ticket_types"`
}

// TicketType is a sellable ticket tier of an event.
type TicketType struct {
	ID           int64     `json:"id"`
	EventID      int64     `json:"event_id"`
	Name         string    `json:"name"`
	PriceCents   int64     `json:"price_cents"`
	Quota        int       `json:"quota"`
	Sold         int       `json:"sold"`
	SaleStartsAt time.Time `json:"sale_starts_at"`
	SaleEndsAt   time.Time `json:"sale_ends_at"`
}

// OrderItem is a line item in a checkout request (not yet a concrete ticket).
type OrderItem struct {
	TicketTypeID   int64  `json:"ticket_type_id"`
	TicketTypeName string `json:"ticket_type_name,omitempty"`
	Qty            int    `json:"qty"`
	PriceCents     int64  `json:"price_cents"`
}

// Order is a buyer's checkout with commission and gateway fee breakdown.
type Order struct {
	ID               int64       `json:"id"`
	OrderCode        string      `json:"order_code"`
	UserID           int64       `json:"user_id"`
	EventID          int64       `json:"event_id"`
	EventTitle       string      `json:"event_title,omitempty"`
	SubtotalCents    int64       `json:"subtotal_cents"`
	DiscountCents    int64       `json:"discount_cents"`
	PromoCode        string      `json:"promo_code,omitempty"`
	CommissionCents  int64       `json:"commission_cents"`
	GatewayFeeCents  int64       `json:"gateway_fee_cents"`
	GatewayFeeBearer string      `json:"gateway_fee_bearer"`
	GrossCents       int64       `json:"gross_cents"`
	Status           string      `json:"status"`
	SnapToken        string      `json:"snap_token,omitempty"`
	Items            []OrderItem `json:"items"`
	CreatedAt        time.Time   `json:"created_at"`
}

// Ticket is a concrete, scannable ticket issued after an order is paid.
type Ticket struct {
	ID             int64      `json:"id"`
	TicketCode     string     `json:"ticket_code"`
	QRPayLoad      string     `json:"qr_payload"`
	EventTitle     string     `json:"event_title"`
	EventStartsAt  time.Time  `json:"event_starts_at"`
	TicketTypeName string     `json:"ticket_type_name"`
	PriceCents     int64      `json:"price_cents"`
	OrderCode      string     `json:"order_code"`
	HolderUserID   *int64     `json:"holder_user_id,omitempty"`
	HolderName     string     `json:"holder_name,omitempty"`
	Status         string     `json:"status"` // unused | used
	UsedAt         *time.Time `json:"used_at"`
}

// Payment records a Midtrans notification for an order.
type Payment struct {
	ID                int64     `json:"id"`
	OrderID           int64     `json:"order_id"`
	MidtransOrderID   string    `json:"midtrans_order_id"`
	TransactionID     string    `json:"transaction_id"`
	PaymentType       string    `json:"payment_type"`
	TransactionStatus string    `json:"transaction_status"`
	GrossAmount       int64     `json:"gross_amount"`
	CreatedAt         time.Time `json:"created_at"`
}

// Refund is a buyer's refund request handled by an admin.
type Refund struct {
	ID               int64     `json:"id"`
	OrderID          int64     `json:"order_id"`
	OrderCode        string    `json:"order_code,omitempty"`
	AmountCents      int64     `json:"amount_cents"`
	Reason           string    `json:"reason"`
	Status           string    `json:"status"`
	HandledBy        *int64    `json:"handled_by"`
	MidtransRefundID string    `json:"midtrans_refund_id,omitempty"`
	RawResponse      string    `json:"raw_response,omitempty"`
	Note             string    `json:"note,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// Payout is a transfer of seller earnings for an event.
type Payout struct {
	ID              int64     `json:"id"`
	SellerID        int64     `json:"seller_id"`
	EventID         int64     `json:"event_id"`
	AmountCents     int64     `json:"amount_cents"`
	Status          string    `json:"status"`
	Method          string    `json:"method"`
	IrisReferenceNo string    `json:"iris_reference_no,omitempty"`
	IrisStatus      string    `json:"iris_status,omitempty"`
	ProofURL        string    `json:"proof_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// AuditLog records admin/sensitive actions.
type AuditLog struct {
	ID        int64     `json:"id"`
	ActorID   *int64    `json:"actor_id"`
	ActorName string    `json:"actor_name,omitempty"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// ReportSummary aggregates sales numbers for a period.
type ReportSummary struct {
	From                     string `json:"from"`
	To                       string `json:"to"`
	GMVCents                 int64  `json:"gmv_cents"`
	CommissionCents          int64  `json:"commission_cents"`
	GatewayFeeCollectedCents int64  `json:"gateway_fee_collected_cents"`
	OrdersPaid               int64  `json:"orders_paid"`
	TicketsSold              int64  `json:"tickets_sold"`
	RefundedCents            int64  `json:"refunded_cents"`
}

// PromoCode is a discount code created by a seller or admin.
type PromoCode struct {
	ID            int64      `json:"id"`
	Code          string     `json:"code"`
	Type          string     `json:"type"` // percent | flat
	Value         float64    `json:"value"`
	MinSpendCents int64      `json:"min_spend_cents"`
	Quota         *int       `json:"quota,omitempty"`
	PerUserLimit  *int       `json:"per_user_limit,omitempty"`
	StartsAt      *time.Time `json:"starts_at,omitempty"`
	EndsAt        *time.Time `json:"ends_at,omitempty"`
	Scope         string     `json:"scope"` // global | event
	EventID       *int64     `json:"event_id,omitempty"`
	SellerID      int64      `json:"seller_id"`
	Active        bool       `json:"active"`
	CreatedAt     time.Time  `json:"created_at"`
}

// PromoUsage records one application of a promo code to an order.
type PromoUsage struct {
	ID        int64     `json:"id"`
	PromoID   int64     `json:"promo_id"`
	UserID    int64     `json:"user_id"`
	OrderID   int64     `json:"order_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Review is a buyer's rating + comment for a finished event.
type Review struct {
	ID        int64     `json:"id"`
	EventID   int64     `json:"event_id"`
	UserID    int64     `json:"user_id"`
	UserName  string    `json:"user_name,omitempty"`
	Rating    int       `json:"rating"`
	Text      string    `json:"text,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReviewSummary aggregates an event's reviews.
type ReviewSummary struct {
	Avg   float64 `json:"avg"`
	Count int64   `json:"count"`
}

// TicketTransfer records a ticket changing hands between users.
type TicketTransfer struct {
	ID           int64     `json:"id"`
	TicketCode   string    `json:"ticket_code"`
	FromUserID   int64     `json:"from_user_id"`
	FromUserName string    `json:"from_user_name,omitempty"`
	ToUserID     int64     `json:"to_user_id"`
	ToUserName   string    `json:"to_user_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
