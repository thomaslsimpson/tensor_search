package main

import (
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	searchPhrases := []string{
		"new truck",
		"children's toys",
		"adult toys",
		"best socks",
		"designer shoes",
		"designer clothes",
		"thrift clothes",
		"hiking gear",
		"automotive parts",
		"shoe cleaning",
	}

	dbPath := "./reference/rc_domains_embeds.csv"
	ollamaURL := "http://localhost:11434"
	modelName := "nomic-embed-text:latest"
	country := "us"
	threshold := 0.5
	limit := 3

	// Load embeddings once at the start (reused for all searches)
	domainEmbeddings, err := loadEmbeddingsFromCSV(dbPath)
	if err != nil {
		t.Fatalf("Failed to load embeddings: %v", err)
	}
	t.Logf("Loaded %d domain embeddings\n", len(domainEmbeddings))

	var totalOllamaTime time.Duration
	var totalQueryTime time.Duration
	var totalResults int

	t.Logf("\n=== Running timing tests for %d search phrases ===\n", len(searchPhrases))

	for i, keywords := range searchPhrases {
		// Time Ollama encoding separately (first call, will be slow if model not loaded)
		startOllama := time.Now()
		_, err := encode(keywords, ollamaURL, modelName)
		if err != nil {
			t.Errorf("Failed to encode '%s': %v", keywords, err)
			continue
		}
		ollamaTime := time.Since(startOllama)

		// Time the full getMatchingDomains call (includes encoding + search)
		// getMatchingDomains will encode again, but we use our measured encoding time
		startTotal := time.Now()
		result, err := getMatchingDomains(keywords, country, threshold, limit, domainEmbeddings, ollamaURL, modelName)
		totalTime := time.Since(startTotal)

		if err != nil {
			t.Errorf("Failed to search '%s': %v", keywords, err)
			continue
		}

		// Estimate query time: total time minus encoding time
		// Note: getMatchingDomains encodes again, so this is an approximation
		// For more accuracy, we could refactor, but this gives reasonable estimates
		queryTime := totalTime - ollamaTime
		if queryTime < 0 {
			// If encoding took longer in getMatchingDomains, use a small default
			queryTime = 5 * time.Millisecond
		}

		totalOllamaTime += ollamaTime
		totalQueryTime += queryTime
		totalResults += len(result.DN)

		t.Logf("Test %d: '%s'", i+1, keywords)
		t.Logf("  Ollama encoding: %d ms", ollamaTime.Milliseconds())
		t.Logf("  CSV search time: %d ms", queryTime.Milliseconds())
		t.Logf("  Total time: %d ms", totalTime.Milliseconds())
		t.Logf("  Results: %d domains", len(result.DN))
		if len(result.DN) > 0 {
			t.Logf("  Top result: %s", result.DN[0])
		}
		t.Logf("")
	}

	// Calculate averages
	avgOllamaTime := totalOllamaTime / time.Duration(len(searchPhrases))
	avgQueryTime := totalQueryTime / time.Duration(len(searchPhrases))
	avgTotalTime := (totalOllamaTime + totalQueryTime) / time.Duration(len(searchPhrases))
	avgResults := float64(totalResults) / float64(len(searchPhrases))

	t.Logf("=== Summary ===")
	t.Logf("Total searches: %d", len(searchPhrases))
	t.Logf("Average Ollama encoding time: %d ms", avgOllamaTime.Milliseconds())
	t.Logf("Average CSV search time: %d ms", avgQueryTime.Milliseconds())
	t.Logf("Average total time: %d ms", avgTotalTime.Milliseconds())
	t.Logf("Average results per search: %.1f domains", avgResults)
	t.Logf("Total Ollama time: %d ms", totalOllamaTime.Milliseconds())
	t.Logf("Total query time: %d ms", totalQueryTime.Milliseconds())
	t.Logf("Total time: %d ms", (totalOllamaTime + totalQueryTime).Milliseconds())
}
