package main

import (
    "code.google.com/p/go-html-transform/h5"
    "code.google.com/p/go-html-transform/html/transform"
    "io/ioutil"
    "net/http"
    "strings"
    "testing"
    "time"
)

func curl(url string) string {
    if r, err := http.Get("http://localhost:8080/" + url); err == nil {
        b, err := ioutil.ReadAll(r.Body)
        r.Body.Close()
        if err == nil {
            return string(b)
        }
    }
    return ""
}

func mustContain(t *testing.T, page string, what string) {
    if !strings.Contains(page, what) {
        t.Errorf("Test page did not contain %q", what)
    }
}

func TestStartServer(t *testing.T) {
    go runServer()
    time.Sleep(50 * time.Millisecond)
}

func TestMainPage(t *testing.T) {
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
    for _, test := range simpleTests {
        mustContain(t, curl(test.url), test.out)
    }
}

func TestBasicStructure(t *testing.T) {
    var blocks = []string{
        "#header", "#subheader", "#content", "#footer", "#sidebar",
    }
    for _, block := range blocks {
        node := query1(t, "", block)
        if node.Data() != "div" {
            t.Errorf("<div> expected, but <%q> found!", node.Data())
        }
    }
}

func TestEmptyDatasetGeneratesFriendlyError(t *testing.T) {
    posts = nil
    html := curl("")
    mustContain(t, html, "No entries")
}

func TestNonEmptyDatasetHasEntries(t *testing.T) {
    posts = loadData("testdata")
    what := "No entries"
    if strings.Contains(curl(""), what) {
        t.Errorf("Test page should not contain %q", what)
    }
}

func TestEntryListHasAuthor(t *testing.T) {
    nodes := query(t, "", "#author")
    for _, node := range nodes {
        if node.Data() != "div" {
            t.Fatalf("<div> expected, but <%q> found!", node.Data())
        }
        if len(node.Children) == 0 {
            t.Fatalf("No author specified in author div!")
        }
        if node.Children[0].Data() != "rtfb" {
            t.Fatalf("'rtfb' expected, but '%q' found!", node.Children[0].Data())
        }
    }
}

func TestEveryEntryHasAuthor(t *testing.T) {
    for _, e := range posts {
        node := query1(t, e.Url, "#author")
        if node.Data() != "div" {
            t.Fatalf("<div> expected, but <%q> found!", node.Data())
        }
        if len(node.Children) == 0 {
            t.Fatalf("No author specified in author div!")
        }
    }
}

func query(t *testing.T, url string, query string) []*h5.Node {
    html := curl(url)
    doc, err := transform.NewDoc(html)
    if err != nil {
        t.Fatal("Error parsing document!")
    }
    q := transform.NewSelectorQuery(query)
    node := q.Apply(doc)
    if len(node) == 0 {
        t.Fatalf("Node not found: %q", query)
    }
    return node
}

func query1(t *testing.T, url string, q string) *h5.Node {
    nodes := query(t, url, q)
    if len(nodes) > 1 {
        t.Fatalf("Too many matches (%d) for node: %q", len(nodes), q)
    }
    return nodes[0]
}
