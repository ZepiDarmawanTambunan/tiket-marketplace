// Command api runs the ticket marketplace REST API service.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/config"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/crypto"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/handler"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/middleware"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/repository"
	"github.com/ZepiDarmawanTambunan/tiket-marketplace/backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	db, err := repository.Open(cfg.DBDSN)
	if err != nil {
		slog.Error("database error", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	masterKey := crypto.DeriveKey(cfg.SettingsMasterKey)
	deps := service.NewDeps(db, masterKey)

	// Wire dependencies: repository -> service -> handler.
	authSvc := service.NewAuthService(deps, cfg.JWTSecret, cfg.AccessTTL)
	eventSvc := service.NewEventService(deps)
	ticketTypeSvc := service.NewTicketTypeService(deps)
	promoSvc := service.NewPromoService(deps)
	reviewSvc := service.NewReviewService(deps)
	orderSvc := service.NewOrderService(deps, cfg.QRSecret, promoSvc)
	ticketSvc := service.NewTicketService(deps, cfg.QRSecret)
	adminSvc := service.NewAdminService(deps)

	authH := handler.NewAuthHandler(authSvc)
	eventH := handler.NewEventHandler(eventSvc, ticketTypeSvc)
	orderH := handler.NewOrderHandler(orderSvc)
	webhookH := handler.NewWebhookHandler(orderSvc)
	ticketH := handler.NewTicketHandler(ticketSvc)
	promoH := handler.NewPromoHandler(promoSvc)
	reviewH := handler.NewReviewHandler(reviewSvc)
	adminH := handler.NewAdminHandler(adminSvc)
	adminEventH := handler.NewAdminEventHandler(eventSvc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, middleware.CORS(cfg.CORSAllowedOrigins))

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"status":"ok"}}`))
	})

	r.Route("/api", func(r chi.Router) {
		// Public.
		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)
		r.Post("/webhooks/midtrans", webhookH.Midtrans)
		r.Get("/config/midtrans-client-key", adminH.MidtransClientKey)
		r.Get("/config/public", adminH.PublicConfig)
		r.With(middleware.OptionalAuth(cfg.JWTSecret)).Get("/events", eventH.List)
		r.With(middleware.OptionalAuth(cfg.JWTSecret)).Get("/events/{id}", eventH.Get)
		r.With(middleware.OptionalAuth(cfg.JWTSecret)).Get("/events/{id}/ticket-types", eventH.ListTicketTypes)
		r.With(middleware.OptionalAuth(cfg.JWTSecret)).Get("/events/{id}/reviews", reviewH.List)

		// Authenticated users.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Get("/auth/me", authH.Me(deps))
			r.Get("/orders", orderH.List)
			r.Post("/orders", orderH.Create)
			r.Get("/orders/{id}", orderH.Get)
			r.Post("/orders/{id}/refund", orderH.Refund)
			r.Get("/tickets", ticketH.MyTickets)
			r.Post("/tickets/transfer", ticketH.Transfer)
			r.Get("/tickets/transfers", ticketH.TransferHistory)
			r.Post("/promos/validate", promoH.Validate)
			r.Post("/events/{id}/reviews", reviewH.Create)
			r.Put("/events/{id}/reviews", reviewH.Update)
		})

		// Sellers (and admins acting as staff).
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Use(middleware.RequireRole("seller", "super_admin"))
			r.Post("/events", eventH.Create)
			r.Put("/events/{id}", eventH.Update)
			r.Delete("/events/{id}", eventH.Delete)
			r.Post("/events/{id}/submit", eventH.Submit)
			r.Post("/events/{id}/ticket-types", eventH.CreateTicketType)
			r.Put("/ticket-types/{id}", eventH.UpdateTicketType)
			r.Delete("/ticket-types/{id}", eventH.DeleteTicketType)
			r.Post("/checkin", ticketH.CheckIn)
			r.Get("/seller/payouts", adminH.SellerPayouts)
			r.Post("/promos", promoH.Create)
			r.Get("/promos", promoH.List)
			r.Put("/promos/{id}", promoH.Update)
			r.Delete("/promos/{id}", promoH.Delete)
		})

		// Super admin.
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Use(middleware.RequireRole("super_admin"))
			r.Post("/events/{id}/approve", adminEventH.Approve)
			r.Post("/events/{id}/reject", adminEventH.Reject)
			r.Get("/events", adminEventH.List)
			r.Get("/users", adminH.ListUsers)
			r.Put("/users/{id}/role", adminH.UpdateRole)
			r.Put("/users/{id}/verify-seller", adminH.VerifySeller)
			r.Get("/transactions", adminH.ListTransactions)
			r.Get("/refunds", adminH.ListRefunds)
			r.Post("/refunds/{id}/approve", adminH.ApproveRefund)
			r.Post("/refunds/{id}/reject", adminH.RejectRefund)
			r.Get("/payouts", adminH.ListPayouts)
			r.Post("/payouts", adminH.CreatePayout)
			r.Post("/payouts/{id}/pay", adminH.PayPayout)
			r.Post("/payouts/{id}/send-iris", adminH.SendIrisPayout)
			r.Get("/payouts/{id}/iris-status", adminH.IrisStatus)
			r.Delete("/reviews/{id}", reviewH.DeleteByAdmin)
			r.Get("/reports/summary", adminH.ReportSummary)
			r.Get("/audit-logs", adminH.AuditLogs)
			r.Get("/settings/midtrans", adminH.GetMidtransSettings)
			r.Put("/settings/midtrans", adminH.UpdateMidtransSettings)
			r.Post("/settings/midtrans/test", adminH.TestMidtrans)
			r.Get("/settings/notifications", adminH.GetNotificationSettings)
			r.Put("/settings/notifications", adminH.UpdateNotificationSettings)
			r.Post("/settings/notifications/test", adminH.TestNotification)
		})
	})

	slog.Info("ticket marketplace listening", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
