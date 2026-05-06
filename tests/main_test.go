package tests_test

import (
	"coachwise/src/app"
	"coachwise/src/config"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	configPath   string
	router       *gin.Engine
	db           *sqlx.DB
	focused      = false
	authExecuted = false
)

// Setup the test environment before any tests run
var _ = BeforeSuite(func() {
	db, router = setupTestEnvironment()
})

// Drop the database after all tests have run
var _ = AfterSuite(func() {
	teardownTestEnvironment(db)
})

func TestSuite(t *testing.T) {
	checkFocus()
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

var _ = Describe("coachwise Test Suite", func() {
	Context("Ping", pingGroup)
	Context("Auth", authGroup)
	Context("Exercise", exerciseGroup)
	Context("Users", usersGroup)
	Context("Plans", plansGroup)
	Context("Schedules", schedulesGroup)
	Context("Sessions", sessionsGroup)
	Context("Feeds", feedGroup)
	Context("Media", mediaGroup)
	Context("Messages", messagesGroup)
	Context("Edge Cases", edgeCasesGroup)
})

func init() {
	// We back to root dir on execute tests
	os.Chdir("../")
	// Define a flag for the config path
	flag.StringVar(&configPath, "c", "test.config.yml", "Path to the configuration file")
}

func replaceAny(a, b gin.H) {
	for key, valueA := range a {
		if valueB, exists := b[key]; exists {
			// If the value in a is a map, recurse into it
			if mapA, ok := valueA.(gin.H); ok {
				if mapB, ok := valueB.(gin.H); ok {
					replaceAny(mapA, mapB)
				}
			} else if valueA == "<ANY>" {
				// Replace "<ANY>" with the corresponding value from b
				a[key] = valueB
			}
		}
	}
}

func decodeBody(responseBody io.Reader) gin.H {
	body := gin.H{}
	decoder := json.NewDecoder(responseBody)
	decoder.Decode(&body)
	return body
}
func bodyExpect(body, expect gin.H) {
	filtered := gin.H{}
	for k, v := range body {
		filtered[k] = v
	}
	for k := range filtered {
		if _, ok := expect[k]; !ok {
			delete(filtered, k)
		}
	}
	replaceAny(expect, filtered)
	Expect(filtered).To(Equal(expect))
}

// createDatabase creates a new database if it doesn't exist
func createDatabase(dbURL string) error {
	// Parse the database URL to extract database name
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Extract database name from path
	dbName := strings.TrimPrefix(parsedURL.Path, "/")

	// Create connection URL to postgres database (default database)
	parsedURL.Path = "/postgres"
	postgresURL := parsedURL.String()

	// Connect to postgres database
	db, err := sqlx.Connect("postgres", postgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer db.Close()

	// Create the database
	query := fmt.Sprintf("CREATE DATABASE %s", dbName)
	if _, err := db.Exec(query); err != nil {
		// Ignore error if database already exists
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create database: %w", err)
		}
	}

	return nil
}

func setupTestEnvironment() (*sqlx.DB, *gin.Engine) {
	config.Init(configPath)

	// Step 1: Drop existing test database
	log.Println("Dropping existing test database...")
	if err := database.DropDatabase(config.Config.Database.URL); err != nil {
		log.Printf("Warning: Failed to drop database (may not exist): %v", err)
	} else {
		log.Println("Database dropped successfully")
	}

	// Step 2: Create fresh test database
	log.Println("Creating fresh test database...")
	if err := createDatabase(config.Config.Database.URL); err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	log.Println("Check database passed successfully")

	// Step 3: Connect to the NEW database
	log.Println("Connecting to fresh database...")
	db := database.Connect(&database.ConnectOption{
		URL:         config.Config.Database.URL,
		SqlDir:      config.Config.Database.SqlDir,
		MaxRequests: 500,
		Interval:    30 * time.Second,
		Timeout:     5 * time.Second,
	})

	// Step 5: Apply migrations (skip if migrations directory is empty)
	log.Println("Checking for migrations...")
	m, err := migrate.New(
		fmt.Sprintf("file://%s", config.Config.Database.Migrations),
		config.Config.Database.URL,
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}
	log.Println("Migration done !")
	// Step 6: Initialize router
	router := app.Init()
	return db, router
}

func teardownTestEnvironment(db *sqlx.DB) {
	db.Close()
	if err := database.DropDatabase(config.Config.Database.URL); err != nil {
		log.Fatalf("Dropping database %v", err)
	}
}

func checkFocus() {
	for _, arg := range os.Args[1:] {
		if strings.Contains(arg, "focus") {
			focused = true
			break
		}
	}
}
