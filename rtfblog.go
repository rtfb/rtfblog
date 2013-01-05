package main

import (
    "crypto/md5"
    "database/sql"
    "fmt"
    "log"
    "os"
    "path"
    "strings"
    "time"

    "github.com/hoisie/web"
    "github.com/lye/mustache"
    _ "github.com/mattn/go-sqlite3"
    "github.com/russross/blackfriday"
)

type Tag struct {
    TagUrl  string
    TagName string
}

type Comment struct {
    Name      string
    Email     string
    EmailHash string
    Website   string
    Ip        string
    Body      string
    Time      string
    CommentId string
}

type Entry struct {
    Author   string
    Title    string
    Date     string
    Body     string
    Url      string
    Tags     []*Tag
    Comments []*Comment
}

var dataset string
var dbName string

func (e *Entry) HasTags() bool {
    return len(e.Tags) > 0
}

func (e *Entry) HasComments() bool {
    return len(e.Comments) > 0
}

func (e *Entry) TagsStr() string {
    parts := make([]string, 0)
    for _, t := range e.Tags {
        part := fmt.Sprintf(`<a href="/tag/%s">%s</a>`, t.TagUrl, t.TagName)
        parts = append(parts, part)
    }
    return strings.Join(parts, ", ")
}

func readDb(dbName string) (entries []*Entry, err error) {
    db, err := sql.Open("sqlite3", dbName)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer db.Close()
    rows, err := db.Query(`select a.disp_name, p.id, p.title, p.date,
                                  p.body, p.url
                           from author as a, post as p
                           where a.id=p.author_id`)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer rows.Close()
    for rows.Next() {
        entry := new(Entry)
        var id int
        var unixDate int64
        var bodyMarkdown string
        rows.Scan(&entry.Author, &id, &entry.Title, &unixDate,
            &bodyMarkdown, &entry.Url)
        entry.Body = string(blackfriday.MarkdownCommon([]byte(bodyMarkdown)))
        entry.Date = time.Unix(unixDate, 0).Format("2006-01-02")
        entry.Tags = queryTags(db, id)
        entry.Comments = queryComments(db, id)
        entries = append(entries, entry)
    }
    return
}

func queryTags(db *sql.DB, postId int) []*Tag {
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
        rows.Scan(&tag.TagName, &tag.TagUrl)
        tags = append(tags, tag)
    }
    return tags
}

func queryComments(db *sql.DB, postId int) []*Comment {
    stmt, err := db.Prepare(`select a.name, a.email, a.www, a.ip,
                                    c.id, c.timestamp, c.body
                             from commenter as a, comment as c
                             where a.id = c.commenter_id
                                   and c.post_id = ?`)
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
        var body string
        data.Scan(&comment.Name, &comment.Email, &comment.Website, &comment.Ip,
            &comment.CommentId, &unixDate, &body)
        hash := md5.New()
        hash.Write([]byte(strings.ToLower(comment.Email)))
        comment.EmailHash = fmt.Sprintf("%x", hash.Sum(nil))
        comment.Time = time.Unix(unixDate, 0).Format("2006-01-02 15:04")
        comment.Body = string(blackfriday.MarkdownCommon([]byte(body)))
        comments = append(comments, comment)
    }
    return comments
}

func render(ctx *web.Context, tmpl string, data map[string]interface{}) {
    html := mustache.RenderFile("tmpl/"+tmpl+".html.mustache", data)
    ctx.WriteString(html)
}

func handler(ctx *web.Context, path string) {
    posts := loadData(dataset, dbName)
    var basicData = map[string]interface{}{
        "PageTitle": "",
        "entries":   posts,
    }
    value, found := ctx.GetSecureCookie("adminlogin")
    basicData["AdminLogin"] = found && value == "yesplease"
    if path == "" {
        basicData["PageTitle"] = "Velkam"
        render(ctx, "main", basicData)
        return
    } else if path == "login" {
        basicData["PageTitle"] = "Login"
        render(ctx, "login", basicData)
        return
    } else if path == "logout" {
        ctx.SetSecureCookie("adminlogin", "", 0)
        ctx.Redirect(301, "/")
        return
    } else {
        for _, e := range posts {
            if e.Url == path {
                basicData["PageTitle"] = e.Title
                basicData["entry"] = e
                render(ctx, "post", basicData)
                return
            }
        }
        ctx.NotFound("Page not found: " + path)
        return
    }
    ctx.Abort(500, "Server Error")
}

func getCommenterId(xaction *sql.Tx, ctx *web.Context) (id int64, err error) {
    name := ctx.Params["name"]
    email := ctx.Params["email"]
    website := ctx.Params["website"]
    ip := ctx.Request.RemoteAddr
    query, _ := xaction.Prepare(`select c.id from commenter as c
                                 where c.name = ?
                                   and c.email = ?
                                   and c.www = ?`)
    defer query.Close()
    err = query.QueryRow(name, email, website).Scan(&id)
    switch err {
    case nil:
        return
    case sql.ErrNoRows:
        insertCommenter, _ := xaction.Prepare(`insert into commenter
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
        return -1, sql.ErrNoRows
    }
    return -1, sql.ErrNoRows
}

func getPostId(xaction *sql.Tx, url string) (id int64, err error) {
    query, _ := xaction.Prepare(`select p.id from post as p
                                 where p.url = ?`)
    defer query.Close()
    err = query.QueryRow(url).Scan(&id)
    return
}

func login_handler(ctx *web.Context) {
    uname := ctx.Params["uname"]
    //passwd := ctx.Request.Form["passwd"][0]
    if uname == "admin" {
        ctx.SetSecureCookie("adminlogin", "yesplease", int64(time.Hour*24))
    }
    ctx.Redirect(301, "/")
}

func comment_handler(ctx *web.Context) {
    db, err := sql.Open("sqlite3", path.Join(dataset, dbName))
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer db.Close()
    xaction, err := db.Begin()
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    commenterId, err := getCommenterId(xaction, ctx)
    if err != nil {
        fmt.Println("getCommenterId failed: " + err.Error())
        ctx.Abort(500, "Server Error")
        return
    }
    referer := ctx.Request.Header["Referer"][0]
    refUrl := referer[strings.LastIndex(referer, "/")+1:]
    postId, err := getPostId(xaction, refUrl)
    if err != nil {
        fmt.Println("getPostId failed: " + err.Error())
        ctx.Abort(500, "Server Error")
        return
    }
    stmt, _ := xaction.Prepare(`insert into comment(commenter_id, post_id,
                                                    timestamp, body)
                                values(?, ?, ?, ?)`)
    defer stmt.Close()
    body := ctx.Params["text"]
    stmt.Exec(commenterId, postId, time.Now().Unix(), body)
    xaction.Commit()
    ctx.Redirect(301, "/"+refUrl)
}

func runServer() {
    f, err := os.Create("server.log")
    if err != nil {
        println(err.Error())
        return
    }
    logger := log.New(f, "", log.Ldate|log.Ltime)
    web.Post("/comment_submit", comment_handler)
    web.Post("/login_submit", login_handler)
    web.Get("/(.*)", handler)
    web.SetLogger(logger)
    web.Config.StaticDir = "static"
    web.Config.CookieSecret = "foobarbaz" // XXX: don't forget to change that!
    web.Run(":8080")
}

func loadData(set string, db string) []*Entry {
    if set == "" || dbName == "" {
        return nil
    }
    data, err := readDb(path.Join(set, dbName))
    if err != nil {
        println(err.Error())
        return nil
    }
    return data
}

func main() {
    dataset = "testdata"
    dbName = "foo.db"
    runServer()
}
