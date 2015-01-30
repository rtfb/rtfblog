package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/rtfb/go-html-transform/h5"
	"golang.org/x/net/html"
)

var (
	tclient *http.Client
	tserver *httptest.Server
)

func initTestClient() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	tclient = &http.Client{
		Jar: jar,
	}
}

func initTestServer(router http.Handler) {
	tserver = httptest.NewServer(router)
}

func mustUnmarshal(t *testing.T, jsonObj string) map[string]interface{} {
	var obj map[string]interface{}
	err := json.Unmarshal([]byte(jsonObj), &obj)
	if err != nil {
		t.Fatalf("json.Unmarshal(%q) =\nerror %q", jsonObj, err.Error())
	}
	return obj
}

func mustContain(t *testing.T, page string, what string) {
	if !strings.Contains(page, what) {
		t.Fatalf("Test page did not contain %q\npage:\n%s", what, page)
	}
}

func mustNotContain(t *testing.T, page string, what string) {
	if strings.Contains(page, what) {
		t.Fatalf("Test page incorrectly contained %q", what)
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
		t.Fatalf("h5.NewFromString(%s) = err %q", html, err.Error())
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

func mkTempFile(t *testing.T, name, content string) func() {
	exists, err := FileExists(name)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Errorf("Refusing to overwrite %q, already exists", name)
	}
	err = ioutil.WriteFile(name, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return func() {
		err := os.Remove(name)
		if err != nil {
			t.Fatal(err)
		}
	}
}
