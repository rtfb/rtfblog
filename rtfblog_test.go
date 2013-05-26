package main

import (
    "./util"
    "io/ioutil"
    "net/http"
    "net/url"
    "regexp"
    "runtime/debug"
    "strings"
    "testing"
    "time"

    "code.google.com/p/go-html-transform/h5"
    "code.google.com/p/go-html-transform/html/transform"
)

type Jar struct {
    cookies []*http.Cookie
}

type T struct {
    *testing.T
}

var (
    jar        = new(Jar)
    tclient    = &http.Client{nil, nil, jar}
    test_comm  = []*Comment{{"N", "@", "@h", "w", "IP", "Body", "Raw", "time", "testid"}}
    test_posts = []*Entry{
        {"Author", "Hi1", "2013-03-19", "Body1", "RawBody1", "hello1", []*Tag{{"u1", "n1"}}, test_comm},
        {"Author", "Hi2", "2013-03-19", "Body2", "RawBody2", "hello2", []*Tag{{"u2", "n2"}}, test_comm},
        {"Author", "Hi3", "2013-03-19", "Body3", "RawBody3", "hello3", []*Tag{{"u3", "n3"}}, test_comm},
        {"Author", "Hi4", "2013-03-19", "Body4", "RawBody4", "hello4", []*Tag{{"u4", "n4"}}, test_comm},
        {"Author", "Hi5", "2013-03-19", "Body5", "RawBody5", "hello5", []*Tag{{"u5", "n5"}}, test_comm},
        {"Author", "Hi6", "2013-03-19", "Body6", "RawBody6", "hello6", []*Tag{{"u6", "n6"}}, test_comm},
        {"Author", "Hi7", "2013-03-19", "Body7", "RawBody7", "hello7", []*Tag{{"u7", "n7"}}, test_comm},
        {"Author", "Hi8", "2013-03-19", "Body8", "RawBody8", "hello8", []*Tag{{"u8", "n8"}}, test_comm},
        {"Author", "Hi9", "2013-03-19", "Body9", "RawBody9", "hello9", []*Tag{{"u9", "n9"}}, test_comm},
        {"Author", "Hi10", "2013-03-19", "Body10", "RawBody10", "hello10", []*Tag{{"u10", "n10"}}, test_comm},
        {"Author", "Hi11", "2013-03-19", "Body11", "RawBody11", "hello11", []*Tag{{"u11", "n11"}}, test_comm},
    }
)

func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
    jar.cookies = cookies
}

func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
    return jar.cookies
}

func login() {
    resp, err := tclient.PostForm("http://localhost:8080/login_submit", url.Values{
        "uname":  {"testuser"},
        "passwd": {"testpasswd"},
    })
    if err != nil {
        println(err.Error())
    }
    resp.Body.Close()
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
    if r, err := tclient.Get("http://localhost:8080/" + url); err == nil {
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
    conf = loadConfig("server.conf")
    db = openDb(conf.Get("database"))
    err := forgeTestUser("testuser", "testpasswd")
    if err != nil {
        t.Error("Failed to set up test account")
    }
    testLoader = func() []*Entry {
        return test_posts
    }
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
    dbtemp := db
    loaderTemp := testLoader
    testLoader = nil
    db = nil
    html := curl("")
    mustContain(t, html, "No entries")
    db = dbtemp
    testLoader = loaderTemp
}

func TestLogin(t *testing.T) {
    login()
    html := curl(test_posts[0].Url)
    mustContain(t, html, "Logout")
}

func TestNonEmptyDatasetHasEntries(t *testing.T) {
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
    if strings.Contains(node.String(), "&nbsp;") {
        return
    }
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
    q := transform.NewSelectorQuery("strong")
    n2 := q.Apply(doc)
    t.failIf(len(n2) != 1, "Author node not found in section: %q", node.String())
    t.failIf(n2[0].Children == nil, "Author node not found in section: %q", node.String())
}

func TestEveryEntryHasAuthor(t *testing.T) {
    for _, e := range test_posts {
        node := query1(t, e.Url, "#author")
        assertElem(t, node, "div")
        if len(node.Children) == 0 {
            t.Fatalf("No author specified in author div!")
        }
        checkAuthorSection(T{t}, node)
    }
}

func TestCommentsFormattingInPostPage(t *testing.T) {
    for _, p := range test_posts {
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
    comments := transform.NewSelectorQuery("strong").Apply(node)
    t.failIf(len(noComments) == 0 && len(comments) == 0,
        "Comments node not found in section: %q", node.String())
    if len(comments) > 0 {
        headers := transform.NewSelectorQuery("#comment-container").Apply(node)
        t.failIf(len(headers) == 0,
            "Comment header not found in section: %q", node.String())
        bodies := transform.NewSelectorQuery("#bubble-container").Apply(node)
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
    for _, e := range test_posts {
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
    for _, p := range test_posts {
        node := query1(t, p.Url, "#comment")
        assertElem(t, node, "form")
    }
}

func TestLoginPage(t *testing.T) {
    node := query1(t, "login", "#login_form")
    assertElem(t, node, "form")
}

func TestOnlyOnePageOfPostsAppearsOnMainPage(t *testing.T) {
    nodes := query0(t, "", "#post")
    T{t}.failIf(len(nodes) != POSTS_PER_PAGE, "Not all posts have been rendered!")
}

func TestArchiveContainsAllEntries(t *testing.T) {
    if len(test_posts) <= NUM_RECENT_POSTS {
        t.Fatalf("This test only makes sense if len(test_posts) > NUM_RECENT_POSTS")
    }
    nodes := query0(t, "archive", "#post")
    T{t}.failIf(len(nodes) != len(test_posts), "Not all posts rendered in archive!")
}

func TestPostPager(t *testing.T) {
    mustContain(t, curl(""), "/page/2")
}

func TestMainPageHasEditPostButtonWhenLoggedIn(t *testing.T) {
    login()
    nodes := query(t, "", "#edit-post-button")
    T{t}.failIf(len(nodes) != POSTS_PER_PAGE, "Not all posts have Edit button!")
}

func TestEveryCommentHasEditFormWhenLoggedId(t *testing.T) {
    login()
    node := query1(t, test_posts[0].Url, "#edit-comment-form")
    assertElem(t, node, "form")
}

func query(t *testing.T, url, query string) []*h5.Node {
    nodes := query0(t, url, query)
    if len(nodes) == 0 {
        t.Fatalf("No nodes found: %q", query)
    }
    return nodes
}

func query0(t *testing.T, url, query string) []*h5.Node {
    html := curl(url)
    doc, err := transform.NewDoc(html)
    if err != nil {
        t.Fatalf("Error parsing document! URL=%q, Err=%s", url, err.Error())
    }
    q := transform.NewSelectorQuery(query)
    return q.Apply(doc)
}

func query1(t *testing.T, url, q string) *h5.Node {
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

func forgeTestUser(uname, passwd string) error {
    salt, passwdHash := util.Encrypt(passwd)
    updateStmt, err := db.Prepare(`update author set disp_name=?, salt=?, passwd=?
                                   where id=?`)
    if err != nil {
        return err
    }
    defer updateStmt.Close()
    _, err = updateStmt.Exec(uname, salt, passwdHash, 1)
    if err != nil {
        return err
    }
    return nil
}
