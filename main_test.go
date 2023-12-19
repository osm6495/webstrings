package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetContents(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond with a 200 OK for successful requests
		if r.URL.Path == "/success" || r.URL.Path == "/relative" {
			fmt.Fprint(w, "Successful response")
		}
		// Respond with a 404 Not Found for error requests
		if r.URL.Path == "/notfound" {
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()
	baseURL := "https://example.com"

	// Test case: Successful request
	url := mockServer.URL + "/success"
	ctx := context.TODO()
	result, err := getContents(ctx, url, baseURL)
	assert.Nil(t, err, "Unexpected error for successful request")
	assert.NotNil(t, result, "Expected non-nil result")
	assert.Equal(t, "Successful response", *result, "Unexpected response body")

	// Test case: Empty URL
	result, err = getContents(ctx, "", baseURL)
	assert.NotNil(t, err, "Expected error for empty URL")
	assert.Nil(t, result, "Expected nil result")

	// Test case: Relative URL
	url = "/relative"
	result, err = getContents(ctx, url, mockServer.URL)
	assert.Nil(t, err, "Unexpected error for relative URL")
	assert.NotNil(t, result, "Expected non-nil result")
	assert.Equal(t, "Successful response", *result, "Unexpected response body")

	// Test case: Error response (404 Not Found) - This will print a Warning, but pass
	url = mockServer.URL + "/notfound"
	result, err = getContents(ctx, url, baseURL)
	assert.Nil(t, err, "Unexpected error for error response")
	assert.Nil(t, result, "Expected nil result")
}

func TestGetScripts(t *testing.T) {
	htmlContent := `
		<html>
			<head>
				<script src="/script1.js"></script>
				<script src="/script2.js"></script>
			</head>
			<body>
				<script src="/script3.js"></script>
			</body>
		</html>
	`

	scripts, err := getScripts(&htmlContent)

	//Test case: Matching scripts
	assert.Nil(t, err, "Unexpected error")
	expectedScripts := []string{"/script1.js", "/script2.js", "/script3.js"}
	assert.ElementsMatch(t, expectedScripts, scripts, "Unexpected scripts")
}

func TestGetStrings(t *testing.T) {
	text := "This is a test response. It should return 'result1', \"result2\", and `result3`."
	empty := ""
	flags := map[string]bool{"secrets": false, "dom": false, "verify": false, "noisy": false, "urls": false}

	//Test case: Empty text
	results, err := getStrings(empty, flags)
	assert.Nil(t, err, "Unexpected error")
	assert.Emptyf(t, results, "Expected empty slice for empty text, got: len(results) = %d", len(results))

	//Test case: Matching strings
	results, err = getStrings(text, flags)
	assert.Nil(t, err, "Unexpected error")
	assert.Equalf(t, 3, len(results), "Expected 3 results for non-empty text, got: len(results) = %d", len(results))
	expectedStrings := []string{"result1", "result2", "result3"}
	assert.ElementsMatch(t, expectedStrings, results, "Unexpected strings")
}

func TestGetSecrets(t *testing.T) {
	text := `
		This is a test response. It should return https://example.com, as well as example.com if
		the noisy flag is enabled. It should also identify "ghp_123456789023456789012345678902345678"
		as a secret.`
	flags := map[string]bool{"secrets": true, "dom": false, "verify": false, "noisy": true, "urls": true}

	//Test Case: Noisy flag (Should return all possible URLs and secrets)
	results := getSecrets(text, flags)
	assert.Equalf(t, 2, len(results), "Expected 2 finding types from text (two URLs, one GH PAT), got: len(results) = %d", len(results))
	expectedResults := map[string][]string{
		"GitHub Personal Access Token (Classic)": {"ghp_123456789023456789012345678902345678"},
		"URL":                                    {"https://example.com", "example.com"},
	}
	assert.Equal(t, expectedResults, results, "Unexpected results")

	//Test Case: Noisy flag disabled (Should return exact URLS and non-noisy secrets)
	flags["noisy"] = false
	results = getSecrets(text, flags)
	assert.Equalf(t, 2, len(results), "Expected 2 finding types from text (one URL, one GH PAT), got: len(results) = %d", len(results))
	expectedResults = map[string][]string{
		"GitHub Personal Access Token (Classic)": {"ghp_123456789023456789012345678902345678"},
		"URL":                                    {"https://example.com"},
	}
	assert.Equal(t, expectedResults, results, "Unexpected results")
}

func TestSearch(t *testing.T) {
	ctx := context.TODO()

	// Test case: Empty URL
	emptyURL := ""
	_, err := search(ctx, emptyURL, make(map[string]bool), nil)
	assert.NotNil(t, err, "Expected error for empty URL")

	// Test case: Valid URL, no errors
	validURL := "https://example.com"
	flags := map[string]bool{"dom": false, "secrets": true, "verify": false, "noisy": false, "urls": false}
	urlQueue := &URLQueue{}
	_, err = search(ctx, validURL, flags, urlQueue)
	assert.Nil(t, err, "Unexpected error")
}
