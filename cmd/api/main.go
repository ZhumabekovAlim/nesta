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
	addressHandlers "nesta/internal/http/handlers/addresses"
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
	"nesta/internal/sms/mobizon"
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
	repoCities := repositories.NewCityRepository(store.DB)
	repoPlans := repositories.NewPlanRepository(store.DB)
	repoTimeWindows := repositories.NewTimeWindowRepository(store.DB)
	repoSubscriptionTypes := repositories.NewSubscriptionTypeRepository(store.DB)
	repoAddresses := repositories.NewAddressRepository(store.DB)
	repoSubscriptions := repositories.NewSubscriptionRepository(store.DB)
	repoProducts := repositories.NewProductRepository(store.DB)
	repoOrders := repositories.NewOrderRepository(store.DB)
	repoPickups := repositories.NewPickupLogRepository(store.DB)

	var mobizonClient *mobizon.Client
	if cfg.OTPDeliveryMode == services.OTPDeliveryModeMobizon || cfg.OTPDeliveryMode == services.OTPDeliveryModeMobizonEcho {
		mobizonClient, err = mobizon.NewClient(nil, cfg.Mobizon.BaseURL, cfg.Mobizon.APIKey)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to init mobizon client")
		}
	}

	authService := &services.AuthService{
		Users:            repoUsers,
		OTP:              repoOTP,
		RefreshTokens:    repoRefresh,
		JWTSecret:        cfg.JWTSecret,
		AccessTTL:        cfg.AccessTokenTTL,
		RefreshTTL:       cfg.RefreshTokenTTL,
		OTPTTL:           cfg.OTPTTL,
		OTPRateLimit:     cfg.OTPRateLimit,
		OTPMaxAttempts:   cfg.OTPMaxAttempts,
		OTPDeliveryMode:  cfg.OTPDeliveryMode,
		OTPSender:        cfg.Mobizon.Sender,
		OTPValidityMin:   cfg.Mobizon.Validity,
		OTPMessagePrefix: cfg.Mobizon.MessagePrefix,
		SMS:              mobizonClient,
	}

	complexService := &services.ComplexService{
		DB:              store.DB,
		Complexes:       repoComplexes,
		Requests:        repoComplexRequests,
		ThresholdStatus: "PLANNED",
	}

	addressService := &services.AddressService{
		Addresses:   repoAddresses,
		Complexes:   repoComplexes,
		Cities:      repoCities,
		TimeWindows: repoTimeWindows,
	}

	subscriptionService := &services.SubscriptionService{
		Subscriptions: repoSubscriptions,
		Addresses:     repoAddresses,
		Complexes:     repoComplexes,
		Types:         repoSubscriptionTypes,
	}

	orderService := &services.OrderService{
		Orders:   repoOrders,
		Products: repoProducts,
	}

	paymentService := &services.PaymentService{
		DB:     store.DB,
		Config: cfg.Robokassa,
	}

	deps := server.Dependencies{
		Health: handlers.HealthHandler{DBPinger: store.Ping},
		Auth:   authHandlers.Handler{Auth: authService},
		Complexes: apiHandlers.ComplexHandler{
			Complexes: repoComplexes,
			Requests:  repoComplexRequests,
			Users:     repoUsers,
			Service:   complexService,
			JWTSecret: cfg.JWTSecret,
		},
		Cities:            apiHandlers.CityHandler{Cities: repoCities},
		Plans:             apiHandlers.PlanHandler{Plans: repoPlans},
		TimeWindows:       apiHandlers.TimeWindowHandler{TimeWindows: repoTimeWindows},
		SubscriptionTypes: apiHandlers.SubscriptionTypeHandler{Types: repoSubscriptionTypes},
		Pickups:           apiHandlers.PickupHandler{Logs: repoPickups},
		Addresses: addressHandlers.Handler{
			Service:   addressService,
			Addresses: repoAddresses,
		},
		Subscriptions: subscriptionHandlers.Handler{
			Service:       subscriptionService,
			Subscriptions: repoSubscriptions,
			Addresses:     repoAddresses,
		},
		Users:          userHandlers.Handler{Users: repoUsers, TimeWindows: repoTimeWindows},
		Products:       storeHandlers.ProductHandler{Products: repoProducts},
		Orders:         storeHandlers.OrderHandler{Service: orderService, Orders: repoOrders},
		Payments:       paymentHandlers.Handler{Payments: paymentService},
		AdminComplexes: adminHandlers.ComplexHandler{Complexes: repoComplexes, Cities: repoCities, Service: complexService},
		AdminPlans:     adminHandlers.PlanHandler{Plans: repoPlans},
		AdminSubTypes:  adminHandlers.SubscriptionTypeHandler{Types: repoSubscriptionTypes},
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
