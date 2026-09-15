package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// userAgent identifies this tool to PokeAPI and Bulbapedia. MediaWiki policy
// asks for an identifiable User-Agent on API clients; project-identifying
// without embedding a personal contact address is enough for a low-volume
// personal-project scraper like this one.
const userAgent = "PokePediaScrapeTool/1.0 (+https://github.com/Rishi2804/pokepedia-api-v2)"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// httpGet issues a single polite GET. Retries and backoff for 429/503 live
// in bulba.go, next to the batching logic that needs them -- PokeAPI has
// never been observed to return either in this project's testing, so this
// bare helper is sufficient for pokeapi.go.
func httpGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
