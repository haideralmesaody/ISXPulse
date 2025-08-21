package app

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"isxcli/internal/config"
	"isxcli/internal/errors"
	handlers "isxcli/internal/transport/http"
	"isxcli/internal/infrastructure"
	"isxcli/internal/license"
	customMiddleware "isxcli/internal/middleware"
	"isxcli/internal/operations"
	"isxcli/internal/services"
	"isxcli/internal/updater"
	ws "isxcli/internal/websocket"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/gorilla/websocket"
)

const (
	VERSION    = "enhanced-v3.0.0"
	REPO_URL   = "https://github.com/haideralmesaody/ISXDailyReportScrapper"
	AppName    = "ISX Pulse - The Heartbeat of Iraqi Markets"
	Executable = "ISXPulse.exe"
)

var (
	// BuildTime is set at compile time
	BuildTime = time.Now().Format(time.RFC3339)
	// BuildID is a unique identifier for this build
	BuildID = generateBuildID()
)

func generateBuildID() string {
	// Generate a deterministic build ID based on version and time
	h := sha256.New()
	h.Write([]byte(VERSION))
	h.Write([]byte(time.Now().Format("2006-01-02")))
	return fmt.Sprintf("%x", h.Sum(nil))[:12]
}

// Application represents the main application container
type Application struct {
	Config          *config.Config
	Router          *chi.Mux
	Server          *http.Server
	LicenseManager  *license.Manager
	WebSocketHub    *ws.Hub
	OperationService *services.OperationService
	DataService     *services.DataService
	HealthService   *services.HealthService
	UpdateChecker   *updater.AutoUpdateChecker
	Logger         *slog.Logger // Single slog instance per CLAUDE.md
	Services        *ServiceContainer
	OTelProviders   *infrastructure.OTelProviders // OpenTelemetry providers
	FrontendFS      fs.FS // Embedded frontend filesystem
	JobQueue        *operations.JobQueue // Async job queue for operations
}

// ServiceContainer holds all application services
type ServiceContainer struct {
	License  *license.Manager
	LicenseService services.LicenseService
	operation *services.OperationService
	Data     *services.DataService
	Health   *services.HealthService
	WebSocket *ws.Hub
	Liquidity *services.LiquidityService
}

// NewApplication creates a new application instance with dependency injection
func NewApplication(frontendFS fs.FS) (*Application, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize single infrastructure logger per CLAUDE.md
	logger, err := infrastructure.InitializeLogger(cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	
	// Log startup information
	logger.Info("Application starting",
		slog.String("name", AppName),
		slog.String("version", VERSION),
		slog.String("executable", Executable))
	
	// Validate and log all paths at startup for debugging
	paths, err := config.GetPaths()
	if err != nil {
		return nil, fmt.Errorf("failed to get paths: %w", err)
	}
	
	// Ensure all required directories exist
	logger.Info("Ensuring required directories exist")
	if err := paths.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to ensure directories: %w", err)
	}
	
	// Log all resolved paths at startup for debugging
	paths.LogPathResolution()
	
	// Validate license file exists (non-fatal if missing, will be handled by license manager)
	if !config.FileExists(paths.LicenseFile) {
		logger.Warn("License file not found",
			slog.String("path", paths.LicenseFile),
			slog.String("action", "License activation will be required"))
	}

	// Initialize OpenTelemetry
	otelProviders, err := infrastructure.InitializeOTel(infrastructure.DefaultOTelConfig(), logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}

	// Initialize global operation tracer
	if err := operations.InitGlobalOperationTracer(otelProviders); err != nil {
		return nil, fmt.Errorf("failed to initialize operation tracer: %w", err)
	}

	// License tracer initialization removed during simplification
	// The license system now uses standard observability through slog and OpenTelemetry

	// Initialize WebSocket OpenTelemetry metrics
	if err := ws.InitOTelMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize WebSocket OpenTelemetry metrics: %w", err)
	}

	// Create application
	app := &Application{
		Config:        cfg,
		Logger:        logger,
		OTelProviders: otelProviders,
		FrontendFS:    frontendFS,
	}

	// Initialize services in order
	if err := app.initializeServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Setup router
	app.setupRouter()

	// Create HTTP server
	app.createServer()

	return app, nil
}

// initializeServices initializes all application services
func (a *Application) initializeServices() error {
	// Initialize license manager
	licensePath := a.Config.GetLicenseFile()
	licenseManager, err := license.NewManager(licensePath)
	if err != nil {
		return fmt.Errorf("failed to initialize license manager: %w", err)
	}
	a.LicenseManager = licenseManager

	// Initialize WebSocket hub
	hub := ws.NewHub(a.Logger)
	hub.Start() // Start the hub's goroutines
	a.WebSocketHub = hub

	// Initialize operation service
	OperationAdapter := services.NewWebSocketOperationAdapter(hub)
	OperationService, err := services.NewOperationService(OperationAdapter, a.Logger)
	if err != nil {
		return fmt.Errorf("failed to initialize operation service: %w", err)
	}
	a.OperationService = OperationService
	
	// Initialize job queue for async operations
	jobStore := operations.NewMemoryJobStore()
	manager := a.OperationService.GetManager()
	a.JobQueue = operations.NewJobQueue(4, jobStore, manager, a.Logger) // 4 workers by default
	
	// Start the job queue
	ctx := context.Background()
	a.JobQueue.Start(ctx)

	// Initialize data service with injected logger
	dataService, err := services.NewDataServiceWithLogger(a.Config, a.Logger)
	if err != nil {
		return fmt.Errorf("failed to initialize data service: %w", err)
	}
	a.DataService = dataService

	// Initialize health service with injected logger
	healthService := services.NewHealthServiceWithBuildInfo(
		VERSION,
		REPO_URL,
		BuildTime,
		BuildID,
		a.Config.Paths,
		a.LicenseManager,
		a.OperationService.GetManager(),
		a.WebSocketHub,
		a.Logger,
	)
	a.HealthService = healthService

	// Initialize update checker
	upd, err := updater.NewUpdater(VERSION, REPO_URL)
	if err != nil {
		return fmt.Errorf("failed to initialize updater: %w", err)
	}
	
	updateChecker := updater.NewAutoUpdateChecker(upd, 24*time.Hour, func(info *updater.UpdateInfo) bool {
		a.Logger.Info("Update available", 
			slog.String("current", info.CurrentVersion), 
			slog.String("latest", info.LatestVersion))
		return false // Don't auto-install
	})
	a.UpdateChecker = updateChecker

	// Initialize license service
	licenseService := services.NewLicenseService(licenseManager, a.Logger)

	// Get paths for liquidity service
	paths, err := config.GetPaths()
	if err != nil {
		return fmt.Errorf("failed to get paths for liquidity service: %w", err)
	}

	// Initialize liquidity service
	liquidityService := services.NewLiquidityService(paths.ReportsDir, a.Logger)


	// Create service container
	a.Services = &ServiceContainer{
		License:   licenseManager,
		LicenseService: licenseService,
		operation:  OperationService,
		Data:      dataService,
		Health:    healthService,
		WebSocket: hub,
		Liquidity: liquidityService,
	}

	return nil
}

// setupRouter configures the HTTP router with all routes
func (a *Application) setupRouter() {
	r := chi.NewRouter()

	// Apply MINIMAL middleware that won't interfere with WebSocket
	// These are safe because they don't wrap the ResponseWriter
	r.Use(customMiddleware.RequestID) // Use our CLAUDE.md compliant RequestID
	r.Use(customMiddleware.RealIP)

	// WebSocket route with minimal middleware and tracing
	// MUST be registered after minimal middleware but before the group
	r.With(customMiddleware.WebSocketTraceMiddleware(a.Logger)).HandleFunc("/ws", a.handleWebSocket)

	// Serve static assets OUTSIDE middleware group to avoid license validation
	if a.FrontendFS != nil {
		a.setupStaticAssetsOnly(r)
	}

	// Create a route group for everything else with FULL middleware
	r.Group(func(r chi.Router) {
		// Apply remaining middleware only to this group
		// Follow CLAUDE.md ordering: RequestID → RealIP → OTel → Logger → Recoverer → Timeout
		
		// OpenTelemetry middleware for tracing and metrics
		otelMiddleware, err := customMiddleware.NewOTelMiddleware(a.OTelProviders)
		if err != nil {
			a.Logger.Error("Failed to create OpenTelemetry middleware", slog.String("error", err.Error()))
		} else {
			r.Use(otelMiddleware.Handler)
		}
		
		// Business metrics middleware
		businessMetrics, _ := infrastructure.CreateBusinessMetrics(a.OTelProviders.Meter)
		r.Use(customMiddleware.BusinessMetricsMiddleware(businessMetrics))
		
		r.Use(customMiddleware.StructuredLogger(a.Logger)) // Use infrastructure logger
		r.Use(customMiddleware.Recoverer(a.Logger)) // Use our CLAUDE.md compliant recoverer
		// NOTE: Timeout middleware moved to specific route groups below to allow different timeouts for operations
		r.Use(customMiddleware.SecurityHeaders)
		
		// CORS middleware - configured for embedded frontend and development
		corsConfig := a.getCORSConfig()
		r.Use(customMiddleware.CORS(corsConfig))
		
		// Rate limiting
		if a.Config.Security.RateLimit.Enabled {
			r.Use(customMiddleware.NewRateLimiter(
				a.Config.Security.RateLimit.RPS,
				a.Config.Security.RateLimit.Burst,
				a.Logger, // Pass infrastructure logger
			).Handler)
		}
		
		// License validation
		licenseValidator := customMiddleware.NewLicenseValidator(a.LicenseManager, a.Logger)
		r.Use(licenseValidator.Handler)
		
		// Now register all other routes within this group
		a.setupAPIRoutes(r)
		a.setupHTMLRoutes(r)
		// Note: setupHTMLRoutes includes embedded frontend serving, which replaces setupStaticRoutes
	})

	// Add Prometheus metrics endpoint (outside the middleware group for performance)
	if a.OTelProviders.PrometheusHTTP != nil {
		r.Handle("/metrics", a.OTelProviders.PrometheusHTTP)
	}

	a.Router = r
}

// setupMiddleware is no longer used - middleware is now applied in setupRouter using route groups
// Keeping this comment for reference to the middleware that was moved

// setupAPIRoutes configures API endpoints
func (a *Application) setupAPIRoutes(r chi.Router) {
	// API routes with common middleware
	r.Route("/api", func(r chi.Router) {
		r.Use(render.SetContentType(render.ContentTypeJSON))

		// Apply standard timeout to most API endpoints
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.Timeout(a.Config.Server.ReadTimeout, a.Logger))
			
			// Health handler
			healthHandler := handlers.NewHealthHandler(a.HealthService, a.Logger)
			r.Get("/health", healthHandler.HealthCheck)
			r.Get("/health/ready", healthHandler.ReadinessCheck)
			r.Get("/health/live", healthHandler.LivenessCheck)
			r.Get("/version", healthHandler.Version)

			// Metrics and observability handler
			metricsHandler := handlers.NewMetricsHandler()
			{
				r.Mount("/metrics", metricsHandler.Routes())
			}

			// License endpoints
			licenseHandler := handlers.NewLicenseHandler(a.Services.LicenseService, a.Logger)
			r.Mount("/license", licenseHandler.Routes())

			// Create error handler
			errorHandler := errors.NewErrorHandler(a.Logger, false)

			// Data handler
			dataHandler := handlers.NewDataHandler(a.DataService, a.Logger, errorHandler)
			r.Mount("/data", dataHandler.Routes())
			
			// Liquidity handler
			liquidityHandler := handlers.NewLiquidityHandler(a.Services.Liquidity, a.Logger)
			liquidityHandler.RegisterRoutes(r)
			
		})

		// Operations handler with longer timeout for long-running operations
		r.Group(func(r chi.Router) {
			// Use operation-specific timeout (2 hours by default)
			r.Use(customMiddleware.Timeout(a.Config.Server.OperationTimeout, a.Logger))
			
			OperationHandler := handlers.NewOperationsHandler(a.OperationService, a.WebSocketHub, a.Logger)
			// Set the job queue for async operations
			OperationHandler.SetJobQueue(a.JobQueue)
			r.Mount("/operations", OperationHandler.Routes())
			
			// Operation shortcuts with tracing - also need longer timeout
			r.Post("/scrape", customMiddleware.PipelineTraceHandler("scrape", func(w http.ResponseWriter, r *http.Request) {
				var params map[string]interface{}
				if err := render.DecodeJSON(r.Body, &params); err != nil {
					render.JSON(w, r, map[string]interface{}{"error": "Invalid request"})
					return
				}
				pipelineID, err := a.OperationService.StartScraping(r.Context(), params)
				if err != nil {
					render.JSON(w, r, map[string]interface{}{"error": err.Error()})
					return
				}
				render.JSON(w, r, map[string]interface{}{"pipeline_id": pipelineID, "status": "started"})
			}))
			r.Post("/process", customMiddleware.PipelineTraceHandler("process", func(w http.ResponseWriter, r *http.Request) {
				var params map[string]interface{}
				if err := render.DecodeJSON(r.Body, &params); err != nil {
					render.JSON(w, r, map[string]interface{}{"error": "Invalid request"})
					return
				}
				pipelineID, err := a.OperationService.StartProcessing(r.Context(), params)
				if err != nil {
					render.JSON(w, r, map[string]interface{}{"error": err.Error()})
					return
				}
				render.JSON(w, r, map[string]interface{}{"pipeline_id": pipelineID, "status": "started"})
			}))
			r.Post("/indexcsv", customMiddleware.PipelineTraceHandler("indexcsv", func(w http.ResponseWriter, r *http.Request) {
				var params map[string]interface{}
				if err := render.DecodeJSON(r.Body, &params); err != nil {
					render.JSON(w, r, map[string]interface{}{"error": "Invalid request"})
					return
				}
				pipelineID, err := a.OperationService.StartIndexExtraction(r.Context(), params)
				if err != nil {
					render.JSON(w, r, map[string]interface{}{"error": err.Error()})
					return
				}
				render.JSON(w, r, map[string]interface{}{"pipeline_id": pipelineID, "status": "started"})
			}))
		})
		
		// Client logging endpoint with standard timeout
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.Timeout(a.Config.Server.ReadTimeout, a.Logger))
			r.Post("/logs", handlers.NewClientLogHandler(a.Logger).Handle)
		})
	})
}

// setupStaticRoutes configures static file serving
func (a *Application) setupStaticRoutes(r chi.Router) {
	staticDir := filepath.Join(a.Config.GetWebDir(), "static")
	templatesDir := filepath.Join(a.Config.GetWebDir(), "templates")

	// Serve static files
	r.Route("/static", func(r chi.Router) {
		r.Use(middleware.Compress(5))
		r.Handle("/*", http.StripPrefix("/static", http.FileServer(http.Dir(staticDir))))
	})

	// Serve templates
	r.Route("/templates", func(r chi.Router) {
		r.Handle("/*", http.StripPrefix("/templates", http.FileServer(http.Dir(templatesDir))))
	})
}

// setupHTMLRoutes configures HTML page routes and embedded Next.js frontend
func (a *Application) setupHTMLRoutes(r chi.Router) {
	// Serve embedded Next.js frontend
	a.setupEmbeddedFrontend(r)
	
	// Legacy routes for backward compatibility (served from filesystem if available)
	r.Get("/legacy/license", handlers.ServeLicensePage(a.Config.GetWebDir()))
	r.Get("/legacy/app", handlers.ServeMainApp(a.Config.GetWebDir()))
	r.Get("/test", handlers.ServeTestPage())
}

// setupStaticAssetsOnly configures ONLY static assets without middleware
func (a *Application) setupStaticAssetsOnly(r chi.Router) {
	frontendFS := a.FrontendFS
	
	// Serve static assets (JS, CSS, images, etc.) with cache-busting headers
	r.Route("/_next", func(r chi.Router) {
		// Use no-cache for development, aggressive caching for production with versioned files
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if file has hash in name (cache-busted)
				if strings.Contains(r.URL.Path, ".") && (strings.Contains(r.URL.Path, "-") || strings.Contains(r.URL.Path, "_")) {
					// Versioned assets can be cached aggressively
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					// Non-versioned assets should not be cached
					w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
					w.Header().Set("Pragma", "no-cache")
					w.Header().Set("Expires", "0")
				}
				next.ServeHTTP(w, r)
			})
		})
		r.HandleFunc("/*", a.serveStaticWithMIME(frontendFS, "/_next").ServeHTTP)
	})

	// Serve other frontend static assets (from public folder)
	r.Route("/assets", func(r chi.Router) {
		r.Use(middleware.SetHeader("Cache-Control", "public, max-age=86400"))
		r.HandleFunc("/*", a.serveStaticWithMIME(frontendFS, "/assets").ServeHTTP)
	})

	// Serve favicon and other root assets
	r.Get("/favicon.ico", a.serveFrontendFile(frontendFS, "favicon.ico"))
	r.Get("/favicon-16x16.png", a.serveFrontendFile(frontendFS, "favicon-16x16.png"))
	r.Get("/favicon-32x32.png", a.serveFrontendFile(frontendFS, "favicon-32x32.png"))
	r.Get("/apple-touch-icon.png", a.serveFrontendFile(frontendFS, "apple-touch-icon.png"))
	r.Get("/android-chrome-192x192.png", a.serveFrontendFile(frontendFS, "android-chrome-192x192.png"))
	r.Get("/android-chrome-512x512.png", a.serveFrontendFile(frontendFS, "android-chrome-512x512.png"))
	r.Get("/site.webmanifest", a.serveFrontendFile(frontendFS, "site.webmanifest"))
	r.Get("/robots.txt", a.serveFrontendFile(frontendFS, "robots.txt"))
	r.Get("/iraqi-investor-logo.svg", a.serveFrontendFile(frontendFS, "iraqi-investor-logo.svg"))
}

// setupEmbeddedFrontend configures the embedded Next.js frontend serving (SPA routes only)
func (a *Application) setupEmbeddedFrontend(r chi.Router) {
	// Check if frontend filesystem is available
	if a.FrontendFS == nil {
		a.Logger.Warn("Frontend filesystem not available, falling back to legacy handlers")
		r.Get("/", handlers.RedirectToLicense)
		return
	}
	
	frontendFS := a.FrontendFS

	// NOTE: Static assets are served by setupStaticAssetsOnly outside middleware group
	
	// SPA routing - serve index.html for all unmatched routes
	// This must be last to catch all routes not handled by API or static assets
	r.Get("/*", a.serveSPAHandler(frontendFS))
}

// serveFrontendFile serves a specific file from the embedded frontend
func (a *Application) serveFrontendFile(frontendFS fs.FS, filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, err := frontendFS.Open(filename)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		// Set appropriate content type
		switch filepath.Ext(filename) {
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
		case ".txt":
			w.Header().Set("Content-Type", "text/plain")
		case ".json":
			w.Header().Set("Content-Type", "application/json")
		}

		// Set caching headers
		w.Header().Set("Cache-Control", "public, max-age=86400")

		io.Copy(w, file)
	}
}

// serveStaticWithMIME creates a file server that properly sets MIME types for embedded files
func (a *Application) serveStaticWithMIME(frontendFS fs.FS, prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For embedded filesystem, we need to map URL paths to the correct embedded structure
		// URLs like /_next/static/... should map to _next/static/... in embedded FS
		path := r.URL.Path
		
		// Remove leading slash for fs.Open, but preserve the prefix structure
		if strings.HasPrefix(path, "/") {
			path = path[1:]
		}
		
		// Debug logging
		a.Logger.InfoContext(r.Context(), "Static file request",
			"original_url", r.URL.Path,
			"prefix", prefix,
			"resolved_path", path)
		
		// Try to open the file
		file, err := frontendFS.Open(path)
		if err != nil {
			a.Logger.WarnContext(r.Context(), "Static file not found",
				"path", path,
				"error", err.Error())
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		
		// Set content type based on extension
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".json":
			w.Header().Set("Content-Type", "application/json")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".gif":
			w.Header().Set("Content-Type", "image/gif")
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
		case ".woff2":
			w.Header().Set("Content-Type", "font/woff2")
		case ".woff":
			w.Header().Set("Content-Type", "font/woff")
		case ".ttf":
			w.Header().Set("Content-Type", "font/ttf")
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		
		// Set security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		
		// Copy file content
		io.Copy(w, file)
	})
}

// serveSPAHandler serves the Next.js SPA with license-first routing
func (a *Application) serveSPAHandler(frontendFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract request ID for logging
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = fmt.Sprintf("spa-%d", time.Now().UnixNano())
		}
		
		ctx := infrastructure.WithTraceID(r.Context(), reqID)
		
		// Log the SPA request
		a.Logger.InfoContext(ctx, "SPA route request",
			slog.String("path", r.URL.Path),
			slog.String("method", r.Method),
			slog.String("user_agent", r.Header.Get("User-Agent")))

		// LICENSE-FIRST FLOW: Always redirect root to /license page
		if r.URL.Path == "/" {
			a.Logger.InfoContext(ctx, "Redirecting root to license page (license-first flow)")
			http.Redirect(w, r, "/license", http.StatusTemporaryRedirect)
			return
		}

		// Check license status for protected routes (redirect to /license if invalid)
		// Skip license check for license page itself and API routes
		if r.URL.Path != "/license" && !strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/_next/") {
			if a.Services != nil && a.Services.LicenseService != nil {
				status, err := a.Services.LicenseService.GetStatus(ctx)
				// Allow both "active" and "warning" states (warning means <30 days left)
				if err != nil || (status.LicenseStatus != "active" && status.LicenseStatus != "warning") {
					a.Logger.InfoContext(ctx, "Redirecting to license page - invalid license",
						slog.String("path", r.URL.Path),
						slog.String("license_status", status.LicenseStatus),
						slog.Bool("has_error", err != nil))
					http.Redirect(w, r, "/license", http.StatusTemporaryRedirect)
					return
				}
			}
		}

		// Clean the path
		urlPath := path.Clean(r.URL.Path)
		
		// Try to serve the specific file first
		if urlPath != "/" {
			exactPath := strings.TrimPrefix(urlPath, "/")
			a.Logger.InfoContext(ctx, "Trying exact path",
				slog.String("url_path", urlPath),
				slog.String("exact_path", exactPath))
			file, err := frontendFS.Open(exactPath)
			if err == nil {
				defer file.Close()
				
				// Check if this is a directory (which we don't want to serve directly)
				if stat, statErr := file.Stat(); statErr == nil && stat.IsDir() {
					a.Logger.InfoContext(ctx, "Exact path is directory, skipping",
						slog.String("exact_path", exactPath))
					// Continue to Next.js route check
				} else {
					a.Logger.InfoContext(ctx, "Serving exact file",
						slog.String("exact_path", exactPath))
					
					// Set content type based on extension
				ext := filepath.Ext(urlPath)
				switch ext {
				case ".html":
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
				case ".js":
					w.Header().Set("Content-Type", "application/javascript")
				case ".css":
					w.Header().Set("Content-Type", "text/css")
				case ".json":
					w.Header().Set("Content-Type", "application/json")
				case ".svg":
					w.Header().Set("Content-Type", "image/svg+xml")
				case ".png":
					w.Header().Set("Content-Type", "image/png")
				case ".jpg", ".jpeg":
					w.Header().Set("Content-Type", "image/jpeg")
				case ".gif":
					w.Header().Set("Content-Type", "image/gif")
				case ".ico":
					w.Header().Set("Content-Type", "image/x-icon")
				}
				
				// Set security headers
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.Header().Set("X-Frame-Options", "DENY")
				w.Header().Set("X-XSS-Protection", "1; mode=block")
				
				io.Copy(w, file)
				return
				}
			}
			
			// For Next.js routes like /license, try /license/index.html
			indexPath := strings.TrimPrefix(urlPath, "/") + "/index.html"
			a.Logger.InfoContext(ctx, "Trying Next.js route index.html",
				slog.String("url_path", urlPath),
				slog.String("index_path", indexPath))
			indexFile, err := frontendFS.Open(indexPath)
			if err == nil {
				defer indexFile.Close()
				
				a.Logger.InfoContext(ctx, "Serving Next.js route index.html",
					slog.String("route", urlPath),
					slog.String("file_path", indexPath))
				
				// Set headers for HTML content
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.Header().Set("X-Frame-Options", "DENY")
				w.Header().Set("X-XSS-Protection", "1; mode=block")
				w.Header().Set("X-Build-Time", time.Now().Format(time.RFC3339))
				
				io.Copy(w, indexFile)
				return
			} else {
				a.Logger.WarnContext(ctx, "Next.js route index.html not found",
					slog.String("url_path", urlPath),
					slog.String("index_path", indexPath),
					slog.String("error", err.Error()))
			}
		}

		// Fallback to index.html for SPA routing
		indexFile, err := frontendFS.Open("index.html")
		if err != nil {
			a.Logger.ErrorContext(ctx, "Failed to open index.html", 
				slog.String("error", err.Error()),
				slog.String("path", urlPath))
			http.Error(w, "Frontend not available", http.StatusServiceUnavailable)
			return
		}
		defer indexFile.Close()

		// Set headers for HTML content
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Serve index.html for client-side routing
		io.Copy(w, indexFile)
		
		a.Logger.DebugContext(ctx, "Served SPA index.html",
			slog.String("original_path", urlPath))
	}
}

// getCORSConfig returns CORS configuration based on environment
func (a *Application) getCORSConfig() customMiddleware.CORSConfig {
	// Detect environment
	isDevelopment := a.isDevelopmentMode()
	
	config := customMiddleware.CORSConfig{
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization", 
			"Content-Type",
			"X-CSRF-Token",
			"X-Request-ID",
			"X-Requested-With",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
		},
		AllowCredentials: true,
		MaxAge:           300,
		Logger:           a.Logger,
	}
	
	if isDevelopment {
		// Development mode: Allow Next.js dev server
		config.AllowedOrigins = []string{
			"http://localhost:3000",  // Next.js dev server
			"http://127.0.0.1:3000",
			"http://localhost:8080",  // Go server
			"http://127.0.0.1:8080",
		}
		a.Logger.Info("CORS configured for development mode",
			slog.Any("allowed_origins", config.AllowedOrigins))
	} else {
		// Production mode: Only allow same origin
		config.AllowedOrigins = []string{
			"http://localhost:8080",
			"http://127.0.0.1:8080",
		}
		
		// Add any configured origins
		if a.Config.Security.EnableCORS && len(a.Config.Security.AllowedOrigins) > 0 {
			config.AllowedOrigins = append(config.AllowedOrigins, a.Config.Security.AllowedOrigins...)
		}
		
		a.Logger.Info("CORS configured for production mode",
			slog.Any("allowed_origins", config.AllowedOrigins))
	}
	
	return config
}

// isDevelopmentMode detects if we're running in development mode
func (a *Application) isDevelopmentMode() bool {
	// Check environment variable
	if env := os.Getenv("NODE_ENV"); env == "development" {
		return true
	}
	if env := os.Getenv("GO_ENV"); env == "development" {
		return true
	}
	
	// Check if Next.js dev files exist (indicates development)
	if _, err := os.Stat("frontend/package.json"); err == nil {
		if _, err := os.Stat("frontend/.next"); err == nil {
			return true
		}
	}
	
	// Check if running from dev directory
	if wd, err := os.Getwd(); err == nil {
		if strings.Contains(wd, "dev") || strings.Contains(wd, "development") {
			return true
		}
	}
	
	return false
}

// createServer creates the HTTP server
func (a *Application) createServer() {
	a.Server = &http.Server{
		Addr:         fmt.Sprintf(":%d", a.Config.Server.Port),
		Handler:      a.Router,
		ReadTimeout:  a.Config.Server.ReadTimeout,
		WriteTimeout: a.Config.Server.WriteTimeout,
		IdleTimeout:  a.Config.Server.IdleTimeout,
	}
}

// Start starts the application
func (a *Application) Start(ctx context.Context, cancel context.CancelFunc) error {
	a.Logger.InfoContext(ctx, "Starting application",
		slog.String("name", AppName),
		slog.String("version", VERSION),
		slog.Int("port", a.Config.Server.Port),
		slog.String("level", a.Config.Logging.Level))
	
	// Log important paths for debugging
	paths, _ := config.GetPaths()
	a.Logger.InfoContext(ctx, "Application paths",
		slog.String("executable_dir", paths.ExecutableDir),
		slog.String("data_dir", paths.DataDir),
		slog.String("web_dir", paths.WebDir),
		slog.String("logs_dir", paths.LogsDir),
		slog.String("license_file", paths.LicenseFile))

	// Start background services
	go a.WebSocketHub.Run()
	go a.UpdateChecker.Start()

	// Start server
	go func() {
		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.Logger.ErrorContext(ctx, "Server error", slog.String("error", err.Error()))
			// Signal shutdown through context instead of os.Exit
			cancel()
		}
	}()

	// Perform health check on critical paths
	err := a.performStartupHealthCheck(ctx)
	if err != nil {
		a.Logger.WarnContext(ctx, "Startup health check warnings", slog.String("warnings", err.Error()))
	}
	
	a.Logger.InfoContext(ctx, "Application started successfully",
		slog.String("address", fmt.Sprintf("http://localhost:%d", a.Config.Server.Port)),
		slog.String("license_status", "checking..."))

	// Open browser after server is ready
	go func() {
		// Create independent context for browser opening
		browserCtx := context.Background()
		
		// Wait for server to be ready by checking health endpoint
		url := fmt.Sprintf("http://localhost:%d", a.Config.Server.Port)
		healthURL := fmt.Sprintf("%s/api/health", url)
		
		// Try to connect to health endpoint with retries
		maxRetries := 10
		for i := 0; i < maxRetries; i++ {
			// Check if main context is cancelled
			select {
			case <-ctx.Done():
				a.Logger.InfoContext(ctx, "Browser opening cancelled - application shutting down")
				return
			default:
			}
			
			// Try health check
			resp, err := http.Get(healthURL)
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				a.Logger.InfoContext(browserCtx, "Server is ready, opening browser", 
					slog.String("url", url),
					slog.Int("attempts", i+1))
				
				// Server is ready, open browser
				if err := openBrowser(url); err != nil {
					a.Logger.ErrorContext(browserCtx, "Failed to open browser after server ready", 
						slog.String("error", err.Error()),
						slog.String("url", url))
					
					// Print manual instruction
					fmt.Printf("\n")
					fmt.Printf("========================================\n")
					fmt.Printf("ISX Pulse is running!\n")
					fmt.Printf("Please open your browser and navigate to:\n")
					fmt.Printf("  %s\n", url)
					fmt.Printf("========================================\n")
					fmt.Printf("\n")
				} else {
					a.Logger.InfoContext(browserCtx, "Browser opened successfully", 
						slog.String("url", url))
				}
				return
			}
			
			if resp != nil {
				resp.Body.Close()
			}
			
			// Wait before retry
			time.Sleep(500 * time.Millisecond)
		}
		
		// If we get here, server never became ready
		a.Logger.ErrorContext(browserCtx, "Server did not become ready for browser opening", 
			slog.String("url", url),
			slog.Int("max_retries", maxRetries))
	}()

	return nil
}

// Stop gracefully stops the application
func (a *Application) Stop(ctx context.Context) error {
	a.Logger.InfoContext(ctx, "Shutting down application")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, a.Config.Server.ShutdownTimeout)
	defer cancel()

	// Stop server
	if err := a.Server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	// Stop background services
	a.UpdateChecker.Stop()
	a.WebSocketHub.Stop()
	
	// Stop job queue with timeout
	if a.JobQueue != nil {
		a.Logger.InfoContext(ctx, "Stopping job queue")
		if err := a.JobQueue.Stop(30 * time.Second); err != nil {
			a.Logger.ErrorContext(ctx, "Failed to stop job queue gracefully", slog.String("error", err.Error()))
		}
	}

	// Cancel running operations
	if err := a.OperationService.CancelAll(ctx); err != nil {
		a.Logger.ErrorContext(ctx, "Error cancelling operations", slog.String("error", err.Error()))
	}

	// Shutdown OpenTelemetry providers
	if a.OTelProviders != nil {
		if err := a.OTelProviders.Shutdown(shutdownCtx); err != nil {
			a.Logger.ErrorContext(ctx, "Error shutting down OpenTelemetry", slog.String("error", err.Error()))
		}
	}

	a.Logger.InfoContext(ctx, "Application shutdown complete")
	return nil
}

// Run runs the application until interrupted
func (a *Application) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start application
	if err := a.Start(ctx, cancel); err != nil {
		return err
	}

	// Wait for interrupt
	<-sigChan
	a.Logger.InfoContext(ctx, "Received interrupt signal")

	// Graceful shutdown
	return a.Stop(ctx)
}

// handleWebSocket handles WebSocket connections
func (a *Application) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract any available request ID (might not have middleware)
	reqID := r.Header.Get("X-Request-ID")
	if reqID == "" {
		reqID = fmt.Sprintf("ws-%d", time.Now().UnixNano())
	}
	
	// Structured logging per CLAUDE.md
	ctx := infrastructure.WithTraceID(r.Context(), reqID)
	a.Logger.InfoContext(ctx, "WebSocket upgrade request",
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("origin", r.Header.Get("Origin")),
		slog.String("host", r.Host),
		slog.String("user_agent", r.UserAgent()))
	
	// Set CORS headers explicitly for WebSocket upgrade
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Handle cases where Origin header is missing (e.g., file:// protocol)
		origin = fmt.Sprintf("http://%s", r.Host)
		a.Logger.WarnContext(ctx, "No Origin header in WebSocket request, using host",
			slog.String("host", r.Host))
	}
	
	// Set WebSocket-specific CORS headers
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
	
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			
			// Allow if no origin (local file or same-origin request)
			if origin == "" {
				a.Logger.DebugContext(ctx, "WebSocket origin check - no origin header, allowing",
					slog.String("host", r.Host))
				return true
			}
			
			// In development mode, be more permissive
			if a.isDevelopmentMode() {
				a.Logger.DebugContext(ctx, "WebSocket origin check - development mode, allowing",
					slog.String("origin", origin))
				return true
			}
			
			// In production, validate against allowed origins
			corsConfig := a.getCORSConfig()
			for _, allowed := range corsConfig.AllowedOrigins {
				if origin == allowed {
					a.Logger.DebugContext(ctx, "WebSocket origin check - origin allowed",
						slog.String("origin", origin))
					return true
				}
			}
			
			a.Logger.WarnContext(ctx, "WebSocket origin check - origin not allowed",
				slog.String("origin", origin),
				slog.Any("allowed_origins", corsConfig.AllowedOrigins))
			return false
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// Add error handler for better debugging
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			a.Logger.ErrorContext(ctx, "WebSocket upgrade error",
				slog.Int("status", status),
				slog.String("reason", reason.Error()),
				slog.String("origin", r.Header.Get("Origin")))
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		a.Logger.ErrorContext(ctx, "WebSocket upgrade failed",
			slog.String("error", err.Error()),
			slog.String("details", fmt.Sprintf("%+v", err)),
			slog.String("origin", origin))
		return
	}

	// Create a new client with trace ID and register with hub
	client := ws.NewClientWithTrace(a.WebSocketHub, conn, reqID, a.Logger)
	a.WebSocketHub.Register(client)
	
	a.Logger.InfoContext(ctx, "WebSocket client connected",
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("request_id", reqID))

	// Start client goroutines with proper error handling
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.Logger.ErrorContext(ctx, "WebSocket write pump panic",
					slog.Any("panic", r),
					slog.String("request_id", reqID))
			}
		}()
		client.WritePump()
	}()
	
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.Logger.ErrorContext(ctx, "WebSocket read pump panic",
					slog.Any("panic", r),
					slog.String("request_id", reqID))
			}
		}()
		client.ReadPump()
	}()
}

// performStartupHealthCheck performs health checks on critical paths and resources
func (a *Application) performStartupHealthCheck(ctx context.Context) error {
	paths, err := config.GetPaths()
	if err != nil {
		return fmt.Errorf("failed to get paths: %w", err)
	}
	
	var warnings []string
	
	// Check critical directories are writable
	directories := map[string]string{
		"Data":      paths.DataDir,
		"Downloads": paths.DownloadsDir,
		"Reports":   paths.ReportsDir,
		"Cache":     paths.CacheDir,
		"Logs":      paths.LogsDir,
	}
	
	for name, dir := range directories {
		// Try to create a test file to verify write access
		testFile := filepath.Join(dir, ".write_test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s directory not writable: %s", name, dir))
		} else {
			// Clean up test file
			os.Remove(testFile)
		}
	}
	
	// Check web directory exists and has content
	if !config.FileExists(paths.WebDir) {
		warnings = append(warnings, fmt.Sprintf("Web directory not found: %s", paths.WebDir))
	}
	
	// Check for critical configuration files (non-fatal)
	configFiles := map[string]string{
		"Credentials": paths.CredentialsFile,
		"Sheets Config": paths.SheetsConfigFile,
	}
	
	for name, file := range configFiles {
		if !config.FileExists(file) {
			a.Logger.InfoContext(ctx, "Configuration file not found",
				slog.String("file", name),
				slog.String("path", file))
		}
	}
	
	if len(warnings) > 0 {
		return fmt.Errorf("startup health check warnings: %s", strings.Join(warnings, "; "))
	}
	
	a.Logger.InfoContext(ctx, "Startup health check passed")
	return nil
}

// openBrowser opens the default browser to the specified URL with retry logic
func openBrowser(url string) error {
	var lastErr error
	
	// Try multiple methods with retries
	methods := getBrowserOpenMethods(url)
	
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Wait before retry with exponential backoff
			time.Sleep(time.Duration(attempt) * time.Second)
			slog.Info("Retrying browser open", 
				slog.Int("attempt", attempt+1),
				slog.String("url", url))
		}
		
		for _, method := range methods {
			slog.Info("Attempting to open browser", 
				slog.String("method", method.name),
				slog.String("command", method.cmd),
				slog.Any("args", method.args),
				slog.String("url", url))
			
			cmd := exec.Command(method.cmd, method.args...)
			
			// Set a timeout for the command
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd = exec.CommandContext(ctx, method.cmd, method.args...)
			
			if err := cmd.Start(); err != nil {
				lastErr = err
				slog.Warn("Browser open method failed", 
					slog.String("method", method.name),
					slog.String("error", err.Error()))
				continue
			}
			
			// Give the browser a moment to start
			time.Sleep(500 * time.Millisecond)
			
			// Success
			slog.Info("Browser opened successfully", 
				slog.String("method", method.name),
				slog.String("url", url))
			return nil
		}
	}
	
	return fmt.Errorf("failed to open browser after all attempts: %w", lastErr)
}

// browserMethod represents a method to open the browser
type browserMethod struct {
	name string
	cmd  string
	args []string
}

// getBrowserOpenMethods returns platform-specific browser opening methods
func getBrowserOpenMethods(url string) []browserMethod {
	switch runtime.GOOS {
	case "windows":
		return []browserMethod{
			{
				name: "start_command",
				cmd:  "cmd",
				args: []string{"/c", "start", "", url},
			},
			{
				name: "rundll32",
				cmd:  "rundll32",
				args: []string{"url.dll,FileProtocolHandler", url},
			},
			{
				name: "powershell",
				cmd:  "powershell",
				args: []string{"-Command", fmt.Sprintf("Start-Process '%s'", url)},
			},
			{
				name: "explorer",
				cmd:  "explorer",
				args: []string{url},
			},
		}
	case "darwin":
		return []browserMethod{
			{
				name: "open",
				cmd:  "open",
				args: []string{url},
			},
		}
	default: // Linux and others
		return []browserMethod{
			{
				name: "xdg-open",
				cmd:  "xdg-open",
				args: []string{url},
			},
			{
				name: "sensible-browser",
				cmd:  "sensible-browser",
				args: []string{url},
			},
			{
				name: "firefox",
				cmd:  "firefox",
				args: []string{url},
			},
			{
				name: "chromium",
				cmd:  "chromium",
				args: []string{url},
			},
		}
	}
}