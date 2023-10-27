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
	logger, err := zap.NewDevelopment() // or NewProduction, or NewDevelopment
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()

	pgxPool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		sugar.Fatalf("Unable to connect to database: %v", err)
	}

	r := routes.MakeHTTPHandler(&demografix.Enricher{}, pgxPool, sugar)

	sugar.Info("listening on 0.0.0.0:3000")
	sugar.Fatal(http.ListenAndServe(":3000", r))
}
