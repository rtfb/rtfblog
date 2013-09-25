package main

import (
    "./util"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "regexp"
    "runtime/debug"
    "strings"
    "testing"

    "code.google.com/p/go-html-transform/css/selector"
    "code.google.com/p/go-html-transform/h5"
    "code.google.com/p/go.net/html"
)

type Jar struct {
    cookies []*http.Cookie
}

type T struct {
    *testing.T
}

var (
    jar         = new(Jar)
    tclient     = &http.Client{nil, nil, jar}
    test_comm   = []*Comment{{"N", "@", "@h", "w", "IP", "Body", "Raw", "time", "testid"}}
    test_posts  = make([]*Entry, 0)
    test_author = new(Author)
)

type TestData struct{}

func (db *TestData) hiddenPosts(flag bool) {
}

func (db *TestData) post(url string) *Entry {
    for _, e := range test_posts {
        if e.Url == url {
            return e
        }
    }
    return nil
}

func (db *TestData) postId(url string) (id int64, err error) {
    id = 0
    return
}

func (db *TestData) posts(limit, offset int) []*Entry {
    if offset < 0 {
        offset = 0
    }
    if limit > 0 && limit < len(test_posts) {
        return test_posts[offset:limit]
    }
    return test_posts
}

func (db *TestData) numPosts() int {
    return len(test_posts)
}

func (dd *TestData) titles(limit int) (links []*EntryLink) {
    for _, p := range test_posts {
        entryLink := &EntryLink{p.Title, p.Url, false}
        links = append(links, entryLink)
    }
    return
}

func (dd *TestData) allComments() []*CommentWithPostTitle {
    comments := make([]*CommentWithPostTitle, 0)
    for _, c := range test_comm {
        comment := new(CommentWithPostTitle)
        comment.Comment = *c
        comment.Url = test_posts[0].Url
        comment.Title = test_posts[0].Title
        comments = append(comments, comment)
    }
    return comments
}

func (dd *TestData) author(username string) (*Author, error) {
    return test_author, nil
}

func (dd *TestData) deleteComment(id string) bool {
    return false
}

func (dd *TestData) updateComment(id, text string) bool {
    return false
}

func (dd *TestData) begin() bool {
    return true
}

func (dd *TestData) commit() {
}

func (dd *TestData) rollback() {
}

func (dd *TestData) selOrInsCommenter(name, email, website, ip string) (id int64, err error) {
    return
}

func (dd *TestData) insertComment(commenterId, postId int64, body string) (id int64, err error) {
    return
}

func (dd *TestData) insertPost(author int64, e *Entry) (id int64, err error) {
    return
}

func (dd *TestData) updatePost(id int64, e *Entry) bool {
    return true
}

func (dd *TestData) updateTags(tags []*Tag, postId int64) {
}

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

func logout() {
    curl("logout")
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

func curlParam(url string, method func(string) (*http.Response, error)) string {
    if r, err := method(url); err == nil {
        b, err := ioutil.ReadAll(r.Body)
        r.Body.Close()
        if err == nil {
            return string(b)
        }
    }
    return ""
}

func curl(url string) string {
    return curlParam(url, tclientGet)
}

func curlPost(url string) string {
    return curlParam(url, tclientPostForm)
}

func localhostUrl(url string) string {
    return "http://localhost:8080/" + url
}

func tclientGet(rqUrl string) (*http.Response, error) {
    return tclient.Get(localhostUrl(rqUrl))
}

func tclientPostForm(rqUrl string) (*http.Response, error) {
    return tclient.PostForm(localhostUrl(rqUrl), url.Values{})
}

func mustContain(t *testing.T, page string, what string) {
    if !strings.Contains(page, what) {
        t.Errorf("Test page did not contain %q", what)
    }
}

func init() {
    conf = loadConfig("server.conf")
    logger = util.MkLogger("tests.log")
    forgeTestUser("testuser", "testpasswd")
    auth := "Author"
    date := "2013-03-19"
    for i := 1; i <= 11; i++ {
        e := &Entry{
            EntryLink: EntryLink{
                Title: fmt.Sprintf("Hi%d", i),
                Url:   fmt.Sprintf("hello%d", i),
            },
            Author:   auth,
            Date:     date,
            Body:     fmt.Sprintf("Body%d", i),
            RawBody:  fmt.Sprintf("RawBody%d", i),
            Tags:     []*Tag{{fmt.Sprintf("u%d", i), fmt.Sprintf("n%d", i)}},
            Comments: test_comm,
        }
        test_posts = append(test_posts, e)
    }
    go runServer(&TestData{})
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
    tmpPosts := test_posts
    test_posts = nil
    html := curl("")
    mustContain(t, html, "No entries")
    test_posts = tmpPosts
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
        if len(h5.Children(node)) == 0 {
            t.Fatalf("No author specified in author div!")
        }
        checkAuthorSection(T{t}, node)
    }
}

func TestEntriesHaveTagsInList(t *testing.T) {
    nodes := query(t, "", "#tags")
    for _, node := range nodes {
        assertElem(t, node, "div")
        if len(h5.Children(node)) == 0 {
            t.Fatalf("Empty tags div found!")
        }
        checkTagsSection(T{t}, node)
    }
}

func cssSelect(t T, node *html.Node, query string) []*html.Node {
    chain, err := selector.Selector(query)
    if err != nil {
        t.Fatalf("WTF? Err=%s", query, err.Error())
    }
    return chain.Find(node)
}

func checkTagsSection(t T, node *html.Node) {
    if strings.Contains(h5.NewTree(node).String(), "&nbsp;") {
        return
    }
    n2 := cssSelect(t, node, "a")
    t.failIf(len(n2) == 0, "Tags node not found in section: %q", h5.NewTree(node).String())
}

func checkAuthorSection(t T, node *html.Node) {
    date := node.FirstChild.Data
    dateRe, _ := regexp.Compile("[0-9]{4}-[0-9]{2}-[0-9]{2}")
    m := dateRe.FindString(date)
    t.failIf(m == "", "No date found in author section!")
    n2 := cssSelect(t, node, "strong")
    t.failIf(len(n2) != 1, "Author node not found in section: %q", h5.NewTree(node).String())
    t.failIf(h5.Children(n2[0]) == nil, "Author node not found in section: %q", h5.NewTree(node).String())
}

func TestEveryEntryHasAuthor(t *testing.T) {
    for _, e := range test_posts {
        node := query1(t, e.Url, "#author")
        assertElem(t, node, "div")
        if len(h5.Children(node)) == 0 {
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

func checkCommentsSection(t T, node *html.Node) {
    noComments := cssSelect(t, node, "p")
    comments := cssSelect(t, node, "strong")
    t.failIf(len(noComments) == 0 && len(comments) == 0,
        "Comments node not found in section: %q", h5.NewTree(node).String())
    if len(comments) > 0 {
        headers := cssSelect(t, node, "#comment-container")
        t.failIf(len(headers) == 0,
            "Comment header not found in section: %q", h5.NewTree(node).String())
        bodies := cssSelect(t, node, "#bubble-container")
        t.failIf(len(bodies) == 0,
            "Comment body not found in section: %q", h5.NewTree(node).String())
    }
}

func emptyChildren(node *html.Node) bool {
    if len(h5.Children(node)) == 0 {
        return true
    }
    sum := ""
    for _, ch := range h5.Children(node) {
        sum += ch.Data
    }
    return strings.TrimSpace(sum) == ""
}

func TestTagFormattingInPostPage(t *testing.T) {
    for _, e := range test_posts {
        nodes := query0(t, e.Url, "#tags")
        if len(nodes) > 0 {
            for _, node := range nodes {
                assertElem(t, node, "div")
                if len(h5.Children(node)) == 0 {
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

func TestNonAdminCantAccessAdminPages(t *testing.T) {
    logout()
    urls := []string{
        "all_comments",
        "admin",
        "edit_post",
        "load_comments",
        "delete_comment",
    }
    for _, u := range urls {
        html := curl(u)
        mustContain(t, html, "Verboten")
    }
    postUrls := []string{
        "moderate_comment",
        "submit_post",
        "upload_images",
    }
    for _, u := range postUrls {
        html := curlPost(u)
        mustContain(t, html, "Verboten")
    }
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

func TestAdminPageHasAllCommentsButton(t *testing.T) {
    login()
    node := query1(t, "/admin", "#display-all-comments")
    assertElem(t, node, "input")
}

func TestAllCommentsPageHasAllComments(t *testing.T) {
    nodes := query(t, "/all_comments", "#comment")
    if len(nodes) != len(test_comm) {
        t.Fatalf("Not all comments in /all_comments!")
    }
}

func query(t *testing.T, url, query string) []*html.Node {
    nodes := query0(t, url, query)
    if len(nodes) == 0 {
        t.Fatalf("No nodes found: %q", query)
    }
    return nodes
}

func query0(t *testing.T, url, query string) []*html.Node {
    html := curl(url)
    doctree, err := h5.NewFromString(html)
    if err != nil {
        t.Fatalf("Error in NewFromString! doc=%q, Err=%s", html, err.Error())
    }
    return cssSelect(T{t}, doctree.Top(), query)
}

func query1(t *testing.T, url, q string) *html.Node {
    nodes := query(t, url, q)
    if len(nodes) > 1 {
        t.Fatalf("Too many matches (%d) for node: %q", len(nodes), q)
    }
    return nodes[0]
}

func assertElem(t *testing.T, node *html.Node, elem string) {
    if !strings.HasPrefix(node.Data, elem) {
        T{t}.failIf(true, "<%s> expected, but <%s> found!", elem, node.Data)
    }
}

func forgeTestUser(uname, passwd string) {
    salt, passwdHash, err := util.Encrypt(passwd)
    if err != nil {
        panic(fmt.Sprintf("Error in util.Encrypt(): %s\n", err))
    }
    test_author.Salt = salt
    test_author.Passwd = passwdHash
    test_author.UserName = uname
}
