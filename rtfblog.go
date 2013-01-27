package main

import (
    "crypto/md5"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
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
    RawBody   string
    Time      string
    CommentId string
}

type Entry struct {
    Author   string
    Title    string
    Date     string
    Body     string
    RawBody  string
    Url      string
    Tags     []*Tag
    Comments []*Comment
}

var dataset string
var testLoader func() []*Entry

func (e *Entry) HasTags() bool {
    return len(e.Tags) > 0
}

func (e *Entry) HasComments() bool {
    return len(e.Comments) > 0
}

func (e *Entry) NumComments() int {
    return len(e.Comments)
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
        var id int64
        var unixDate int64
        rows.Scan(&entry.Author, &id, &entry.Title, &unixDate,
            &entry.RawBody, &entry.Url)
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
        rows.Scan(&tag.TagName, &tag.TagUrl)
        tags = append(tags, tag)
    }
    return tags
}

func queryComments(db *sql.DB, postId int64) []*Comment {
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
        data.Scan(&comment.Name, &comment.Email, &comment.Website, &comment.Ip,
            &comment.CommentId, &unixDate, &comment.RawBody)
        hash := md5.New()
        hash.Write([]byte(strings.ToLower(comment.Email)))
        comment.EmailHash = fmt.Sprintf("%x", hash.Sum(nil))
        comment.Time = time.Unix(unixDate, 0).Format("2006-01-02 15:04")
        comment.Body = string(blackfriday.MarkdownCommon([]byte(comment.RawBody)))
        comments = append(comments, comment)
    }
    return comments
}

func render(ctx *web.Context, tmpl string, data map[string]interface{}) {
    html := mustache.RenderFile("tmpl/"+tmpl+".html.mustache", data)
    ctx.WriteString(html)
}

func xtractReferer(ctx *web.Context) string {
    referers := ctx.Request.Header["Referer"]
    if len(referers) == 0 {
        return ""
    }
    referer := referers[0]
    return referer[strings.LastIndex(referer, "/")+1:]
}

func handler(ctx *web.Context, path string) {
    posts := loadData(dataset)
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
    } else if path == "admin" {
        basicData["PageTitle"] = "Admin Console"
        render(ctx, "admin", basicData)
        return
    } else if path == "login" {
        basicData["RedirectTo"] = xtractReferer(ctx)
        basicData["PageTitle"] = "Login"
        render(ctx, "login", basicData)
        return
    } else if path == "logout" {
        ctx.SetSecureCookie("adminlogin", "", 0)
        ctx.Redirect(301, "/"+xtractReferer(ctx))
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
    redir := ctx.Params["redirect_to"]
    ctx.Redirect(301, "/"+redir)
}

func load_comments_handler(ctx *web.Context) {
    post := ctx.Params["post"]
    posts := loadData(dataset)
    for _, p := range posts {
        if p.Url == post {
            b, err := json.Marshal(p)
            if err != nil {
                fmt.Println(err.Error())
                return
            }
            ctx.WriteString(string(b))
            return
        }
    }
}

func delete_comment_handler(ctx *web.Context) {
    action := ctx.Params["action"]
    redir := ctx.Params["redirect_to"]
    id := ctx.Params["id"]
    if action == "delete" {
        db, err := sql.Open("sqlite3", dataset)
        if err != nil {
            fmt.Println(err.Error())
            return
        }
        defer db.Close()
        _, err = db.Exec(`delete from comment where id=?`, id)
        if err != nil {
            fmt.Println(err.Error())
            return
        }
    }
    ctx.Redirect(301, "/"+redir)
}

func moderate_comment_handler(ctx *web.Context) {
    action := ctx.Params["action"]
    redir := ctx.Params["redirect_to"]
    text := ctx.Params["text"]
    id := ctx.Params["id"]
    if action == "edit" {
        db, err := sql.Open("sqlite3", dataset)
        if err != nil {
            fmt.Println(err.Error())
            return
        }
        defer db.Close()
        _, err = db.Exec(`update comment set body=? where id=?`, text, id)
        if err != nil {
            fmt.Println(err.Error())
            return
        }
    }
    ctx.Redirect(301, "/"+redir)
}

func comment_handler(ctx *web.Context) {
    db, err := sql.Open("sqlite3", dataset)
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
    refUrl := xtractReferer(ctx)
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
    result, err := stmt.Exec(commenterId, postId, time.Now().Unix(), body)
    if err != nil {
        fmt.Println("Failed to insert comment: " + err.Error())
        ctx.Abort(500, "Server Error")
    }
    commentId, _ := result.LastInsertId()
    xaction.Commit()
    redir := fmt.Sprintf("/%s#comment-%d", refUrl, commentId)
    ctx.Redirect(301, redir)
}

func serve_favicon(ctx *web.Context) {
    http.ServeFile(ctx, ctx.Request, "static/snifter.png")
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
    web.Get("/load_comments", load_comments_handler)
    web.Get("/delete_comment", delete_comment_handler)
    web.Post("/moderate_comment", moderate_comment_handler)
    web.Get("/favicon.ico", serve_favicon)
    web.Get("/(.*)", handler)
    web.SetLogger(logger)
    web.Config.StaticDir = "static"
    web.Config.CookieSecret = "foobarbaz" // XXX: don't forget to change that!
    web.Run(":8080")
}

func loadData(db string) []*Entry {
    if testLoader != nil {
        return testLoader()
    }
    if db == "" {
        return nil
    }
    data, err := readDb(db)
    if err != nil {
        println(err.Error())
        return nil
    }
    return data
}

func main() {
    dataset = "testdata/foo.db"
    runServer()
}
