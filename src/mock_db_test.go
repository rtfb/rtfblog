package main

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"
)

type CallSpec struct {
	function interface{}
	params   string
}

type TestDataI interface {
	reset()
	calls() string
	pushCall(paramStr string)
	expect(t *testing.T, f interface{}, paramStr string)
}

type TestData struct {
	Data
	TestDataI
	lastCalls []string
}

var (
	testData TestData
)

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

func (td *TestData) post(url string, includeHidden bool) *Entry {
	for _, e := range td.testPosts(includeHidden) {
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

func (td *TestData) testPosts(includeHidden bool) []*Entry {
	if includeHidden {
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

func (td *TestData) posts(limit, offset int, includeHidden bool) []*Entry {
	if offset < 0 {
		offset = 0
	}
	tp := td.testPosts(includeHidden)
	if limit > 0 && limit < len(tp) {
		return tp[offset:(offset + limit)]
	}
	return tp
}

func (td *TestData) numPosts(includeHidden bool) int {
	return len(td.testPosts(includeHidden))
}

func (td *TestData) titles(limit int, includeHidden bool) (links []EntryLink, err error) {
	err = nil
	for _, p := range td.testPosts(includeHidden) {
		links = append(links, EntryLink{p.Title, p.URL, false})
	}
	return
}

func (td *TestData) titlesByTag(tag string, includeHidden bool) ([]EntryLink, error) {
	td.pushCall(tag)
	return nil, nil
}

func (td *TestData) allComments() ([]*CommentWithPostTitle, error) {
	td.pushCall("")
	var comments []*CommentWithPostTitle
	for _, c := range testComm {
		comments = append(comments, &CommentWithPostTitle{
			Comment: *c,
			EntryLink: EntryLink{
				URL:   testPosts[0].URL,
				Title: testPosts[0].Title,
			},
		})
	}
	return comments, nil
}

func (td *TestData) author(username string) (*Author, error) {
	if username == testAuthor.UserName {
		return testAuthor, nil
	}
	return nil, gorm.RecordNotFound
}

func (td *TestData) deleteComment(id string) error {
	td.pushCall(id)
	return nil
}

func (td *TestData) deletePost(url string) error {
	td.pushCall(url)
	return nil
}

func (td *TestData) updateComment(id, text string) error {
	td.pushCall(fmt.Sprintf("%s - %s", id, text))
	return nil
}

func (td *TestData) queryAllTags() ([]*Tag, error) {
	return nil, nil
}

func (td *TestData) begin() error {
	return nil
}

func (td *TestData) commit() {
}

func (td *TestData) rollback() {
}

func (td *TestData) insertCommenter(c *Commenter) (id int64, err error) {
	td.pushCall(c.Name)
	return
}

func (td *TestData) commenterID(c *Commenter) (id int64, err error) {
	tc := testComm[0]
	if c.Name == tc.Name && c.Email == tc.Email && c.Website == tc.Website {
		return 1, nil
	}
	return -1, gorm.RecordNotFound
}

func (td *TestData) insertComment(commenterID, postID int64, body string) (id int64, err error) {
	return
}

func (td *TestData) insertPost(e *EntryTable) (id int64, err error) {
	return
}

func (td *TestData) updatePost(e *EntryTable) error {
	td.pushCall("0")
	return nil
}

func (td *TestData) updateTags(tags []*Tag, postID int64) error {
	td.pushCall(fmt.Sprintf("%d: %+v", postID, *tags[0]))
	return nil
}
