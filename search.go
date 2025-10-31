package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
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

// DomainEmbedding represents a domain with its embedding vector
type DomainEmbedding struct {
	Domain    string
	Country   string
	Embedding []float32 // 768-dimensional vector as float32
}

// cosineSimilarity calculates cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 is perfect match, -1 is opposite
// a is []float64 (from Ollama), b is []float32 (from CSV)
func cosineSimilarity(a []float64, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		valB := float64(b[i])
		dotProduct += a[i] * valB
		normA += a[i] * a[i]
		normB += valB * valB
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// cosineDistance converts cosine similarity to cosine distance
// Cosine distance = 1 - cosine_similarity (ranges from 0 to 2)
func cosineDistance(similarity float64) float64 {
	return 1.0 - similarity
}

// parseEmbeddingString parses a numpy-style array string into a float32 slice
// Input format: "[-3.14846630e-02  4.35670600e-02 ...]" (768 values)
func parseEmbeddingString(embedStr string) ([]float32, error) {
	// Remove brackets and trim whitespace
	embedStr = strings.TrimSpace(embedStr)
	embedStr = strings.TrimPrefix(embedStr, "[")
	embedStr = strings.TrimSuffix(embedStr, "]")
	embedStr = strings.TrimSpace(embedStr)

	// Split by whitespace (handles both spaces and newlines)
	fields := strings.Fields(embedStr)

	if len(fields) != 768 {
		return nil, fmt.Errorf("expected 768 embedding values, got %d", len(fields))
	}

	embedding := make([]float32, 768)
	for i, field := range fields {
		val, err := strconv.ParseFloat(strings.TrimSpace(field), 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse embedding value at index %d: %w", i, err)
		}
		embedding[i] = float32(val)
	}

	return embedding, nil
}

// loadEmbeddingsFromCSV loads all domain embeddings from the CSV file
// CSV format: index,domain,country,embed (where embed is a numpy-style array string)
func loadEmbeddingsFromCSV(csvPath string) ([]DomainEmbedding, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Skip header row if present
	startIdx := 0
	if len(records) > 0 && (records[0][0] == "" || records[0][1] == "domain") {
		startIdx = 1
	}

	var embeddings []DomainEmbedding
	for i := startIdx; i < len(records); i++ {
		record := records[i]
		if len(record) < 4 {
			continue // Skip invalid rows
		}

		// Column format: index,domain,country,embed
		domain := strings.TrimSpace(record[1])
		country := strings.TrimSpace(record[2])
		embedStr := record[3]

		embedding, err := parseEmbeddingString(embedStr)
		if err != nil {
			// Log error but continue with other rows
			fmt.Fprintf(os.Stderr, "Warning: failed to parse embedding for %s: %v\n", domain, err)
			continue
		}

		embeddings = append(embeddings, DomainEmbedding{
			Domain:    domain,
			Country:   country,
			Embedding: embedding,
		})
	}

	return embeddings, nil
}

// getMatchingDomains searches for matching domains using keywords and returns results
// This version uses in-memory cosine similarity calculation instead of SQLite vector search
// threshold: similarity score cutoff (0.0-1.0), results must have similarity >= threshold to be returned
// limit: maximum number of results to return
func getMatchingDomains(keywords string, country string, threshold float64, limit int, dbPath string, ollamaURL string, modelName string) (Response, error) {
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

	// Get embedding from Ollama
	queryEmbedding, err := encode(keywords, ollamaURL, modelName)
	if err != nil {
		resp.ERR = 1
		resp.MS = time.Since(startTime).Milliseconds()
		return resp, fmt.Errorf("failed to encode keywords: %w", err)
	}

	// Load all embeddings from CSV file (dbPath is now treated as CSV path)
	domainEmbeddings, err := loadEmbeddingsFromCSV(dbPath)
	if err != nil {
		resp.ERR = 2
		resp.MS = time.Since(startTime).Milliseconds()
		return resp, fmt.Errorf("failed to load embeddings: %w", err)
	}

	// Calculate similarity scores for all domains
	type Match struct {
		Domain     string
		Country    string
		Similarity float64
		Distance   float64
	}

	matches := make([]Match, 0, len(domainEmbeddings))
	for _, de := range domainEmbeddings {
		similarity := cosineSimilarity(queryEmbedding, de.Embedding)
		distance := cosineDistance(similarity)

		// Convert cosine similarity (-1 to 1) to 0-1 scale for threshold comparison
		// Similarity on 0-1 scale = (cosine_similarity + 1) / 2
		similarity01 := (similarity + 1.0) / 2.0

		matches = append(matches, Match{
			Domain:     de.Domain,
			Country:    de.Country,
			Similarity: similarity01,
			Distance:   distance,
		})
	}

	// Sort by distance (ascending - lower distance = better match)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Distance < matches[j].Distance
	})

	// Filter by threshold and country, then apply limit
	domains := []string{}
	for _, match := range matches {
		// Filter by threshold: similarity must be >= threshold
		if match.Similarity < threshold {
			continue
		}

		// Filter by country if specified (and country matches)
		if country == "" || match.Country == country {
			domains = append(domains, match.Domain)

			// Stop if we've reached the limit
			if limit > 0 && len(domains) >= limit {
				break
			}
		}
	}

	resp.DN = domains
	resp.MS = time.Since(startTime).Milliseconds()
	return resp, nil
}
