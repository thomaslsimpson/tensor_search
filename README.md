# Tensor Search

A Go module for semantic domain search using text embeddings and cosine similarity. This tool searches a database of domain names by converting keyword queries into embeddings via Ollama and finding the most similar domains using in-memory cosine similarity calculations.

## Features

- **Semantic Search**: Uses text embeddings to find domains semantically similar to your keywords
- **Fast In-Memory Search**: Loads embeddings once and performs searches entirely in memory
- **No CGO Dependencies**: Pure Go implementation with zero external dependencies (no CGO required)
- **Configurable**: Supports custom thresholds, result limits, and country filtering

## Requirements

- Go 1.25.0 or later
- Ollama server running (default: `http://localhost:11434`)
- CSV file with domain embeddings (format: `index,domain,country,embed`)

## Installation

```bash
git clone <repository-url>
cd tensor_search
go build
```

## Usage

### Command-Line Tool

The `tensor_search` binary provides a command-line interface for searching domains.

#### Single Search

```bash
./tensor_search "new truck"
```

#### With Custom Options

```bash
./tensor_search \
  -db=./reference/rc_domains_embeds.csv \
  -ollama=http://localhost:11434 \
  -model=nomic-embed-text:latest \
  -country=us \
  -threshold=0.5 \
  -limit=3 \
  "buy new automobile"
```

#### Batch Processing from stdin

```bash
echo -e "new truck\nbest socks\nhiking gear" | ./tensor_search
```

#### Command-Line Flags

- `-db` - Path to CSV file with embeddings (default: `./reference/rc_domains_embeds.csv`)
- `-ollama` - Ollama server URL (default: `http://localhost:11434`)
- `-model` - Model name for embeddings (default: `nomic-embed-text:latest`)
- `-country` - Country code for filtering results (default: `us`)
- `-threshold` - Similarity threshold 0.0-1.0 (default: `0.5`)
- `-limit` - Maximum number of results to return (default: `3`)

### As a Go Module

```go
package main

import (
    "fmt"
    "tensor_search"
)

func main() {
    // Load embeddings once at startup
    domainEmbeddings, err := loadEmbeddingsFromCSV("./reference/rc_domains_embeds.csv")
    if err != nil {
        panic(err)
    }

    // Search for matching domains
    result, err := getMatchingDomains(
        "new truck",           // keywords
        "us",                  // country
        0.5,                   // threshold
        3,                     // limit
        domainEmbeddings,      // pre-loaded embeddings
        "http://localhost:11434", // ollama URL
        "nomic-embed-text:latest", // model name
    )
    
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found %d domains: %v\n", len(result.DN), result.DN)
}
```

## Response Format

The search function returns a JSON response with the following structure:

```json
{
  "kw": "new truck",              // Original keywords
  "cn": "us",                     // Country code
  "dn": [                         // List of matching domains
    "napaonline.com",
    "ford.com",
    "suncentauto.com"
  ],
  "ms": 60,                       // Processing time in milliseconds
  "err": 0                        // Error code (0 = success)
}
```

### Error Codes

- `0` - Success
- `1` - Failed to encode keywords (Ollama error)
- `2` - Failed to load embeddings (CSV file error)

## CSV File Format

The CSV file should have the following format:

```csv
,domain,country,embed
0,example.com,us,"[-0.031 0.044 -0.137 ...]"
1,another.com,us,"[-0.025 0.038 -0.142 ...]"
```

Where:
- Column 1: Index (can be ignored)
- Column 2: Domain name
- Column 3: Country code
- Column 4: Embedding vector as a numpy-style array string (768 float values)

The embedding string format should be: `"[value1 value2 value3 ...]"` with 768 space-separated floating-point values in scientific notation.

## Examples

### Example 1: Basic Search

```bash
$ ./tensor_search "new truck"
Loaded 9544 domain embeddings
{"kw":"new truck","cn":"us","dn":["napaonline.com","ford.com","suncentauto.com"],"ms":412,"err":0}
```

### Example 2: Search with Higher Threshold

```bash
$ ./tensor_search -threshold=0.6 "buy new automobile"
Loaded 9544 domain embeddings
{"kw":"buy new automobile","cn":"us","dn":["cargurus.com","ford.com"],"ms":445,"err":0}
```

### Example 3: More Results

```bash
$ ./tensor_search -limit=10 "automotive parts"
Loaded 9544 domain embeddings
{"kw":"automotive parts","cn":"us","dn":["partsgeek.com","napaonline.com",...],"ms":398,"err":0}
```

## Testing

Run the test suite:

```bash
go test -v
```

Run timing tests with custom Ollama URL:

```bash
go test -v -run TestTime -ollama-url=http://localhost:11434
```

## Performance

- **Initial Load**: ~300-400ms to load ~10,000 domain embeddings from CSV
- **Search Time**: ~5-10ms per search (after initial load)
- **Ollama Encoding**: ~15-50ms per query (depends on model)

The embeddings are loaded once at startup and reused for all subsequent searches, making batch operations very efficient.

## How It Works

1. **Embedding Generation**: Keywords are sent to Ollama's embedding API to generate a 768-dimensional vector
2. **Similarity Calculation**: Cosine similarity is calculated between the query embedding and all domain embeddings in memory
3. **Filtering & Sorting**: Results are filtered by:
   - Similarity threshold (must be >= threshold)
   - Country code (if specified)
   - Limited to top N results (by distance, ascending)

