package main

import (
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "html/template"
    "io/ioutil"
    "net/http"
    "net/http/cookiejar"
    "net/http/httptest"
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
    "github.com/gorilla/sessions"
)

type T struct {
    *testing.T
}

type CallSpec struct {
    function interface{}
    params   string
}

var (
    jar, _  = cookiejar.New(nil)
    tclient = &http.Client{
        Jar: jar,
    }
    tserver    *httptest.Server
    testComm   = []*Comment{{Commenter{"N", "@", "@h", "http://w", "IP"}, "Body", "Raw", "time", "testid"}}
    testPosts  = make([]*Entry, 0)
    testAuthor = new(Author)
    testData   TestData
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
        t.Fatalf("%s() exptected, but got %s", sig, testData.calls())
    }
}

func (td *TestData) expectSeries(t *testing.T, series []CallSpec) {
    var seriesWithPackage []string
    for _, call := range series {
        seriesWithPackage = append(seriesWithPackage, getCallSig(call))
    }
    seriesWithPackageStr := strings.Join(seriesWithPackage, "\n")
    if td.calls() != seriesWithPackageStr {
        t.Fatalf("%s exptected, but got %s", seriesWithPackageStr, testData.calls())
    }
}

func (td *TestData) hiddenPosts(flag bool) {
    td.includeHidden = flag
}

func (td *TestData) post(url string) *Entry {
    for _, e := range td.testPosts() {
        if e.URL == url {
            return e
        }
    }
    return nil
}

func (td *TestData) postID(url string) (id int64, err error) {
    td.pushCall(fmt.Sprintf("%s", url))
    id = 0
    return
}

func (td *TestData) testPosts() []*Entry {
    if td.includeHidden {
        return testPosts
    }
    var posts []*Entry
    for _, p := range testPosts {
        if p.Hidden {
            continue
        }
        posts = append(posts, p)
    }
    return posts
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
        entryLink := &EntryLink{p.Title, p.URL, false}
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
    var comments []*CommentWithPostTitle
    for _, c := range testComm {
        comment := new(CommentWithPostTitle)
        comment.Comment = *c
        comment.URL = testPosts[0].URL
        comment.Title = testPosts[0].Title
        comments = append(comments, comment)
    }
    return comments
}

func (td *TestData) author(username string) (*Author, error) {
    if username == testAuthor.UserName {
        return testAuthor, nil
    }
    return nil, sql.ErrNoRows
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
    td.pushCall(fmt.Sprintf("%s - %s", id, text))
    return false
}

func (td *TestData) begin() bool {
    return true
}

func (td *TestData) commit() {
}

func (td *TestData) rollback() {
}

func (td *TestData) insertCommenter(c Commenter) (id int64, err error) {
    td.pushCall(c.Name)
    return
}

func (td *TestData) commenter(c Commenter) (id int64, err error) {
    tc := testComm[0]
    if c.Name == tc.Name && c.Email == tc.Email && c.Website == tc.Website {
        return 1, nil
    }
    return -1, sql.ErrNoRows
}

func (td *TestData) insertComment(commenterID, postID int64, body string) (id int64, err error) {
    return
}

func (td *TestData) insertPost(author int64, e *Entry) (id int64, err error) {
    return
}

func (td *TestData) updatePost(id int64, e *Entry) bool {
    td.pushCall("0")
    return true
}

func (td *TestData) updateTags(tags []*Tag, postID int64) {
    td.pushCall(fmt.Sprintf("%d: %+v", postID, *tags[0]))
}

func loginWithCred(username, passwd string) string {
    resp, err := tclient.PostForm(localhostURL("login"), url.Values{
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
        }
        println(err.Error())
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

func localhostURL(u string) string {
    if u == "" {
        return tserver.URL
    } else if u[0] == '/' {
        return tserver.URL + u
    } else {
        return tserver.URL + "/" + u
    }
}

func tclientGet(rqURL string) (*http.Response, error) {
    return tclient.Get(localhostURL(rqURL))
}

func tclientPostForm(rqURL string) (*http.Response, error) {
    return tclient.PostForm(localhostURL(rqURL), url.Values{})
}

func postForm(t *testing.T, path string, values *url.Values, testFunc func(html string)) {
    defer testData.reset()
    login()
    if r, err := tclient.PostForm(localhostURL(path), *values); err == nil {
        defer r.Body.Close()
        body, err := ioutil.ReadAll(r.Body)
        if err != nil {
            t.Error(err)
        }
        testFunc(string(body))
    } else {
        t.Error(err)
    }
}

func mustContain(t *testing.T, page string, what string) {
    if !strings.Contains(page, what) {
        t.Errorf("Test page did not contain %q\npage:\n%s", what, page)
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
            URL:    fmt.Sprintf("hello%d", i),
            Hidden: hidden,
        },
        Author:   auth,
        Date:     date,
        Body:     template.HTML(fmt.Sprintf("Body%d", i)),
        RawBody:  fmt.Sprintf("RawBody%d", i),
        Tags:     []*Tag{{fmt.Sprintf("u%d", i), fmt.Sprintf("n%d", i)}},
        Comments: testComm,
    }
}

func init() {
    conf = obtainConfiguration("")
    conf["staticdir"] = "../static"
    InitL10n("../l10n", "en-US")
    tmplDir = "../tmpl"
    logger = MkLogger("tests.log")
    store = sessions.NewCookieStore([]byte("aaabbbcccddd"))
    forgeTestUser("testuser", "testpasswd")
    for i := 1; i <= 11; i++ {
        testPosts = append(testPosts, mkTestEntry(i, false))
    }
    for i := 1; i <= 2; i++ {
        testPosts = append(testPosts, mkTestEntry(i+1000, true))
    }
    DetectLanguage = func(string) string {
        return "foo"
    }
    Decrypt = func(hash, passwd []byte) error {
        if string(passwd) == "testpasswd" {
            return nil
        }
        return errors.New("bad passwd")
    }
    testData = TestData{}
    initData(&testData)
    initRoutes()
    tserver = httptest.NewServer(Router)
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
        {"", "Ribs"},
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
    tmpPosts := testPosts
    testPosts = nil
    html := curl("")
    mustContain(t, html, "No entries")
    testPosts = tmpPosts
}

func TestLogin(t *testing.T) {
    login()
    html := curl(testPosts[0].URL)
    mustContain(t, html, "Logout")
}

func TestBadLogin(t *testing.T) {
    html := loginWithCred("wronguser", "wrongpasswd")
    mustContain(t, html, "Login failed")
    html = loginWithCred("testuser", "wrongpasswd")
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
        t.Fatalf("WTF? query=%q, Err=%s", query, err.Error())
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
    for _, e := range testPosts {
        mustContain(t, curl(e.URL), "captcha-id")
    }
}

func TestEveryEntryHasCaptchaSection(t *testing.T) {
    for _, e := range testPosts {
        node := query1(t, e.URL, ".author")
        assertElem(t, node, "div")
        if len(h5.Children(node)) == 0 {
            t.Fatalf("No author specified in author div!")
        }
        checkAuthorSection(T{t}, node)
    }
}

func TestCommentsFormattingInPostPage(t *testing.T) {
    for _, p := range testPosts {
        nodes := query0(t, p.URL, "#comments")
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
    for _, e := range testPosts {
        nodes := query0(t, e.URL, ".tags")
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
    for _, p := range testPosts {
        node := query1(t, p.URL, "#comment")
        assertElem(t, node, "form")
    }
}

func TestLoginPage(t *testing.T) {
    node := query1(t, "login", "#login_form")
    assertElem(t, node, "form")
}

func TestOnlyOnePageOfPostsAppearsOnMainPage(t *testing.T) {
    nodes := query0(t, "", ".post-title")
    T{t}.failIf(len(nodes) != PostsPerPage, "Not all posts have been rendered!")
}

func TestArchiveContainsAllEntries(t *testing.T) {
    if len(testPosts) <= NumRecentPosts {
        t.Fatalf("This test only makes sense if len(testPosts) > NUM_RECENT_POSTS")
    }
    nodes := query0(t, "archive", ".post-title")
    T{t}.failIf(len(nodes) != len(testPosts), "Not all posts rendered in archive!")
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

func TestModerateCommentCallsDbFunc(t *testing.T) {
    postForm(t, "moderate_comment", &url.Values{
        "action":            {"edit"},
        "id":                {"foo"},
        "edit-comment-text": {"bar"},
    }, func(html string) {
        testData.expect(t, (*TestData).updateComment, "foo - bar")
    })
}

func TestModerateCommentIgnoresWrongAction(t *testing.T) {
    postForm(t, "moderate_comment", &url.Values{
        "action":            {"wrong-action"},
        "id":                {"testid"},
        "redirect_to":       {"hello1"},
        "edit-comment-text": {"bar"},
    }, func(html string) {
        mustContain(t, html, "@h")
    })
}

func TestLoadComments(t *testing.T) {
    login()
    json := curl("/load_comments?post=hello1")
    mustContain(t, json, `"Comments":[{"Name":"N","Email":"@"`)
}

func TestSubmitPost(t *testing.T) {
    postForm(t, "submit_post", &url.Values{
        "title":  {"T1tlE"},
        "url":    {"shiny-url"},
        "tags":   {"tagzorz"},
        "hidden": {"off"},
        "text":   {"contentzorz"},
    }, func(html string) {
        testData.expectSeries(t, []CallSpec{{(*TestData).postID, "shiny-url"},
            {(*TestData).updatePost, "0"},
            {(*TestData).updateTags, "0: {TagURL:tagzorz TagName:tagzorz}"}})
    })
}

func TestExplodeTags(t *testing.T) {
    var tagSpecs = []struct {
        spec, expected string
    }{
        {"tag", "{TagURL:tag TagName:tag}"},
        {"Tag>taag", "{TagURL:taag TagName:Tag}"},
        {",tagg", "{TagURL:tagg TagName:tagg}"},
    }
    for _, ts := range tagSpecs {
        result := fmt.Sprintf("%+v", *explodeTags(ts.spec)[0])
        T{t}.assertEqual(ts.expected, result)
    }
}

func TestMainPageHasEditPostButtonWhenLoggedIn(t *testing.T) {
    login()
    nodes := query(t, "", ".edit-post-button")
    T{t}.failIf(len(nodes) != PostsPerPage, "Not all posts have Edit button!")
}

func TestEveryCommentHasEditFormWhenLoggedId(t *testing.T) {
    login()
    node := query1(t, testPosts[0].URL, "#edit-comment-form")
    assertElem(t, node, "form")
}

func TestAdminPageHasAllCommentsButton(t *testing.T) {
    login()
    node := query1(t, "/admin", "#display-all-comments")
    assertElem(t, node, "input")
}

func TestAllCommentsPageHasAllComments(t *testing.T) {
    defer testData.reset()
    login()
    nodes := query(t, "/all_comments", "#comment")
    if len(nodes) != len(testComm) {
        t.Fatalf("Not all comments in /all_comments!")
    }
    testData.expect(t, (*TestData).allComments, "")
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

func TestHiddenPostDoesNotAppearInRss(t *testing.T) {
    bak := testPosts
    testPosts = make([]*Entry, 0)
    testPosts = append(testPosts, mkTestEntry(1, false))
    testPosts = append(testPosts, mkTestEntry(1000, true))
    testPosts = append(testPosts, mkTestEntry(2, false))
    login()
    xml := curl("feeds/rss.xml")
    mustNotContain(t, xml, "hello1000")
    testPosts = bak
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
    defer testData.reset()
    tag := "taaag"
    html := curl("/tag/" + tag)
    testData.expect(t, (*TestData).titlesByTag, tag)
    mustContain(t, html, "Posts tagged ")
    mustContain(t, html, tag)
}

func TestDeletePostCallsDbFunc(t *testing.T) {
    defer testData.reset()
    curl("delete_post?id=hello1001")
    testData.expect(t, (*TestData).deletePost, "hello1001")
}

func TestDeleteCommentCallsDbFunc(t *testing.T) {
    defer testData.reset()
    curl("delete_comment?id=1&action=delete")
    testData.expect(t, (*TestData).deleteComment, "1")
}

func TestShowCaptcha(t *testing.T) {
    url := "comment_submit?name=joe&captcha=&email=snailmail&text=cmmnt%20txt"
    respJSON := curl(url)
    var resp map[string]interface{}
    err := json.Unmarshal([]byte(respJSON), &resp)
    if err != nil {
        t.Fatalf("Failed to parse json %q\nwith error %q", respJSON, err.Error())
    }
    T{t}.failIf(resp["status"] != "showcaptcha", "No captcha box")
}

func TestReturningCommenterSkipsCaptcha(t *testing.T) {
    url := "comment_submit?name=N&captcha=&email=@&website=w&text=cmmnt%20txt"
    respJSON := curl(url)
    var resp map[string]interface{}
    err := json.Unmarshal([]byte(respJSON), &resp)
    if err != nil {
        t.Fatalf("Failed to parse json %q\nwith error %q", respJSON, err.Error())
    }
    T{t}.failIf(resp["status"] != "accepted", "Comment by returning commenter not accepted")
}

func TestDetectedLtLanguageCommentApprove(t *testing.T) {
    defer testData.reset()
    temp := DetectLanguage
    DetectLanguage = func(string) string {
        return `"lt"`
    }
    url := "comment_submit?name=UnknownCommenter&captcha=&email=@&website=w&text=cmmnt%20txt"
    respJSON := curl(url)
    var resp map[string]interface{}
    err := json.Unmarshal([]byte(respJSON), &resp)
    if err != nil {
        t.Fatalf("Failed to parse json %q\nwith error %q", respJSON, err.Error())
    }
    T{t}.failIf(resp["status"] != "accepted", "Comment w/ detected language 'lt' not accepted")
    testData.expectSeries(t, []CallSpec{{(*TestData).postID, ""},
        {(*TestData).postID, ""},
        {(*TestData).postID, ""},
        {(*TestData).insertCommenter, "UnknownCommenter"}})
    DetectLanguage = temp
}

func TestUndetectedLanguageCommentDismiss(t *testing.T) {
    defer testData.reset()
    url := "comment_submit?name=UnknownCommenter&captcha=&email=@&website=w&text=cmmnt%20txt&captcha-id=666"
    respJSON := curl(url)
    var resp map[string]interface{}
    err := json.Unmarshal([]byte(respJSON), &resp)
    if err != nil {
        t.Fatalf("Failed to parse json %q\nwith error %q", respJSON, err.Error())
    }
    T{t}.failIf(resp["status"] != "rejected", "Comment with undetected language not rejected")
    testData.expect(t, (*TestData).postID, "")
}

func TestCorrectCaptchaReply(t *testing.T) {
    defer testData.reset()
    SetNextTask(0)
    task := GetTask()
    captchaURL := fmt.Sprintf("&captcha-id=%s&captcha=%s", task.ID, task.Answer)
    url := "comment_submit?name=UnknownCommenter&email=@&website=w&text=cmmnt%20txt" + captchaURL
    respJSON := curl(url)
    var resp map[string]interface{}
    err := json.Unmarshal([]byte(respJSON), &resp)
    if err != nil {
        t.Fatalf("Failed to parse json %q\nwith error %q", respJSON, err.Error())
    }
    T{t}.failIf(resp["status"] != "accepted", "Comment with correct captcha reply not accepted")
    testData.expectSeries(t, []CallSpec{{(*TestData).postID, ""},
        {(*TestData).insertCommenter, "UnknownCommenter"}})
}

func TestRssFeed(t *testing.T) {
    xml := curl("feeds/rss.xml")
    url := tserver.URL
    mustContain(t, xml, fmt.Sprintf("<link>%s</link>", url))
    mustContain(t, xml, "<title>Hi3</title>")
    mustContain(t, xml, fmt.Sprintf("<link>%s/%s</link>", url, "hello3"))
}

func TestRobotsTxtGetsServed(t *testing.T) {
    robots := curl("robots.txt")
    mustContain(t, robots, "Disallow")
}

func TestPagination(t *testing.T) {
    nodes := query0(t, "page/2", ".post-title")
    T{t}.failIf(len(nodes) != PostsPerPage, "Not all posts have been rendered!")
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

func TestGetUnknownKeyFromConfigReturnsEmptyString(t *testing.T) {
    val := conf.Get("unknown-key")
    if val != "" {
        t.Fatalf("val should be empty: %+v", val)
    }
}

func TestLoadUnexistantConfig(t *testing.T) {
    c := loadConfig("unexistant-file")
    if len(c) != 0 {
        t.Fatalf("Config should be empty: %+v", c)
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
    passwdHash, err := Encrypt(passwd)
    if err != nil {
        panic(fmt.Sprintf("Error in Encrypt(): %s\n", err))
    }
    testAuthor.Passwd = passwdHash
    testAuthor.UserName = uname
}
