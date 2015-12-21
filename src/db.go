package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/jinzhu/gorm"
)

type Data interface {
	post(url string, includeHidden bool) (*Entry, error)
	postID(url string) (id int64, err error)
	posts(limit, offset int, includeHidden bool) ([]*Entry, error)
	titles(limit int, includeHidden bool) ([]EntryLink, error)
	titlesByTag(tag string, includeHidden bool) ([]EntryLink, error)
	allComments() ([]*CommentWithPostTitle, error)
	numPosts(includeHidden bool) (int, error)
	author() (*Author, error)
	insertAuthor(a *Author) (id int64, err error)
	updateAuthor(a *Author) error
	deleteAuthor(id int64) error
	deleteComment(id string) error
	deletePost(url string) error
	updateComment(id, text string) error
	commenterID(c *Commenter) (id int64, err error)
	insertCommenter(c *Commenter) (id int64, err error)
	insertComment(commenterID, postID int64, body string) (id int64, err error)
	insertPost(e *EntryTable) (id int64, err error)
	updatePost(e *EntryTable) error
	updateTags(tags []*Tag, postID int64) error
	queryAllTags() ([]*Tag, error)
	begin() error
	commit()
	rollback()
}

type DbData struct {
	db *gorm.DB
	tx *gorm.DB
}

func InitDB(conn string) *DbData {
	dialect := "postgres"
	if conn == "" {
		dialect = "sqlite3"
		conn = "default.db"
	}
	logDbConn(dialect, conn)
	db, err := gorm.Open(dialect, conn)
	err = db.DB().Ping()
	if err != nil {
		panic(err)
	}
	db.SingularTable(true)
	return &DbData{
		db: &db,
		tx: nil,
	}
}

func logDbConn(dialect, conn string) {
	if dialect == "postgres" {
		conn = CensorPostgresConnStr(conn)
	}
	logger.Printf("Connecting to %q DB via conn %q\n", dialect, conn)
}

func notInXactionErr() error {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("runtime.Caller(1) != ok, dafuq?")
	}
	funcName := runtime.FuncForPC(pc).Name()
	msg := "Error! %s() can only be called within transaction!"
	return fmt.Errorf(msg, funcName)
}

func (dd *DbData) begin() error {
	dd.tx = dd.db.Begin()
	return dd.tx.Error
}

func (dd *DbData) commit() {
	dd.tx.Commit()
	logger.LogIf(dd.db.Error)
	dd.tx = nil
}

func (dd *DbData) rollback() {
	if dd.tx == nil {
		return
	}
	dd.tx.Rollback()
	logger.LogIf(dd.db.Error)
	dd.tx = nil
}

func (dd *DbData) post(url string, includeHidden bool) (*Entry, error) {
	posts, err := queryPosts(dd, -1, -1, url, includeHidden)
	if err != nil {
		return nil, err
	}
	if len(posts) != 1 {
		msg := "DbData.post(%q) should return 1 post, but returned %d"
		return nil, fmt.Errorf(msg, url, len(posts))
	}
	return posts[0], nil
}

func (dd *DbData) postID(url string) (int64, error) {
	var post Entry
	rows := dd.db.Table("post").Where("url = ?", url).Select("id")
	err := rows.First(&post).Error
	return post.ID, err
}

func (dd *DbData) posts(limit, offset int, includeHidden bool) ([]*Entry, error) {
	return queryPosts(dd, limit, offset, "", includeHidden)
}

func (dd *DbData) numPosts(includeHidden bool) (int, error) {
	var count int
	if includeHidden {
		dd.db.Table("post").Count(&count)
	} else {
		dd.db.Table("post").Where("hidden=?", false).Count(&count)
	}
	return count, dd.db.Error
}

func (dd *DbData) titles(limit int, includeHidden bool) ([]EntryLink, error) {
	var results []EntryLink
	posts := dd.db.Table("post").Select("title, url, hidden")
	if !includeHidden {
		posts = posts.Where("hidden=?", false)
	}
	err := posts.Order("date desc").Limit(limit).Scan(&results).Error
	return results, err
}

func (dd *DbData) titlesByTag(tag string, includeHidden bool) ([]EntryLink, error) {
	var postIDs []int64
	var results []EntryLink
	join := "inner join tag on tagmap.tag_id = tag.id"
	rows := dd.db.Table("tagmap").Joins(join).Where("tag.tag=?", tag)
	err := rows.Pluck("post_id", &postIDs).Error
	if err != nil {
		return nil, err
	}
	columns := "title, url, hidden"
	posts := dd.db.Table("post").Select(columns).Where("id in (?)", postIDs)
	if !includeHidden {
		posts = posts.Where("hidden=?", false)
	}
	err = posts.Order("date desc").Scan(&results).Error
	return results, err
}

func (dd *DbData) allComments() ([]*CommentWithPostTitle, error) {
	var results []*CommentWithPostTitle
	sel := `commenter.name, commenter.email, commenter.www, commenter.ip,
		comment.id, comment.timestamp, comment.body,
		post.title, post.url`
	join := `right join comment on commenter.id = comment.commenter_id
		inner join post on comment.post_id = post.id`
	joined := dd.db.Table("commenter").Select(sel).Joins(join)
	err := joined.Order("comment.timestamp desc").Scan(&results).Error
	// TODO: there's an identical loop in queryComments, but it loops over
	// []Comment instead of []CommentWithPostTitle. Would be nice to unify.
	for _, c := range results {
		c.EmailHash = Md5Hash(c.Email)
		c.Time = time.Unix(c.Timestamp, 0).Format("2006-01-02 15:04")
		c.Body = sanitizeHTML(mdToHTML(c.RawBody))
	}
	return results, err
}

func (dd *DbData) commenterID(c *Commenter) (id int64, err error) {
	var result CommenterTable
	where := "name = ? and email = ? and www = ?"
	rows := dd.db.Table("commenter").Select("id")
	err = rows.Where(where, c.Name, c.Email, c.Website).Scan(&result).Error
	id = result.ID
	return
}

func (dd *DbData) insertCommenter(c *Commenter) (id int64, err error) {
	if dd.tx == nil {
		return -1, notInXactionErr()
	}
	entry := CommenterTable{ID: 0, Commenter: *c}
	err = dd.tx.Save(&entry).Error
	return entry.ID, err
}

func (dd *DbData) insertComment(commenterID, postID int64, body string) (id int64, err error) {
	if dd.tx == nil {
		return -1, notInXactionErr()
	}
	c := CommentTable{
		CommenterID: commenterID,
		PostID:      postID,
		RawBody:     body,
		Timestamp:   time.Now().Unix(),
	}
	err = dd.tx.Save(&c).Error
	return c.CommentID, err
}

func (dd *DbData) insertPost(e *EntryTable) (id int64, err error) {
	if dd.tx == nil {
		return -1, notInXactionErr()
	}
	e.UnixDate = time.Now().Unix()
	err = dd.tx.Save(e).Error
	return e.ID, err
}

func (dd *DbData) updatePost(e *EntryTable) error {
	if dd.tx == nil {
		return notInXactionErr()
	}
	return dd.tx.Save(e).Error
}

func (dd *DbData) updateTags(tags []*Tag, postID int64) error {
	if dd.tx == nil {
		return notInXactionErr()
	}
	dd.tx.Where("post_id = ?", postID).Delete(TagMap{})
	for _, t := range tags {
		tagID, err := insertOrGetTagID(dd.tx, t)
		if err != nil {
			return err
		}
		err = updateTagMap(dd.tx, postID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dd *DbData) author() (*Author, error) {
	var a Author
	err := dd.db.First(&a).Error
	return &a, err
}

func (dd *DbData) insertAuthor(a *Author) (id int64, err error) {
	if dd.tx == nil {
		return -1, notInXactionErr()
	}
	err = dd.tx.Save(a).Error
	return a.ID, err
}

func (dd *DbData) updateAuthor(a *Author) error {
	if dd.tx == nil {
		return notInXactionErr()
	}
	return dd.tx.Save(a).Error
}

func (dd *DbData) deleteAuthor(id int64) error {
	return dd.db.Where("id=?", id).Delete(Author{}).Error
}

func (dd *DbData) deleteComment(id string) error {
	return dd.db.Where("id=?", id).Delete(CommentTable{}).Error
}

func (dd *DbData) deletePost(url string) error {
	return dd.db.Where("url=?", url).Delete(Entry{}).Error
}

func (dd *DbData) updateComment(id, text string) error {
	return dd.db.Model(CommentTable{}).Where("id=?", id).Update("body", text).Error
}

func queryPosts(dd *DbData, limit, offset int, url string,
	includeHidden bool) ([]*Entry, error) {
	var results []*Entry
	cols := `author.disp_name, post.id, post.title, post.date, post.body,
		post.url, post.hidden`
	join := "inner join author on post.author_id=author.id"
	posts := dd.db.Table("post").Select(cols).Joins(join)
	if !includeHidden {
		posts = posts.Where("hidden=?", false)
	}
	if url != "" {
		posts = posts.Where("url=?", url)
	}
	rows := posts.Order("date desc").Limit(limit).Offset(offset)
	err := rows.Scan(&results).Error
	for _, p := range results {
		p.Body = sanitizeTrustedHTML(mdToHTML(p.RawBody))
		p.Date = time.Unix(p.UnixDate, 0).Format("2006-01-02")
		p.Tags = queryTags(dd.db, p.ID)
		p.Comments = queryComments(dd.db, p.ID)
	}
	return results, err
}

func queryTags(db *gorm.DB, postID int64) []*Tag {
	var results []*Tag
	join := "inner join tagmap on tagmap.tag_id = tag.id"
	tables := db.Table("tag").Select("tag.tag").Joins(join)
	tables.Where("tagmap.post_id = ?", postID).Scan(&results)
	logger.LogIff(db.Error, "error querying tags for post %d", postID)
	return results
}

func (dd *DbData) queryAllTags() ([]*Tag, error) {
	var tags []*Tag
	err := dd.db.Find(&tags).Error
	return tags, err
}

func queryComments(db *gorm.DB, postID int64) []*Comment {
	var comments []*Comment
	join := "inner join commenter on comment.commenter_id = commenter.id"
	order := "timestamp asc"
	tables := db.Table("comment").Select("*").Joins(join)
	rows := tables.Where("post_id = ?", postID).Order(order)
	err := rows.Scan(&comments).Error
	logger.LogIff(err, "error querying comments")
	for _, c := range comments {
		c.EmailHash = Md5Hash(c.Email)
		c.Time = time.Unix(c.Timestamp, 0).Format("2006-01-02 15:04")
		c.Body = sanitizeHTML(mdToHTML(c.RawBody))
	}
	return comments
}

func insertOrGetTagID(db *gorm.DB, tag *Tag) (tagID int64, err error) {
	var result Tag
	err = db.Where("tag = ?", tag.Name).First(&result).Error
	switch err {
	case nil:
		return result.ID, nil
	case gorm.RecordNotFound:
		err = db.Save(tag).Error
		return tag.ID, err
	default:
		logger.Log(err)
		return -1, err
	}
}

func updateTagMap(db *gorm.DB, postID int64, tagID int64) error {
	return db.Save(&TagMap{TagID: tagID, EntryID: postID}).Error
}
