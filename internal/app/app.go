package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/internal/admin"
	"github.com/epoxx-arch/stoa/internal/storefront"
	"github.com/epoxx-arch/stoa/internal/auth"
	"github.com/epoxx-arch/stoa/internal/config"
	"github.com/epoxx-arch/stoa/internal/database"
	"github.com/epoxx-arch/stoa/internal/domain/audit"
	"github.com/epoxx-arch/stoa/internal/domain/cart"
	"github.com/epoxx-arch/stoa/internal/domain/category"
	"github.com/epoxx-arch/stoa/internal/domain/customer"
	"github.com/epoxx-arch/stoa/internal/domain/discount"
	domainmedia "github.com/epoxx-arch/stoa/internal/domain/media"
	"github.com/epoxx-arch/stoa/internal/domain/order"
	"github.com/epoxx-arch/stoa/internal/domain/payment"
	"github.com/epoxx-arch/stoa/internal/domain/product"
	"github.com/epoxx-arch/stoa/internal/domain/shipping"
	"github.com/epoxx-arch/stoa/internal/domain/tag"
	"github.com/epoxx-arch/stoa/internal/domain/tax"
	storagemedia "github.com/epoxx-arch/stoa/internal/media"
	"github.com/epoxx-arch/stoa/internal/plugin"
	"github.com/epoxx-arch/stoa/internal/search"
	"github.com/epoxx-arch/stoa/internal/server"
)

type App struct {
	Config         *config.Config
	DB             *database.DB
	Server         *server.Server
	JWTManager     *auth.JWTManager
	AuthMiddleware *auth.Middleware
	PluginRegistry *plugin.Registry
	Logger         zerolog.Logger
}

func New(cfg *config.Config) (*App, error) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().Caller().Logger()

	db, err := database.New(cfg.Database, logger)
	if err != nil {
		return nil, err
	}

	jwtManager := auth.NewJWTManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)

	apiKeyManager := auth.NewAPIKeyManager(db.Pool)
	authMiddleware := auth.NewMiddleware(jwtManager, apiKeyManager)

	pluginRegistry := plugin.NewRegistry(logger)

	srv := server.New(cfg, db, logger)

	a := &App{
		Config:         cfg,
		DB:             db,
		Server:         srv,
		JWTManager:     jwtManager,
		AuthMiddleware: authMiddleware,
		PluginRegistry: pluginRegistry,
		Logger:         logger,
	}

	if err := a.setupDomains(cfg); err != nil {
		return nil, fmt.Errorf("setting up domains: %w", err)
	}

	return a, nil
}

// setupDomains wires all domain repos, services, handlers and mounts their routes.
func (a *App) setupDomains(cfg *config.Config) error {
	pool := a.DB.Pool
	hooks := a.PluginRegistry.Hooks()
	log := a.Logger
	validate := validator.New()

	// ── Repositories ──────────────────────────────────────────────────────────

	productRepo  := product.NewPostgresRepository(pool)
	categoryRepo := category.NewPostgresRepository(pool, log)
	customerRepo := customer.NewPostgresRepository(pool, log)
	orderRepo    := order.NewPostgresRepository(pool, log)
	cartRepo     := cart.NewPostgresRepository(pool, log)
	taxRepo      := tax.NewPostgresRepository(pool, log)
	shippingRepo := shipping.NewPostgresRepository(pool, log)
	pmethodRepo  := payment.NewPostgresMethodRepository(pool, log)
	ptxRepo      := payment.NewPostgresTransactionRepository(pool, log)
	discountRepo := discount.NewPostgresRepository(pool, log)
	tagRepo      := tag.NewPostgresRepository(pool, log)
	auditRepo    := audit.NewPostgresRepository(pool, log)
	mediaRepo    := domainmedia.NewPostgresRepository(pool, log)

	// ── Media storage backend ──────────────────────────────────────────────────

	var storageBackend storagemedia.Storage
	switch cfg.Media.Storage {
	case "s3":
		s3Store, err := storagemedia.NewS3Storage(context.Background(), storagemedia.S3Config{
			Bucket:          cfg.Media.S3.Bucket,
			Region:          cfg.Media.S3.Region,
			Endpoint:        cfg.Media.S3.Endpoint,
			AccessKeyID:     cfg.Media.S3.AccessKeyID,
			SecretAccessKey: cfg.Media.S3.SecretAccessKey,
		})
		if err != nil {
			return fmt.Errorf("initializing S3 storage: %w", err)
		}
		storageBackend = s3Store
	default: // "local"
		localStore, err := storagemedia.NewLocalStorage(cfg.Media.LocalPath, "/uploads")
		if err != nil {
			return fmt.Errorf("initializing local media storage: %w", err)
		}
		storageBackend = localStore
	}
	storage := &storageAdapter{s: storageBackend}

	// ── Services ──────────────────────────────────────────────────────────────

	categorySvc := category.NewService(categoryRepo, hooks, log)
	customerSvc := customer.NewCustomerService(customerRepo, hooks, log)
	orderSvc    := order.NewService(orderRepo, hooks, log)
	cartSvc     := cart.NewCartService(cartRepo, productRepo, hooks, log)
	taxSvc      := tax.NewService(taxRepo, hooks, log)
	pmethodSvc  := payment.NewMethodService(pmethodRepo, hooks, log)
	ptxSvc      := payment.NewTransactionService(ptxRepo, hooks, log)
	discountSvc := discount.NewService(discountRepo, hooks, log)
	tagSvc      := tag.NewService(tagRepo, hooks, log)
	auditSvc    := audit.NewService(auditRepo, log)
	mediaSvc    := domainmedia.NewService(mediaRepo, storage, hooks, log)

	// Tax rate fn for product service (taxSvc must be initialised before productSvc).
	productTaxRateFn := product.TaxRateFn(func(ctx context.Context, id uuid.UUID) (int, error) {
		tr, err := taxSvc.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		return tr.Rate, nil
	})
	productSvc := product.NewService(productRepo, hooks, log, storage.URL, productTaxRateFn)

	// Tax rate fn for shipping service.
	shippingTaxRateFn := shipping.TaxRateFn(func(ctx context.Context, id uuid.UUID) (int, error) {
		tr, err := taxSvc.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		return tr.Rate, nil
	})
	shippingSvc := shipping.NewService(shippingRepo, hooks, log, shippingTaxRateFn)

	// ── Handlers ──────────────────────────────────────────────────────────────

	authH     := auth.NewHandler(pool, a.JWTManager, log)
	productH  := product.NewHandler(productSvc, validate, log)
	categoryH := category.NewHandler(categorySvc, log)
	customerH := customer.NewHandler(customerSvc, validate, log)
	shippingCostFn := order.ShippingCostFn(func(ctx context.Context, id uuid.UUID) (int, error) {
		sm, err := shippingSvc.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		return sm.PriceGross, nil
	})
	// Product tax rate fn for checkout: resolves the tax rate via the product's tax rule.
	checkoutTaxRateFn := order.ProductTaxRateFn(func(ctx context.Context, pid uuid.UUID) (int, error) {
		p, err := productSvc.GetByID(ctx, pid)
		if err != nil || p.TaxRuleID == nil {
			return 0, fmt.Errorf("no tax rule")
		}
		tr, err := taxSvc.GetByID(ctx, *p.TaxRuleID)
		if err != nil {
			return 0, err
		}
		return tr.Rate, nil
	})
	orderH    := order.NewHandler(orderSvc, shippingCostFn, checkoutTaxRateFn, validate, log)
	cartH     := cart.NewHandler(cartSvc, log)
	taxH      := tax.NewHandler(taxSvc, log)
	shippingH := shipping.NewHandler(shippingSvc, log)
	paymentH  := payment.NewHandler(pmethodSvc, ptxSvc, log)
	discountH := discount.NewHandler(discountSvc, log)
	tagH      := tag.NewHandler(tagSvc, log)
	auditH    := audit.NewHandler(auditSvc, log)
	mediaH    := domainmedia.NewHandler(mediaSvc, log)

	// ── Routes ────────────────────────────────────────────────────────────────

	r := a.Server.Router()

	// /api/v1/auth/* – no authentication required
	authH.RegisterRoutes(r)

	// /api/v1/admin/* – JWT required, staff roles only
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(a.AuthMiddleware.Authenticate)
		r.Use(a.AuthMiddleware.RequireRole(
			auth.RoleSuperAdmin, auth.RoleAdmin, auth.RoleManager,
		))
		r.Use(audit.Middleware(auditSvc, log))

		productH.RegisterAdminRoutes(r)
		r.Route("/categories", categoryH.RegisterAdminRoutes)
		customerH.RegisterAdminRoutes(r)
		orderH.RegisterAdminRoutes(r)
		taxH.RegisterAdminRoutes(r)
		shippingH.RegisterAdminRoutes(r)
		paymentH.RegisterAdminRoutes(r)
		discountH.RegisterAdminRoutes(r)
		tagH.RegisterAdminRoutes(r)
		auditH.RegisterAdminRoutes(r)
		mediaH.RegisterAdminRoutes(r)
	})

	// ── Search ────────────────────────────────────────────────────────────────

	searchEngine := search.NewPostgresEngine(pool, log)
	searchH := search.NewHandler(searchEngine, log)

	// /api/v1/store/* – public; optional auth enriches context for customer routes
	r.Route("/api/v1/store", func(r chi.Router) {
		r.Use(a.AuthMiddleware.OptionalAuth)
		r.Use(audit.Middleware(auditSvc, log))

		productH.RegisterStoreRoutes(r)
		r.Route("/categories", categoryH.RegisterStoreRoutes)
		customerH.RegisterStoreRoutes(r)
		orderH.RegisterStoreRoutes(r)
		cartH.RegisterStoreRoutes(r)
		shippingH.RegisterStoreRoutes(r)
		paymentH.RegisterStoreRoutes(r)
		searchH.RegisterStoreRoutes(r)
	})

	// ── Uploaded media files ──────────────────────────────────────────────────

	// Serve local uploads at /uploads/*  (no-op when using S3 storage)
	if cfg.Media.Storage != "s3" {
		uploadsDir := http.Dir(cfg.Media.LocalPath)
		r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(uploadsDir)))
	}

	// ── Admin Frontend ────────────────────────────────────────────────────────

	// Serve embedded SvelteKit SPA under /admin/*
	adminHandler := admin.Handler()
	r.Handle("/admin", adminHandler)
	r.Handle("/admin/*", adminHandler)

	// ── Storefront ────────────────────────────────────────────────────────────

	// Serve embedded SvelteKit storefront SPA at the root.
	// Registered last so that /api and /admin take priority.
	storefrontHandler := storefront.Handler()
	r.Handle("/", storefrontHandler)
	r.Handle("/*", storefrontHandler)

	return nil
}

func (a *App) Run() error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Server.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-quit:
		a.Logger.Info().Msg("shutdown signal received")
	}

	return a.Shutdown()
}

func (a *App) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.PluginRegistry.ShutdownAll(); err != nil {
		a.Logger.Error().Err(err).Msg("plugin shutdown error")
	}

	if err := a.Server.Shutdown(ctx); err != nil {
		a.Logger.Error().Err(err).Msg("server shutdown error")
	}

	a.DB.Close()
	a.Logger.Info().Msg("application stopped")
	return nil
}

// storageAdapter adapts any storagemedia.Storage to the domain/media.StorageBackend
// interface. The content is buffered in memory to determine size before upload,
// which is required by S3-compatible backends.
type storageAdapter struct {
	s storagemedia.Storage
}

func (a *storageAdapter) Store(ctx context.Context, filename string, src io.Reader) (string, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return "", fmt.Errorf("reading upload data: %w", err)
	}
	stored, err := a.s.Store(ctx, filename, bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	return stored.Path, nil
}

func (a *storageAdapter) Delete(ctx context.Context, storagePath string) error {
	// Strip URL prefix if a full URL was accidentally passed.
	path := storagePath
	if idx := strings.Index(path, "/uploads/"); idx != -1 {
		path = path[idx+len("/uploads/"):]
	}
	return a.s.Delete(ctx, path)
}

func (a *storageAdapter) URL(storagePath string) string {
	return a.s.URL(storagePath)
}
