package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/do/v2"
	"github.com/spf13/viper"
	"github.com/zaffka/jigsaw/internal/handler"
	"github.com/zaffka/jigsaw/internal/middleware"
	"github.com/zaffka/jigsaw/internal/migrate"
	"github.com/zaffka/jigsaw/internal/store"
	"github.com/zaffka/jigsaw/internal/worker"
	"github.com/zaffka/jigsaw/pkg/di"
	"github.com/zaffka/jigsaw/pkg/s3"
	"go.uber.org/zap"
)

var serviceVersion = "0.0.0"

func main() {
	ctx, stopFn := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopFn()

	viper.AutomaticEnv()

	cfgReader := di.NewViperConfigReader()
	appConfig := di.AppConfig{
		ServiceName:    "jigsaw",
		ServiceVersion: serviceVersion,
	}

	container, err := di.InitializeContainer(ctx, cfgReader, appConfig)
	if err != nil {
		print(err.Error())
		os.Exit(-1)
	}

	log, err := do.Invoke[*zap.Logger](container)
	if err != nil {
		print(err.Error())
		os.Exit(-1)
	}

	pool, err := do.InvokeNamed[*pgxpool.Pool](container, "default")
	if err != nil {
		log.Error("failed to invoke pgxpool", zap.Error(err))
		return
	}

	if _, err := migrate.Run(pool, log); err != nil {
		log.Error("failed to run migrations", zap.Error(err))
		return
	}

	if err := migrate.SeedAdmin(ctx, pool); err != nil {
		log.Error("failed to seed admin", zap.Error(err))
		return
	}

	s3Client, err := do.Invoke[*s3.BucketCli](container)
	if err != nil {
		log.Error("failed to invoke s3 client", zap.Error(err))
		return
	}

	mux, err := do.Invoke[*http.ServeMux](container)
	if err != nil {
		log.Error("failed to invoke serve mux", zap.Error(err))
		return
	}

	st := store.New(pool)
	h := &handler.Handler{
		Store:        st,
		S3:           s3Client,
		Log:          log,
		CookieSecure: viper.GetBool("COOKIE_SECURE"),
	}

	authMiddleware := middleware.Auth(st)
	adminChain := func(next http.Handler) http.Handler {
		return authMiddleware(middleware.RequireAuth(middleware.RequireAdmin(next)))
	}
	authChain := func(next http.Handler) http.Handler {
		return authMiddleware(middleware.RequireAuth(next))
	}

	childAuthMiddleware := middleware.ChildAuth(st)
	childChain := func(next http.Handler) http.Handler {
		return childAuthMiddleware(middleware.RequireChild(next))
	}
	parentChain := func(next http.Handler) http.Handler {
		return authMiddleware(middleware.RequireAuth(next))
	}
	_ = childChain // used below

	mux.HandleFunc("GET /healthz", h.HandleHealthz)

	mux.Handle("POST /api/auth/register", middleware.Locale(http.HandlerFunc(h.HandleRegister)))
	mux.HandleFunc("POST /api/auth/login", h.HandleLogin)
	mux.HandleFunc("POST /api/auth/logout", h.HandleLogout)
	mux.Handle("GET /api/auth/me", authChain(http.HandlerFunc(h.HandleMe)))

	mux.HandleFunc("GET /api/media/{path...}", h.HandleMedia)

	mux.HandleFunc("GET /api/categories", h.HandleListCategories)

	mux.Handle("GET /api/admin/categories", adminChain(http.HandlerFunc(h.HandleAdminListCategories)))
	mux.Handle("POST /api/admin/categories", adminChain(http.HandlerFunc(h.HandleAdminCreateCategory)))
	mux.Handle("PUT /api/admin/categories/{id}", adminChain(http.HandlerFunc(h.HandleAdminUpdateCategory)))
	mux.Handle("DELETE /api/admin/categories/{id}", adminChain(http.HandlerFunc(h.HandleAdminDeleteCategory)))

	mux.Handle("GET /api/catalog", authMiddleware(http.HandlerFunc(h.HandleListCatalog)))
	mux.Handle("GET /api/catalog/{id}", authMiddleware(http.HandlerFunc(h.HandleGetCatalogPuzzle)))

	mux.Handle("GET /api/admin/catalog/puzzles", adminChain(http.HandlerFunc(h.HandleAdminListCatalog)))
	mux.Handle("POST /api/admin/catalog/puzzles", adminChain(http.HandlerFunc(h.HandleAdminCreateCatalogPuzzle)))
	mux.Handle("GET /api/admin/catalog/puzzles/{id}", adminChain(http.HandlerFunc(h.HandleAdminGetCatalogPuzzle)))
	mux.Handle("PUT /api/admin/catalog/puzzles/{id}", adminChain(http.HandlerFunc(h.HandleAdminUpdateCatalogPuzzle)))
	mux.Handle("DELETE /api/admin/catalog/puzzles/{id}", adminChain(http.HandlerFunc(h.HandleAdminDeleteCatalogPuzzle)))
	mux.Handle("GET /api/admin/users", adminChain(http.HandlerFunc(h.HandleAdminListUsers)))

	mux.Handle("GET /api/admin/moderation", adminChain(http.HandlerFunc(h.HandleAdminListModeration)))
	mux.Handle("POST /api/admin/moderation/{id}/approve", adminChain(http.HandlerFunc(h.HandleAdminApprove)))
	mux.Handle("POST /api/admin/moderation/{id}/reject", adminChain(http.HandlerFunc(h.HandleAdminReject)))

	// Child auth
	mux.HandleFunc("POST /api/children/auth", h.HandleChildAuth)

	// Parent → children
	mux.Handle("GET /api/parent/children", parentChain(http.HandlerFunc(h.HandleParentListChildren)))
	mux.Handle("POST /api/parent/children", parentChain(http.HandlerFunc(h.HandleParentCreateChild)))
	mux.Handle("GET /api/parent/children/{id}", parentChain(http.HandlerFunc(h.HandleParentGetChild)))
	mux.Handle("PUT /api/parent/children/{id}", parentChain(http.HandlerFunc(h.HandleParentUpdateChild)))
	mux.Handle("DELETE /api/parent/children/{id}", parentChain(http.HandlerFunc(h.HandleParentDeleteChild)))

	// Parent → puzzles
	mux.Handle("GET /api/parent/puzzles", parentChain(http.HandlerFunc(h.HandleParentListPuzzles)))
	mux.Handle("POST /api/parent/puzzles", parentChain(http.HandlerFunc(h.HandleParentCreatePuzzle)))
	mux.Handle("GET /api/parent/puzzles/{id}", parentChain(http.HandlerFunc(h.HandleParentGetPuzzle)))
	mux.Handle("PUT /api/parent/puzzles/{id}", parentChain(http.HandlerFunc(h.HandleParentUpdatePuzzle)))
	mux.Handle("DELETE /api/parent/puzzles/{id}", parentChain(http.HandlerFunc(h.HandleParentDeletePuzzle)))

	// Parent → layers
	mux.Handle("GET /api/parent/puzzles/{id}/layers", parentChain(http.HandlerFunc(h.HandleParentListLayers)))
	mux.Handle("POST /api/parent/puzzles/{id}/layers", parentChain(http.HandlerFunc(h.HandleParentCreateLayer)))
	mux.Handle("POST /api/parent/puzzles/{id}/layers/reorder", parentChain(http.HandlerFunc(h.HandleParentReorderLayers)))
	mux.Handle("PUT /api/parent/puzzles/{id}/layers/{lid}", parentChain(http.HandlerFunc(h.HandleParentUpdateLayer)))
	mux.Handle("DELETE /api/parent/puzzles/{id}/layers/{lid}", parentChain(http.HandlerFunc(h.HandleParentDeleteLayer)))

	// Parent → moderation
	mux.Handle("POST /api/parent/puzzles/{id}/submit", parentChain(http.HandlerFunc(h.HandleParentSubmitPuzzle)))
	mux.Handle("GET /api/parent/notifications", parentChain(http.HandlerFunc(h.HandleParentListNotifications)))

	// Play (best-effort, no auth required)
	mux.HandleFunc("GET /api/play/completed", h.HandlePlayCompleted)
	mux.Handle("POST /api/play/{id}/complete", http.HandlerFunc(h.HandlePlayComplete))

	httpServer, err := do.Invoke[*http.Server](container)
	if err != nil {
		log.Error("failed to invoke http server", zap.Error(err))
		return
	}

	go func() {
		log.Info("starting http server", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", zap.Error(err))
		}
	}()

	go worker.New(st, s3Client, log).Run(ctx)

	<-ctx.Done()
	log.Info("shutting down")
	httpServer.Shutdown(context.Background())
}
