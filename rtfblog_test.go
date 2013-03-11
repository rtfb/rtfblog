package main

import (
    "io/ioutil"
    "net/http"
    "regexp"
    "runtime/debug"
    "strings"
    "testing"
    "time"

    "code.google.com/p/go-html-transform/h5"
    "code.google.com/p/go-html-transform/html/transform"
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
        {"", "skeleton"},
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
    dataset = "testdata/foo.db"
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
            t.Fatalf("Empty tags div found!")
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
    posts := loadData("testdata/foo.db")
    for _, e := range posts {
        node := query1(t, e.Url, "#author")
        assertElem(t, node, "div")
        if len(node.Children) == 0 {
            t.Fatalf("No author specified in author div!")
        }
        checkAuthorSection(T{t}, node)
    }
}

func TestCommentsFormattingInPostPage(t *testing.T) {
    posts := loadData("testdata/foo.db")
    for _, p := range posts {
        nodes := query0(t, p.Url, "#comments")
        if len(nodes) != 1 {
            t.Fatal("There should be only one comments section!")
        }
        for _, node := range nodes {
            assertElem(t, node, "div")
            if emptyChildren(node) {
                t.Fatalf("Empty comments div found!")
            }
            checkCommentsSection(T{t}, node)
        }
    }
}

func checkCommentsSection(t T, node *h5.Node) {
    noComments := transform.NewSelectorQuery("p").Apply(node)
    comments := transform.NewSelectorQuery("b").Apply(node)
    t.failIf(len(noComments) == 0 && len(comments) == 0,
        "Comments node not found in section: %q", node.String())
    if len(comments) > 0 {
        headers := transform.NewSelectorQuery("#comment-container").Apply(node)
        t.failIf(len(headers) == 0,
            "Comment header not found in section: %q", node.String())
        bodies := transform.NewSelectorQuery(".body-container").Apply(node)
        t.failIf(len(bodies) == 0,
            "Comment body not found in section: %q", node.String())
    }
}

func emptyChildren(node *h5.Node) bool {
    if len(node.Children) == 0 {
        return true
    }
    sum := ""
    for _, ch := range node.Children {
        sum += ch.Data()
    }
    return strings.TrimSpace(sum) == ""
}

func TestTagFormattingInPostPage(t *testing.T) {
    posts := loadData("testdata/foo.db")
    for _, e := range posts {
        nodes := query0(t, e.Url, "#tags")
        if len(nodes) > 0 {
            for _, node := range nodes {
                assertElem(t, node, "div")
                if len(node.Children) == 0 {
                    t.Fatalf("Empty tags div found!")
                }
                checkTagsSection(T{t}, node)
            }
        }
    }
}

func TestPostPageHasCommentEditor(t *testing.T) {
    posts := loadData("testdata/foo.db")
    for _, p := range posts {
        node := query1(t, p.Url, "#comment")
        assertElem(t, node, "form")
    }
}

func TestLoginPage(t *testing.T) {
    node := query1(t, "login", "#login_form")
    assertElem(t, node, "form")
}

func TestAllLoadedPostsAppearOnMainPage(t *testing.T) {
    testLoader = func() []*Entry {
        return []*Entry{{"", "LD", "", "B", "labadena", "RB", []*Tag{{"u", "n"}}, nil},
            {},
            {},
            {},
            {},
            {},
        }
    }
    nodes := query0(t, "", "#post")
    T{t}.failIf(len(nodes) != 6, "Not all posts have been rendered!")
}

func query(t *testing.T, url string, query string) []*h5.Node {
    nodes := query0(t, url, query)
    if len(nodes) == 0 {
        t.Fatalf("No nodes found: %q", query)
    }
    return nodes
}

func query0(t *testing.T, url string, query string) []*h5.Node {
    html := curl(url)
    doc, err := transform.NewDoc(html)
    if err != nil {
        t.Fatalf("Error parsing document! URL=%q, Err=%s", url, err.Error())
    }
    q := transform.NewSelectorQuery(query)
    return q.Apply(doc)
}

func query1(t *testing.T, url string, q string) *h5.Node {
    nodes := query(t, url, q)
    if len(nodes) > 1 {
        t.Fatalf("Too many matches (%d) for node: %q", len(nodes), q)
    }
    return nodes[0]
}

func assertElem(t *testing.T, node *h5.Node, elem string) {
    if !strings.HasPrefix(node.Data(), elem) {
        T{t}.failIf(true, "<%s> expected, but <%s> found!", elem, node.Data())
    }
}
