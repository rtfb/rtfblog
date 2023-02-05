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

type HT struct {
	client *http.Client
	server *httptest.Server
}

func initClient() *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &http.Client{
		Jar: jar,
	}
}

func initServer(router http.Handler) *httptest.Server {
	return httptest.NewServer(router)
}

func New(router http.Handler) HT {
	return HT{
		client: initClient(),
		server: initServer(router),
	}
}

func (ht *HT) Client() *http.Client {
	return ht.client
}

func (ht *HT) CssSelect(t *testing.T, node *html.Node, query string) []*html.Node {
	chain, err := selector.Selector(query)
	if err != nil {
		t.Fatalf("Error: query=%q, err=%s", query, err.Error())
	}
	return chain.Find(node)
}

func (ht *HT) Query(t *testing.T, url, method, query string) []*html.Node {
	switch method {
	case "+":
		return ht.queryPlus(t, url, query)
	case "*":
		return ht.queryStar(t, url, query)
	case "1":
		return []*html.Node{ht.QueryOne(t, url, query)}
	default:
		t.Fatalf("Error: unknown query method: %q", method)
	}
	return nil
}

// QueryOne ensures there's only one thing that matches.
func (ht *HT) QueryOne(t *testing.T, url, q string) *html.Node {
	nodes := ht.queryPlus(t, url, q)
	if len(nodes) > 1 {
		t.Fatalf("Too many matches (%d) for node: %q", len(nodes), q)
	}
	return nodes[0]
}

// queryPlus is like + in regular expressions: requires one or more matches.
func (ht *HT) queryPlus(t *testing.T, url, query string) []*html.Node {
	nodes := ht.queryStar(t, url, query)
	if len(nodes) == 0 {
		t.Fatalf("No nodes found: %q", query)
	}
	return nodes
}

// queryStar is like * in regular expressions: requires zero or more matches.
func (ht *HT) queryStar(t *testing.T, url, query string) []*html.Node {
	html := ht.Curl(url)
	doctree, err := h5.NewFromString(html)
	if err != nil {
		t.Fatalf("h5.NewFromString(%s) = err %q", html, err.Error())
	}
	return ht.CssSelect(t, doctree.Top(), query)
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

func (ht *HT) Curl(url string) string {
	return curlParam(url, ht.clientGet)
}

func (ht *HT) CurlPost(url string) string {
	return curlParam(url, ht.clientPostForm)
}

func (ht *HT) PostForm(path string, values *url.Values) (string, error) {
	resp, err := ht.client.PostForm(ht.PathToURL(path), *values)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body), err
}

func (ht *HT) PathToURL(path string) string {
	if path == "" {
		return ht.server.URL
	} else if path[0] == '/' {
		return ht.server.URL + path
	} else {
		return ht.server.URL + "/" + path
	}
}

func (ht *HT) clientGet(rqURL string) (*http.Response, error) {
	return ht.client.Get(ht.PathToURL(rqURL))
}

func (ht *HT) clientPostForm(rqURL string) (*http.Response, error) {
	return ht.client.PostForm(ht.PathToURL(rqURL), url.Values{})
}
