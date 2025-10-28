package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"time"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/pkg/hasher"
	"github.com/Temutjin2k/ride-hail-system/pkg/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	configPath = flag.String("config-path", "config.yaml", "Path to the config yaml file")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	cfg, err := config.NewConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	client, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		log.Fatal(err)
	}

	migrateDefaultUsers(client.Pool)
}

func migrateDefaultUsers(db *pgxpool.Pool) {
	// short timeout for migration operations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type defaultUser struct {
		Email     string
		Role      string
		Status    string
		PlainPass string
		Attrs     map[string]any
	}

	users := []defaultUser{
		{
			Email:     "beka@ride.kz",
			Role:      "PASSENGER",
			Status:    "ACTIVE",
			PlainPass: "password",
		},
		{
			Email:     "mans@ride.kz",
			Role:      "DRIVER",
			Status:    "ACTIVE",
			PlainPass: "password",
		},
		{
			Email:     "temu@ride.kz",
			Role:      "ADMIN",
			Status:    "ACTIVE",
			PlainPass: "password",
		},
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatalf("migrateDefaultUsers: begin tx: %v", err)
	}
	// ensure rollback if commit doesn't happen
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const q = `
INSERT INTO users (email, role, status, password_hash, attrs)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (email) DO NOTHING;
`

	for _, u := range users {
		// hash password
		hashed := hasher.Hash(u.PlainPass)

		attrsJSON, err := json.Marshal(u.Attrs)
		if err != nil {
			log.Fatalf("migrateDefaultUsers: marshal attrs: %v", err)
		}

		if _, err := tx.Exec(ctx, q, u.Email, u.Role, u.Status, hashed, attrsJSON); err != nil {
			log.Fatalf("migrateDefaultUsers: insert user %s: %v", u.Email, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("migrateDefaultUsers: commit: %v", err)
	}

	log.Printf("migrateDefaultUsers: inserted/ensured %d default users", len(users))
}
