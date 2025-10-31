package main

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

// Response represents the JSON response format
type Response struct {
	KW  string   `json:"kw"`  // keywords
	CN  string   `json:"cn"`  // country
	DN  []string `json:"dn"`  // domains
	MS  int64    `json:"ms"`  // processing time in milliseconds
	ERR int      `json:"err"` // error code (0 on success)
}

// encode sends text to Ollama and returns the embedding vector
func encode(text string, ollamaURL string, modelName string) ([]float64, error) {
	// Construct the API endpoint
	url := strings.TrimSuffix(ollamaURL, "/") + "/api/embed"

	// Create the request payload
	payload := map[string]interface{}{
		"model": modelName,
		"input": text,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Create HTTP POST request
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Parse response - Ollama returns {"embeddings": [[...vector...]]}
	var result struct {
		Embeddings [][]float64 `json:"embeddings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Ollama returns embeddings as an array of arrays, where each inner array is one embedding
	// For a single input text, we expect one embedding
	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings in response")
	}

	return result.Embeddings[0], nil
}

// getMatchingDomains searches for matching domains using keywords and returns results
// threshold: similarity score cutoff (0.0-1.0), results must have similarity >= threshold to be returned
func getMatchingDomains(keywords string, country string, threshold float64, dbPath string, ollamaURL string, modelName string) (Response, error) {
	startTime := time.Now()

	resp := Response{
		KW:  keywords,
		CN:  country,
		DN:  []string{},
		MS:  0,
		ERR: 0,
	}

	// Default country to "us" if empty
	if country == "" {
		country = "us"
		resp.CN = "us"
	}

	// Enable sqlite-vec extension
	sqlite_vec.Auto()

	// Get embedding from Ollama
	embedding, err := encode(keywords, ollamaURL, modelName)
	if err != nil {
		resp.ERR = 1
		resp.MS = time.Since(startTime).Milliseconds()
		return resp, fmt.Errorf("failed to encode keywords: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		resp.ERR = 2
		resp.MS = time.Since(startTime).Milliseconds()
		return resp, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		resp.ERR = 3
		resp.MS = time.Since(startTime).Milliseconds()
		return resp, fmt.Errorf("failed to ping database: %w", err)
	}

	// Convert float64 embedding to float32 blob (vec0 stores embeddings as float32)
	// vec0 expects embeddings as binary blobs with float32 values in little-endian format
	embeddingBlob := make([]byte, len(embedding)*4) // 4 bytes per float32
	for i, val := range embedding {
		f32 := float32(val)
		bits := math.Float32bits(f32)
		binary.LittleEndian.PutUint32(embeddingBlob[i*4:(i+1)*4], bits)
	}

	// Query using the embedding blob as a parameter
	query := `
		SELECT d.domain, d.country, d.distance
		FROM domains AS d
		WHERE d.embedding MATCH ?
		AND k = 10
		ORDER BY d.distance
	`

	rows, err := db.Query(query, embeddingBlob)
	if err != nil {
		resp.ERR = 4
		resp.MS = time.Since(startTime).Milliseconds()
		return resp, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	domains := []string{}
	for rows.Next() {
		var domain string
		var resultCountry string
		var distance float64
		if err := rows.Scan(&domain, &resultCountry, &distance); err != nil {
			resp.ERR = 5
			resp.MS = time.Since(startTime).Milliseconds()
			return resp, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert cosine distance to similarity score
		// Cosine distance ranges from 0 (perfect match) to 2 (opposite)
		// Similarity = 1 - (distance / 2), where similarity ranges from 1.0 (perfect) to 0.0 (worst)
		similarity := 1.0 - (distance / 2.0)

		// Filter by threshold: similarity must be >= threshold
		if similarity < threshold {
			continue
		}

		// Filter by country if specified (and country matches)
		if country == "" || resultCountry == country {
			domains = append(domains, domain)
		}
	}

	if err := rows.Err(); err != nil {
		resp.ERR = 6
		resp.MS = time.Since(startTime).Milliseconds()
		return resp, fmt.Errorf("error iterating rows: %w", err)
	}

	resp.DN = domains
	resp.MS = time.Since(startTime).Milliseconds()
	return resp, nil
}
