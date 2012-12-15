package main

import (
    "net/http"
    "io/ioutil"
    "strings"
    "testing"
)

func curl(url string) string {
    if r, err := http.Get("http://localhost:8080/" + url); err == nil {
        b, err := ioutil.ReadAll(r.Body)
        r.Body.Close()
        if err == nil {
            return string(b)
        }
    }
    return""
}

func mustContain(t *testing.T, page string, what string) {
    if !strings.Contains(page, what) {
        t.Errorf("Test page did not contain %q", what)
    }
}

var simpleTests = []struct {
    url string
    out string
}{
    {"", "container"},
    {"", "header"},
    {"", "subheader"},
    {"", "content"},
    {"", "sidebar"},
    {"", "footer"},
    {"", "blueprint"},
    {"", "utf-8"},
    {"", "gopher.png"},
    {"", "vim_created.png"},
}

func TestMainPage(t *testing.T) {
    go main()
    for _, test := range simpleTests {
        mustContain(t, curl(test.url), test.out)
    }
}
