package main

import "testing"

func TestGetContents(t *testing.T) {
	url := ""
	_, _, err := getContents(url, url)
	if err.Error() != "Attempted to get contents of empty URL" {
		t.Error("Expected error for empty URL, got: ", err)
	}
}

func TestGetScripts(t *testing.T) {
	url := ""
	_, err := getScripts(url, url)
	if err.Error() != "Attempted to get contents of empty URL" {
		t.Error("Expected error for empty URL, got: ", err)
	}
}

func TestGetStrings(t *testing.T) {
	text := "This is a test response. It should return 'result1', \"result2\", and `result3`."
	empty := ""
	flags := map[string]bool{"secrets": false, "dom": false, "verify": false, "noisy": false, "urls": false}

	//Should successfully return an empty slice when text is empty
	results, err := getStrings(empty, flags)
	if err != nil {
		t.Error("Expected no error for empty text, got: ", err)
	}
	if len(results) != 0 {
		t.Error("Expected empty slice for empty text, got: len(results) = ", len(results))
	}

	//Should successfully return a slice of 3 results
	results, err = getStrings(text, flags)
	if err != nil {
		t.Error("Expected no error for non-empty text, got: ", err)
	}
	if len(results) != 3 {
		t.Error("Expected 3 results for non-empty text, got: len(results) = ", len(results))
	}
	if results[0] != "result1" || results[1] != "result2" || results[2] != "result3" {
		t.Error("Expected results to be 'result1', 'result2', and 'result3'")
	}
}

func TestGetSecrets(t *testing.T) {
	text := `
		This is a test response. It should return https://example.com, as well as example.com if
		the noisy flag is enabled. It should also identify "ghp_123456789023456789012345678902345678"
		as a secret.`
	flags := map[string]bool{"secrets": true, "dom": false, "verify": false, "noisy": true, "urls": true}

	//Should return all possible URLs and secrets when using noisy flag
	results := getSecrets(text, flags)
	if len(results) != 2 {
		t.Error("Expected 2 finding types from text (2 URLs, one GH PAT), got: len(results) = ", len(results))
	} else if len(results) == 0 {
		t.Error("Expected 2 finding types from text (2 URLs, one GH PAT), got: len(results) = ", len(results))
		return
	}
	if results["URL"][0] != "https://example.com" || results["URL"][1] != "example.com" {
		t.Error("Expected https://example.com and example.com to be identified as URLs")
	}
	if results["GitHub Personal Access Token (Classic)"][0] != "ghp_123456789023456789012345678902345678" {
		t.Error("Expected GitHub PAT to be identified as a secret")
	}

	flags["noisy"] = false
	//Should successfully return a slice of 3 results
	results = getSecrets(text, flags)
	if len(results) != 2 {
		t.Error("Expected 3 results from text, got: len(results) = ", len(results))
	} else if len(results) == 0 {
		t.Error("Expected 2 finding types from text (2 URLs, one GH PAT), got: len(results) = ", len(results))
		return
	}
	if results["URL"][0] != "https://example.com" {
		t.Error("Expected https://example.com to be identified as URLs")
	}
	if results["GitHub Personal Access Token (Classic)"][0] != "ghp_123456789023456789012345678902345678" {
		t.Error("Expected GitHub PAT to be identified as a secret")
	}
}
