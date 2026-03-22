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
		Store: st,
		S3:    s3Client,
		Log:   log,
	}

	authMiddleware := middleware.Auth(st)
	adminChain := func(next http.Handler) http.Handler {
		return authMiddleware(middleware.RequireAuth(middleware.RequireAdmin(next)))
	}
	authChain := func(next http.Handler) http.Handler {
		return authMiddleware(middleware.RequireAuth(next))
	}

	mux.HandleFunc("GET /healthz", h.HandleHealthz)

	mux.Handle("POST /api/auth/register", middleware.Locale(http.HandlerFunc(h.HandleRegister)))
	mux.HandleFunc("POST /api/auth/login", h.HandleLogin)
	mux.HandleFunc("POST /api/auth/logout", h.HandleLogout)
	mux.Handle("GET /api/auth/me", authChain(http.HandlerFunc(h.HandleMe)))

	mux.HandleFunc("GET /api/media/{path...}", h.HandleMedia)

	mux.Handle("GET /api/catalog", authMiddleware(http.HandlerFunc(h.HandleListCatalog)))
	mux.Handle("GET /api/catalog/{id}", authMiddleware(http.HandlerFunc(h.HandleGetCatalogPuzzle)))

	mux.Handle("GET /api/admin/catalog/puzzles", adminChain(http.HandlerFunc(h.HandleAdminListCatalog)))
	mux.Handle("POST /api/admin/catalog/puzzles", adminChain(http.HandlerFunc(h.HandleAdminCreateCatalogPuzzle)))
	mux.Handle("GET /api/admin/catalog/puzzles/{id}", adminChain(http.HandlerFunc(h.HandleAdminGetCatalogPuzzle)))
	mux.Handle("PUT /api/admin/catalog/puzzles/{id}", adminChain(http.HandlerFunc(h.HandleAdminUpdateCatalogPuzzle)))
	mux.Handle("DELETE /api/admin/catalog/puzzles/{id}", adminChain(http.HandlerFunc(h.HandleAdminDeleteCatalogPuzzle)))
	mux.Handle("GET /api/admin/catalog/puzzles/{id}/reward", adminChain(http.HandlerFunc(h.HandleAdminGetReward)))
	mux.Handle("POST /api/admin/catalog/puzzles/{id}/reward", adminChain(http.HandlerFunc(h.HandleAdminUpsertReward)))
	mux.Handle("GET /api/admin/users", adminChain(http.HandlerFunc(h.HandleAdminListUsers)))

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
