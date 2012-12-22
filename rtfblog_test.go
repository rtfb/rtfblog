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

func TestStartServer(t *testing.T) {
    go main()
    time.Sleep(50 * time.Millisecond)
}

func TestMainPage(t *testing.T) {
    for _, test := range simpleTests {
        mustContain(t, curl(test.url), test.out)
    }
}

func TestBasicStructure(t *testing.T) {
    var blocks = []string{
        "#header", "#subheader", "#content", "#footer",
    }
    for _, block := range blocks {
        node := query1(t, "", block)
        if node[0].Data() != "div" {
            t.Errorf("<div> expected, but <%q> found!", node[0].Data())
        }
    }
}

func query(t *testing.T, url string, query string) []*h5.Node {
    html := curl("")
    doc, err := transform.NewDoc(html)
    if err != nil {
        t.Error("Error parsing document!")
        t.FailNow()
    }
    q := transform.NewSelectorQuery(query)
    node := q.Apply(doc)
    if len(node) == 0 {
        t.Errorf("Node not found: %q", query)
        t.FailNow()
    }
    return node
}

func query1(t *testing.T, url string, q string) []*h5.Node {
    node := query(t, url, q)
    if len(node) > 1 {
        t.Errorf("Too many matches (%d) for node: %q", len(node), q)
        t.FailNow()
    }
    return node
}
