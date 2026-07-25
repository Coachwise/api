// Package database is the app's handle on godatabase.
//
// It owns the single *godatabase.DB and re-exports the calls models make, so
// model code stays free of connection plumbing. Every call logs the underlying
// error before returning it: the driver's circuit breaker opens after four
// consecutive failures and from then on every caller sees only "circuit breaker
// is open", which hides the fault that tripped it. Logging here keeps the first
// error — the useful one — in the server log.
package database

import (
	"context"
	"database/sql"
	"log"
	"time"

	"coachwise/src/metrics"

	"github.com/jeyem/godatabase"
	"github.com/jmoiron/sqlx"
)

// ConnectOption configures the connection and its circuit breaker.
type ConnectOption struct {
	URL         string
	SqlDir      string
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
}

// Filter is a single `?filter.key=value` pair from a list request.
type Filter struct {
	Key   string
	Value string
}

// Paginate carries the window a list endpoint asked for.
type Paginate struct {
	Limit   int
	Offset  int
	Filters []Filter
}

// FetchList is the id + window total that list queries select before hydrating
// their rows (`COUNT(*) OVER () AS total_count`).
type FetchList = godatabase.FetchList

var db *godatabase.DB

// Connect opens the pool and returns it. It fails hard: without a database
// there is nothing to serve.
func Connect(options *ConnectOption) *sqlx.DB {
	d, err := godatabase.Connect(context.Background(), &godatabase.ConnectOption{
		URL:         options.URL,
		SqlDir:      options.SqlDir,
		MaxRequests: options.MaxRequests,
		Interval:    options.Interval,
		Timeout:     options.Timeout,
	})
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	db = d
	return d.Conn()
}

// GetDB returns the raw pool, for transactions and anything the named-query
// helpers don't cover.
func GetDB() *sqlx.DB { return db.Conn() }

// DropDatabase drops the database in the URL. Tests use it; nothing else should.
func DropDatabase(url string) error {
	return godatabase.DropDatabase(context.Background(), url)
}

// Query runs a named query and returns rows for manual scanning.
func Query(ctx context.Context, queryName string, args ...interface{}) (*sqlx.Rows, error) {
	start := time.Now()
	rows, err := db.Query(ctx, queryName, args...)
	metrics.ObserveQuery(queryName, time.Since(start), err)
	return rows, logErr(queryName, err)
}

// QuerySelect runs a named query into a slice.
func QuerySelect(queryName string, dest interface{}, args ...interface{}) error {
	start := time.Now()
	err := db.Select(context.Background(), dest, queryName, args...)
	metrics.ObserveQuery(queryName, time.Since(start), err)
	return logErr(queryName, err)
}

// Get runs a named query expected to return exactly one row.
func Get(dest interface{}, queryName string, args ...interface{}) error {
	start := time.Now()
	err := db.Get(context.Background(), dest, queryName, args...)
	metrics.ObserveQuery(queryName, time.Since(start), err)
	return logErr(queryName, err)
}

// Fetch loads records by id. The query name comes from the destination's
// FetchQuery, so dest must implement godatabase.Model.
func Fetch(dest interface{}, ids ...interface{}) error {
	start := time.Now()
	err := db.Fetch(context.Background(), dest, ids...)
	metrics.ObserveQuery("fetch", time.Since(start), err)
	return logErr("fetch", err)
}

// TxQuery runs a named query inside tx and returns rows.
func TxQuery(ctx context.Context, tx *sqlx.Tx, queryName string, args ...interface{}) (*sqlx.Rows, error) {
	start := time.Now()
	rows, err := db.TxQuery(ctx, tx, queryName, args...)
	metrics.ObserveQuery(queryName, time.Since(start), err)
	return rows, logErr(queryName, err)
}

// TxExecuteQuery runs a named write query inside tx, binding the named
// parameters (:id, :email) from data.
func TxExecuteQuery(tx *sqlx.Tx, queryName string, data interface{}) (sql.Result, error) {
	start := time.Now()
	res, err := db.TxExec(context.Background(), tx, queryName, data)
	metrics.ObserveQuery(queryName, time.Since(start), err)
	return res, logErr(queryName, err)
}

// UnmarshalJSONTextFields hydrates a row's JSONText columns into their typed
// siblings. Fetch and Get already do this; callers scanning rows by hand don't.
func UnmarshalJSONTextFields(input interface{}) error {
	return godatabase.UnmarshalJSONTextFields(input)
}

// logErr records which named query failed and why, then hands the error back
// untouched. sql.ErrNoRows is an ordinary "not found" and stays quiet.
func logErr(queryName string, err error) error {
	if err != nil && err != sql.ErrNoRows {
		log.Printf("database: query %q failed: %v", queryName, err)
	}
	return err
}
