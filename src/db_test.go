package main

import (
	"os"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/rtfb/bark"
)

var (
	realDB Data
	data   Data
)

func init() {
	conf = readConfigs("")
	config := "$RTFBLOG_DB_TEST_URL"
	envVar := os.ExpandEnv(config)
	if envVar == "" {
		return
	}
	if !strings.HasPrefix(envVar, "host=/tmp/PGSQL-") {
		return
	}
	conf.Server.DBConn = config
	logger = bark.CreateFile("tests.log")
	realDB = InitDB(getDBConnString(), Bindir())
}

func failIfErr(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatalf("%s, err = %s", msg, err.Error())
	}
}

func testInsertAuthor(t *testing.T) {
	passwd, err := EncryptBcrypt([]byte("testpasswd"))
	failIfErr(t, err, "Failed to encrypt passwd")
	err = data.begin()
	failIfErr(t, err, "Failed to start xaction")
	defer data.rollback()
	// XXX: panics when trying to insert a second copy. Investigate.
	id, err := data.insertAuthor(&Author{
		UserName: "testuser",
		Passwd:   passwd,
		FullName: "Joe Blogger",
		Email:    "joe@blogg.er",
		Www:      "http://test.blog",
	})
	if err != nil || id != 1 {
		t.Fatalf("Failed to insert author: %s", err.Error())
	}
	data.commit()
}

func testUpdateAuthor(t *testing.T) {
	data.begin()
	defer data.rollback()
	a, err := data.author()
	failIfErr(t, err, "Failed to query author")
	newName := "Zoe Vlogger"
	a.FullName = newName
	err = data.updateAuthor(a)
	failIfErr(t, err, "Failed to updateAuthor")
	data.commit()
	a, err = data.author()
	failIfErr(t, err, "Failed to query author")
	if a.FullName != newName {
		t.Fatalf("a.FullName = %q, expected %q", a.FullName, newName)
	}
}

func testDeleteAuthor(t *testing.T) {
	data.begin()
	defer data.rollback()
	err := data.deleteAuthor(1)
	failIfErr(t, err, "deleteAuthor failed")
	data.commit()
	_, err = data.author()
	if err != gorm.RecordNotFound {
		t.Fatalf("Unexpected error querying author: %s", err.Error())
	}
}

func testExistingAuthor(t *testing.T) {
	a, err := data.author()
	failIfErr(t, err, "Failed to query author")
	if a == nil {
		t.Fatal("Failed to query author: a == nil")
	}
	if a.FullName != "Zoe Vlogger" {
		t.Fatalf(`a.FullName != "Zoe Vlogger"`)
	}
}

func testPost(t *testing.T) {
	post, err := data.post("url", true)
	failIfErr(t, err, "Failed to query post")
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
	failIfErr(t, err, "Failed to query post ID")
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
	failIfErr(t, err, "Failed to query tags")
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
	failIfErr(t, err, "Failed to insert post")
	data.commit()

	// Now test a few methods
	numPosts, err := data.numPosts(true)
	failIfErr(t, err, "Failed to get numPosts")
	if numPosts != 3 {
		t.Errorf("Wrong numPosts: expected %d, but got %d", 3, numPosts)
	}

	allPosts, err := data.posts(-1, 0, true)
	failIfErr(t, err, "Failed to query posts")
	if len(allPosts) != 3 {
		t.Errorf("Wrong len(allPosts): expected %d, but got %d", 3, len(allPosts))
	}

	secondPost, err := data.posts(1, 1, true)
	failIfErr(t, err, "Failed to query posts")
	if len(secondPost) != 1 {
		t.Errorf("Wrong len(secondPost): expected %d, but got %d", 1, len(secondPost))
	}

	titles, err := data.titles(-1, true)
	failIfErr(t, err, "Failed to query titles")
	if len(titles) != 3 {
		t.Errorf("Wrong len(titles): expected %d, but got %d", 3, len(titles))
	}

	firstTitle, err := data.titles(1, true)
	failIfErr(t, err, "Failed to query titles")
	if len(firstTitle) != 1 {
		t.Errorf("Wrong len(firstTitle): expected %d, but got %d", 1, len(firstTitle))
	}
	if firstTitle[0].Title != "title" {
		t.Errorf("Wrong firstTitle.Title: expected %q, but got %q", "title", firstTitle[0].Title)
	}
}

func testTitlesByTag(t *testing.T) {
	titles, err := data.titlesByTag("tag1", true)
	failIfErr(t, err, "Failed to query titles")
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
		ID:       3,
		Date:     "2014-12-28",
		RawBody:  "*markdown*",
	})
	failIfErr(t, err, "Failed to updatePost")
	data.commit()
	post, err := data.post("url-three", true)
	failIfErr(t, err, "Failed to query post")
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
	failIfErr(t, err, "Failed to insert commenter")
	if commenterID != 1 {
		t.Fatalf("Wrong commenterID = %d, expected %d", commenterID, 1)
	}
	commentID, err := data.insertComment(commenterID, 1, "comment body")
	failIfErr(t, err, "Failed to insert comment")
	if commentID != 1 {
		t.Fatalf("Wrong commentID = %d, expected %d", commentID, 1)
	}
	data.commit()
}

func testQueryCommenterID(t *testing.T) {
	id, err := data.commenterID(&Commenter{
		Name:    "cname",
		Email:   "cemail",
		Website: "cwebsite",
	})
	failIfErr(t, err, "Error querying commenter ID")
	if id != 1 {
		t.Fatalf("Wrong commenter id = %d, expected %d", id, 1)
	}
}

func testQueryAllComments(t *testing.T) {
	comms, err := data.allComments()
	failIfErr(t, err, "Error querying comments")
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
	failIfErr(t, err, "updateComment failed")
	data.commit()
}

func testDeleteComment(t *testing.T) {
	data.begin()
	defer data.rollback()
	err := data.deleteComment("1")
	failIfErr(t, err, "deleteComment failed")
	data.commit()
	comms, err := data.allComments()
	failIfErr(t, err, "Error querying comments")
	if len(comms) != 0 {
		t.Fatalf("Wrong len(comms) = %d, expected %d", len(comms), 0)
	}
}

func testDeletePost(t *testing.T) {
	data.begin()
	defer data.rollback()
	err := data.deletePost("url-three")
	failIfErr(t, err, "deletePost failed")
	data.commit()
	posts, err := data.posts(-1, 0, true)
	failIfErr(t, err, "Failed to query posts")
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
	testInsertAuthor(t)
	testUpdateAuthor(t)
	testExistingAuthor(t)
	testInsertPost(t)
	testPost(t)
	testUpdateTags(t)
	testTags(t)
	testNumPosts(t)
	testTitlesByTag(t)
	testUpdatePost(t)
	testInsertComment(t)
	testQueryCommenterID(t)
	testQueryAllComments(t)
	testUpdateComment(t)
	testDeleteComment(t)
	testDeletePost(t)
	testDeleteAuthor(t)
}
