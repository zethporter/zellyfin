package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SearchResult struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
}

type searchResponse struct {
	Results []SearchResult `json:"results"`
}

func Search(apiKey, title string) (*SearchResult, error) {
	params := url.Values{}
	params.Set("api_key", apiKey)
	params.Set("query", title)
	params.Set("language", "en-US")
	params.Set("page", "1")

	resp, err := http.Get(
		fmt.Sprintf("https://api.themoviedb.org/3/search/movie?%s", params.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("TMDB request failed: %w", err)
	}
	defer resp.Body.Close()

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode TMDB response: %w", err)
	}
	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no results found for: %s", title)
	}

	return &result.Results[0], nil
}

func ExtractYear(releaseDate string) string {
	if len(releaseDate) >= 4 {
		return releaseDate[:4]
	}
	return "Unknown"
}

func SanitizeFilename(name string) string {
	r := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-",
		"*", "", "?", "", "\"", "",
		"<", "", ">", "", "|", "",
	)
	return strings.TrimSpace(r.Replace(name))
}
