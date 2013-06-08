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
    post(url string) *Entry
    postId(url string) (id int64, err error)
    posts(limit, offset int) []*Entry
    titles(limit int) []*EntryLink
    numPosts() int
    author(username string) (*Author, error)
    deleteComment(id string) bool
    updateComment(id, text string) bool
    selOrInsCommenter(name, email, website, ip string) (id int64, err error)
    insertComment(commenterId, postId int64, body string) (id int64, err error)
    insertPost(author int64, title, url, body string) (id int64, err error)
    updatePost(id int64, tiel, url, body string) bool
    updateTags(tags []*Tag, postId int64)
    begin() bool
    commit()
    rollback()
}

type DbData struct {
    db  *sql.DB
    tx  *sql.Tx
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
    posts := loadPosts(dd.db, -1, -1, url)
    if len(posts) != 1 {
        msg := "Error! DbData.post(%q) should return 1 post, but returned %d\n"
        logger.Println(fmt.Sprintf(msg, url, len(posts)))
        return nil
    }
    return posts[0]
}

func (dd *DbData) postId(url string) (id int64, err error) {
    query, err := dd.db.Prepare("select id from post where url = ?")
    if err != nil {
        return
    }
    defer query.Close()
    err = query.QueryRow(url).Scan(&id)
    return
}

func (dd *DbData) posts(limit, offset int) []*Entry {
    return loadPosts(dd.db, limit, offset, "")
}

func (dd *DbData) numPosts() int {
    rows, err := dd.db.Query(`select count(*) from post`)
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
    selectSql := `select p.title, p.url
                  from post as p
                  order by p.date desc`
    if limit > 0 {
        selectSql = selectSql + " limit ?"
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
        err = rows.Scan(&entryLink.Title, &entryLink.Url)
        if err != nil {
            logger.Println(err.Error())
            continue
        }
        links = append(links, entryLink)
    }
    return
}

func (dd *DbData) selOrInsCommenter(name, email, website, ip string) (id int64, err error) {
    id = -1
    err = sql.ErrNoRows
    if dd.tx == nil {
        logger.Println("DbData.selOrInsCommenter() can only be called within xaction!")
        return
    }
    query, _ := dd.tx.Prepare(`select c.id from commenter as c
                               where c.name = ?
                                 and c.email = ?
                                 and c.www = ?`)
    defer query.Close()
    err = query.QueryRow(name, email, website).Scan(&id)
    switch err {
    case nil:
        return
    case sql.ErrNoRows:
        insertCommenter, _ := dd.tx.Prepare(`insert into commenter
                                             (name, email, www, ip)
                                             values (?, ?, ?, ?)`)
        defer insertCommenter.Close()
        result, err := insertCommenter.Exec(name, email, website, ip)
        if err != nil {
            logger.Println("Failed to insert commenter: " + err.Error())
        }
        return result.LastInsertId()
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
                                values (?, ?, ?, ?)`)
    if err != nil {
        logger.Println("Failed to prepare insert comment stmt: " + err.Error())
        return
    }
    defer stmt.Close()
    result, err := stmt.Exec(commenterId, postId, time.Now().Unix(), body)
    if err != nil {
        logger.Println("Failed to insert comment: " + err.Error())
        return
    }
    return result.LastInsertId()
}

func (dd *DbData) insertPost(author int64, title, url, body string) (id int64, err error) {
    id = -1
    err = sql.ErrNoRows
    if dd.tx == nil {
        logger.Println("DbData.insertPost() can only be called within xaction!")
        return
    }
    insertPostSql, _ := dd.tx.Prepare(`insert into post
                                       (author_id, title, date, url, body)
                                       values (?, ?, ?, ?, ?)`)
    defer insertPostSql.Close()
    date := time.Now().Unix()
    result, err := insertPostSql.Exec(author, title, date, url, body)
    if err != nil {
        logger.Println("Failed to insert post: " + err.Error())
        return
    }
    return result.LastInsertId()
}

func (dd *DbData) updatePost(id int64, title, url, body string) bool {
    updateStmt, _ := dd.tx.Prepare(`update post
                                    set title=?, url=?, body=?
                                    where id=?`)
    defer updateStmt.Close()
    _, err := updateStmt.Exec(title, url, body, id)
    if err != nil {
        logger.Println(err.Error())
        return false
    }
    return true
}

func (dd *DbData) updateTags(tags []*Tag, postId int64) {
    delStmt, _ := dd.tx.Prepare("delete from tagmap where post_id=?")
    defer delStmt.Close()
    delStmt.Exec(postId)
    for _, t := range tags {
        tagId, _ := insertOrGetTagId(dd.tx, t)
        updateTagMap(dd.tx, postId, tagId)
    }
}

func (dd *DbData) author(username string) (*Author, error) {
    row := dd.db.QueryRow(`select salt, passwd, full_name, email, www
                           from author where disp_name=?`, username)
    var a Author
    a.UserName = username
    err := row.Scan(&a.Salt, &a.Passwd, &a.FullName, &a.Email, &a.Www)
    return &a, err
}

func (dd *DbData) deleteComment(id string) bool {
    _, err := dd.db.Exec("delete from comment where id=?", id)
    if err != nil {
        logger.Println(err.Error())
        return false
    }
    return true
}

func (dd *DbData) updateComment(id, text string) bool {
    _, err := dd.db.Exec("update comment set body=? where id=?", text, id)
    if err != nil {
        logger.Println(err.Error())
        return false
    }
    return true
}

func loadPosts(db *sql.DB, limit, offset int, url string) []*Entry {
    if db == nil {
        return nil
    }
    data, err := queryPosts(db, limit, offset, url)
    if err != nil {
        logger.Println(err.Error())
        return nil
    }
    return data
}

func queryPosts(db *sql.DB, limit, offset int, url string) (entries []*Entry, err error) {
    postUrlWhereClause := ""
    if url != "" {
        postUrlWhereClause = fmt.Sprintf("and p.url='%s'", url)
    }
    limitClause := ""
    if limit >= 0 {
        limitClause = fmt.Sprintf("limit %d", limit)
    }
    offsetClause := ""
    if offset > 0 {
        offsetClause = fmt.Sprintf("offset %d", offset)
    }
    queryFmt := `select a.disp_name, p.id, p.title, p.date, p.body, p.url
                 from author as a, post as p
                 where a.id=p.author_id
                 %s
                 order by p.date desc
                 %s %s`
    query := fmt.Sprintf(queryFmt, postUrlWhereClause, limitClause, offsetClause)
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
            &entry.RawBody, &entry.Url)
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
                                   and tm.post_id = ?`)
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
                                   and c.post_id = ?
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
    query, err := xaction.Prepare("select id from tag where url=?")
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
                                              values (?, ?)`)
        if err != nil {
            logger.Println("Failed to prepare insert tag stmt: " + err.Error())
            return -1, err
        }
        defer insertTagSql.Close()
        result, err := insertTagSql.Exec(tag.TagName, tag.TagUrl)
        if err != nil {
            logger.Println("Failed to insert tag: " + err.Error())
        }
        return result.LastInsertId()
    default:
        logger.Printf("err: %s", err.Error())
        return -1, sql.ErrNoRows
    }
    return -1, sql.ErrNoRows
}

func updateTagMap(xaction *sql.Tx, postId int64, tagId int64) {
    stmt, err := xaction.Prepare(`insert into tagmap
                                  (tag_id, post_id)
                                  values (?, ?)`)
    if err != nil {
        logger.Println("Failed to prepare insrt tagmap stmt: " + err.Error())
    }
    defer stmt.Close()
    stmt.Exec(tagId, postId)
}
