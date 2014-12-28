package main

import (
	"database/sql"
	"fmt"
	"time"
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
	deleteComment(id string) bool
	deletePost(url string) bool
	updateComment(id, text string) bool
	commenter(c Commenter) (id int64, err error)
	insertCommenter(c Commenter) (id int64, err error)
	insertComment(commenterID, postID int64, body string) (id int64, err error)
	insertPost(author int64, e *Entry) (id int64, err error)
	updatePost(id int64, e *Entry) bool
	updateTags(tags []*Tag, postID int64) error
	queryAllTags() []*Tag
	begin() bool
	commit()
	rollback()
}

type DbData struct {
	db            *sql.DB
	tx            *sql.Tx
	includeHidden bool
}

func (dd *DbData) hiddenPosts(flag bool) {
	dd.includeHidden = flag
}

func (dd *DbData) begin() bool {
	if dd.tx != nil {
		logger.Println("Error! DbData.begin() called within transaction!")
		return false
	}
	xaction, err := dd.db.Begin()
	if err != nil {
		logger.Println(err.Error())
		return false
	}
	dd.tx = xaction
	return true
}

func (dd *DbData) commit() {
	if dd.tx == nil {
		logger.Println("Error! DbData.commit() called outside of transaction!")
		return
	}
	dd.tx.Commit()
	dd.tx = nil
}

func (dd *DbData) rollback() {
	if dd.tx == nil {
		logger.Println("Error! DbData.rollback() called outside of transaction!")
		return
	}
	dd.tx.Rollback()
	dd.tx = nil
}

func (dd *DbData) post(url string) *Entry {
	posts := loadPosts(dd.db, -1, -1, url, dd.includeHidden)
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
	return loadPosts(dd.db, limit, offset, "", dd.includeHidden)
}

func (dd *DbData) numPosts() int {
	selectSql := "select count(*) from post as p"
	if !dd.includeHidden {
		selectSql = selectSql + " where p.hidden=FALSE"
	}
	rows, err := dd.db.Query(selectSql)
	if err != nil {
		logger.Println(err.Error())
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
		logger.Println(err.Error())
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
		logger.Println(err.Error())
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
		logger.Println(err.Error())
		return
	}
	defer stmt.Close()
	rows, err := stmt.Query(tag)
	if err != nil {
		logger.Println(err.Error())
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
			logger.Println(err.Error())
			continue
		}
		links = append(links, entryLink)
	}
	err := rows.Err()
	if err != nil {
		logger.Println(err.Error())
	}
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
		logger.Println(err.Error())
		return nil
	}
	defer stmt.Close()
	data, err := stmt.Query()
	if err != nil {
		logger.Println(err.Error())
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
		if err != nil {
			logger.Printf("error scanning comment row: %s\n", err.Error())
		}
		comment.EmailHash = Md5Hash(comment.Email)
		comment.Time = time.Unix(unixDate, 0).Format("2006-01-02 15:04")
		comment.Body = sanitizeHTML(mdToHTML(comment.RawBody))
		comments = append(comments, comment)
	}
	err = data.Err()
	if err != nil {
		logger.Printf("error scanning comment row: %s\n", err.Error())
	}
	return comments
}

func (dd *DbData) commenter(c Commenter) (id int64, err error) {
	id = -1
	query, err := dd.db.Prepare(`select c.id from commenter as c
                                 where c.name = $1
                                   and c.email = $2
                                   and c.www = $3`)
	if err != nil {
		logger.Println("err: " + err.Error())
		return
	}
	defer query.Close()
	err = query.QueryRow(c.Name, c.Email, c.Website).Scan(&id)
	if err != nil {
		logger.Println("err: " + err.Error())
	}
	return
}

func (dd *DbData) insertCommenter(c Commenter) (id int64, err error) {
	if dd.tx == nil {
		return -1, fmt.Errorf("DbData.insertCommenter() can only be called within xaction!")
	}
	insertCommenter, _ := dd.tx.Prepare(`insert into commenter
                                         (name, email, www, ip)
                                         values ($1, $2, $3, $4)
                                         returning id`)
	defer insertCommenter.Close()
	err = insertCommenter.QueryRow(c.Name, c.Email, c.Website, c.IP).Scan(&id)
	if err != nil {
		logger.Println("Failed to insert commenter: " + err.Error())
	}
	return
}

func (dd *DbData) insertComment(commenterID, postID int64, body string) (id int64, err error) {
	if dd.tx == nil {
		return -1, fmt.Errorf("DbData.insertComment() can only be called within xaction!")
	}
	stmt, err := dd.tx.Prepare(`insert into comment
                                (commenter_id, post_id, timestamp, body)
                                values ($1, $2, $3, $4)
                                returning id`)
	if err != nil {
		logger.Println("Failed to prepare insert comment stmt: " + err.Error())
		return
	}
	defer stmt.Close()
	err = stmt.QueryRow(commenterID, postID, time.Now().Unix(), body).Scan(&id)
	if err != nil {
		logger.Println("Failed to insert comment: " + err.Error())
		return
	}
	return
}

func (dd *DbData) insertPost(author int64, e *Entry) (id int64, err error) {
	if dd.tx == nil {
		return -1, fmt.Errorf("DbData.insertPost() can only be called within xaction!")
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
		logger.Println("Failed to insert post: " + err.Error())
		return
	}
	return
}

func (dd *DbData) updatePost(id int64, e *Entry) bool {
	updateStmt, _ := dd.tx.Prepare(`update post
                                    set title=$1, url=$2, body=$3, hidden=$4
                                    where id=$5`)
	defer updateStmt.Close()
	_, err := updateStmt.Exec(e.Title, e.URL, string(e.Body), e.Hidden, id)
	if err != nil {
		logger.Println(err.Error())
		return false
	}
	return true
}

func (dd *DbData) updateTags(tags []*Tag, postID int64) error {
	if dd.tx == nil {
		return fmt.Errorf("DbData.updateTags() can only be called within xaction!")
	}
	delStmt, _ := dd.tx.Prepare("delete from tagmap where post_id=$1")
	defer delStmt.Close()
	delStmt.Exec(postID)
	for _, t := range tags {
		tagID, err := insertOrGetTagID(dd.tx, t)
		if err != nil {
			return err
		}
		updateTagMap(dd.tx, postID, tagID)
	}
	return nil
}

func (dd *DbData) author(username string) (*Author, error) {
	row := dd.db.QueryRow(`select passwd, full_name, email, www
                           from author where disp_name=$1`, username)
	var a Author
	a.UserName = username
	err := row.Scan(&a.Passwd, &a.FullName, &a.Email, &a.Www)
	return &a, err
}

func (dd *DbData) deleteComment(id string) bool {
	_, err := dd.db.Exec("delete from comment where id=$1", id)
	if err != nil {
		logger.Println(err.Error())
		return false
	}
	return true
}

func (dd *DbData) deletePost(url string) bool {
	_, err := dd.db.Exec("delete from post where url=$1", url)
	if err != nil {
		logger.Println(err.Error())
		return false
	}
	return true
}

func (dd *DbData) updateComment(id, text string) bool {
	_, err := dd.db.Exec("update comment set body=$1 where id=$2", text, id)
	if err != nil {
		logger.Println(err.Error())
		return false
	}
	return true
}

func loadPosts(db *sql.DB, limit, offset int, url string, includeHidden bool) []*Entry {
	if db == nil {
		return nil
	}
	data, err := queryPosts(db, limit, offset, url, includeHidden)
	if err != nil {
		logger.Println(err.Error())
		return nil
	}
	return data
}

func queryPosts(db *sql.DB, limit, offset int,
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
	rows, err := db.Query(query)
	if err != nil {
		logger.Println(err.Error())
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
			logger.Println(err.Error())
			continue
		}
		entry.Body = sanitizeTrustedHTML(mdToHTML(entry.RawBody))
		entry.Date = time.Unix(unixDate, 0).Format("2006-01-02")
		entry.Tags = queryTags(db, id)
		entry.Comments = queryComments(db, id)
		entries = append(entries, entry)
	}
	err = rows.Err()
	if err != nil {
		logger.Printf("error scanning post row: %s\n", err.Error())
	}
	return
}

func queryTags(db *sql.DB, postID int64) []*Tag {
	stmt, err := db.Prepare(`select t.tag
                             from tag as t, tagmap as tm
                             where t.id = tm.tag_id
                                   and tm.post_id = $1`)
	if err != nil {
		logger.Println(err.Error())
		return nil
	}
	defer stmt.Close()
	rows, err := stmt.Query(postID)
	if err != nil {
		logger.Println(err.Error())
		return nil
	}
	defer rows.Close()
	var tags []*Tag
	for rows.Next() {
		tag := new(Tag)
		err = rows.Scan(&tag.Name)
		if err != nil {
			logger.Println(err.Error())
			continue
		}
		tags = append(tags, tag)
	}
	err = rows.Err()
	if err != nil {
		logger.Printf("error scanning tag row: %s\n", err.Error())
	}
	return tags
}

func (dd *DbData) queryAllTags() []*Tag {
	stmt, err := dd.db.Prepare(`select tag from tag`)
	if err != nil {
		logger.Println(err.Error())
		return nil
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		logger.Println(err.Error())
		return nil
	}
	defer rows.Close()
	var tags []*Tag
	for rows.Next() {
		tag := new(Tag)
		err = rows.Scan(&tag.Name)
		if err != nil {
			logger.Println(err.Error())
			continue
		}
		tags = append(tags, tag)
	}
	err = rows.Err()
	if err != nil {
		logger.Printf("error scanning tag row: %s\n", err.Error())
	}
	return tags
}

func queryComments(db *sql.DB, postID int64) []*Comment {
	stmt, err := db.Prepare(`select a.name, a.email, a.www, a.ip,
                                    c.id, c.timestamp, c.body
                             from commenter as a, comment as c
                             where a.id = c.commenter_id
                                   and c.post_id = $1
                             order by c.timestamp asc`)
	if err != nil {
		logger.Println(err.Error())
		return nil
	}
	defer stmt.Close()
	data, err := stmt.Query(postID)
	if err != nil {
		logger.Println(err.Error())
		return nil
	}
	defer data.Close()
	var comments []*Comment
	for data.Next() {
		comment := new(Comment)
		var unixDate int64
		err = data.Scan(&comment.Name, &comment.Email, &comment.Website, &comment.IP,
			&comment.CommentID, &unixDate, &comment.RawBody)
		if err != nil {
			logger.Printf("error scanning comment row: %s\n", err.Error())
		}
		comment.EmailHash = Md5Hash(comment.Email)
		comment.Time = time.Unix(unixDate, 0).Format("2006-01-02 15:04")
		comment.Body = sanitizeHTML(mdToHTML(comment.RawBody))
		comments = append(comments, comment)
	}
	err = data.Err()
	if err != nil {
		logger.Printf("error scanning comment row: %s\n", err.Error())
	}
	return comments
}

func insertOrGetTagID(xaction *sql.Tx, tag *Tag) (tagID int64, err error) {
	query, err := xaction.Prepare("select id from tag where tag=$1")
	if err != nil {
		logger.Println("Failed to prepare select tag stmt: " + err.Error())
		return
	}
	defer query.Close()
	err = query.QueryRow(tag.Name).Scan(&tagID)
	switch err {
	case nil:
		return
	case sql.ErrNoRows:
		insertTagSql, err := xaction.Prepare(`insert into tag
                                              (tag)
                                              values ($1)
                                              returning id`)
		if err != nil {
			logger.Println("Failed to prepare insert tag stmt: " + err.Error())
			return -1, err
		}
		defer insertTagSql.Close()
		err = insertTagSql.QueryRow(tag.Name).Scan(&tagID)
		if err != nil {
			logger.Println("Failed to insert tag: " + err.Error())
		}
		return tagID, err
	default:
		logger.Printf("err: %s", err.Error())
		return -1, err
	}
}

func updateTagMap(xaction *sql.Tx, postID int64, tagID int64) {
	stmt, err := xaction.Prepare(`insert into tagmap
                                  (tag_id, post_id)
                                  values ($1, $2)`)
	if err != nil {
		logger.Println("Failed to prepare insrt tagmap stmt: " + err.Error())
	}
	defer stmt.Close()
	stmt.Exec(tagID, postID)
}

func insertTestAuthor(db *sql.DB, a *Author) error {
	passwdHash, err := cryptoHelper.Encrypt(a.Passwd)
	if err != nil {
		return err
	}
	stmt, _ := db.Prepare(`insert into author
		(disp_name, passwd, full_name, email, www)
		values ($1, $2, $3, $4, $5)`)
	defer stmt.Close()
	stmt.Exec(a.UserName, passwdHash, a.FullName, a.Email, a.Www)
	return nil
}
