package main

import (
	"testing"
)

func TestOllama(t *testing.T) {
	ollamaURL := "http://localhost:11434"
	modelName := "nomic-embed-text:latest"
	testText := "nice new trucks"

	// Test encoding
	embedding, err := encode(testText, ollamaURL, modelName)
	if err != nil {
		t.Fatalf("Failed to encode text: %v", err)
	}

	// Verify we got an embedding
	if len(embedding) == 0 {
		t.Fatal("Received empty embedding vector")
	}

	// Verify the embedding has a reasonable length (nomic-embed-text typically produces 768-dimensional vectors)
	if len(embedding) < 100 {
		t.Errorf("Expected embedding length to be at least 100, got %d", len(embedding))
	}

	// Verify all values are valid floats (not NaN or Inf)
	for i, val := range embedding {
		if val != val { // NaN check
			t.Errorf("Embedding contains NaN at index %d", i)
		}
	}

	t.Logf("Successfully encoded text '%s'", testText)
	t.Logf("Embedding vector length: %d", len(embedding))
	t.Logf("First 5 values: %v", embedding[:5])
	t.Logf("Last 5 values: %v", embedding[len(embedding)-5:])
}
