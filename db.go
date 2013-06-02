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
    begin() bool
    commit()
    rollback()
    xaction() *sql.Tx
}

type DbData struct {
    db  *sql.DB
    tx  *sql.Tx
}

func (dd *DbData) begin() bool {
    if dd.tx != nil {
        fmt.Println("Error! DbData.begin() called within transaction!")
        return false
    }
    xaction, err := dd.db.Begin()
    if err != nil {
        fmt.Println(err.Error())
        return false
    }
    dd.tx = xaction
    return true
}

func (dd *DbData) commit() {
    if dd.tx == nil {
        fmt.Println("Error! DbData.commit() called outside of transaction!")
        return
    }
    dd.tx.Commit()
    dd.tx = nil
}

func (dd *DbData) rollback() {
    if dd.tx == nil {
        fmt.Println("Error! DbData.rollback() called outside of transaction!")
        return
    }
    dd.tx.Rollback()
    dd.tx = nil
}

func (dd *DbData) xaction() *sql.Tx {
    return dd.tx
}

func (dd *DbData) post(url string) *Entry {
    posts := loadPosts(dd.db, -1, -1, url)
    if len(posts) != 1 {
        msg := "Error! DbData.post(%q) should return 1 post, but returned %d\n"
        println(fmt.Sprintf(msg, url, len(posts)))
        return nil
    }
    return posts[0]
}

func (dd *DbData) postId(url string) (id int64, err error) {
    query, err := dd.db.Prepare("select id from post where url = ?")
    defer query.Close()
    if err != nil {
        return
    }
    err = query.QueryRow(url).Scan(&id)
    return
}

func (dd *DbData) posts(limit, offset int) []*Entry {
    return loadPosts(dd.db, limit, offset, "")
}

func (dd *DbData) numPosts() int {
    rows, err := dd.db.Query(`select count(*) from post`)
    if err != nil {
        fmt.Println(err.Error())
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
        fmt.Println(err.Error())
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
        fmt.Println(err.Error())
        return
    }
    defer rows.Close()
    for rows.Next() {
        entryLink := new(EntryLink)
        err = rows.Scan(&entryLink.Title, &entryLink.Url)
        if err != nil {
            fmt.Println(err.Error())
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
        fmt.Println("DbData.selOrInsCommenter() can only be called within xaction!")
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
            fmt.Println("Failed to insert commenter: " + err.Error())
        }
        return result.LastInsertId()
    default:
        fmt.Println("err")
        fmt.Println(err.Error())
        return
    }
    return
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
        fmt.Println(err.Error())
        return false
    }
    return true
}

func (dd *DbData) updateComment(id, text string) bool {
    _, err := dd.db.Exec("update comment set body=? where id=?", text, id)
    if err != nil {
        fmt.Println(err.Error())
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
        println(err.Error())
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
        fmt.Println(err.Error())
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
            fmt.Println(err.Error())
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
        fmt.Println(err.Error())
        return nil
    }
    defer stmt.Close()
    rows, err := stmt.Query(postId)
    if err != nil {
        fmt.Println(err.Error())
        return nil
    }
    defer rows.Close()
    tags := make([]*Tag, 0)
    for rows.Next() {
        tag := new(Tag)
        err = rows.Scan(&tag.TagName, &tag.TagUrl)
        if err != nil {
            fmt.Println(err.Error())
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
        fmt.Println(err.Error())
        return nil
    }
    defer stmt.Close()
    data, err := stmt.Query(postId)
    if err != nil {
        fmt.Println(err.Error())
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
            fmt.Printf("error scanning comment row: %s\n", err.Error())
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
