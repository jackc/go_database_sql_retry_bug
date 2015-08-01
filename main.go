package main

import (
	"database/sql"
	"fmt"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	_ "github.com/lib/pq"
	"os"
	"strings"
)

func main() {
	connPoolConfig, err := extractConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "extractConfig failed:", err)
		os.Exit(1)
	}

	pgxStdlib, err := openPgxStdlib(connPoolConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, "openPgxStdLib failed: %v", err)
		os.Exit(1)
	}

	pq, err := openPq(connPoolConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, "openPq failed: %v", err)
		os.Exit(1)
	}

	nameDb := map[string]*sql.DB{
		"github.com/jackc/pgx/stdlib": pgxStdlib,
		"github.com/lib/pq":           pq,
	}

	for name, db := range nameDb {
		err = resetSchema(db)
		if err != nil {
			fmt.Fprintln(os.Stderr, "resetSchema failed:", err)
			os.Exit(1)
		}

		fmt.Println("Testing with", name)
		testUpdates(db, 10000)
	}
}

func extractConfig() (config pgx.ConnPoolConfig, err error) {
	config.ConnConfig, err = pgx.ParseEnvLibpq()
	if err != nil {
		return config, err
	}

	if config.Host == "" {
		config.Host = "localhost"
	}

	if config.User == "" {
		config.User = os.Getenv("USER")
	}

	if config.Database == "" {
		config.Database = "go_database_sql_retry_bug"
	}

	config.MaxConnections = 10

	return config, nil
}

func resetSchema(db *sql.DB) error {
	var err error
	for i := 0; i < 100; i++ {
		_, err = db.Exec(`
		drop table if exists t;
		create table t(n int not null);
		insert into t(n) values(0);`)
		if err == nil {
			return nil
		}

	}

	return err
}

func openPgxStdlib(config pgx.ConnPoolConfig) (*sql.DB, error) {
	connPool, err := pgx.NewConnPool(config)
	if err != nil {
		return nil, err
	}

	return stdlib.OpenFromConnPool(connPool)
}

func openPq(config pgx.ConnPoolConfig) (*sql.DB, error) {
	var options []string
	options = append(options, fmt.Sprintf("host=%s", config.Host))
	options = append(options, fmt.Sprintf("user=%s", config.User))
	options = append(options, fmt.Sprintf("dbname=%s", config.Database))
	options = append(options, "sslmode=disable")
	if config.Password != "" {
		options = append(options, fmt.Sprintf("password=%s", config.Password))
	}

	return sql.Open("postgres", strings.Join(options, " "))
}

func testUpdates(db *sql.DB, updateCount int) {
	errCount := 0
	for i := 0; i < updateCount; i++ {
		_, err := db.Exec("update t set n=n+1")
		if err != nil {
			errCount += 1
		}
	}

	var actualUpdates int64
	var err error
	for i := 0; i < 100; i++ {
		err = db.QueryRow("select n from t").Scan(&actualUpdates)
		if err == nil {
			break
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error selecting number of actual updates: %v", err)
		os.Exit(1)
	}

	fmt.Println("Exec'ed statements:", updateCount)
	fmt.Println("Reported error count:", errCount)
	fmt.Println("Actual updates:", actualUpdates)
}
