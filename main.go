package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define command line flags
	dbPath := flag.String("db", "./reference/rc_domains_embeds.csv", "Path to the CSV file with embeddings")
	ollamaURL := flag.String("ollama", "http://localhost:11434", "Ollama server URL")
	modelName := flag.String("model", "nomic-embed-text:latest", "Model name for embeddings")
	country := flag.String("country", "us", "Country code for filtering results")
	threshold := flag.Float64("threshold", 0.5, "Similarity threshold (0.0-1.0)")
	limit := flag.Int("limit", 3, "Maximum number of results to return")

	flag.Parse()

	// Load embeddings once at startup (reused for all searches)
	domainEmbeddings, err := loadEmbeddingsFromCSV(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading embeddings: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Loaded %d domain embeddings\n", len(domainEmbeddings))

	// Get keywords from command line (remaining positional arguments)
	keywords := ""
	if len(flag.Args()) > 0 {
		// Join all remaining args as keywords (in case keywords contain spaces)
		for i, arg := range flag.Args() {
			if i > 0 {
				keywords += " "
			}
			keywords += arg
		}
	}

	// If keywords provided on command line, run a single search
	if keywords != "" {
		result, err := getMatchingDomains(keywords, *country, *threshold, *limit, domainEmbeddings, *ollamaURL, *modelName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Output as JSON
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes))
		return
	}

	// Otherwise, read from stdin one line at a time
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue // Skip empty lines
		}

		result, err := getMatchingDomains(line, *country, *threshold, *limit, domainEmbeddings, *ollamaURL, *modelName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error searching '%s': %v\n", line, err)
			continue
		}

		// Output as JSON
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
			continue
		}
		fmt.Println(string(jsonBytes))
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
		os.Exit(1)
	}
}
