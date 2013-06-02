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
}

type DbData struct{}

func (db *DbData) post(url string) *Entry {
    posts := loadPosts(-1, -1, url)
    if len(posts) != 1 {
        msg := "Error! DbData.post(%q) should return 1 post, but returned %d\n"
        println(fmt.Sprintf(msg, url, len(posts)))
        return nil
    }
    return posts[0]
}

func (dd *DbData) postId(url string) (id int64, err error) {
    query, err := db.Prepare("select id from post where url = ?")
    defer query.Close()
    if err != nil {
        return
    }
    err = query.QueryRow(url).Scan(&id)
    return
}

func (db *DbData) posts(limit, offset int) []*Entry {
    return loadPosts(limit, offset, "")
}

func (dd *DbData) numPosts() int {
    rows, err := db.Query(`select count(*) from post`)
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
    stmt, err := db.Prepare(selectSql)
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

func (dd *DbData) author(username string) (*Author, error) {
    row := db.QueryRow(`select salt, passwd, full_name, email, www
                        from author where disp_name=?`, username)
    var a Author
    a.UserName = username
    err := row.Scan(&a.Salt, &a.Passwd, &a.FullName, &a.Email, &a.Www)
    return &a, err
}

func loadPosts(limit, offset int, url string) []*Entry {
    if db == nil {
        return nil
    }
    data, err := queryPosts(limit, offset, url)
    if err != nil {
        println(err.Error())
        return nil
    }
    return data
}

func queryPosts(limit, offset int, url string) (entries []*Entry, err error) {
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
