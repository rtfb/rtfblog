package main

import (
    "crypto/md5"
    "database/sql"
    "fmt"
    "strings"
    "time"

    "github.com/russross/blackfriday"
)

type Data interface {
    hiddenPosts(flag bool)
    post(url string) *Entry
    postId(url string) (id int64, err error)
    posts(limit, offset int) []*Entry
    titles(limit int) []*EntryLink
    allComments() []*CommentWithPostTitle
    numPosts() int
    author(username string) (*Author, error)
    deleteComment(id string) bool
    updateComment(id, text string) bool
    selOrInsCommenter(name, email, website, ip string) (id int64, err error)
    insertComment(commenterId, postId int64, body string) (id int64, err error)
    insertPost(author int64, e *Entry) (id int64, err error)
    updatePost(id int64, e *Entry) bool
    updateTags(tags []*Tag, postId int64)
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

func (dd *DbData) postId(url string) (id int64, err error) {
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
    selectSql := "select count(*) from post"
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
    for rows.Next() {
        entryLink := new(EntryLink)
        err = rows.Scan(&entryLink.Title, &entryLink.Url, &entryLink.Hidden)
        if err != nil {
            logger.Println(err.Error())
            continue
        }
        links = append(links, entryLink)
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
    comments := make([]*CommentWithPostTitle, 0)
    for data.Next() {
        comment := new(CommentWithPostTitle)
        var unixDate int64
        err = data.Scan(&comment.Name, &comment.Email, &comment.Website, &comment.Ip,
            &comment.CommentId, &unixDate, &comment.RawBody,
            &comment.PostTitle, &comment.PostUrl)
        if err != nil {
            logger.Printf("error scanning comment row: %s\n", err.Error())
        }
        hash := md5.New()
        hash.Write([]byte(strings.ToLower(comment.Email)))
        comment.EmailHash = fmt.Sprintf("%x", hash.Sum(nil))
        comment.Time = time.Unix(unixDate, 0).Format("2006-01-02 15:04")
        comment.Body = string(blackfriday.MarkdownCommon([]byte(comment.RawBody)))
        comments = append(comments, comment)
    }
    return comments
}

func (dd *DbData) selOrInsCommenter(name, email, website, ip string) (id int64, err error) {
    id = -1
    err = sql.ErrNoRows
    if dd.tx == nil {
        logger.Println("DbData.selOrInsCommenter() can only be called within xaction!")
        return
    }
    query, _ := dd.tx.Prepare(`select c.id from commenter as c
                               where c.name = $1
                                 and c.email = $2
                                 and c.www = $3`)
    defer query.Close()
    err = query.QueryRow(name, email, website).Scan(&id)
    switch err {
    case nil:
        return
    case sql.ErrNoRows:
        insertCommenter, _ := dd.tx.Prepare(`insert into commenter
                                             (name, email, www, ip)
                                             values ($1, $2, $3, $4)
                                             returning id`)
        defer insertCommenter.Close()
        err = insertCommenter.QueryRow(name, email, website, ip).Scan(&id)
        if err != nil {
            logger.Println("Failed to insert commenter: " + err.Error())
        }
        return
    default:
        logger.Println("err: " + err.Error())
        return
    }
    return
}

func (dd *DbData) insertComment(commenterId, postId int64, body string) (id int64, err error) {
    id = -1
    err = sql.ErrNoRows
    if dd.tx == nil {
        logger.Println("DbData.insertComment() can only be called within xaction!")
        return
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
    err = stmt.QueryRow(commenterId, postId, time.Now().Unix(), body).Scan(&id)
    if err != nil {
        logger.Println("Failed to insert comment: " + err.Error())
        return
    }
    return
}

func (dd *DbData) insertPost(author int64, e *Entry) (id int64, err error) {
    id = -1
    err = sql.ErrNoRows
    if dd.tx == nil {
        logger.Println("DbData.insertPost() can only be called within xaction!")
        return
    }
    insertPostSql, _ := dd.tx.Prepare(`insert into post
                                       (author_id, title, date, url, body, hidden)
                                       values ($1, $2, $3, $4, $5, $6)
                                       returning id`)
    defer insertPostSql.Close()
    date := time.Now().Unix()
    err = insertPostSql.QueryRow(e.Author, e.Title, date, e.Url, e.Body, e.Hidden).Scan(&id)
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
    _, err := updateStmt.Exec(e.Title, e.Url, e.Body, e.Hidden, id)
    if err != nil {
        logger.Println(err.Error())
        return false
    }
    return true
}

func (dd *DbData) updateTags(tags []*Tag, postId int64) {
    delStmt, _ := dd.tx.Prepare("delete from tagmap where post_id=$1")
    defer delStmt.Close()
    delStmt.Exec(postId)
    for _, t := range tags {
        tagId, _ := insertOrGetTagId(dd.tx, t)
        updateTagMap(dd.tx, postId, tagId)
    }
}

func (dd *DbData) author(username string) (*Author, error) {
    row := dd.db.QueryRow(`select salt, passwd, full_name, email, www
                           from author where disp_name=$1`, username)
    var a Author
    a.UserName = username
    err := row.Scan(&a.Salt, &a.Passwd, &a.FullName, &a.Email, &a.Www)
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
    postUrlWhereClause := ""
    if url != "" {
        postUrlWhereClause = fmt.Sprintf("and p.url='%s'", url)
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
    query := fmt.Sprintf(queryFmt, postUrlWhereClause, postHiddenWhereClause,
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
            &entry.RawBody, &entry.Url, &entry.Hidden)
        if err != nil {
            logger.Println(err.Error())
            continue
        }
        entry.Body = string(blackfriday.MarkdownCommon([]byte(entry.RawBody)))
        entry.Date = time.Unix(unixDate, 0).Format("2006-01-02")
        entry.Tags = queryTags(db, id)
        entry.Comments = queryComments(db, id)
        entries = append(entries, entry)
    }
    return
}

func queryTags(db *sql.DB, postId int64) []*Tag {
    stmt, err := db.Prepare(`select t.name, t.url
                             from tag as t, tagmap as tm
                             where t.id = tm.tag_id
                                   and tm.post_id = $1`)
    if err != nil {
        logger.Println(err.Error())
        return nil
    }
    defer stmt.Close()
    rows, err := stmt.Query(postId)
    if err != nil {
        logger.Println(err.Error())
        return nil
    }
    defer rows.Close()
    tags := make([]*Tag, 0)
    for rows.Next() {
        tag := new(Tag)
        err = rows.Scan(&tag.TagName, &tag.TagUrl)
        if err != nil {
            logger.Println(err.Error())
            continue
        }
        tags = append(tags, tag)
    }
    return tags
}

func queryComments(db *sql.DB, postId int64) []*Comment {
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
    data, err := stmt.Query(postId)
    if err != nil {
        logger.Println(err.Error())
        return nil
    }
    defer data.Close()
    comments := make([]*Comment, 0)
    for data.Next() {
        comment := new(Comment)
        var unixDate int64
        err = data.Scan(&comment.Name, &comment.Email, &comment.Website, &comment.Ip,
            &comment.CommentId, &unixDate, &comment.RawBody)
        if err != nil {
            logger.Printf("error scanning comment row: %s\n", err.Error())
        }
        hash := md5.New()
        hash.Write([]byte(strings.ToLower(comment.Email)))
        comment.EmailHash = fmt.Sprintf("%x", hash.Sum(nil))
        comment.Time = time.Unix(unixDate, 0).Format("2006-01-02 15:04")
        comment.Body = string(blackfriday.MarkdownCommon([]byte(comment.RawBody)))
        comments = append(comments, comment)
    }
    return comments
}

func insertOrGetTagId(xaction *sql.Tx, tag *Tag) (tagId int64, err error) {
    query, err := xaction.Prepare("select id from tag where url=$1")
    if err != nil {
        logger.Println("Failed to prepare select tag stmt: " + err.Error())
        return
    }
    defer query.Close()
    err = query.QueryRow(tag.TagUrl).Scan(&tagId)
    switch err {
    case nil:
        return
    case sql.ErrNoRows:
        insertTagSql, err := xaction.Prepare(`insert into tag
                                              (name, url)
                                              values ($1, $2)
                                              returning id`)
        if err != nil {
            logger.Println("Failed to prepare insert tag stmt: " + err.Error())
            return -1, err
        }
        defer insertTagSql.Close()
        err = insertTagSql.QueryRow(tag.TagName, tag.TagUrl).Scan(&tagId)
        if err != nil {
            logger.Println("Failed to insert tag: " + err.Error())
        }
        return tagId, err
    default:
        logger.Printf("err: %s", err.Error())
        return -1, sql.ErrNoRows
    }
    return -1, sql.ErrNoRows
}

func updateTagMap(xaction *sql.Tx, postId int64, tagId int64) {
    stmt, err := xaction.Prepare(`insert into tagmap
                                  (tag_id, post_id)
                                  values ($1, $2)`)
    if err != nil {
        logger.Println("Failed to prepare insrt tagmap stmt: " + err.Error())
    }
    defer stmt.Close()
    stmt.Exec(tagId, postId)
}
