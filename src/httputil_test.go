package main

import (
	"net/http"
	"testing"
)

func TestExtractReferer(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Error(err)
	}
	refURL := ExtractReferer(req)
	T{t}.failIf(refURL != "", "Empty string expected, but <%s> found!", refURL)
	req.Header.Add("Referer", "foo/bar/baz")
	refURL = ExtractReferer(req)
	T{t}.failIf(refURL != "baz", "<baz> expected, but <%s> found!", refURL)
}
