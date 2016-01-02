package main

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/rtfb/htmltest"
	"golang.org/x/net/html"
)

func mustUnmarshal(t *testing.T, jsonObj string) map[string]interface{} {
	var obj map[string]interface{}
	err := json.Unmarshal([]byte(jsonObj), &obj)
	if err != nil {
		t.Fatalf("json.Unmarshal(%v) =\nerror %q", jsonObj, err.Error())
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

func postForm(t *testing.T, path string, values *url.Values, testFunc func(html string)) {
	defer testData.reset()
	login()
	body, err := htmltest.PostForm(path, values)
	if err != nil {
		t.Error(err)
	}
	testFunc(body)
}

func loginWithCred(username, passwd string) string {
	body, err := htmltest.PostForm("login", &url.Values{
		"uname":  {username},
		"passwd": {passwd},
	})
	if err != nil {
		println(err.Error())
		return ""
	}
	return body
}

func login() {
	loginWithCred("testuser", "testpasswd")
}

func logout() {
	htmltest.Curl("logout")
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

func mkQueryURL(qry string, params map[string]string) string {
	bits := []string{}
	for k, v := range params {
		bits = append(bits, k+"="+v)
	}
	return qry + "?" + strings.Join(bits, "&")
}
