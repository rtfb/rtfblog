package main

import (
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

var (
	realDB Data
	data   Data
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
	realDB = InitDB(getDBConnString())
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
	post, err := data.post("url", true)
	if err != nil {
		t.Fatalf("Failed to query post, err = %s", err.Error())
	}
	if post == nil {
		t.Fatalf("Failed to query post")
	}
	if post.Title != "title" {
		t.Errorf("Wrong title, expected %q, got %q", "title", post.Title)
	}
	post, err = data.post("non-existant", true)
	if err == nil {
		t.Fatalf("Expected to fail querying non-existant post, but err == nil")
	}
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
	defer data.rollback()
	id, err := data.insertPost(&EntryTable{
		EntryLink: EntryLink{
			Title:  "title",
			URL:    "url",
			Hidden: false,
		},
		AuthorID: 1,
		Date:     "2014-12-28",
		RawBody:  "*markdown*",
	})
	if err != nil || id != 1 {
		t.Fatalf("Failed to insert post, err = %s", err.Error())
	}
	data.commit()
}

func testUpdateTags(t *testing.T) {
	data.begin()
	defer data.rollback()
	tags := []*Tag{{Name: "tag1"}, {Name: "tag2"}}
	err := data.updateTags(tags, 1)
	if err != nil {
		t.Fatalf("Failed to update tags, err = %s", err.Error())
	}
	data.commit()
}

func testTags(t *testing.T) {
	tags, err := data.queryAllTags()
	if err != nil {
		t.Fatalf("Failed to query tags")
	}
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
	defer data.rollback()
	id, err := data.insertPost(&EntryTable{
		EntryLink: EntryLink{
			Title:  "title2",
			URL:    "url2",
			Hidden: false,
		},
		AuthorID: 1,
		Date:     "2014-12-30",
		RawBody:  "*markdown 2*",
	})
	if err != nil || id != 2 {
		t.Fatalf("Failed to insert post, err = %s", err.Error())
	}
	id, err = data.insertPost(&EntryTable{
		EntryLink: EntryLink{
			Title:  "title3",
			URL:    "url3",
			Hidden: false,
		},
		AuthorID: 1,
		Date:     "2014-12-30",
		RawBody:  "*markdown 3*",
	})
	if err != nil || id != 3 {
		t.Fatalf("Failed to insert post, err = %s", err.Error())
	}
	data.commit()

	// Now test a few methods
	numPosts, err := data.numPosts(true)
	if err != nil {
		t.Fatalf("Failed to get numPosts, err = %s", err.Error())
	}
	if numPosts != 3 {
		t.Errorf("Wrong numPosts: expected %d, but got %d", 3, numPosts)
	}

	allPosts, err := data.posts(-1, 0, true)
	if err != nil {
		t.Fatalf("Failed to query posts, err = %s", err.Error())
	}
	if len(allPosts) != 3 {
		t.Errorf("Wrong len(allPosts): expected %d, but got %d", 3, len(allPosts))
	}

	secondPost, err := data.posts(1, 1, true)
	if err != nil {
		t.Fatalf("Failed to query posts, err = %s", err.Error())
	}
	if len(secondPost) != 1 {
		t.Errorf("Wrong len(secondPost): expected %d, but got %d", 1, len(secondPost))
	}

	titles, err := data.titles(-1, true)
	if err != nil {
		t.Fatalf("Failed to query titles, err = %s", err.Error())
	}
	if len(titles) != 3 {
		t.Errorf("Wrong len(titles): expected %d, but got %d", 3, len(titles))
	}

	firstTitle, err := data.titles(1, true)
	if err != nil {
		t.Fatalf("Failed to query titles, err = %s", err.Error())
	}
	if len(firstTitle) != 1 {
		t.Errorf("Wrong len(firstTitle): expected %d, but got %d", 1, len(firstTitle))
	}
	if firstTitle[0].Title != "title" {
		t.Errorf("Wrong firstTitle.Title: expected %q, but got %q", "title", firstTitle[0].Title)
	}
}

func testTitlesByTag(t *testing.T) {
	titles, err := data.titlesByTag("tag1", true)
	if err != nil {
		t.Fatalf("Failed to query titles, err = %s", err.Error())
	}
	if len(titles) != 1 {
		t.Fatalf("Wrong len(titles), expected %d, but got %d", 1, len(titles))
	}
	if titles[0].Title != "title" {
		t.Fatalf("titles[0].Title != %q, got %q", "title", titles[0].Title)
	}
}

func testUpdatePost(t *testing.T) {
	data.begin()
	defer data.rollback()
	err := data.updatePost(&EntryTable{
		EntryLink: EntryLink{
			Title:  "title three",
			URL:    "url-three",
			Hidden: false,
		},
		AuthorID: 1,
		Id:       3,
		Date:     "2014-12-28",
		RawBody:  "*markdown*",
	})
	if err != nil {
		t.Fatalf("Failed to updatePost: " + err.Error())
	}
	data.commit()
	post, err := data.post("url-three", true)
	if err != nil {
		t.Fatalf("Failed to query post, err = %s", err.Error())
	}
	if post == nil {
		t.Fatalf("Failed to query post")
	}
	if post.Title != "title three" {
		t.Errorf("Wrong title, expected %q, got %q", "title three", post.Title)
	}
}

func testInsertComment(t *testing.T) {
	data.begin()
	defer data.rollback()
	commenterID, err := data.insertCommenter(&Commenter{
		Name:    "cname",
		Email:   "cemail",
		Website: "cwebsite",
		IP:      "cip",
	})
	if err != nil {
		t.Fatalf("Failed to insert commenter: " + err.Error())
	}
	if commenterID != 1 {
		t.Fatalf("Wrong commenterID = %d, expected %d", commenterID, 1)
	}
	commentID, err := data.insertComment(commenterID, 1, "comment body")
	if err != nil {
		t.Fatalf("Failed to insert comment: " + err.Error())
	}
	if commentID != 1 {
		t.Fatalf("Wrong commentID = %d, expected %d", commentID, 1)
	}
	data.commit()
}

func testQueryAllComments(t *testing.T) {
	comms, err := data.allComments()
	if err != nil {
		t.Fatalf("Error querying comments: %s", err.Error())
	}
	if len(comms) != 1 {
		t.Fatalf("Wrong len(comms) = %d, expected %d", len(comms), 1)
	}
	if comms[0].RawBody != "comment body" {
		t.Fatalf("Wrong comms[0].RawBody = %q, expected %q", comms[0].RawBody, "comment body")
	}
	if comms[0].Title != "title" {
		t.Fatalf("Wrong comms[0].Title = %q, expected %q", comms[0].Title, "title")
	}
}

func testUpdateComment(t *testing.T) {
	data.begin()
	defer data.rollback()
	err := data.updateComment("1", "new body")
	if err != nil {
		t.Fatalf("updateComment failed: " + err.Error())
	}
	data.commit()
}

func testDeleteComment(t *testing.T) {
	data.begin()
	defer data.rollback()
	err := data.deleteComment("1")
	if err != nil {
		t.Fatalf("deleteComment failed: " + err.Error())
	}
	data.commit()
	comms, err := data.allComments()
	if err != nil {
		t.Fatalf("Error querying comments: %s", err.Error())
	}
	if len(comms) != 0 {
		t.Fatalf("Wrong len(comms) = %d, expected %d", len(comms), 0)
	}
}

func testDeletePost(t *testing.T) {
	data.begin()
	defer data.rollback()
	err := data.deletePost("url-three")
	if err != nil {
		t.Fatalf("deletePost failed: " + err.Error())
	}
	data.commit()
	posts, err := data.posts(-1, 0, true)
	if err != nil {
		t.Fatalf("Failed to query posts, err = %s", err.Error())
	}
	if len(posts) != 2 {
		t.Fatalf("Wrong len(posts) = %d, expected %d", len(posts), 2)
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
	testTitlesByTag(t)
	testUpdatePost(t)
	testInsertComment(t)
	testQueryAllComments(t)
	testUpdateComment(t)
	testDeleteComment(t)
	testDeletePost(t)
}
