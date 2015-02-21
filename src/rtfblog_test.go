package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
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

	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/rtfb/bark"
	"github.com/rtfb/go-html-transform/css/selector"
	"github.com/rtfb/go-html-transform/h5"
	"golang.org/x/net/html"
)

type T struct {
	*testing.T
}

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
		EntryLink: EntryLink{
			Title:  fmt.Sprintf("Hi%d", i),
			URL:    fmt.Sprintf("hello%d", i),
			Hidden: hidden,
		},
		Author:   auth,
		Date:     date,
		Body:     template.HTML(fmt.Sprintf("Body%d", i)),
		RawBody:  fmt.Sprintf("RawBody%d", i),
		Tags:     []*Tag{{Id: int64(i), Name: fmt.Sprintf("u%d", i)}},
		Comments: testComm,
	}
}

type TestCryptoHelper struct{}

func (h TestCryptoHelper) Encrypt(passwd string) (hash string, err error) {
	return "", nil
}

func (h TestCryptoHelper) Decrypt(hash, passwd []byte) error {
	if string(passwd) == "testpasswd" {
		return nil
	}
	return errors.New("bad passwd")
}

type TestLangDetector struct{}

func (d TestLangDetector) Detect(text string) string {
	return "foo"
}

type LTLangDetector struct{}

func (d LTLangDetector) Detect(text string) string {
	return `"lt"`
}

func forgeTestUser(uname, passwd string) {
	passwdHash, err := cryptoHelper.Encrypt(passwd)
	if err != nil {
		panic(fmt.Sprintf("Error in Encrypt(): %s\n", err))
	}
	testAuthor.Passwd = passwdHash
	testAuthor.UserName = uname
}

func init() {
	root := ".."
	conf = obtainConfiguration("")
	conf["staticdir"] = filepath.Join(root, "static")
	InitL10n(root, "en-US")
	logger = bark.CreateFile("tests.log")
	forgeTestUser("testuser", "testpasswd")
	for i := 1; i <= 11; i++ {
		testPosts = append(testPosts, mkTestEntry(i, false))
	}
	for i := 1; i <= 2; i++ {
		testPosts = append(testPosts, mkTestEntry(i+1000, true))
	}
	langDetector = TestLangDetector{}
	cryptoHelper = TestCryptoHelper{}
	testData = TestData{}
	initTestClient()
	initTestServer(initRoutes(&GlobalContext{
		Router: pat.New(),
		Db:     &testData,
		Root:   root,
		Store:  sessions.NewCookieStore([]byte("aaabbbcccddd")),
	}))
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
			{(*TestData).updateTags, "0: {Id:0 Name:tagzorz}"}})
	})
}

func TestUploadImageHandlesWrongRequest(t *testing.T) {
	postForm(t, "upload_images", &url.Values{
		"foo": {"bar"},
	}, func(html string) {
		T{t}.assertEqual("HTTP Error 500", html)
	})
}

// Creates a new file upload http request with optional extra params
func mkFakeFileUploadRequest(uri string, params map[string]string, paramName, fileName, contents string) (*http.Request, error) {
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
	req, err := http.NewRequest("POST", localhostURL(uri), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
	return req, nil
}

func TestUploadImage(t *testing.T) {
	uploadedFile := "../static/testupload.md"
	testContent := "Foobarbaz"
	defer func() {
		err := os.Remove(uploadedFile)
		if err != nil {
			t.Fatal(err)
		}
	}()
	extraParams := map[string]string{
		"title":       "My Document",
		"author":      "The Author",
		"description": "The finest document",
	}
	request, err := mkFakeFileUploadRequest("upload_images", extraParams, "file", "testupload.md", testContent)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := tclient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	T{t}.assertEqual("200", fmt.Sprintf("%d", resp.StatusCode))
	T{t}.assertEqual("[foo]: /testupload.md", string(body.Bytes()))
	fileBytes, err := ioutil.ReadFile(uploadedFile)
	if err != nil {
		t.Fatal(err)
	}
	prefix := string(fileBytes)[:len(testContent)]
	T{t}.assertEqual(testContent, prefix)
}

func TestExplodeTags(t *testing.T) {
	var tagSpecs = []struct {
		spec, expected string
	}{
		{"tag", "{Id:0 Name:tag}"},
		{",tagg", "{Id:0 Name:tagg}"},
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
	resp := mustUnmarshal(t, curl(url))
	T{t}.failIf(resp["status"] != "showcaptcha", "No captcha box")
}

func TestReturningCommenterSkipsCaptcha(t *testing.T) {
	url := "comment_submit?name=N&captcha=&email=@&website=w&text=cmmnt%20txt"
	resp := mustUnmarshal(t, curl(url))
	T{t}.failIf(resp["status"] != "accepted", "Comment by returning commenter not accepted")
}

func TestDetectedLtLanguageCommentApprove(t *testing.T) {
	defer testData.reset()
	temp := langDetector
	defer func() {
		langDetector = temp
	}()
	langDetector = LTLangDetector{}
	url := "comment_submit?name=UnknownCommenter&captcha=&email=@&website=w&text=cmmnt%20txt"
	resp := mustUnmarshal(t, curl(url))
	T{t}.failIf(resp["status"] != "accepted", "Comment w/ detected language 'lt' not accepted")
	testData.expectSeries(t, []CallSpec{{(*TestData).postID, ""},
		{(*TestData).postID, ""},
		{(*TestData).postID, ""},
		{(*TestData).insertCommenter, "UnknownCommenter"}})
}

func TestUndetectedLanguageCommentDismiss(t *testing.T) {
	defer testData.reset()
	url := "comment_submit?name=UnknownCommenter&captcha=&email=@&website=w&text=cmmnt%20txt&captcha-id=666"
	resp := mustUnmarshal(t, curl(url))
	T{t}.failIf(resp["status"] != "rejected", "Comment with undetected language not rejected")
	testData.expect(t, (*TestData).postID, "")
}

func TestCorrectCaptchaReply(t *testing.T) {
	defer testData.reset()
	deck := NewDeck()
	deck.SetNextTask(0)
	task := deck.GetTask()
	captchaURL := fmt.Sprintf("&captcha-id=%s&captcha=%s", task.ID, task.Answer)
	url := "comment_submit?name=UnknownCommenter&email=@&website=w&text=cmmnt%20txt" + captchaURL
	resp := mustUnmarshal(t, curl(url))
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
	html := curl("page/2")
	mustContain(t, html, "<a href=\"/page/1\">1</a>\n2\n<a href=\"/page/3\">3</a>\n")
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

func TestPathToFullPath(t *testing.T) {
	T{t}.assertEqual("/a/b/c", PathToFullPath("/a/b/c"))
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	T{t}.assertEqual(filepath.Join(cwd, "b/c"), PathToFullPath("./b/c"))
}

func TestVersionString(t *testing.T) {
	expected := "foobar"
	del := mkTempFile(t, "VERSION", expected)
	defer del()
	T{t}.assertEqual(expected, versionString())
}

func TestObtainConfiguration(t *testing.T) {
	del := mkTempFile(t, ".rtfblogrc", `{"foo": "bar"}`)
	defer del()
	config := obtainConfiguration(".")
	foo, ok := config["foo"]
	if !ok || foo == nil {
		t.Fatal("No foo")
	}
	T{t}.assertEqual("bar", foo.(string))
}

func TestMkNotifEmail(t *testing.T) {
	subj, body := mkCommentNotifEmail(Commenter{
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
	T{t}.assertEqual("d3b07384d113edec49eaa6238ad5ff00", Md5Hash("foo\n"))
}
