package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "reflect"
    "regexp"
    "runtime"
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

type CallSpec struct {
    function interface{}
    params   string
}

var (
    jar         = new(Jar)
    tclient     = &http.Client{nil, nil, jar}
    test_comm   = []*Comment{{"N", "@", "@h", "w", "IP", "Body", "Raw", "time", "testid"}}
    test_posts  = make([]*Entry, 0)
    test_author = new(Author)
    test_data   TestData
)

type TestDataI interface {
    reset()
    calls() string
    pushCall(paramStr string)
    expect(t *testing.T, f interface{}, paramStr string)
}

type TestData struct {
    Data
    TestDataI
    includeHidden bool
    lastCalls     []string
}

func (td *TestData) reset() {
    td.lastCalls = nil
}

func (td *TestData) calls() string {
    return strings.Join(td.lastCalls, "\n")
}

func (td *TestData) pushCall(paramStr string) {
    pc, _, _, ok := runtime.Caller(1)
    if !ok {
        panic("runtime.Caller(1) != ok, dafuq?")
    }
    funcName := runtime.FuncForPC(pc).Name()
    sig := fmt.Sprintf("%s('%s')", funcName, paramStr)
    td.lastCalls = append(td.lastCalls, sig)
}

func getCallSig(call CallSpec) string {
    funcName := runtime.FuncForPC(reflect.ValueOf(call.function).Pointer()).Name()
    return fmt.Sprintf("%s('%s')", funcName, call.params)
}

func (td *TestData) expect(t *testing.T, f interface{}, paramStr string) {
    sig := getCallSig(CallSpec{f, paramStr})
    if td.calls() != sig {
        t.Fatalf("%s() exptected, but got %s", sig, test_data.calls())
    }
}

func (td *TestData) expectSeries(t *testing.T, series []CallSpec) {
    var seriesWithPackage []string
    for _, call := range series {
        seriesWithPackage = append(seriesWithPackage, getCallSig(call))
    }
    seriesWithPackageStr := strings.Join(seriesWithPackage, "\n")
    if td.calls() != seriesWithPackageStr {
        t.Fatalf("%s exptected, but got %s", seriesWithPackageStr, test_data.calls())
    }
}

func (td *TestData) hiddenPosts(flag bool) {
    td.includeHidden = flag
}

func (td *TestData) post(url string) *Entry {
    for _, e := range td.testPosts() {
        if e.Url == url {
            return e
        }
    }
    return nil
}

func (td *TestData) postId(url string) (id int64, err error) {
    td.pushCall(fmt.Sprintf("%s", url))
    id = 0
    return
}

func (td *TestData) testPosts() []*Entry {
    if td.includeHidden {
        return test_posts
    } else {
        posts := make([]*Entry, 0)
        for _, p := range test_posts {
            if p.Hidden {
                continue
            }
            posts = append(posts, p)
        }
        return posts
    }
}

func (td *TestData) posts(limit, offset int) []*Entry {
    if offset < 0 {
        offset = 0
    }
    tp := td.testPosts()
    if limit > 0 && limit < len(tp) {
        return tp[offset:(offset + limit)]
    }
    return tp
}

func (td *TestData) numPosts() int {
    return len(td.testPosts())
}

func (td *TestData) titles(limit int) (links []*EntryLink) {
    for _, p := range td.testPosts() {
        entryLink := &EntryLink{p.Title, p.Url, false}
        links = append(links, entryLink)
    }
    return
}

func (td *TestData) titlesByTag(tag string) (links []*EntryLink) {
    td.pushCall(tag)
    return
}

func (td *TestData) allComments() []*CommentWithPostTitle {
    td.pushCall("")
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

func (td *TestData) author(username string) (*Author, error) {
    return test_author, nil
}

func (td *TestData) deleteComment(id string) bool {
    td.pushCall(id)
    return false
}

func (td *TestData) deletePost(url string) bool {
    td.pushCall(url)
    return false
}

func (td *TestData) updateComment(id, text string) bool {
    return false
}

func (td *TestData) begin() bool {
    return true
}

func (td *TestData) commit() {
}

func (td *TestData) rollback() {
}

func (td *TestData) insertCommenter(name, email, website, ip string) (id int64, err error) {
    return
}

func (td *TestData) commenter(name, email, website, ip string) (id int64, err error) {
    if name == "N" && email == "@" && website == "w" {
        return 1, nil
    }
    return -1, sql.ErrNoRows
}

func (td *TestData) insertComment(commenterId, postId int64, body string) (id int64, err error) {
    return
}

func (td *TestData) insertPost(author int64, e *Entry) (id int64, err error) {
    return
}

func (td *TestData) updatePost(id int64, e *Entry) bool {
    td.pushCall("0")
    return true
}

func (td *TestData) updateTags(tags []*Tag, postId int64) {
    td.pushCall(fmt.Sprintf("%d: %+v", postId, *tags[0]))
}

func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
    jar.cookies = cookies
}

func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
    return jar.cookies
}

func loginWithCred(username, passwd string) string {
    resp, err := tclient.PostForm(localhostUrl("login"), url.Values{
        "uname":  {username},
        "passwd": {passwd},
    })
    if err != nil {
        println(err.Error())
        return ""
    }
    b, err := ioutil.ReadAll(resp.Body)
    resp.Body.Close()
    if err != nil {
        println(err.Error())
        return ""
    }
    return string(b)
}

func login() {
    loginWithCred("testuser", "testpasswd")
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

func (t T) assertEqual(expected, actual string) {
    if expected != actual {
        t.T.Fatalf("Expected %q, but got %q", expected, actual)
    }
}

func curlParam(url string, method func(string) (*http.Response, error)) string {
    if r, err := method(url); err == nil {
        b, err := ioutil.ReadAll(r.Body)
        r.Body.Close()
        if err == nil {
            return string(b)
        } else {
            println(err.Error())
        }
    } else {
        println(err.Error())
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

func mustNotContain(t *testing.T, page string, what string) {
    if strings.Contains(page, what) {
        t.Errorf("Test page incorrectly contained %q", what)
    }
}

func mkTestEntry(i int, hidden bool) *Entry {
    auth := "Author"
    date := "2013-03-19"
    return &Entry{
        EntryLink: EntryLink{
            Title:  fmt.Sprintf("Hi%d", i),
            Url:    fmt.Sprintf("hello%d", i),
            Hidden: hidden,
        },
        Author:   auth,
        Date:     date,
        Body:     fmt.Sprintf("Body%d", i),
        RawBody:  fmt.Sprintf("RawBody%d", i),
        Tags:     []*Tag{{fmt.Sprintf("u%d", i), fmt.Sprintf("n%d", i)}},
        Comments: test_comm,
    }
}

func init() {
    conf = obtainConfiguration("")
    logger = MkLogger("tests.log")
    forgeTestUser("testuser", "testpasswd")
    for i := 1; i <= 11; i++ {
        test_posts = append(test_posts, mkTestEntry(i, false))
    }
    for i := 1; i <= 2; i++ {
        test_posts = append(test_posts, mkTestEntry(i+1000, true))
    }
    test_data = TestData{}
    go runServer(&test_data)
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

func TestBadLogin(t *testing.T) {
    html := loginWithCred("wronguser", "wrongpasswd")
    mustContain(t, html, "Login failed")
}

func TestNonEmptyDatasetHasEntries(t *testing.T) {
    what := "No entries"
    if strings.Contains(curl(""), what) {
        t.Errorf("Test page should not contain %q", what)
    }
}

func TestEntryListHasAuthor(t *testing.T) {
    nodes := query(t, "", ".author")
    for _, node := range nodes {
        assertElem(t, node, "div")
        if len(h5.Children(node)) == 0 {
            t.Fatalf("No author specified in author div!")
        }
        checkAuthorSection(T{t}, node)
    }
}

func TestEntriesHaveTagsInList(t *testing.T) {
    nodes := query(t, "", ".tags")
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
        node := query1(t, e.Url, ".author")
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
        headers := cssSelect(t, node, ".comment-container")
        t.failIf(len(headers) == 0,
            "Comment header not found in section: %q", h5.NewTree(node).String())
        bodies := cssSelect(t, node, ".bubble-container")
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
        nodes := query0(t, e.Url, ".tags")
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
    nodes := query0(t, "", ".post-title")
    T{t}.failIf(len(nodes) != POSTS_PER_PAGE, "Not all posts have been rendered!")
}

func TestArchiveContainsAllEntries(t *testing.T) {
    if len(test_posts) <= NUM_RECENT_POSTS {
        t.Fatalf("This test only makes sense if len(test_posts) > NUM_RECENT_POSTS")
    }
    nodes := query0(t, "archive", ".post-title")
    T{t}.failIf(len(nodes) != len(test_posts), "Not all posts rendered in archive!")
}

func TestPostPager(t *testing.T) {
    mustContain(t, curl(""), "/page/2")
}

func TestInvalidPageDefaultsToPageOne(t *testing.T) {
    page1 := curl("/page/1")
    pageFoo := curl("/page/foo")
    T{t}.failIf(page1 != pageFoo, "Invalid page did not produce /page/1")
}

func TestNonAdminCantAccessAdminPages(t *testing.T) {
    logout()
    urls := []string{
        "all_comments",
        "admin",
        "edit_post",
        "load_comments",
        "delete_comment",
        "delete_post",
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

func TestLoadComments(t *testing.T) {
    login()
    json := curl("/load_comments?post=hello1")
    mustContain(t, json, `"Comments":[{"Name":"N","Email":"@"`)
}

func TestSubmitPost(t *testing.T) {
    defer test_data.reset()
    login()
    values := url.Values{
        "title":  {"T1tlE"},
        "url":    {"shiny-url"},
        "tags":   {"tagzorz"},
        "hidden": {"off"},
        "text":   {"contentzorz"},
    }
    if r, err := tclient.PostForm(localhostUrl("submit_post"), values); err == nil {
        r.Body.Close()
    } else {
        println(err.Error())
    }
    test_data.expectSeries(t, []CallSpec{{(*TestData).postId, "shiny-url"},
        {(*TestData).updatePost, "0"},
        {(*TestData).updateTags, "0: {TagUrl:tagzorz TagName:tagzorz}"}})
}

func TestExplodeTags(t *testing.T) {
    var tagSpecs = []struct {
        spec, expected string
    }{
        {"tag", "{TagUrl:tag TagName:tag}"},
        {"Tag>taag", "{TagUrl:taag TagName:Tag}"},
        {",tagg", "{TagUrl:tagg TagName:tagg}"},
    }
    for _, ts := range tagSpecs {
        result := fmt.Sprintf("%+v", *explodeTags(ts.spec)[0])
        T{t}.assertEqual(ts.expected, result)
    }
}

func TestMainPageHasEditPostButtonWhenLoggedIn(t *testing.T) {
    login()
    nodes := query(t, "", ".edit-post-button")
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
    defer test_data.reset()
    login()
    nodes := query(t, "/all_comments", "#comment")
    if len(nodes) != len(test_comm) {
        t.Fatalf("Not all comments in /all_comments!")
    }
    test_data.expect(t, (*TestData).allComments, "")
}

func TestHiddenPosts(t *testing.T) {
    var positiveTests = []struct {
        url, content string
    }{
        {"hello1001", "Body"},
        {"", "hello1001"},
        {"archive", "hello1001"},
    }
    var negativeTests = []struct {
        url, content string
    }{
        {"", "hello1001"},
        {"archive", "hello1001"},
    }
    login()
    for _, i := range positiveTests {
        html := curl(i.url)
        mustContain(t, html, i.content)
    }
    logout()
    for _, i := range negativeTests {
        html := curl(i.url)
        mustNotContain(t, html, i.content)
    }
}

func TestHiddenPostAccess(t *testing.T) {
    login()
    html := curl("hello1001")
    mustContain(t, html, "Body")
    logout()
    html = curl("hello1001")
    mustContain(t, html, "Page Not Found")
}

func TestEditPost(t *testing.T) {
    login()
    // test with non-hidden post
    html := curl("edit_post?post=hello3")
    mustContain(t, html, "Body3")
    mustContain(t, html, "Hi3")
    mustContain(t, html, "u3")
    mustContain(t, html, "Delete!")
    mustNotContain(t, html, "checked")
    // now test with hidden post
    html = curl("edit_post?post=hello1002")
    mustContain(t, html, "Body1002")
    mustContain(t, html, "Hi1002")
    mustContain(t, html, "u1002")
    mustContain(t, html, "Delete!")
    mustContain(t, html, "checked")
}

func TestTitleByTagGetsCalled(t *testing.T) {
    defer test_data.reset()
    tag := "taaag"
    html := curl("/tag/" + tag)
    test_data.expect(t, (*TestData).titlesByTag, tag)
    mustContain(t, html, "Posts tagged ")
    mustContain(t, html, tag)
}

func TestDeletePostCallsDbFunc(t *testing.T) {
    defer test_data.reset()
    curl("delete_post?id=hello1001")
    test_data.expect(t, (*TestData).deletePost, "hello1001")
}

func TestDeleteCommentCallsDbFunc(t *testing.T) {
    defer test_data.reset()
    curl("delete_comment?id=1&action=delete")
    test_data.expect(t, (*TestData).deleteComment, "1")
}

func TestShowCaptcha(t *testing.T) {
    url := "comment_submit?name=joe&captcha=&email=snailmail&text=cmmnt%20txt"
    respJson := curl(url)
    var resp map[string]interface{}
    err := json.Unmarshal([]byte(respJson), &resp)
    if err != nil {
        t.Fatalf("Failed to parse json %q\nwith error %q", respJson, err.Error())
    }
    T{t}.failIf(resp["status"] != "showcaptcha", "No captcha box")
}

func TestReturningCommenterSkipsCaptcha(t *testing.T) {
    url := "comment_submit?name=N&captcha=&email=@&website=w&text=cmmnt%20txt"
    respJson := curl(url)
    var resp map[string]interface{}
    err := json.Unmarshal([]byte(respJson), &resp)
    if err != nil {
        t.Fatalf("Failed to parse json %q\nwith error %q", respJson, err.Error())
    }
    T{t}.failIf(resp["status"] != "accepted", "Comment by returning commenter not accepted")
}

func TestRssFeed(t *testing.T) {
    xml := curl("feeds/rss.xml")
    mustContain(t, xml, "<title>rtfb&#39;s blog</title>")
    mustContain(t, xml, "<title>Hi3</title>")
    mustContain(t, xml, "<link>hello3</link>")
}

func TestPagination(t *testing.T) {
    nodes := query0(t, "page/2", ".post-title")
    T{t}.failIf(len(nodes) != POSTS_PER_PAGE, "Not all posts have been rendered!")
    if nodes[0].Attr[1].Val != "/hello6" {
        t.Fatalf("Wrong post!")
    }
    if nodes[4].Attr[1].Val != "/hello10" {
        t.Fatalf("Wrong post!")
    }
}

func TestNewPostShowsEmptyForm(t *testing.T) {
    titleInput := query1(t, "edit_post", "#post_title")
    assertElem(t, titleInput, "input")
    bodyTextArea := query1(t, "edit_post", "#wmd-input")
    assertElem(t, bodyTextArea, "textarea")
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
    passwdHash, err := Encrypt(passwd)
    if err != nil {
        panic(fmt.Sprintf("Error in Encrypt(): %s\n", err))
    }
    test_author.Passwd = passwdHash
    test_author.UserName = uname
}
