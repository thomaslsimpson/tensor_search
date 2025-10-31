package main

import (
	"database/sql"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

func TestSQLite(t *testing.T) {
	// Enable sqlite-vec extension for all database connections
	sqlite_vec.Auto()

	// Open the database
	db, err := sql.Open("sqlite3", "./rc_domain_embeds.sqlite3")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Verify vec0 extension is loaded by checking vec_version
	var vecVersion string
	err = db.QueryRow("SELECT vec_version()").Scan(&vecVersion)
	if err != nil {
		t.Fatalf("Failed to verify vec0 extension (vec_version check failed): %v", err)
	}
	t.Logf("vec0 extension loaded successfully, version: %s", vecVersion)

	// Test query: Find 10 matches for ford.com
	query := `
		SELECT d2.domain, d2.country, d2.distance
		FROM domains AS d2
		WHERE d2.embedding MATCH (SELECT embedding FROM domains WHERE domain = 'ford.com')
		AND k = 10
		ORDER BY d2.distance
	`

	rows, err := db.Query(query)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}
	defer rows.Close()

	var results []struct {
		domain   string
		country  string
		distance float64
	}

	for rows.Next() {
		var domain string
		var country string
		var distance float64
		if err := rows.Scan(&domain, &country, &distance); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		results = append(results, struct {
			domain   string
			country  string
			distance float64
		}{domain, country, distance})
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("Error iterating rows: %v", err)
	}

	// Verify we got results
	if len(results) == 0 {
		t.Fatal("Query returned no results")
	}

	// Verify we got 10 results (or at least some results)
	if len(results) < 1 {
		t.Fatalf("Expected at least 1 result, got %d", len(results))
	}

	// Verify we got exactly 10 results
	if len(results) != 10 {
		t.Errorf("Expected exactly 10 results, got %d", len(results))
	}

	// Verify ford.com is in the results
	foundFord := false
	for _, result := range results {
		if result.domain == "ford.com" {
			foundFord = true
			break
		}
	}

	if !foundFord {
		t.Errorf("Expected to find 'ford.com' in results, but didn't. Got %d results", len(results))
		// Print what we got for debugging
		for _, result := range results {
			t.Logf("Domain: %s, Country: %s, Distance: %f", result.domain, result.country, result.distance)
		}
	} else {
		// Log all results for verification
		t.Logf("Retrieved %d matching domains:", len(results))
		for _, result := range results {
			t.Logf("  Domain: %s, Country: %s, Distance: %f", result.domain, result.country, result.distance)
		}
	}
}
