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
    posts(limit int) []*Entry
    numPosts() int
}

type DbData struct{}

func (db *DbData) post(url string) *Entry {
    posts := loadData()
    for _, e := range posts {
        if e.Url == url {
            return e
        }
    }
    return nil
}

func (db *DbData) posts(limit int) []*Entry {
    posts := loadData()
    if limit > 0 && limit < len(posts) {
        return posts[:limit]
    }
    return posts
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

func loadData() []*Entry {
    if db == nil {
        return nil
    }
    data, err := readDb()
    if err != nil {
        println(err.Error())
        return nil
    }
    return data
}

func readDb() (entries []*Entry, err error) {
    rows, err := db.Query(`select a.disp_name, p.id, p.title, p.date,
                                  p.body, p.url
                           from author as a, post as p
                           where a.id=p.author_id
                           order by p.date desc`)
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
