package htmltest

import (
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/rtfb/go-html-transform/css/selector"
	"github.com/rtfb/go-html-transform/h5"
	"golang.org/x/net/html"
)

var (
	tclient *http.Client
	tserver *httptest.Server
)

func initClient() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	tclient = &http.Client{
		Jar: jar,
	}
}

func initServer(router http.Handler) {
	tserver = httptest.NewServer(router)
}

func Init(router http.Handler) {
	initClient()
	initServer(router)
}

func Client() *http.Client {
	return tclient
}

func CssSelect(t *testing.T, node *html.Node, query string) []*html.Node {
	chain, err := selector.Selector(query)
	if err != nil {
		t.Fatalf("Error: query=%q, err=%s", query, err.Error())
	}
	return chain.Find(node)
}

func Query(t *testing.T, url, method, query string) []*html.Node {
	switch method {
	case "+":
		return queryPlus(t, url, query)
	case "*":
		return queryStar(t, url, query)
	case "1":
		return []*html.Node{QueryOne(t, url, query)}
	default:
		t.Fatalf("Error: unknown query method: %q", method)
	}
	return nil
}

// QueryOne ensures there's only one thing that matches.
func QueryOne(t *testing.T, url, q string) *html.Node {
	nodes := queryPlus(t, url, q)
	if len(nodes) > 1 {
		t.Fatalf("Too many matches (%d) for node: %q", len(nodes), q)
	}
	return nodes[0]
}

// queryPlus is like + in regular expressions: requires one or more matches.
func queryPlus(t *testing.T, url, query string) []*html.Node {
	nodes := queryStar(t, url, query)
	if len(nodes) == 0 {
		t.Fatalf("No nodes found: %q", query)
	}
	return nodes
}

// queryStar is like * in regular expressions: requires zero or more matches.
func queryStar(t *testing.T, url, query string) []*html.Node {
	html := Curl(url)
	doctree, err := h5.NewFromString(html)
	if err != nil {
		t.Fatalf("h5.NewFromString(%s) = err %q", html, err.Error())
	}
	return CssSelect(t, doctree.Top(), query)
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

func Curl(url string) string {
	return curlParam(url, tclientGet)
}

func CurlPost(url string) string {
	return curlParam(url, tclientPostForm)
}

func PostForm(path string, values *url.Values) (string, error) {
	resp, err := tclient.PostForm(PathToURL(path), *values)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body), err
}

func PathToURL(path string) string {
	if path == "" {
		return tserver.URL
	} else if path[0] == '/' {
		return tserver.URL + path
	} else {
		return tserver.URL + "/" + path
	}
}

func tclientGet(rqURL string) (*http.Response, error) {
	return tclient.Get(PathToURL(rqURL))
}

func tclientPostForm(rqURL string) (*http.Response, error) {
	return tclient.PostForm(PathToURL(rqURL), url.Values{})
}
