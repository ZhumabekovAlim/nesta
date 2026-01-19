package main

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nesta/internal/config"
	"nesta/internal/http/handlers"
	adminHandlers "nesta/internal/http/handlers/admin"
	apiHandlers "nesta/internal/http/handlers/api"
	authHandlers "nesta/internal/http/handlers/auth"
	paymentHandlers "nesta/internal/http/handlers/payments"
	storeHandlers "nesta/internal/http/handlers/store"
	subscriptionHandlers "nesta/internal/http/handlers/subscriptions"
	userHandlers "nesta/internal/http/handlers/users"
	"nesta/internal/http/server"
	"nesta/internal/repositories"
	"nesta/internal/services"
	"nesta/internal/storage"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()
	logger := setupLogger(cfg.Env)
	rand.Seed(time.Now().UnixNano())

	store, err := storage.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init database")
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			logger.Error().Err(closeErr).Msg("failed to close database")
		}
	}()

	repoUsers := repositories.NewUserRepository(store.DB)
	repoOTP := repositories.NewOTPRepository(store.DB)
	repoRefresh := repositories.NewRefreshTokenRepository(store.DB)
	repoComplexes := repositories.NewComplexRepository(store.DB)
	repoComplexRequests := repositories.NewComplexRequestRepository(store.DB)
	repoPlans := repositories.NewPlanRepository(store.DB)
	repoSubscriptions := repositories.NewSubscriptionRepository(store.DB)
	repoProducts := repositories.NewProductRepository(store.DB)
	repoOrders := repositories.NewOrderRepository(store.DB)
	repoPayments := repositories.NewPaymentRepository(store.DB)
	repoPickups := repositories.NewPickupLogRepository(store.DB)

	authService := &services.AuthService{
		Users:          repoUsers,
		OTP:            repoOTP,
		RefreshTokens:  repoRefresh,
		JWTSecret:      cfg.JWTSecret,
		AccessTTL:      cfg.AccessTokenTTL,
		RefreshTTL:     cfg.RefreshTokenTTL,
		OTPTTL:         cfg.OTPTTL,
		OTPRateLimit:   cfg.OTPRateLimit,
		OTPMaxAttempts: cfg.OTPMaxAttempts,
	}

	complexService := &services.ComplexService{
		DB:              store.DB,
		Complexes:       repoComplexes,
		Requests:        repoComplexRequests,
		ThresholdStatus: "PLANNED",
	}

	subscriptionService := &services.SubscriptionService{
		Subscriptions: repoSubscriptions,
		Complexes:     repoComplexes,
		Plans:         repoPlans,
	}

	orderService := &services.OrderService{
		Orders:   repoOrders,
		Products: repoProducts,
	}

	paymentService := &services.PaymentService{
		DB:            store.DB,
		Payments:      repoPayments,
		Orders:        repoOrders,
		Subscriptions: repoSubscriptions,
	}

	deps := server.Dependencies{
		Health: handlers.HealthHandler{DBPinger: store.Ping},
		Auth:   authHandlers.Handler{Auth: authService},
		Complexes: apiHandlers.ComplexHandler{
			Complexes: repoComplexes,
			Requests:  repoComplexRequests,
			Service:   complexService,
			JWTSecret: cfg.JWTSecret,
		},
		Plans:   apiHandlers.PlanHandler{Plans: repoPlans},
		Pickups: apiHandlers.PickupHandler{Logs: repoPickups},
		Subscriptions: subscriptionHandlers.Handler{
			Service:       subscriptionService,
			Subscriptions: repoSubscriptions,
		},
		Users:          userHandlers.Handler{Users: repoUsers},
		Products:       storeHandlers.ProductHandler{Products: repoProducts},
		Orders:         storeHandlers.OrderHandler{Service: orderService, Orders: repoOrders},
		Payments:       paymentHandlers.Handler{Payments: paymentService},
		AdminComplexes: adminHandlers.ComplexHandler{Complexes: repoComplexes, Service: complexService},
		AdminPlans:     adminHandlers.PlanHandler{Plans: repoPlans},
		AdminSubs:      adminHandlers.SubscriptionHandler{Subscriptions: repoSubscriptions, Service: subscriptionService},
		AdminProducts:  adminHandlers.ProductHandler{Products: repoProducts},
		AdminOrders:    adminHandlers.OrderHandler{Orders: repoOrders},
		AdminPickups:   adminHandlers.PickupLogHandler{Logs: repoPickups},
	}

	appServer := server.New(logger, deps, cfg.JWTSecret)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      appServer.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", cfg.Port).Msg("server started")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("server failed")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("server shutdown error")
	}
}

func setupLogger(env string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	if env == "development" {
		logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
		return logger
	}
	return log.Logger
}
