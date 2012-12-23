package main

import (
    "code.google.com/p/go-html-transform/h5"
    "code.google.com/p/go-html-transform/html/transform"
    "io/ioutil"
    "net/http"
    "regexp"
    "runtime/debug"
    "strings"
    "testing"
    "time"
)

type T struct {
    *testing.T
}

func (t T) failIf(cond bool, msg string, params ...interface{}) {
    if cond {
        println("============================================")
        println("STACK:")
        println("======")
        debug.PrintStack()
        println("--------")
        println("FAILURE:")
        t.T.Fatalf(msg, params...)
    }
}

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
        checkAuthorSection(T{t}, node)
    }
}

func TestEntriesHaveTagsInList(t *testing.T) {
    nodes := query(t, "", "#tags")
    for _, node := range nodes {
        assertElem(t, node, "div")
        if len(node.Children) == 0 {
            t.Fatalf("No tags specified in tags div!")
        }
        checkTagsSection(T{t}, node)
    }
}

func checkTagsSection(t T, node *h5.Node) {
    doc, err := transform.NewDoc(node.String())
    t.failIf(err != nil, "Error parsing tags section!")
    q := transform.NewSelectorQuery("a")
    n2 := q.Apply(doc)
    t.failIf(len(n2) == 0, "Tags node not found in section: %q", node.String())
}

func checkAuthorSection(t T, node *h5.Node) {
    date := node.Children[0].Data()
    dateRe, _ := regexp.Compile("[0-9]{4}-[0-9]{2}-[0-9]{2}")
    m := dateRe.FindString(date)
    t.failIf(m == "", "No date found in author section!")
    doc, err := transform.NewDoc(node.String())
    t.failIf(err != nil, "Error parsing author section!")
    q := transform.NewSelectorQuery("b")
    n2 := q.Apply(doc)
    t.failIf(len(n2) != 1, "Author node not found in section: %q", node.String())
    t.failIf(n2[0].Children == nil, "Author node not found in section: %q", node.String())
}

func TestEveryEntryHasAuthor(t *testing.T) {
    posts := loadData("testdata")
    for _, e := range posts {
        node := query1(t, e.Url, "#author")
        assertElem(t, node, "div")
        if len(node.Children) == 0 {
            t.Fatalf("No author specified in author div!")
        }
        checkAuthorSection(T{t}, node)
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
