package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/anton-yak/person-enricher/internal/demografix"
	"github.com/anton-yak/person-enricher/internal/routes"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		sugar.Fatalf("environment variable SERVER_PORT is not set")
	}

	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		sugar.Fatalf("environment variable DATABASE_URL is not set")
	}

	pgxPool, err := pgxpool.New(context.Background(), databaseUrl)
	if err != nil {
		sugar.Fatalf("Unable to connect to database: %v", err)
	}

	r := routes.MakeHTTPHandler(&demografix.Enricher{}, pgxPool, sugar)

	sugar.Infof("listening on 0.0.0.0:%s", serverPort)
	sugar.Fatal(http.ListenAndServe(":"+serverPort, r))
}
