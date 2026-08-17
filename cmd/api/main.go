package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"security-central/ent"
	"security-central/internal/db"
	"security-central/internal/endpoint"
	"security-central/internal/repository"
	"security-central/internal/service"
	"security-central/internal/transport"
	"security-central/internal/worker"
	auditpb "security-central/proto"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/go-kit/kit/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := log.NewLogfmtLogger(os.Stdout)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/security_central?sslmode=disable"
	}

	sqlDB, err := db.OpenPostgres(dsn)
	if err != nil {
		_ = logger.Log("fatal", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	drv := entsql.OpenDB(dialect.Postgres, sqlDB)
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		_ = logger.Log("fatal", err)
		os.Exit(1)
	}

	workerAddr := os.Getenv("AUDIT_WORKER_ADDR")
	if workerAddr == "" {
		workerAddr = "localhost:9090"
	}
	grpcConn, err := grpc.NewClient(workerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = logger.Log("fatal", err)
		os.Exit(1)
	}
	defer grpcConn.Close()

	workerClient := worker.NewClient(auditpb.NewAuditExecutorClient(grpcConn))
	repo := repository.NewEntRepository(client, sqlDB)
	svc := service.New(repo, workerClient)
	eps := endpoint.New(svc)
	handler := transport.NewHTTPHandler(eps, logger)

	if err := svc.EnsureDefaultUsers(ctx); err != nil {
		_ = logger.Log("error", fmt.Sprintf("failed to seed users: %v", err))
	}

	svc.StartScheduler(ctx)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		_ = logger.Log("msg", fmt.Sprintf("HTTP server listening on :%s", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			_ = logger.Log("fatal", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
