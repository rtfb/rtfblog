package rtfblog

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/rtfb/go-html-transform/h5"
	"github.com/rtfb/rtfblog/src/assets"
	"github.com/rtfb/rtfblog/src/htmltest"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

type T struct {
	*testing.T
}

const (
	buildRoot = "../build"
)

var (
	testComm = []*Comment{{Commenter{"N", "@", "@h", "http://w", "IP"},
		CommentTable{0, 0, "Body", "Raw", "time", time.Now().Unix(), 0}}}
	testPosts  = make([]*Entry, 0)
	testAuthor = new(Author)
)

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

func mkTestEntry(i int, hidden bool) *Entry {
	auth := "Author"
	date := "2013-03-19"
	return &Entry{
		EntryTable: EntryTable{
			EntryLink: EntryLink{
				Title:  fmt.Sprintf("Hi%d", i),
				URL:    fmt.Sprintf("hello%d", i),
				Hidden: hidden,
			},
			Date:    date,
			Body:    template.HTML(fmt.Sprintf("Body%d", i)),
			RawBody: fmt.Sprintf("RawBody%d", i),
		},
		Author:   auth,
		Tags:     []*Tag{{ID: int64(i), Name: fmt.Sprintf("u%d", i)}},
		Comments: testComm,
	}
}

type TestCryptoHelper struct{}

func (h TestCryptoHelper) Encrypt(passwd string) (hash string, err error) {
	return passwd, nil
}

func (h TestCryptoHelper) Decrypt(hash, passwd []byte) error {
	if string(passwd) == "testpasswd" {
		return nil
	}
	return errors.New("bad passwd")
}

type TestLangDetector struct{}

func (d TestLangDetector) Detect(text string, log *slog.Logger) string {
	return "foo"
}

type LTLangDetector struct{}

func (d LTLangDetector) Detect(text string, log *slog.Logger) string {
	return `"lt"`
}

func forgeTestUser(s server, uname, passwd string) {
	passwdHash, err := s.cryptoHelper.Encrypt(passwd)
	if err != nil {
		panic(fmt.Sprintf("Error in Encrypt(): %s\n", err))
	}
	testAuthor.Passwd = passwdHash
	testAuthor.UserName = uname
}

var tserver htmltest.HT

func initTests(uploadsDir string) server {
	conf := readConfigs()
	conf.Server.StaticDir = "static"
	if uploadsDir == "" {
		uploadsDir = conf.Server.UploadsRoot
	}
	slogger := newMainLogger("tests.log")
	assets, err := assets.NewBin(buildRoot, uploadsDir, slogger)
	if err != nil {
		panic(err)
	}
	InitL10n(assets, "en-US")
	for i := 1; i <= 11; i++ {
		testPosts = append(testPosts, mkTestEntry(i, false))
	}
	for i := 1; i <= 2; i++ {
		testPosts = append(testPosts, mkTestEntry(i+1000, true))
	}
	langDetector = TestLangDetector{}
	testData = TestData{}
	gctx := newGlobalContext(&testData, assets, "aaabbbcccddd", slogger)
	s := newServer(&TestCryptoHelper{}, gctx, conf)
	forgeTestUser(s, "testuser", "testpasswd")
	return s
}

func init() {
	s := initTests("")
	tserver = htmltest.New(s.initRoutes(slog.Default()))
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
		mustContain(t, tserver.Curl(test.url), test.out)
	}
}

func TestBasicStructure(t *testing.T) {
	var blocks = []string{
		"#header", "#subheader", "#content", "#footer", "#sidebar",
	}
	for _, block := range blocks {
		node := tserver.QueryOne(t, "", block)
		assertElem(t, node, "div")
	}
}

func TestEmptyDatasetGeneratesFriendlyError(t *testing.T) {
	tmpPosts := testPosts
	testPosts = nil
	defer func() {
		testPosts = tmpPosts
	}()
	html := tserver.Curl("")
	mustContain(t, html, "No entries")
}

func TestLogin(t *testing.T) {
	ensureLogin()
	html := tserver.Curl(testPosts[0].URL)
	mustContain(t, html, "Logout")
}

func TestBadLogin(t *testing.T) {
	html := loginWithCred("wronguser", "wrongpasswd")
	mustContain(t, html, "Login failed")
	html = loginWithCred("testuser", "wrongpasswd")
	mustContain(t, html, "Login failed")
}

func TestNonEmptyDatasetHasEntries(t *testing.T) {
	mustNotContain(t, tserver.Curl(""), "No entries")
}

func TestEntryListHasAuthor(t *testing.T) {
	nodes := tserver.Query(t, "", "+", ".author")
	for _, node := range nodes {
		assertElem(t, node, "div")
		require.NotEmpty(t, h5.Children(node), "Empty author div!")
		checkAuthorSection(T{t}, node)
	}
}

func TestEntriesHaveTagsInList(t *testing.T) {
	nodes := tserver.Query(t, "", "+", ".tags")
	for _, node := range nodes {
		assertElem(t, node, "div")
		require.NotEmpty(t, h5.Children(node), "Empty tags div!")
		checkTagsSection(T{t}, node)
	}
}

func checkTagsSection(t T, node *html.Node) {
	if strings.Contains(h5.NewTree(node).String(), "&nbsp;") {
		return
	}
	n2 := tserver.CssSelect(t.T, node, "a")
	t.failIf(len(n2) == 0, "Tags node not found in section: %q", h5.NewTree(node).String())
}

func checkAuthorSection(t T, node *html.Node) {
	date := node.FirstChild.Data
	dateRe, _ := regexp.Compile("[0-9]{4}-[0-9]{2}-[0-9]{2}")
	m := dateRe.FindString(date)
	t.failIf(m == "", "No date found in author section!")
	n2 := tserver.CssSelect(t.T, node, "strong")
	t.failIf(len(n2) != 1, "Author node not found in section: %q", h5.NewTree(node).String())
	t.failIf(h5.Children(n2[0]) == nil, "Author node not found in section: %q", h5.NewTree(node).String())
}

func TestEveryEntryHasAuthor(t *testing.T) {
	for _, e := range testPosts {
		mustContain(t, tserver.Curl(e.URL), "captcha-id")
	}
}

func TestEveryEntryHasCaptchaSection(t *testing.T) {
	for _, e := range testPosts {
		node := tserver.QueryOne(t, e.URL, ".author")
		assertElem(t, node, "div")
		require.NotEmpty(t, h5.Children(node), "Empty author div!")
		checkAuthorSection(T{t}, node)
	}
}

func TestCommentsFormattingInPostPage(t *testing.T) {
	for _, p := range testPosts {
		nodes := tserver.Query(t, p.URL, "*", "#comments")
		require.Len(t, nodes, 1, "There should be only one comments section!")
		for _, node := range nodes {
			assertElem(t, node, "div")
			require.False(t, emptyChildren(node), "Empty comments div found!")
			checkCommentsSection(T{t}, node)
		}
	}
}

func checkCommentsSection(t T, node *html.Node) {
	noComments := tserver.CssSelect(t.T, node, "p")
	comments := tserver.CssSelect(t.T, node, "strong")
	t.failIf(len(noComments) == 0 && len(comments) == 0,
		"Comments node not found in section: %q", h5.NewTree(node).String())
	if len(comments) > 0 {
		headers := tserver.CssSelect(t.T, node, ".comment-container")
		t.failIf(len(headers) == 0,
			"Comment header not found in section: %q", h5.NewTree(node).String())
		bodies := tserver.CssSelect(t.T, node, ".bubble-container")
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
		nodes := tserver.Query(t, e.URL, "*", ".tags")
		if len(nodes) > 0 {
			for _, node := range nodes {
				assertElem(t, node, "div")
				require.NotEmpty(t, h5.Children(node), "Empty tags div!")
				checkTagsSection(T{t}, node)
			}
		}
	}
}

func TestPostPageHasCommentEditor(t *testing.T) {
	for _, p := range testPosts {
		node := tserver.QueryOne(t, p.URL, "#comment")
		assertElem(t, node, "form")
	}
}

func TestLoginPage(t *testing.T) {
	node := tserver.QueryOne(t, "login", "#login_form")
	assertElem(t, node, "form")
}

func TestOnlyOnePageOfPostsAppearsOnMainPage(t *testing.T) {
	nodes := tserver.Query(t, "", "*", ".post-title")
	require.Len(t, nodes, PostsPerPage, "Not all posts have been rendered!")
}

func TestArchiveContainsAllEntries(t *testing.T) {
	if len(testPosts) <= NumRecentPosts {
		t.Fatalf("This test only makes sense if len(testPosts) > NUM_RECENT_POSTS")
	}
	nodes := tserver.Query(t, "archive", "*", ".post-title")
	require.Len(t, nodes, len(testPosts), "Not all posts rendered in archive!")
}

func TestPostPager(t *testing.T) {
	mustContain(t, tserver.Curl(""), "/page/2")
}

func TestInvalidPageDefaultsToPageOne(t *testing.T) {
	page1 := tserver.Curl("/page/1")
	pageFoo := tserver.Curl("/page/foo")
	T{t}.failIf(page1 != pageFoo, "Invalid page did not produce /page/1")
}

func TestNonAdminCantAccessAdminPages(t *testing.T) {
	doLogout()
	urls := []string{
		"all_comments",
		"admin",
		"edit_post",
		"load_comments",
		"delete_comment",
		"delete_post",
	}
	for _, u := range urls {
		html := tserver.Curl(u)
		mustContain(t, html, "Verboten")
	}
	postUrls := []string{
		"moderate_comment",
		"submit_post",
		"upload_images",
	}
	for _, u := range postUrls {
		html := tserver.CurlPost(u)
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
	ensureLogin()
	json := tserver.Curl("/load_comments?post=hello1")
	mustContain(t, json, `"Comments":[{"Name":"N","Email":"@"`)
}

func TestSubmitNewPost(t *testing.T) {
	defer testData.reset()
	postForm(t, "submit_post", &url.Values{
		"title":  {"T1tlE"},
		"url":    {"shiny-url"},
		"tags":   {"tagzorz"},
		"hidden": {"off"},
		"text":   {"contentzorz"},
	}, func(html string) {
		testData.expectChain(t, []CallSpec{
			{(*TestData).insertPost, fmt.Sprintf("%+v", &EntryTable{
				EntryLink: EntryLink{
					Title:  "T1tlE",
					URL:    "shiny-url",
					Hidden: false,
				},
				RawBody: "contentzorz",
			})},
			{(*TestData).updateTags, "0: {ID:0 Name:tagzorz}"}})
	})
}

func TestSubmitPost(t *testing.T) {
	defer testData.reset()
	postForm(t, "submit_post", &url.Values{
		"title":  {"T1tlE"},
		"url":    {testPosts[0].URL},
		"tags":   {"tagzorz"},
		"hidden": {"off"},
		"text":   {"contentzorz"},
	}, func(html string) {
		testData.expectChain(t, []CallSpec{
			{(*TestData).updatePost, "0"},
			{(*TestData).updateTags, "0: {ID:0 Name:tagzorz}"}})
	})
}

func TestUploadImageHandlesWrongRequest(t *testing.T) {
	postForm(t, "upload_images", &url.Values{
		"foo": {"bar"},
	}, func(html string) {
		T{t}.assertEqual("HTTP Error 500\n", html)
	})
}

// Creates a new file upload http request with optional extra params
func mkFakeFileUploadRequest(ht htmltest.HT, uri string, params map[string]string, paramName, fileName, contents string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, fileName)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, strings.NewReader(contents))
	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, ht.PathToURL(uri), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
	return req, nil
}

func TestUploadImage(t *testing.T) {
	tempDir := t.TempDir()
	s := initTests(tempDir)
	tserver := htmltest.New(s.initRoutes(slog.Default()))

	const username = "testuser"
	const passwd = "testpasswd"
	_, err := tserver.PostForm("login", &url.Values{
		"uname":  {username},
		"passwd": {passwd},
	})
	require.NoError(t, err)

	uploadedFile := filepath.Join(s.gctx.assets.WriteRoot(), "testupload.md")
	testContent := "Foobarbaz"
	extraParams := map[string]string{
		"title":       "My Document",
		"author":      "The Author",
		"description": "The finest document",
	}
	request, err := mkFakeFileUploadRequest(tserver, "upload_images", extraParams, "file", "testupload.md", testContent)
	require.NoError(t, err)
	resp, err := tserver.Client().Do(request)
	require.NoError(t, err)
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	T{t}.assertEqual("200", fmt.Sprintf("%d", resp.StatusCode))
	T{t}.assertEqual("[foo]: /static/testupload.md", string(body.Bytes()))
	fileBytes, err := ioutil.ReadFile(uploadedFile)
	require.NoError(t, err)
	prefix := string(fileBytes)[:len(testContent)]
	T{t}.assertEqual(testContent, prefix)
}

func TestExplodeTags(t *testing.T) {
	var tagSpecs = []struct {
		spec, expected string
	}{
		{"tag", "{ID:0 Name:tag}"},
		{",tagg", "{ID:0 Name:tagg}"},
	}
	for _, ts := range tagSpecs {
		result := fmt.Sprintf("%+v", *explodeTags(ts.spec)[0])
		T{t}.assertEqual(ts.expected, result)
	}
}

func TestMainPageHasEditPostButtonWhenLoggedIn(t *testing.T) {
	ensureLogin()
	nodes := tserver.Query(t, "", "+", ".edit-post-button")
	require.Len(t, nodes, PostsPerPage, "Not all posts have Edit button!")
}

func TestEveryCommentHasEditFormWhenLoggedId(t *testing.T) {
	ensureLogin()
	node := tserver.QueryOne(t, testPosts[0].URL, "#edit-comment-form")
	assertElem(t, node, "form")
}

func TestAdminPageHasAllCommentsButton(t *testing.T) {
	ensureLogin()
	node := tserver.QueryOne(t, "/admin", "#display-all-comments")
	assertElem(t, node, "input")
}

func TestAllCommentsPageHasAllComments(t *testing.T) {
	defer testData.reset()
	ensureLogin()
	nodes := tserver.Query(t, "/all_comments", "+", "#comment")
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
	ensureLogin()
	for _, i := range positiveTests {
		html := tserver.Curl(i.url)
		mustContain(t, html, i.content)
	}
	doLogout()
	for _, i := range negativeTests {
		html := tserver.Curl(i.url)
		mustNotContain(t, html, i.content)
	}
}

func TestHiddenPostDoesNotAppearInRss(t *testing.T) {
	bak := testPosts
	testPosts = make([]*Entry, 0)
	testPosts = append(testPosts, mkTestEntry(1, false))
	testPosts = append(testPosts, mkTestEntry(1000, true))
	testPosts = append(testPosts, mkTestEntry(2, false))
	ensureLogin()
	xml := tserver.Curl("feeds/rss.xml")
	mustNotContain(t, xml, "hello1000")
	testPosts = bak
}

func TestHiddenPostAccess(t *testing.T) {
	ensureLogin()
	html := tserver.Curl("hello1001")
	mustContain(t, html, "Body")
	doLogout()
	html = tserver.Curl("hello1001")
	mustContain(t, html, "Page Not Found")
}

func TestEditPost(t *testing.T) {
	ensureLogin()
	// test with non-hidden post
	html := tserver.Curl("edit_post?post=hello3")
	mustContain(t, html, "Body3")
	mustContain(t, html, "Hi3")
	mustContain(t, html, "u3")
	mustContain(t, html, "Delete!")
	mustNotContain(t, html, "checked")
	// now test with hidden post
	html = tserver.Curl("edit_post?post=hello1002")
	mustContain(t, html, "Body1002")
	mustContain(t, html, "Hi1002")
	mustContain(t, html, "u1002")
	mustContain(t, html, "Delete!")
	mustContain(t, html, "checked")
}

func TestTitleByTagGetsCalled(t *testing.T) {
	defer testData.reset()
	tag := "taaag"
	html := tserver.Curl("/tag/" + tag)
	testData.expect(t, (*TestData).titlesByTag, tag)
	mustContain(t, html, "Posts tagged ")
	mustContain(t, html, tag)
}

func TestDeletePostCallsDbFunc(t *testing.T) {
	defer testData.reset()
	tserver.Curl("delete_post?id=hello1001")
	testData.expect(t, (*TestData).deletePost, "hello1001")
}

func TestDeleteCommentCallsDbFunc(t *testing.T) {
	defer testData.reset()
	tserver.Curl("delete_comment?id=1&action=delete")
	testData.expect(t, (*TestData).deleteComment, "1")
}

func TestShowCaptcha(t *testing.T) {
	url := mkQueryURL("comment_submit", map[string]string{
		"name":    "joe",
		"captcha": "",
		"email":   "snailmail",
		"text":    "cmmnt%20txt",
	})
	resp := mustUnmarshal(t, tserver.Curl(url))
	T{t}.failIf(resp["status"] != "showcaptcha", "No captcha box")
}

func TestReturningCommenterSkipsCaptcha(t *testing.T) {
	url := mkQueryURL("comment_submit", map[string]string{
		"name":    "N",
		"captcha": "",
		"email":   "@",
		"website": "w",
		"text":    "cmmnt%20txt",
	})
	resp := mustUnmarshal(t, tserver.Curl(url))
	T{t}.failIf(resp["status"] != "accepted", "Comment by returning commenter not accepted")
}

func TestDetectedLtLanguageCommentApprove(t *testing.T) {
	defer testData.reset()
	temp := langDetector
	defer func() {
		langDetector = temp
	}()
	langDetector = LTLangDetector{}
	url := mkQueryURL("comment_submit", map[string]string{
		"name":    "UnknownCommenter",
		"captcha": "",
		"email":   "@",
		"website": "w",
		"text":    "cmmnt%20txt",
	})
	resp := mustUnmarshal(t, tserver.Curl(url))
	T{t}.failIf(resp["status"] != "accepted", "Comment w/ detected language 'lt' not accepted")
	testData.expectChain(t, []CallSpec{{(*TestData).postID, ""},
		{(*TestData).postID, ""},
		{(*TestData).postID, ""},
		{(*TestData).insertCommenter, "UnknownCommenter"}})
}

func TestUndetectedLanguageCommentDismiss(t *testing.T) {
	defer testData.reset()
	url := mkQueryURL("comment_submit", map[string]string{
		"name":       "UnknownCommenter",
		"captcha":    "",
		"email":      "@",
		"website":    "w",
		"text":       "cmmnt%20txt",
		"captcha-id": "666",
	})
	resp := mustUnmarshal(t, tserver.Curl(url))
	T{t}.failIf(resp["status"] != "rejected", "Comment with undetected language not rejected")
	testData.expect(t, (*TestData).postID, "")
}

func TestCorrectCaptchaReply(t *testing.T) {
	defer testData.reset()
	deck := NewDeck()
	deck.SetNextTask(0)
	task := deck.NextTask()
	url := mkQueryURL("comment_submit", map[string]string{
		"name":       "UnknownCommenter",
		"captcha":    task.Answer,
		"email":      "@",
		"website":    "w",
		"text":       "cmmnt%20txt",
		"captcha-id": task.ID,
	})
	resp := mustUnmarshal(t, tserver.Curl(url))
	T{t}.failIf(resp["status"] != "accepted", "Comment with correct captcha reply not accepted")
	testData.expectChain(t, []CallSpec{{(*TestData).postID, ""},
		{(*TestData).insertCommenter, "UnknownCommenter"}})
}

func TestRssFeed(t *testing.T) {
	xml := tserver.Curl("feeds/rss.xml")
	url := tserver.PathToURL("")
	mustContain(t, xml, fmt.Sprintf("<link>%s</link>", url))
	mustContain(t, xml, "<title>Hi3</title>")
	mustContain(t, xml, fmt.Sprintf("<link>%s/%s</link>", url, "hello3"))
}

func TestRobotsTxtGetsServed(t *testing.T) {
	robots := tserver.Curl("robots.txt")
	mustContain(t, robots, "Disallow")
}

func TestPagination(t *testing.T) {
	nodes := tserver.Query(t, "page/2", "*", ".post-title")
	T{t}.failIf(len(nodes) != PostsPerPage, "Not all posts have been rendered!")
	if nodes[0].Attr[1].Val != "/hello6" {
		t.Fatalf("Wrong post!")
	}
	if nodes[4].Attr[1].Val != "/hello10" {
		t.Fatalf("Wrong post!")
	}
	html := tserver.Curl("page/2")
	mustContain(t, html, "<a href=\"/page/1\">1</a>\n2\n<a href=\"/page/3\">3</a>\n")
}

func TestNewPostShowsEmptyForm(t *testing.T) {
	titleInput := tserver.QueryOne(t, "edit_post", "#post_title")
	assertElem(t, titleInput, "input")
	bodyTextArea := tserver.QueryOne(t, "edit_post", "#wmd-input")
	assertElem(t, bodyTextArea, "textarea")
}

func TestPathToFullPath(t *testing.T) {
	T{t}.assertEqual("/a/b/c", pathToFullPath("/a/b/c"))
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	T{t}.assertEqual(filepath.Join(cwd, "b/c"), pathToFullPath("./b/c"))
}

func TestVersionString(t *testing.T) {
	expected := "foobar"
	// tmp := t.TempDir()
	// err := ioutil.WriteFile(path.Join(tmp, "VERSION"), expected)
	// require.NoError(t, err)
	del := mkTempFile(t, "VERSION", expected)
	defer del()
	T{t}.assertEqual(expected, versionString())
}

func TestReadConfigs(t *testing.T) {
	del := mkTempFile(t, ".rtfblogrc", `server:
    port: 666
`)
	defer del()
	config := readConfigs()
	T{t}.assertEqual("666", config.Server.Port)
}

func TestMkNotifEmail(t *testing.T) {
	subj, body := mkCommentNotifEmail(&Commenter{
		Name:    "Commenter",
		Email:   "comm@ent.er",
		Website: "wwweb",
	}, "text", "foo", "refURL")
	T{t}.assertEqual("New comment in 'refURL'", subj)
	mustContain(t, body, "Commenter")
	mustContain(t, body, "comm@ent.er")
	mustContain(t, body, "wwweb")
	mustContain(t, body, "text")
	mustContain(t, body, "New comment from")
}

func TestMarkdown(t *testing.T) {
	md := "foo _bar_ **baz**"
	html := mdToHTML(md)
	expected := "<p>foo <em>bar</em> <strong>baz</strong></p>\n"
	if string(html) != expected {
		t.Errorf("mdToHTML(%s) = %q; want %q", md, html, expected)
	}
	html = []byte(`<p>foo</p><script>evil</script><a href="xyzzy"></a>`)
	expected = `<p>foo</p><a href="xyzzy" rel="nofollow"></a>`
	sanitized := sanitizeHTML(html)
	if string(sanitized) != expected {
		t.Errorf("sanitizeHTML(%s) = %q; want %q", html, sanitized, expected)
	}
	html = []byte(`<p>foo</p><script>evil</script><img alt="xyzzy"></img>`)
	expected = `<p>foo</p><img alt="xyzzy"></img>`
	sanitized = sanitizeTrustedHTML(html)
	if string(sanitized) != expected {
		t.Errorf("sanitizeTrustedHTML(%s) = %q; want %q",
			html, sanitized, expected)
	}
}

func TestMd5(t *testing.T) {
	T{t}.assertEqual("d3b07384d113edec49eaa6238ad5ff00", md5Hash("foo\n"))
}

func TestAdminPageHasEditAuthorButton(t *testing.T) {
	mustContain(t, tserver.Curl("/admin"), "Edit Author Profile")
}

func TestMainPageShowsCreateAuthorPage(t *testing.T) {
	tmp := testAuthor
	testAuthor = nil
	html := tserver.Curl("/")
	mustContain(t, html, "New Password")
	mustContain(t, html, "Confirm Password")
	mustNotContain(t, html, "Old Password")
	testAuthor = tmp
}

func TestEditAuthor(t *testing.T) {
	ensureLogin()
	html := tserver.Curl("/edit_author")
	mustContain(t, html, "New Password")
	mustContain(t, html, "Confirm Password")
	mustContain(t, html, "Old Password")
}
