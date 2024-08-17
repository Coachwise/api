package database

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sony/gobreaker"
)

var (
	db          *sqlx.DB
	cb          *gobreaker.CircuitBreaker
	cbStateChan chan gobreaker.State
	sqlDir      string
)

type ConnectOption struct {
	URL         string
	SqlDir      string
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
}

// Connect initializes the database connection and sets up the circuit breaker
func Connect(options *ConnectOption) *sqlx.DB {
	if err := checkOrCreateDB(options.URL); err != nil {
		log.Fatal(err)
	}
	database, err := sqlx.Open("postgres", options.URL)
	if err != nil {
		log.Fatalf("could not connect to the database: %v", err)
	}

	if err := database.Ping(); err != nil {
		log.Fatalf("could not ping the database: %v", err)
	}
	// Initialize the circuit breaker
	cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "DBCircuitBreaker",
		MaxRequests: options.MaxRequests,
		Interval:    options.Interval,
		Timeout:     options.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 3
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Printf("Circuit breaker state changed from %s to %s\n", from, to)
			if cbStateChan != nil {
				cbStateChan <- to
			}
		},
	})
	sqlDir = options.SqlDir
	db = database
	return database
}

func DropDatabase(dbURL string) error {
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		log.Fatalf("Failed to parse connection URL: %v", err)
	}

	// Remove the database name from the URL (set the Path to empty)
	serverURL := *parsedURL
	serverURL.Path = ""

	// Convert the URL back to a string
	serverConnStr := serverURL.String()

	serverDB, err := sqlx.Connect("postgres", serverConnStr)
	if err != nil {
		return err
	}
	defer serverDB.Close()

	dbName := parsedURL.Path[1:] // Get the database name from the original URL's Path

	// Terminate all active connections to the database
	terminateConnQuery := fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		  AND pid <> pg_backend_pid();
	`, dbName)

	_, err = serverDB.Exec(terminateConnQuery)
	if err != nil {
		return fmt.Errorf("failed to terminate active connections: %v", err)
	}

	dropQuery := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
	if _, err := serverDB.Exec(dropQuery); err != nil {
		return err
	}
	log.Printf("Database %s dropped successfully\n", dbName)
	return nil
}

// GetDB returns the database connection
func GetDB() *sqlx.DB {
	return db
}

// Close closes the database connection
func Close() {
	if db != nil {
		db.Close()
	}
}

func checkOrCreateDB(dbURL string) error {
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		log.Fatalf("Failed to parse connection URL: %v", err)
	}

	// Remove the database name from the URL (set the Path to empty)
	serverURL := *parsedURL
	serverURL.Path = ""

	// Convert the URL back to a string
	serverConnStr := serverURL.String()

	serverDB, err := sqlx.Connect("postgres", serverConnStr)
	if err != nil {
		return err
	}
	defer serverDB.Close()

	dbName := parsedURL.Path[1:] // Get the database name from the original URL's Path

	// Check if the database exists
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = '%s')", dbName)
	if err := serverDB.Get(&exists, query); err != nil {
		return err
	}
	if exists {
		log.Printf("Check database %s passed successfully\n", dbName)
		return nil
	}
	// If the database doesn't exist, create it
	_, err = serverDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return err
	}
	log.Printf("Database %s created successfully\n", dbName)
	return nil
}
