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
	"github.com/go-chi/httprate"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/crypto"
	"github.com/stoa-hq/stoa/internal/admin"
	"github.com/stoa-hq/stoa/internal/storefront"
	"github.com/stoa-hq/stoa/internal/auth"
	"github.com/stoa-hq/stoa/internal/config"
	"github.com/stoa-hq/stoa/internal/database"
	"github.com/stoa-hq/stoa/pkg/sdk"
	"github.com/stoa-hq/stoa/internal/domain/audit"
	"github.com/stoa-hq/stoa/internal/domain/cart"
	"github.com/stoa-hq/stoa/internal/domain/category"
	"github.com/stoa-hq/stoa/internal/domain/customer"
	"github.com/stoa-hq/stoa/internal/domain/discount"
	domainmedia "github.com/stoa-hq/stoa/internal/domain/media"
	"github.com/stoa-hq/stoa/internal/domain/order"
	"github.com/stoa-hq/stoa/internal/domain/warehouse"
	"github.com/stoa-hq/stoa/internal/domain/payment"
	"github.com/stoa-hq/stoa/internal/domain/product"
	"github.com/stoa-hq/stoa/internal/domain/shipping"
	"github.com/stoa-hq/stoa/internal/domain/tag"
	"github.com/stoa-hq/stoa/internal/domain/tax"
	storagemedia "github.com/stoa-hq/stoa/internal/media"
	"github.com/stoa-hq/stoa/internal/plugin"
	"github.com/stoa-hq/stoa/internal/search"
	"github.com/stoa-hq/stoa/internal/server"
	"github.com/stoa-hq/stoa/internal/settings"
)

type App struct {
	Config          *config.Config
	DB              *database.DB
	Server          *server.Server
	JWTManager      *auth.JWTManager
	AuthMiddleware  *auth.Middleware
	TokenBlacklist  *auth.TokenBlacklist
	PluginRegistry  *plugin.Registry
	Logger          zerolog.Logger
}

func New(cfg *config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().Caller().Logger()

	db, err := database.New(cfg.Database, logger)
	if err != nil {
		return nil, err
	}

	jwtManager, err := auth.NewJWTManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("initializing jwt manager: %w", err)
	}

	apiKeyManager := auth.NewAPIKeyManager(db.Pool)
	tokenBlacklist := auth.NewTokenBlacklist()
	authMiddleware := auth.NewMiddleware(jwtManager, apiKeyManager, tokenBlacklist)

	pluginRegistry := plugin.NewRegistry(logger)

	srv := server.New(cfg, db, logger)

	// Auto-register plugins that called sdk.Register() in their init().
	for _, p := range sdk.RegisteredPlugins() {
		// Create a per-plugin asset router at /plugins/{name}/assets/*
		// Use StripPrefix so http.FileServer sees paths relative to the FS root.
		assetPrefix := "/plugins/" + p.Name() + "/assets"
		assetRouter := chi.NewRouter()
		srv.Router().Handle(assetPrefix+"/*", http.StripPrefix(assetPrefix, assetRouter))

		pluginAppCtx := &plugin.AppContext{
			DB:          db.Pool,
			Router:      srv.Router(),
			AssetRouter: assetRouter,
			Config:      cfg.Plugins,
			Logger:      logger,
			Auth: &sdk.AuthHelper{
				OptionalAuth: authMiddleware.OptionalAuth,
				Required:     authMiddleware.Authenticate,
				RequireRole: func(roles ...string) func(http.Handler) http.Handler {
					authRoles := make([]auth.Role, len(roles))
					for i, r := range roles {
						authRoles[i] = auth.Role(r)
					}
					return authMiddleware.RequireRole(authRoles...)
				},
				UserID:   auth.UserID,
				UserType: auth.UserType,
			},
		}
		if err := pluginRegistry.Register(p, pluginAppCtx); err != nil {
			logger.Warn().Err(err).Str("plugin", p.Name()).Msg("failed to register plugin")
		}
	}

	// Collect UI extensions from all registered plugins.
	pluginRegistry.CollectUIExtensions()

	a := &App{
		Config:          cfg,
		DB:              db,
		Server:          srv,
		JWTManager:      jwtManager,
		AuthMiddleware:  authMiddleware,
		TokenBlacklist:  tokenBlacklist,
		PluginRegistry:  pluginRegistry,
		Logger:          logger,
	}

	if err := a.setupDomains(cfg); err != nil {
		return nil, fmt.Errorf("setting up domains: %w", err)
	}

	if err := a.migratePaymentEncryption(cfg); err != nil {
		return nil, fmt.Errorf("migrating payment encryption: %w", err)
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
	paymentKey, err := crypto.ParseKey(cfg.Payment.EncryptionKey)
	if err != nil {
		return fmt.Errorf("payment encryption key: %w", err)
	}
	pmethodRepo  := payment.NewPostgresMethodRepository(pool, log, paymentKey)
	ptxRepo      := payment.NewPostgresTransactionRepository(pool, log)
	discountRepo := discount.NewPostgresRepository(pool, log)
	tagRepo      := tag.NewPostgresRepository(pool, log)
	auditRepo    := audit.NewPostgresRepository(pool, log)
	mediaRepo    := domainmedia.NewPostgresRepository(pool, log)
	warehouseRepo := warehouse.NewPostgresRepository(pool, log)

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

	warehouseSvc := warehouse.NewService(warehouseRepo, hooks, log)
	categorySvc := category.NewService(categoryRepo, hooks, log)
	customerSvc := customer.NewCustomerService(customerRepo, hooks, log)
	orderSvc    := order.NewService(orderRepo, &orderStockAdapter{ws: warehouseSvc}, hooks, log)
	cartSvc     := cart.NewCartService(cartRepo, warehouseSvc, hooks, log)
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

	apiKeyManager := auth.NewAPIKeyManager(pool)
	bruteForce := auth.NewBruteForceTracker(
		cfg.Security.BruteForce.MaxAttempts,
		cfg.Security.BruteForce.LockDuration,
	)
	tokenStore := auth.NewRefreshTokenStore(pool)
	authH     := auth.NewHandler(pool, a.JWTManager, apiKeyManager, bruteForce, tokenStore, a.TokenBlacklist, log)
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
	productPriceFn := order.ProductPriceFn(func(ctx context.Context, productID uuid.UUID, variantID *uuid.UUID) (int, int, string, string, error) {
		p, err := productSvc.GetByID(ctx, productID)
		if err != nil {
			return 0, 0, "", "", fmt.Errorf("product not found: %w", err)
		}
		// Resolve the default locale name for the order line item.
		name := ""
		for _, t := range p.Translations {
			name = t.Name
			break
		}
		// When a variant is specified, use variant prices and SKU.
		if variantID != nil {
			for _, v := range p.Variants {
				if v.ID == *variantID {
					sku := v.SKU
					priceNet := p.PriceNet
					priceGross := p.PriceGross
					if v.PriceNet != nil {
						priceNet = *v.PriceNet
					}
					if v.PriceGross != nil {
						priceGross = *v.PriceGross
					}
					return priceNet, priceGross, name, sku, nil
				}
			}
			return 0, 0, "", "", fmt.Errorf("variant not found")
		}
		return p.PriceNet, p.PriceGross, name, p.SKU, nil
	})
	paymentMethodCheckFn := order.PaymentMethodCheckFn(func(ctx context.Context, id *uuid.UUID) (bool, bool, string, error) {
		active := true
		methods, _, err := pmethodSvc.List(ctx, payment.PaymentMethodFilter{Page: 1, Limit: 1, Active: &active})
		if err != nil {
			return false, false, "", err
		}
		hasActive := len(methods) > 0
		if id == nil {
			return hasActive, false, "", nil
		}
		m, err := pmethodSvc.GetByID(ctx, *id)
		if err != nil {
			return hasActive, false, "", nil
		}
		return hasActive, m.Active, m.Provider, nil
	})
	orderH    := order.NewHandler(orderSvc, shippingCostFn, checkoutTaxRateFn, productPriceFn, paymentMethodCheckFn, validate, log, cfg.Security.CSRF.Secure)
	cartH     := cart.NewHandler(cartSvc, log)
	taxH      := tax.NewHandler(taxSvc, log)
	shippingH := shipping.NewHandler(shippingSvc, log)
	orderOwnershipFn := payment.OrderOwnershipFn(func(ctx context.Context, orderID uuid.UUID) (*uuid.UUID, string, error) {
		o, err := orderRepo.FindByID(ctx, orderID)
		if err != nil {
			return nil, "", err
		}
		return o.CustomerID, o.GuestToken, nil
	})
	paymentH  := payment.NewHandler(pmethodSvc, ptxSvc, orderOwnershipFn, log)
	discountH := discount.NewHandler(discountSvc, log)
	tagH      := tag.NewHandler(tagSvc, log)
	auditH    := audit.NewHandler(auditSvc, log)
	mediaH    := domainmedia.NewHandler(mediaSvc, log)
	warehouseH := warehouse.NewHandler(warehouseSvc, validate, log)
	settingsRepo := settings.NewPostgresRepository(pool, log)
	settingsSvc := settings.NewService(settingsRepo, log)
	settingsH := settings.NewHandler(settingsSvc, cfg, validate, log)

	// ── Routes ────────────────────────────────────────────────────────────────

	r := a.Server.Router()

	// ── Plugin Manifest ──────────────────────────────────────────────────────
	manifestH := plugin.NewManifestHandler(a.PluginRegistry)

	// /api/v1/auth/* – no authentication required, login has dedicated rate limit
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.With(httprate.LimitByIP(cfg.Security.RateLimit.Login.RequestsPerMinute, time.Minute)).
			Post("/login", authH.HandleLogin)
		r.Post("/refresh", authH.HandleRefresh)
		r.Post("/logout", authH.HandleLogout)
	})

	// /api/v1/admin/* – JWT required, staff roles only
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(a.AuthMiddleware.Authenticate)
		r.Use(a.AuthMiddleware.RequireRole(
			auth.RoleSuperAdmin, auth.RoleAdmin, auth.RoleManager,
		))
		r.Use(audit.Middleware(auditSvc, log))

		productH.RegisterAdminRoutes(r)
		r.Group(func(r chi.Router) {
			r.Use(a.AuthMiddleware.RequireRole(auth.RoleSuperAdmin, auth.RoleAdmin))
			authH.RegisterAdminRoutes(r)
		})
		r.Route("/categories", categoryH.RegisterAdminRoutes)
		customerH.RegisterAdminRoutes(r)
		orderH.RegisterAdminRoutes(r)
		taxH.RegisterAdminRoutes(r)
		shippingH.RegisterAdminRoutes(r)
		paymentH.RegisterAdminRoutes(r)
		r.Get("/orders/{orderID}/transactions", paymentH.ListTransactionsByOrder)
		discountH.RegisterAdminRoutes(r)
		tagH.RegisterAdminRoutes(r)
		auditH.RegisterAdminRoutes(r)
		mediaH.RegisterAdminRoutes(r)
		warehouseH.RegisterAdminRoutes(r)
		settingsH.RegisterAdminRoutes(r)
		r.Get("/plugin-manifest", manifestH.AdminManifest)
	})

	// ── Search ────────────────────────────────────────────────────────────────

	var searchEngine search.Engine
	if sdkEngine := a.PluginRegistry.SearchEngine(); sdkEngine != nil {
		searchEngine = search.NewSDKEngineAdapter(sdkEngine)
		log.Info().Msg("using plugin search engine")
	} else {
		searchEngine = search.NewPostgresEngine(pool, log)
	}
	searchH := search.NewHandler(searchEngine, log)

	// /api/v1/store/* – public; optional auth enriches context for customer routes
	r.Route("/api/v1/store", func(r chi.Router) {
		r.Use(a.AuthMiddleware.OptionalAuth)
		r.Use(audit.Middleware(auditSvc, log))

		productH.RegisterStoreRoutes(r)
		r.Route("/categories", categoryH.RegisterStoreRoutes)
		// Customer: /register with dedicated rate limit, /account without
		r.With(httprate.LimitByIP(cfg.Security.RateLimit.Register.RequestsPerMinute, time.Minute)).
			Post("/register", customerH.StoreRegister)
		r.Get("/account", customerH.StoreGetAccount)
		r.Put("/account", customerH.StoreUpdateAccount)

		// Order: /checkout with dedicated rate limit, /account/orders without
		r.With(httprate.LimitByIP(cfg.Security.RateLimit.Checkout.RequestsPerMinute, time.Minute)).
			Post("/checkout", orderH.StoreCheckout)
		r.Get("/account/orders", orderH.StoreListOrders)
		cartH.RegisterStoreRoutes(r)
		shippingH.RegisterStoreRoutes(r)
		paymentH.RegisterStoreRoutes(r)
		// Guest order transaction lookup with dedicated rate limit.
		r.With(httprate.LimitByIP(cfg.Security.RateLimit.GuestOrder.RequestsPerMinute, time.Minute)).
			Get("/orders/{orderID}/transactions", paymentH.ListTransactionsByOrderStore)
		r.With(httprate.LimitByIP(cfg.Security.RateLimit.Search.RequestsPerMinute, time.Minute)).
			Get("/search", searchH.Search)
		settingsH.RegisterStoreRoutes(r)
		r.Get("/plugin-manifest", manifestH.StoreManifest)
	})

	// ── Uploaded media files ──────────────────────────────────────────────────

	// Serve local uploads at /uploads/*  (no-op when using S3 storage)
	if cfg.Media.Storage != "s3" {
		uploadsDir := http.Dir(cfg.Media.LocalPath)
		r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(uploadsDir)))
	}

	// ── Dynamic CSP from plugin external scripts ────────────────────────────

	csp := buildCSP(a.PluginRegistry.UIExtensions())

	// ── Admin Frontend ────────────────────────────────────────────────────────

	// Serve embedded SvelteKit SPA under /admin/*
	adminHandler := admin.HandlerWithCSP(csp)
	r.Handle("/admin", adminHandler)
	r.Handle("/admin/*", adminHandler)

	// ── Storefront ────────────────────────────────────────────────────────────

	// Serve embedded SvelteKit storefront SPA at the root.
	// Registered last so that /api and /admin take priority.
	storefrontHandler := storefront.HandlerWithCSP(csp)
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

	a.TokenBlacklist.Stop()

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

// buildCSP constructs a CSP string that includes plugin external script sources.
// Domains from ExternalScripts are also added to frame-src and connect-src
// because payment providers like Stripe embed iframes and make API calls.
func buildCSP(extensions []sdk.UIExtension) string {
	scriptSources := "'self' 'nonce-{{NONCE}}' 'strict-dynamic'"
	frameSources := "'self'"
	connectSources := "'self'"
	seen := make(map[string]bool)
	for _, ext := range extensions {
		if ext.Component == nil {
			continue
		}
		for _, src := range ext.Component.ExternalScripts {
			if !seen[src] {
				seen[src] = true
				scriptSources += " " + src
				frameSources += " " + src
				connectSources += " " + src
			}
		}
	}
	return "default-src 'self'; script-src " + scriptSources +
		"; frame-src " + frameSources +
		"; connect-src " + connectSources +
		"; style-src 'self' 'unsafe-inline'"
}

func (a *App) migratePaymentEncryption(cfg *config.Config) error {
	paymentKey, err := crypto.ParseKey(cfg.Payment.EncryptionKey)
	if err != nil {
		return err
	}
	repo := payment.NewPostgresMethodRepository(a.DB.Pool, a.Logger, paymentKey)
	migrator, ok := repo.(interface{ MigrateEncryption(context.Context) error })
	if !ok {
		return nil
	}
	return migrator.MigrateEncryption(context.Background())
}

// orderStockAdapter bridges warehouse.Service to order.stockDeductor by converting
// between the two StockDeductionItem types (avoiding circular imports).
type orderStockAdapter struct {
	ws *warehouse.Service
}

func (a *orderStockAdapter) DeductStock(ctx context.Context, items []order.StockDeductionItem) error {
	whItems := make([]warehouse.StockDeductionItem, len(items))
	for i, it := range items {
		whItems[i] = warehouse.StockDeductionItem{
			ProductID: it.ProductID,
			VariantID: it.VariantID,
			Quantity:  it.Quantity,
			OrderID:   it.OrderID,
		}
	}
	return a.ws.DeductStock(ctx, whItems)
}

func (a *orderStockAdapter) RestoreStock(ctx context.Context, orderID uuid.UUID) error {
	return a.ws.RestoreStock(ctx, orderID)
}
