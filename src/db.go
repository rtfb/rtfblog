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
	titles(limit int) []*EntryLink
	titlesByTag(tag string) []*EntryLink
	allComments() []*CommentWithPostTitle
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
	queryAllTags() []*Tag
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

func (dd *DbData) postID(url string) (id int64, err error) {
	query, err := dd.db.Prepare("select id from post where url = $1")
	if err != nil {
		return
	}
	defer query.Close()
	err = query.QueryRow(url).Scan(&id)
	return
}

func (dd *DbData) posts(limit, offset int) []*Entry {
	return loadPosts(dd, limit, offset, "", dd.includeHidden)
}

func (dd *DbData) numPosts() int {
	selectSql := "select count(*) from post as p"
	if !dd.includeHidden {
		selectSql = selectSql + " where p.hidden=FALSE"
	}
	rows, err := dd.db.Query(selectSql)
	if err != nil {
		logger.Log(err)
		return 0
	}
	defer rows.Close()
	num := 0
	if rows.Next() {
		rows.Scan(&num)
	}
	return num
}

func (dd *DbData) titles(limit int) (links []*EntryLink) {
	selectSql := `select p.title, p.url, p.hidden
                  from post as p`
	if !dd.includeHidden {
		selectSql = selectSql + " where p.hidden=FALSE"
	}
	selectSql = selectSql + " order by p.date desc"
	if limit > 0 {
		selectSql = selectSql + " limit $1"
	}
	stmt, err := dd.db.Prepare(selectSql)
	if err != nil {
		logger.Log(err)
		return
	}
	defer stmt.Close()
	var rows *sql.Rows
	if limit > 0 {
		rows, err = stmt.Query(limit)
	} else {
		rows, err = stmt.Query()
	}
	if err != nil {
		logger.Log(err)
		return
	}
	defer rows.Close()
	return scanEntryLinks(rows)
}

func (dd *DbData) titlesByTag(tag string) (links []*EntryLink) {
	selectSql := `select p.title, p.url, p.hidden
                  from post as p
                  where p.id in (select tm.post_id from tagmap as tm
                                 inner join tag as t
                                 on tm.tag_id = t.id and t.tag=$1)`
	if !dd.includeHidden {
		selectSql = selectSql + " and p.hidden=FALSE"
	}
	selectSql = selectSql + " order by p.date desc"
	stmt, err := dd.db.Prepare(selectSql)
	if err != nil {
		logger.Log(err)
		return
	}
	defer stmt.Close()
	rows, err := stmt.Query(tag)
	if err != nil {
		logger.Log(err)
		return
	}
	defer rows.Close()
	return scanEntryLinks(rows)
}

func scanEntryLinks(rows *sql.Rows) (links []*EntryLink) {
	for rows.Next() {
		entryLink := new(EntryLink)
		err := rows.Scan(&entryLink.Title, &entryLink.URL, &entryLink.Hidden)
		if err != nil {
			logger.Log(err)
			continue
		}
		links = append(links, entryLink)
	}
	err := rows.Err()
	logger.LogIf(err)
	return
}

func (dd *DbData) allComments() []*CommentWithPostTitle {
	stmt, err := dd.db.Prepare(`select a.name, a.email, a.www, a.ip,
                                       c.id, c.timestamp, c.body,
                                       p.title, p.url
                                from commenter as a, comment as c, post as p
                                where a.id = c.commenter_id
                                      and c.post_id = p.id
                                order by c.timestamp desc`)
	if err != nil {
		logger.Log(err)
		return nil
	}
	defer stmt.Close()
	data, err := stmt.Query()
	if err != nil {
		logger.Log(err)
		return nil
	}
	defer data.Close()
	var comments []*CommentWithPostTitle
	for data.Next() {
		comment := new(CommentWithPostTitle)
		var unixDate int64
		err = data.Scan(&comment.Name, &comment.Email, &comment.Website, &comment.IP,
			&comment.CommentID, &unixDate, &comment.RawBody,
			&comment.Title, &comment.URL)
		logger.LogIff(err, "error scanning comment row")
		comment.EmailHash = Md5Hash(comment.Email)
		comment.Time = time.Unix(unixDate, 0).Format("2006-01-02 15:04")
		comment.Body = sanitizeHTML(mdToHTML(comment.RawBody))
		comments = append(comments, comment)
	}
	logger.LogIff(data.Err(), "error scanning comment row")
	return comments
}

func (dd *DbData) commenterID(c Commenter) (id int64, err error) {
	where := "name = ? and email = ? and www = ?"
	err = dd.gormDB.Select("id").Where(where, c.Name, c.Email, c.Website).Scan(&id).Error
	return
}

func (dd *DbData) insertCommenter(c Commenter) (id int64, err error) {
	err = dd.gormDB.Save(&c).Error
	return c.Id, err
}

func (dd *DbData) insertComment(commenterID, postID int64, body string) (id int64, err error) {
	if dd.tx == nil {
		return -1, notInXactionErr()
	}
	stmt, err := dd.tx.Prepare(`insert into comment
                                (commenter_id, post_id, timestamp, body)
                                values ($1, $2, $3, $4)
                                returning id`)
	if err != nil {
		logger.LogIff(err, "Failed to prepare insert comment stmt")
		return
	}
	defer stmt.Close()
	err = stmt.QueryRow(commenterID, postID, time.Now().Unix(), body).Scan(&id)
	if err != nil {
		logger.LogIff(err, "Failed to insert comment")
		return
	}
	return
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
	_, err := dd.db.Exec("delete from comment where id=$1", id)
	if err != nil {
		return err
	}
	return nil
}

func (dd *DbData) deletePost(url string) error {
	_, err := dd.db.Exec("delete from post where url=$1", url)
	if err != nil {
		return err
	}
	return nil
}

func (dd *DbData) updateComment(id, text string) error {
	_, err := dd.db.Exec("update comment set body=$1 where id=$2", text, id)
	if err != nil {
		return err
	}
	return nil
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

func queryPosts(dd *DbData, limit, offset int,
	url string, includeHidden bool) (entries []*Entry, err error) {
	postURLWhereClause := ""
	if url != "" {
		postURLWhereClause = fmt.Sprintf("and p.url='%s'", url)
	}
	postHiddenWhereClause := ""
	if !includeHidden {
		postHiddenWhereClause = "and p.hidden=FALSE"
	}
	limitClause := ""
	if limit >= 0 {
		limitClause = fmt.Sprintf("limit %d", limit)
	}
	offsetClause := ""
	if offset > 0 {
		offsetClause = fmt.Sprintf("offset %d", offset)
	}
	queryFmt := `select a.disp_name, p.id, p.title, p.date, p.body,
                        p.url, p.hidden
                 from author as a, post as p
                 where a.id=p.author_id
                 %s %s
                 order by p.date desc
                 %s %s`
	query := fmt.Sprintf(queryFmt, postURLWhereClause, postHiddenWhereClause,
		limitClause, offsetClause)
	rows, err := dd.db.Query(query)
	if err != nil {
		logger.Log(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		entry := new(Entry)
		var id int64
		var unixDate int64
		err = rows.Scan(&entry.Author, &id, &entry.Title, &unixDate,
			&entry.RawBody, &entry.URL, &entry.Hidden)
		if err != nil {
			logger.Log(err)
			continue
		}
		entry.Body = sanitizeTrustedHTML(mdToHTML(entry.RawBody))
		entry.Date = time.Unix(unixDate, 0).Format("2006-01-02")
		entry.Tags = queryTags(dd.gormDB, id)
		entry.Comments = queryComments(dd.gormDB, id)
		entries = append(entries, entry)
	}
	err = rows.Err()
	logger.LogIff(err, "error scanning post row")
	return
}

func queryTags(db *gorm.DB, postID int64) []*Tag {
	var results []*Tag
	join := "inner join tagmap on tagmap.tag_id = tag.id"
	tables := db.Table("tag").Select("tag.tag").Joins(join)
	tables.Where("tagmap.post_id = ?", postID).Scan(&results)
	return results // TODO: err
}

func (dd *DbData) queryAllTags() []*Tag {
	stmt, err := dd.db.Prepare(`select tag from tag`)
	if err != nil {
		logger.Log(err)
		return nil
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		logger.Log(err)
		return nil
	}
	defer rows.Close()
	var tags []*Tag
	for rows.Next() {
		tag := new(Tag)
		err = rows.Scan(&tag.Name)
		if err != nil {
			logger.Log(err)
			continue
		}
		tags = append(tags, tag)
	}
	err = rows.Err()
	logger.LogIff(err, "error scanning tag row")
	return tags
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
