package main

import (
	"database/sql"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/jinzhu/gorm"
)

type Data interface {
	hiddenPosts(flag bool)
	post(url string) *Entry
	postID(url string) (id int64, err error)
	posts(limit, offset int) []*Entry
	titles(limit int) ([]EntryLink, error)
	titlesByTag(tag string) ([]EntryLink, error)
	allComments() ([]*CommentWithPostTitle, error)
	numPosts() int
	author(username string) (*Author, error)
	deleteComment(id string) error
	deletePost(url string) error
	updateComment(id, text string) error
	commenterID(c Commenter) (id int64, err error)
	insertCommenter(c Commenter) (id int64, err error)
	insertComment(commenterID, postID int64, body string) (id int64, err error)
	insertPost(author int64, e *Entry) (id int64, err error)
	updatePost(id int64, e *Entry) error
	updateTags(tags []*Tag, postID int64) error
	queryAllTags() ([]*Tag, error)
	begin() error
	commit()
	rollback()
}

type DbData struct {
	gormDB        *gorm.DB
	db            *sql.DB
	tx            *sql.Tx
	includeHidden bool
}

func (dd *DbData) hiddenPosts(flag bool) {
	dd.includeHidden = flag
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
	if dd.tx != nil {
		return errors.New("Error! DbData.begin() called within transaction!")
	}
	xaction, err := dd.db.Begin()
	if err != nil {
		return err
	}
	dd.tx = xaction
	return nil
}

func (dd *DbData) commit() {
	if dd.tx == nil {
		logger.Log(notInXactionErr())
		return
	}
	dd.tx.Commit()
	dd.tx = nil
}

func (dd *DbData) rollback() {
	if dd.tx == nil {
		logger.Log(notInXactionErr())
		return
	}
	dd.tx.Rollback()
	dd.tx = nil
}

func (dd *DbData) post(url string) *Entry {
	posts := loadPosts(dd, -1, -1, url, dd.includeHidden)
	if len(posts) != 1 {
		msg := "Error! DbData.post(%q) should return 1 post, but returned %d\n"
		logger.Println(fmt.Sprintf(msg, url, len(posts)))
		return nil
	}
	return posts[0]
}

func (dd *DbData) postID(url string) (int64, error) {
	var post Entry
	rows := dd.gormDB.Table("post").Where("url = ?", url).Select("id")
	err := rows.First(&post).Error
	return post.Id, err
}

func (dd *DbData) posts(limit, offset int) []*Entry {
	return loadPosts(dd, limit, offset, "", dd.includeHidden)
}

func (dd *DbData) numPosts() int {
	var count int
	if dd.includeHidden {
		dd.gormDB.Table("post").Count(&count)
	} else {
		dd.gormDB.Table("post").Where("hidden=?", false).Count(&count)
	}
	return count
}

func (dd *DbData) titles(limit int) ([]EntryLink, error) {
	var results []EntryLink
	posts := dd.gormDB.Table("post").Select("title, url, hidden")
	posts = posts.Where("hidden=?", false)
	err := posts.Order("date desc").Limit(limit).Scan(&results).Error
	return results, err
}

func (dd *DbData) titlesByTag(tag string) ([]EntryLink, error) {
	var postIDs []int64
	var results []EntryLink
	join := "inner join tag on tagmap.tag_id = tag.id"
	rows := dd.gormDB.Table("tagmap").Joins(join).Where("tag.tag=?", tag)
	err := rows.Pluck("post_id", &postIDs).Error
	if err != nil {
		return nil, err
	}
	columns := "title, url, hidden"
	posts := dd.gormDB.Table("post").Select(columns).Where("id in (?)", postIDs)
	posts = posts.Where("hidden=?", false)
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
	joined := dd.gormDB.Table("commenter").Select(sel).Joins(join)
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

func (dd *DbData) commenterID(c Commenter) (id int64, err error) {
	where := "name = ? and email = ? and www = ?"
	err = dd.gormDB.Select("id").Where(where, c.Name, c.Email, c.Website).Scan(&id).Error
	return
}

func (dd *DbData) insertCommenter(c Commenter) (id int64, err error) {
	entry := CommenterTable{Id: 0, Commenter: c}
	err = dd.gormDB.Save(&entry).Error
	return entry.Id, err
}

func (dd *DbData) insertComment(commenterID, postID int64, body string) (id int64, err error) {
	c := CommentTable{
		CommenterID: commenterID,
		PostID:      postID,
		RawBody:     body,
		Timestamp:   time.Now().Unix(),
	}
	err = dd.gormDB.Save(&c).Error
	return c.CommentID, err
}

func (dd *DbData) insertPost(author int64, e *Entry) (id int64, err error) {
	if dd.tx == nil {
		return -1, notInXactionErr()
	}
	insertPostSql, _ := dd.tx.Prepare(`insert into post
                                       (author_id, title, date, url, body, hidden)
                                       values ($1, $2, $3, $4, $5, $6)
                                       returning id`)
	defer insertPostSql.Close()
	date := time.Now().Unix()
	err = insertPostSql.QueryRow(author, e.Title, date, e.URL,
		string(e.Body), e.Hidden).Scan(&id)
	if err != nil {
		logger.LogIff(err, "Failed to insert post")
		return
	}
	return
}

func (dd *DbData) updatePost(id int64, e *Entry) error {
	updateStmt, _ := dd.tx.Prepare(`update post
                                    set title=$1, url=$2, body=$3, hidden=$4
                                    where id=$5`)
	defer updateStmt.Close()
	_, err := updateStmt.Exec(e.Title, e.URL, string(e.Body), e.Hidden, id)
	if err != nil {
		return err
	}
	return nil
}

func (dd *DbData) updateTags(tags []*Tag, postID int64) error {
	if dd.tx == nil {
		return notInXactionErr()
	}
	dd.gormDB.Where("post_id = ?", postID).Delete(TagMap{})
	for _, t := range tags {
		tagID, err := insertOrGetTagID(dd.gormDB, t)
		if err != nil {
			return err
		}
		err = updateTagMap(dd.gormDB, postID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dd *DbData) author(username string) (*Author, error) {
	var a Author
	err := dd.gormDB.Where("disp_name = ?", username).First(&a).Error
	return &a, err
}

func (dd *DbData) deleteComment(id string) error {
	return dd.gormDB.Where("id=?", id).Delete(CommentTable{}).Error
}

func (dd *DbData) deletePost(url string) error {
	return dd.gormDB.Where("url=?", url).Delete(Entry{}).Error
}

func (dd *DbData) updateComment(id, text string) error {
	return dd.gormDB.Model(CommentTable{}).Where("id=?", id).Update("body", text).Error
}

func loadPosts(dd *DbData, limit, offset int, url string, includeHidden bool) []*Entry {
	if dd.db == nil {
		return nil
	}
	data, err := queryPosts(dd, limit, offset, url, includeHidden)
	if err != nil {
		logger.Log(err)
		return nil
	}
	return data
}

func queryPosts(dd *DbData, limit, offset int, url string,
	includeHidden bool) ([]*Entry, error) {
	var results []*Entry
	cols := `author.disp_name, post.id, post.title, post.date, post.body,
		post.url, post.hidden`
	join := "inner join author on post.author_id=author.id"
	posts := dd.gormDB.Table("post").Select(cols).Joins(join)
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
		p.Tags = queryTags(dd.gormDB, p.Id)
		p.Comments = queryComments(dd.gormDB, p.Id)
	}
	return results, err
}

func queryTags(db *gorm.DB, postID int64) []*Tag {
	var results []*Tag
	join := "inner join tagmap on tagmap.tag_id = tag.id"
	tables := db.Table("tag").Select("tag.tag").Joins(join)
	tables.Where("tagmap.post_id = ?", postID).Scan(&results)
	return results // TODO: err
}

func (dd *DbData) queryAllTags() ([]*Tag, error) {
	var tags []*Tag
	err := dd.gormDB.Find(&tags).Error
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
		return result.Id, nil
	case gorm.RecordNotFound:
		err = db.Save(tag).Error
		return tag.Id, err
	default:
		logger.Log(err)
		return -1, err
	}
}

func updateTagMap(db *gorm.DB, postID int64, tagID int64) error {
	return db.Save(&TagMap{TagID: tagID, EntryID: postID}).Error
}
