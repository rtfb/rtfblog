package main

import (
    "code.google.com/p/go-html-transform/h5"
    "code.google.com/p/go-html-transform/html/transform"
    "io/ioutil"
    "net/http"
    "regexp"
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
        assertElem(t, node, "div")
    }
}

func TestEmptyDatasetGeneratesFriendlyError(t *testing.T) {
    dataset = ""
    html := curl("")
    mustContain(t, html, "No entries")
}

func TestNonEmptyDatasetHasEntries(t *testing.T) {
    dataset = "testdata"
    what := "No entries"
    if strings.Contains(curl(""), what) {
        t.Errorf("Test page should not contain %q", what)
    }
}

func TestEntryListHasAuthor(t *testing.T) {
    nodes := query(t, "", "#author")
    for _, node := range nodes {
        assertElem(t, node, "div")
        if len(node.Children) == 0 {
            t.Fatalf("No author specified in author div!")
        }
        checkAuthorSection(t, node.Children[0].Data())
    }
}

func checkAuthorSection(t *testing.T, text string) {
    re := "[0-9]{4}-[0-9]{2}-[0-9]{2}, by rtfb"
    m, err := regexp.MatchString(re, text)
    if err != nil {
        t.Fatalf("Failed to parse author section %q!", text)
    }
    if !m {
        t.Fatalf("Author section %q doesn't match %q!", text, re)
    }
}

func TestEveryEntryHasAuthor(t *testing.T) {
    posts := loadData("testdata")
    for _, e := range posts {
        node := query1(t, e.Url, "#author")
        assertElem(t, node, "div")
        if len(node.Children) == 0 {
            t.Fatalf("No author specified in author div!")
        }
        checkAuthorSection(t, node.Children[0].Data())
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

func assertElem(t *testing.T, node *h5.Node, elem string) {
    if node.Data() != elem {
        t.Errorf("<%s> expected, but <%s> found!", elem, node.Data())
    }
}
