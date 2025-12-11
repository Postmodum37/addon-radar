package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"addon-radar/internal/database"
	"addon-radar/internal/trending"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL required")
	}
	
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	
	slog.Info("connected to database")
	
	calc := trending.NewCalculator(database.New(pool))
	if err := calc.CalculateAll(ctx); err != nil {
		log.Fatal(err)
	}
}
