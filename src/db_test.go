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
}
