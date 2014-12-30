package main

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

var (
	realDB Data
)

func init() {
	conf = obtainConfiguration("")
	config := "$RTFBLOG_DB_TEST_URL"
	envVar := os.ExpandEnv(config)
	if envVar == "" {
		return
	}
	if !strings.HasPrefix(envVar, "host=/tmp/PGSQL-") {
		return
	}
	conf["database"] = config
	db, err := sql.Open("postgres", getDBConnString())
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	realDB = &DbData{
		db:            db,
		tx:            nil,
		includeHidden: false,
	}
	// TODO: insertTestAuthor is not needed, I inserted the entry in
	// testdb.sql. However, that row has an empty passwd field, which should be
	// altered.
}

func testExistingAuthor(t *testing.T) {
	a, err := data.author("testuser")
	if err != nil {
		t.Fatalf("Failed to query author: %s", err.Error())
	}
	if a == nil {
		t.Fatal("Failed to query author: a == nil")
	}
	if a.FullName != "Joe Blogger" {
		t.Fatalf(`a.FullName != "Joe Blogger"`)
	}
}

func testPost(t *testing.T) {
	post := data.post("url")
	if post == nil {
		t.Fatalf("Failed to query post")
	}
	if post.Title != "title" {
		t.Errorf("Wrong title, expected %q, got %q", "title", post.Title)
	}
	post = data.post("non-existant")
	if post != nil {
		t.Fatalf("Should not find this post")
	}
	id, err := data.postID("url")
	if err != nil {
		t.Fatalf("Failed to query post ID")
	}
	if id != 1 {
		t.Errorf("Wrong post ID, expected %d, got %d", 1, id)
	}
}

func testInsertPost(t *testing.T) {
	data.begin()
	id, err := data.insertPost(1, &Entry{
		EntryLink: EntryLink{
			Title:  "title",
			URL:    "url",
			Hidden: false,
		},
		Author:  "me",
		Date:    "2014-12-28",
		RawBody: "*markdown*",
	})
	if err != nil || id != 1 {
		data.rollback()
		t.Fatalf("Failed to insert post, err = %s", err.Error())
	}
	data.commit()
}

func testUpdateTags(t *testing.T) {
	data.begin()
	tags := []*Tag{{Name: "tag1"}, {Name: "tag2"}}
	err := data.updateTags(tags, 1)
	if err != nil {
		data.rollback()
		t.Fatalf("Failed to update tags, err = %s", err.Error())
	}
	data.commit()
}

func testTags(t *testing.T) {
	tags := data.queryAllTags()
	if tags == nil {
		t.Fatalf("Failed to query tags")
	}
	if len(tags) != 2 {
		t.Fatalf("Wrong num tags, expected %d, got %d", 2, len(tags))
	}
	if tags[0].Name != "tag1" {
		t.Fatalf("Wrong tag, expected %q, got %q", "tag1", tags[0].Name)
	}
}

func testNumPosts(t *testing.T) {
	// Insert couple more posts
	data.begin()
	id, err := data.insertPost(1, &Entry{
		EntryLink: EntryLink{
			Title:  "title2",
			URL:    "url2",
			Hidden: false,
		},
		Author:  "me",
		Date:    "2014-12-30",
		RawBody: "*markdown 2*",
	})
	if err != nil || id != 2 {
		data.rollback()
		t.Fatalf("Failed to insert post, err = %s", err.Error())
	}
	id, err = data.insertPost(1, &Entry{
		EntryLink: EntryLink{
			Title:  "title3",
			URL:    "url3",
			Hidden: false,
		},
		Author:  "me",
		Date:    "2014-12-30",
		RawBody: "*markdown 3*",
	})
	if err != nil || id != 3 {
		data.rollback()
		t.Fatalf("Failed to insert post, err = %s", err.Error())
	}
	data.commit()

	// Now test a few methods
	numPosts := data.numPosts()
	if numPosts != 3 {
		t.Errorf("Wrong numPosts: expected %d, but got %d", 3, numPosts)
	}

	allPosts := data.posts(-1, 0)
	if len(allPosts) != 3 {
		t.Errorf("Wrong len(allPosts): expected %d, but got %d", 3, len(allPosts))
	}

	secondPost := data.posts(1, 1)
	if len(secondPost) != 1 {
		t.Errorf("Wrong len(secondPost): expected %d, but got %d", 1, len(secondPost))
	}

	titles := data.titles(-1)
	if len(titles) != 3 {
		t.Errorf("Wrong len(titles): expected %d, but got %d", 3, len(titles))
	}

	firstTitle := data.titles(1)
	if len(firstTitle) != 1 {
		t.Errorf("Wrong len(firstTitle): expected %d, but got %d", 1, len(firstTitle))
	}
	if firstTitle[0].Title != "title" {
		t.Errorf("Wrong firstTitle.Title: expected %q, but got %q", "title", firstTitle[0].Title)
	}
}

func TestDB(t *testing.T) {
	if realDB == nil {
		return
	}
	tempData := data
	data = realDB
	defer func() {
		data = tempData
	}()
	testExistingAuthor(t)
	testInsertPost(t)
	testPost(t)
	testUpdateTags(t)
	testTags(t)
	testNumPosts(t)
}
