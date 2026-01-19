package server

import (
	"net/http"

	"nesta/internal/http/handlers"
	adminHandlers "nesta/internal/http/handlers/admin"
	apiHandlers "nesta/internal/http/handlers/api"
	authHandlers "nesta/internal/http/handlers/auth"
	paymentHandlers "nesta/internal/http/handlers/payments"
	storeHandlers "nesta/internal/http/handlers/store"
	subscriptionHandlers "nesta/internal/http/handlers/subscriptions"
	userHandlers "nesta/internal/http/handlers/users"
	"nesta/internal/http/middleware"

	"github.com/rs/zerolog"
)

type Server struct {
	mux    *http.ServeMux
	logger zerolog.Logger
}

type Dependencies struct {
	Health         handlers.HealthHandler
	Auth           authHandlers.Handler
	Complexes      apiHandlers.ComplexHandler
	Plans          apiHandlers.PlanHandler
	Pickups        apiHandlers.PickupHandler
	Subscriptions  subscriptionHandlers.Handler
	Users          userHandlers.Handler
	Products       storeHandlers.ProductHandler
	Orders         storeHandlers.OrderHandler
	Payments       paymentHandlers.Handler
	AdminComplexes adminHandlers.ComplexHandler
	AdminPlans     adminHandlers.PlanHandler
	AdminSubs      adminHandlers.SubscriptionHandler
	AdminProducts  adminHandlers.ProductHandler
	AdminOrders    adminHandlers.OrderHandler
	AdminPickups   adminHandlers.PickupLogHandler
}

func New(logger zerolog.Logger, deps Dependencies, jwtSecret string) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", deps.Health.Health)
	mux.HandleFunc("/ready", deps.Health.Ready)

	mux.HandleFunc("/api/v1/complexes", deps.Complexes.List)
	mux.HandleFunc("/api/v1/complexes/", deps.Complexes.HandleItem)

	mux.HandleFunc("/api/v1/plans", deps.Plans.List)

	mux.HandleFunc("/api/v1/auth/otp/send", deps.Auth.SendOTP)
	mux.HandleFunc("/api/v1/auth/otp/verify", deps.Auth.VerifyOTP)
	mux.HandleFunc("/api/v1/auth/refresh", deps.Auth.Refresh)
	mux.HandleFunc("/api/v1/auth/logout", deps.Auth.Logout)

	mux.Handle("/api/v1/me", middleware.Auth(jwtSecret)(http.HandlerFunc(deps.Users.Me)))
	mux.Handle("/api/v1/subscriptions", middleware.Auth(jwtSecret)(http.HandlerFunc(deps.Subscriptions.Create)))
	mux.Handle("/api/v1/subscriptions/me", middleware.Auth(jwtSecret)(http.HandlerFunc(deps.Subscriptions.ListMine)))
	mux.Handle("/api/v1/subscriptions/", middleware.Auth(jwtSecret)(http.HandlerFunc(deps.Subscriptions.Update)))
	mux.Handle("/api/v1/pickups/", middleware.Auth(jwtSecret)(http.HandlerFunc(deps.Pickups.ListBySubscription)))

	mux.HandleFunc("/api/v1/products", deps.Products.List)
	mux.HandleFunc("/api/v1/products/", deps.Products.Get)

	mux.Handle("/api/v1/orders", middleware.Auth(jwtSecret)(http.HandlerFunc(deps.Orders.Create)))
	mux.Handle("/api/v1/orders/me", middleware.Auth(jwtSecret)(http.HandlerFunc(deps.Orders.ListMine)))
	mux.Handle("/api/v1/orders/", middleware.Auth(jwtSecret)(http.HandlerFunc(deps.Orders.Get)))

	mux.Handle("/api/v1/payments/init", middleware.Auth(jwtSecret)(http.HandlerFunc(deps.Payments.Init)))
	mux.HandleFunc("/api/v1/payments/webhook/", deps.Payments.Webhook)

	adminAuth := func(handler http.HandlerFunc) http.Handler {
		return middleware.Auth(jwtSecret)(middleware.RequireRole("admin")(handler))
	}

	mux.Handle("/api/v1/admin/complexes", adminAuth(http.HandlerFunc(deps.AdminComplexes.HandleCollection)))
	mux.Handle("/api/v1/admin/complexes/", adminAuth(http.HandlerFunc(deps.AdminComplexes.UpdateStatus)))
	mux.Handle("/api/v1/admin/plans", adminAuth(http.HandlerFunc(deps.AdminPlans.HandleCollection)))
	mux.Handle("/api/v1/admin/plans/", adminAuth(http.HandlerFunc(deps.AdminPlans.Update)))
	mux.Handle("/api/v1/admin/subscriptions", adminAuth(http.HandlerFunc(deps.AdminSubs.HandleCollection)))
	mux.Handle("/api/v1/admin/subscriptions/", adminAuth(http.HandlerFunc(deps.AdminSubs.Update)))
	mux.Handle("/api/v1/admin/products", adminAuth(http.HandlerFunc(deps.AdminProducts.HandleCollection)))
	mux.Handle("/api/v1/admin/products/", adminAuth(http.HandlerFunc(deps.AdminProducts.Update)))
	mux.Handle("/api/v1/admin/orders", adminAuth(http.HandlerFunc(deps.AdminOrders.HandleCollection)))
	mux.Handle("/api/v1/admin/orders/", adminAuth(http.HandlerFunc(deps.AdminOrders.Update)))
	mux.Handle("/api/v1/admin/pickup-logs", adminAuth(http.HandlerFunc(deps.AdminPickups.HandleCollection)))
	mux.Handle("/api/v1/admin/pickup-logs/", adminAuth(http.HandlerFunc(deps.AdminPickups.Update)))

	return &Server{mux: mux, logger: logger}
}

func (s *Server) Handler() http.Handler {
	h := middleware.RequestID(s.mux)
	h = middleware.Logging(s.logger)(h)
	return h
}
