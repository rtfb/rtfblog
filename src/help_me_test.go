package rtfblog

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/rtfb/rtfblog/src/assets"
	"github.com/stretchr/testify/require"
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

func mustContain(t *testing.T, page, what string) {
	require.Contains(t, page, what)
}

func mustNotContain(t *testing.T, page, what string) {
	require.NotContains(t, page, what)
}

func postForm(t *testing.T, path string, values *url.Values, testFunc func(html string)) {
	defer testData.reset()
	ensureLogin()
	body, err := tserver.PostForm(path, values)
	if err != nil {
		t.Error(err)
	}
	testFunc(body)
}

func loginWithCred(username, passwd string) string {
	body, err := tserver.PostForm("login", &url.Values{
		"uname":  {username},
		"passwd": {passwd},
	})
	if err != nil {
		println(err.Error())
		return ""
	}
	return body
}

func ensureLogin() {
	loginWithCred("testuser", "testpasswd")
}

func doLogout() {
	tserver.Curl("logout")
}

func assertElem(t *testing.T, node *html.Node, elem string) {
	if !strings.HasPrefix(node.Data, elem) {
		T{t}.failIf(true, "<%s> expected, but <%s> found!", elem, node.Data)
	}
}

func mkTempFile(t *testing.T, name, content string) func() {
	exists, err := assets.FileExists(name)
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
