package main

import (
	"context"
	"net"
	"os"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/go-kit/kit/log"
	"google.golang.org/grpc"
	"security-central/ent"
	"security-central/internal/db"
	"security-central/internal/repository"
	"security-central/internal/service"
	"security-central/internal/worker"
	auditpb "security-central/proto"
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

	if err := client.Schema.Create(context.Background()); err != nil {
		_ = logger.Log("fatal", err)
		os.Exit(1)
	}

	repo := repository.NewEntRepository(client, sqlDB)
	svc := service.New(repo, nil)
	grpcServer := grpc.NewServer()
	auditpb.RegisterAuditExecutorServer(grpcServer, worker.NewServer(svc, logger))

	port := os.Getenv("WORKER_PORT")
	if port == "" {
		port = "9090"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		_ = logger.Log("fatal", err)
		os.Exit(1)
	}

	_ = logger.Log("msg", "gRPC worker listening", "addr", ":"+port)
	if err := grpcServer.Serve(lis); err != nil {
		_ = logger.Log("fatal", err)
		os.Exit(1)
	}
}
