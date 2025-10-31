package main

import (
	"encoding/json"
	"testing"
)

func TestSearch(t *testing.T) {
	keywords := "new truck"
	country := "us"
	threshold := 0.5
	limit := 3
	dbPath := "./reference/rc_domains_embeds.csv"
	ollamaURL := "http://localhost:11434"
	modelName := "nomic-embed-text:latest"

	// Run the search
	result, err := getMatchingDomains(keywords, country, threshold, limit, dbPath, ollamaURL, modelName)
	if err != nil {
		t.Fatalf("getMatchingDomains failed: %v", err)
	}

	// Verify error code
	if result.ERR != 0 {
		t.Errorf("Expected error code 0, got %d", result.ERR)
	}

	// Verify keywords and country
	if result.KW != keywords {
		t.Errorf("Expected keywords '%s', got '%s'", keywords, result.KW)
	}
	if result.CN != country {
		t.Errorf("Expected country '%s', got '%s'", country, result.CN)
	}

	// Verify we got results
	if len(result.DN) == 0 {
		t.Fatal("Expected at least one result, got none")
	}

	// Verify limit is enforced
	if len(result.DN) > limit {
		t.Errorf("Expected at most %d results (limit), got %d", limit, len(result.DN))
	}

	// Expected results from README
	expectedDomains := []string{"napaonline.com", "ford.com", "suncentauto.com"}

	// Check that we got the expected top matches (at least the first few should match)
	t.Logf("Search results for '%s':", keywords)
	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("Full result:\n%s", string(jsonBytes))

	// Verify we got the expected domains (they should be in the top results)
	foundExpected := make(map[string]bool)
	for _, expected := range expectedDomains {
		foundExpected[expected] = false
		for _, domain := range result.DN {
			if domain == expected {
				foundExpected[expected] = true
				break
			}
		}
	}

	// Log which expected domains we found
	allFound := true
	for domain, found := range foundExpected {
		if found {
			t.Logf("✓ Found expected domain: %s", domain)
		} else {
			t.Logf("✗ Missing expected domain: %s", domain)
			allFound = false
		}
	}

	if !allFound {
		t.Errorf("Did not find all expected domains. Expected: %v, Got: %v", expectedDomains, result.DN)
	} else {
		t.Logf("Successfully found all expected domains in results!")
	}
}
