package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/hi-im-yan/jwt-with-go/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// @title			CRUD with Go API
// @version		1.0
// @description	Simple CRUD API using Go and PostgreSQL
func main() {
	db := connectDB()
	defer db.Close()

	if err := ensureAdminExists(db); err != nil {
		log.Fatal(err)
	}

	server := server.NewServer("8080", db)

	fmt.Println("Starting server on port " + server.Port)

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}

func ensureAdminExists(db *pgxpool.Pool) error {
	var count int
	err := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(os.Getenv("ADMIN_PASSWORD")), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		_, err = db.Exec(context.Background(), "INSERT INTO users (name, email, password, role) VALUES ($1, $2, $3, $4)",
			"Admin", os.Getenv("ADMIN_EMAIL"), string(hashedPassword), "admin")
		if err != nil {
			return err
		}
		fmt.Println("✅ Admin account created: ", os.Getenv("ADMIN_EMAIL"))
	}
	return nil
}

func connectDB() *pgxpool.Pool {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Read database credentials from environment variables
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")

	// Construct database URL
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPass, dbHost, dbPort, dbName)

	// Run Migrations
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		log.Fatal("Migration error:", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Migration failed:", err)
	}

	fmt.Println("Migrations completed successfully!")

	// Connect to PostgreSQL
	db, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	fmt.Println("Connected to PostgreSQL successfully!")
	return db
}
