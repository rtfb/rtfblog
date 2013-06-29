package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "regexp"
    "strings"
    "time"

    _ "github.com/mattn/go-sqlite3"
    _ "github.com/ziutek/mymysql/godrv"
)

// evo_categories
type Category struct {
    id        int64
    parent_id int64
    name      string
    url       string
}

// evo_items__item
type Post struct {
    id        int64
    id_sqlite int64
    date      time.Time
    body      string
    title     string
    url       string
}

// evo_comments
type Comment struct {
    postId        int64
    postId_sqlite int64
    authorId      sql.NullInt64
    author        sql.NullString
    authorEmail   sql.NullString
    authorUrl     sql.NullString
    authorIp      sql.NullString
    date          time.Time
    content       string
}

// evo_users
type User struct {
    id        int64
    id_sqlite int64
    login     string
    firstname string
    lastname  string
    nickname  string
    email     string
    url       string
    ip        string
}

type DbConfig map[string]string

func xferPosts(sconn, mconn *sql.DB) (posts []*Post, err error) {
    asterisk := regexp.MustCompile(`(\n\[\*\](.*)\n)`)
    links := regexp.MustCompile(`silavinimui:\r\n`)
    rows, err := mconn.Query(`select post_ID, post_datecreated, post_content,
                                     post_title, post_urltitle
                              from evo_items__item
                              where post_creator_user_ID=?`, 15)
    //posts := make([]*Post, 0, 10)
    for rows.Next() {
        var p Post
        err = rows.Scan(&p.id, &p.date, &p.body, &p.title, &p.url)
        if err != nil {
            fmt.Printf("err: %s\n" + err.Error())
        }
        posts = append(posts, &p)
    }
    fmt.Printf("#posts: %d\n", len(posts))
    xaction, err := sconn.Begin()
    if err != nil {
        fmt.Println(err)
        return
    }
    for _, p := range posts {
        stmt, err := xaction.Prepare(`insert into post
                                      (author_id, title, date, url, body)
                                      values(?, ?, ?, ?, ?)`)
        if err != nil {
            fmt.Println(err)
            return posts, err
        }
        defer stmt.Close()
        if p.url == "arrr" {
            p.body = asterisk.ReplaceAllString(p.body, "$1\n")
        }
        if p.url == "uzreferenduma-lt-the-good-the-bad-and-the-ugly" {
            prefix := links.FindStringIndex(p.body)
            startLinks := prefix[1]
            p.body = p.body[:startLinks] + fixupLinks(p.body[startLinks:startLinks+357]) + p.body[startLinks+357:]
        }
        newBody := fixupBody(p.body)
        result, err := stmt.Exec(1, p.title, p.date.Unix(), p.url, newBody)
        p.id_sqlite, _ = result.LastInsertId()
        //fmt.Printf("%+v\n", p)
        //fmt.Printf("%q | %q\n", p.title, p.url)
    }
    xaction.Commit()
    return posts, err
}

func fixupLinks(olinks string) (nlinks string) {
    lst := strings.Split(olinks, "\n")
    nlst := make([]string, 0, len(lst))
    for _, line := range lst {
        s := strings.TrimSpace(line)
        if s == "" {
            continue
        }
        nlst = append(nlst, fmt.Sprintf(" - %s\n", s))
    }
    nlinks = "\n" + strings.Join(nlst, "\n") + "\n"
    return
}

func fixupBody(obody string) (nbody string) {
    nbody = fixupPre(obody)
    nbody = fixupTt(nbody)
    nbody = fixupOl(nbody)
    nbody = fixupImgLinks(nbody)
    nbody = strings.Replace(nbody, "pasistatyi", "pasistatyti", -1)
    nbody = strings.Replace(nbody, "sąngražinės", "sangrąžinės", -1)
    return
}

func fixupImgLinks(obody string) (nbody string) {
    ilines := strings.Split(obody, "\n")
    olines := make([]string, 0, len(ilines))
    for _, line := range ilines {
        newline := strings.Replace(line, "http://blog.stent.lt/media/blogs/rtfb", "", -1)
        newline = strings.Replace(newline, "h3", "h4", -1)
        olines = append(olines, newline)
    }
    nbody = strings.Join(olines, "\n")
    return
}

func fixupPre(obody string) (nbody string) {
    ilines := strings.Split(obody, "\n")
    olines := make([]string, 0, len(ilines))
    inPre := false
    for _, line := range ilines {
        if strings.Contains(line, "<pre>") {
            inPre = true
            line = strings.Replace(line, "<pre>", "", -1)
        }
        if strings.Contains(line, "</pre>") {
            inPre = false
            line = strings.Replace(line, "</pre>", "", -1)
        }
        if inPre {
            olines = append(olines, "    "+line)
        } else {
            olines = append(olines, line)
        }
    }
    nbody = strings.Join(olines, "\n")
    return
}

func fixupTt(obody string) (nbody string) {
    nbody = strings.Replace(obody, "<tt>", "`", -1)
    nbody = strings.Replace(nbody, "</tt>", "`", -1)
    return
}

func fixupOl(obody string) (nbody string) {
    ilines := strings.Split(obody, "\n")
    olines := make([]string, 0, len(ilines))
    inList := false
    for _, line := range ilines {
        if strings.Contains(line, "<ol") {
            inList = true
        }
        if strings.Contains(line, "</ol>") {
            inList = false
            s := strings.Replace(strings.TrimSpace(line), "<li>", "1. ", -1)
            s = strings.Replace(s, "</li>", "", -1)
            s = strings.Replace(s, "</ol>", "", -1)
            olines = append(olines, s)
            continue
        }
        if inList && strings.TrimSpace(line) == "" {
            continue
        } else if inList {
            s := strings.Replace(strings.TrimSpace(line), "<li>", "1. ", -1)
            s = strings.Replace(s, "</li>", "", -1)
            s = strings.Replace(s, "<ol>", "", -1)
            s = strings.Replace(s, "<ol class=\"withalpha\">", "", -1)
            s = strings.Replace(s, "</ol>", "", -1)
            olines = append(olines, s)
        } else {
            olines = append(olines, line)
        }
    }
    nbody = strings.Join(olines, "\n")
    return
}

func xferComments(sconn, mconn *sql.DB, posts []*Post) {
    comms := make([]*Comment, 0, 10)
    for _, p := range posts {
        rows, err := mconn.Query(`select comment_post_ID, comment_author_ID,
                                         comment_author, comment_author_email,
                                         comment_author_url, comment_author_IP,
                                         comment_date, comment_content,
                                         comment_ID
                                  from evo_comments
                                  where comment_post_ID=?`, p.id)
        for rows.Next() {
            var c Comment
            var cid int
            err = rows.Scan(&c.postId, &c.authorId, &c.author, &c.authorEmail,
                &c.authorUrl, &c.authorIp, &c.date, &c.content, &cid)
            if err != nil {
                fmt.Printf("err: %s\n" + err.Error())
            }
            if strings.Contains(c.content, "Honesty is the rarest wealth anyone can possess") ||
                strings.Contains(c.content, "I do that you are going to be elaborating more on this issue") {
                fmt.Printf("skipping spam comment, id=%d\n", cid)
                continue
            }
            c.postId_sqlite = p.id_sqlite
            comms = append(comms, &c)
        }
    }
    fmt.Printf("#comms: %d\n", len(comms))
    xaction, err := sconn.Begin()
    if err != nil {
        fmt.Println(err)
        return
    }
    for _, c := range comms {
        //fmt.Printf("%+v\n", c)
        authorId, err := getCommenterId(xaction, mconn, c)
        if err == sql.ErrNoRows {
            insertCommenter, _ := xaction.Prepare(`insert into commenter
                                                   (name, email, www, ip)
                                                   values (?, ?, ?, ?)`)
            defer insertCommenter.Close()
            ip := ""
            if c.authorIp.Valid {
                ip = c.authorIp.String
            }
            result, err := insertCommenter.Exec(c.author, c.authorEmail,
                c.authorUrl, ip)
            if err != nil {
                fmt.Println("Failed to insert commenter: " + err.Error())
            }
            authorId, err = result.LastInsertId()
            if err != nil {
                fmt.Println("Failed to insert commenter: " + err.Error())
            }
        } else if err != nil {
            fmt.Println("err: " + err.Error())
            continue
        }
        stmt, err := xaction.Prepare(`insert into comment
                                      (commenter_id, post_id, timestamp, body)
                                      values(?, ?, ?, ?)`)
        defer stmt.Close()
        if err != nil {
            fmt.Printf("err: %s\n", err.Error())
        }
        _, err = stmt.Exec(authorId, c.postId_sqlite, c.date.Unix(), c.content)
        if c.authorId.Int64 == 10 {
            //fmt.Printf("%+v\n", c)
        }
        if err != nil {
            fmt.Printf("err: %s\n", err.Error())
        }
    }
    xaction.Commit()
}

func getCommenterId(xaction *sql.Tx, mconn *sql.DB, comment *Comment) (id int64, err error) {
    if comment.authorId.Valid {
        query, err := mconn.Prepare(`select user_nickname, user_email,
                                              user_url, user_ip
                                       from evo_users
                                       where user_ID=?`)
        defer query.Close()
        if err != nil {
            fmt.Printf("err: %s\n", err.Error())
        }
        err = query.QueryRow(comment.authorId.Int64).Scan(&comment.author,
            &comment.authorEmail,
            &comment.authorUrl,
            &comment.authorIp)
    }
    query, _ := xaction.Prepare(`select c.id from commenter as c
                                 where c.email like ?`)
    defer query.Close()
    err = query.QueryRow(comment.authorEmail.String).Scan(&id)
    return
}

func xferTags(sconn, mconn *sql.DB, posts []*Post) {
    for _, p := range posts {
        rows, err := mconn.Query(`select t.tag_name
                                  from evo_items__tag as t,
                                       evo_items__itemtag as it
                                  where t.tag_ID=it.itag_tag_ID
                                    and it.itag_itm_ID=?`, p.id)
        if err != nil {
            fmt.Printf("err: %s\n", err.Error())
        }
        for rows.Next() {
            var tag string
            err = rows.Scan(&tag)
            if err != nil {
                fmt.Printf("err: %s\n" + err.Error())
            }
            fixedTag := strings.Replace(tag, " ", "-", -1)
            row := sconn.QueryRow(`select id from tag where url=?`, fixedTag)
            var tagId int64
            err = row.Scan(&tagId)
            if err != nil {
                if err == sql.ErrNoRows {
                    stmt, err := sconn.Prepare(`insert into tag
                                                (name, url)
                                                values(?, ?)`)
                    if err != nil {
                        fmt.Println(err)
                        continue
                    }
                    defer stmt.Close()
                    result, err := stmt.Exec(strings.Title(tag), fixedTag)
                    tagId, _ = result.LastInsertId()
                } else {
                    fmt.Printf("err: %s\n", err.Error())
                }
            }
            stmt, err := sconn.Prepare(`insert into tagmap
                                        (tag_id, post_id)
                                        values(?, ?)`)
            if err != nil {
                fmt.Println(err)
                continue
            }
            defer stmt.Close()
            _, err = stmt.Exec(tagId, p.id_sqlite)
            if err != nil {
                fmt.Printf("err inserting tagmap: %s\n", err.Error())
            }
        }
    }
}

func readConf(path string) (db, uname, passwd string) {
    b, err := ioutil.ReadFile(path)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    var config DbConfig
    err = json.Unmarshal(b, &config)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    return config["db_name"], config["uname"], config["passwd"]
}

func importLegacyDb(sqliteFile, dbConf string) {
    db, uname, passwd := readConf(dbConf)
    mconn, err := sql.Open("mymysql", fmt.Sprintf("%s/%s/%s", db, uname, passwd))
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer mconn.Close()
    sconn, err := sql.Open("sqlite3", sqliteFile)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer sconn.Close()
    row := mconn.QueryRow(`select blog_shortname, blog_name, blog_owner_user_ID
                           from evo_blogs where blog_ID=?`, 19)
    shortname := ""
    name := ""
    uid := 0
    err = row.Scan(&shortname, &name, &uid)
    if err != nil {
        fmt.Printf("err: " + err.Error())
    } else {
        fmt.Printf("shortname: %q, name: %q, id=%d\n", shortname, name, uid)
    }
    rows, err := mconn.Query(`select cat_ID, cat_parent_ID,
                                     cat_name, cat_urlname
                              from evo_categories
                              where cat_blog_ID=?`, 19)
    cat := make([]*Category, 0, 10)
    var id int64
    var parent_id sql.NullInt64
    var url string
    for rows.Next() {
        err = rows.Scan(&id, &parent_id, &name, &url)
        if err != nil {
            fmt.Printf("err: %s\n" + err.Error())
        }
        cat = append(cat, &Category{id, parent_id.Int64, name, url})
    }
    fmt.Printf("#categories: %d\n", len(cat))
    posts, err := xferPosts(sconn, mconn)
    if err != nil {
        fmt.Printf("err: " + err.Error())
    }
    xferComments(sconn, mconn, posts)
    xferTags(sconn, mconn, posts)
}
