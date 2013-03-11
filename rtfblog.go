package main

import (
    "bufio"
    "crypto/md5"
    "database/sql"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "mime/multipart"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/hoisie/web"
    "github.com/lye/mustache"
    _ "github.com/mattn/go-sqlite3"
    "github.com/russross/blackfriday"
)

const (
    MAX_FILE_SIZE = 50 * 1024 * 1024 // bytes
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

var (
    dataset    string
    testLoader func() []*Entry
)

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

func (e *Entry) TagsWithUrls() string {
    parts := make([]string, 0)
    for _, t := range e.Tags {
        part := fmt.Sprintf("%s", t.TagName)
        if t.TagUrl != t.TagName {
            part = fmt.Sprintf("%s>%s", t.TagName, t.TagUrl)
        }
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

func listOfPages(numPosts, currPage int) (list string) {
    numPages := numPosts / 5
    for p := 0; p < numPages; p++ {
        if p == currPage {
            list += fmt.Sprintf("%d\n", p+1)
        } else {
            list += fmt.Sprintf("<a href=\"/page/%d\">%d</a>\n", p+1, p+1)
        }
    }
    return
}

func handler(ctx *web.Context, path string) {
    posts := loadData(dataset)
    postsPerPage := 5
    if postsPerPage >= len(posts) {
        postsPerPage = len(posts)
    }
    recentPosts := 10
    if recentPosts >= len(posts) {
        recentPosts = len(posts)
    }
    var basicData = map[string]interface{}{
        "PageTitle":       "",
        "NeedPagination":  len(posts) > 5,
        "ListOfPages":     listOfPages(len(posts), 0),
        "entries":         posts[:postsPerPage],
        "sidebar_entries": posts[:recentPosts],
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
    } else if path == "archive" {
        basicData["PageTitle"] = "Archive"
        basicData["all_entries"] = posts
        render(ctx, "archive", basicData)
        return
    } else if path == "login" {
        referer := xtractReferer(ctx)
        if referer == "login" {
            basicData["LoginFailed"] = true
        } else {
            basicData["RedirectTo"] = referer
        }
        basicData["PageTitle"] = "Login"
        render(ctx, "login", basicData)
        return
    } else if path == "logout" {
        ctx.SetSecureCookie("adminlogin", "", 0)
        ctx.Redirect(http.StatusFound, "/"+xtractReferer(ctx))
        return
    } else if path == "edit_post" {
        basicData["PageTitle"] = "Edit Post"
        post := ctx.Params["post"]
        for _, e := range posts {
            if e.Url == post {
                basicData["Title"] = e.Title
                basicData["Url"] = e.Url
                basicData["TagsWithUrls"] = e.TagsWithUrls()
                basicData["RawBody"] = e.RawBody
            }
        }
        render(ctx, "edit_post", basicData)
        return
    } else if strings.HasPrefix(path, "page/") {
        pgNo, err := strconv.Atoi(strings.Replace(path, "page/", "", -1))
        if err != nil {
            pgNo = 1
        }
        lwr := (pgNo - 1) * 5
        upr := pgNo * 5
        if lwr >= len(posts) {
            lwr = 0
        }
        if upr >= len(posts) {
            upr = len(posts)
        }
        basicData["PageTitle"] = "Velkam"
        basicData["entries"] = posts[lwr:upr]
        basicData["ListOfPages"] = listOfPages(len(posts), pgNo-1)
        render(ctx, "main", basicData)
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
    ctx.Abort(http.StatusInternalServerError, "Server Error")
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
    passwd := ctx.Request.Form["passwd"][0]
    if uname == "admin" && passwd == "asd" {
        ctx.SetSecureCookie("adminlogin", "yesplease", int64(time.Hour*24))
        redir := ctx.Params["redirect_to"]
        if redir == "login" {
            redir = ""
        }
        ctx.Redirect(http.StatusFound, "/"+redir)
    } else {
        ctx.Params["HiddenShite"] = "foo"
        ctx.Redirect(http.StatusFound, "/login")
    }
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
    ctx.Redirect(http.StatusFound, "/"+redir)
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
    ctx.Redirect(http.StatusFound, "/"+redir)
}

func submit_post_handler(ctx *web.Context) {
    title := ctx.Params["title"]
    url := ctx.Params["url"]
    tagsWithUrls := ctx.Params["tags"]
    text := ctx.Params["text"]
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
    postId, err := getPostId(xaction, url)
    if err != nil {
        if err == sql.ErrNoRows {
            insertPostSql, _ := xaction.Prepare(`insert into post
                                                 (author_id, title, date,
                                                  url, body)
                                                 values (?, ?, ?, ?, ?)`)
            defer insertPostSql.Close()
            authorId := 1 // XXX: it's only me now
            date := time.Now().Unix()
            result, err := insertPostSql.Exec(authorId, title, date, url, text)
            if err != nil {
                fmt.Println("Failed to insert post: " + err.Error())
                ctx.Abort(http.StatusInternalServerError, "Server Error")
                return
            }
            postId, _ = result.LastInsertId()
        } else {
            fmt.Println("getPostId failed: " + err.Error())
            ctx.Abort(http.StatusInternalServerError, "Server Error")
            return
        }
    }
    updateStmt, _ := xaction.Prepare(`update post set title=?, url=?, body=?
                                      where id=?`)
    defer updateStmt.Close()
    _, err = updateStmt.Exec(title, url, text, postId)
    if err != nil {
        fmt.Println(err.Error())
        ctx.Abort(http.StatusInternalServerError, "Server Error")
        return
    }
    updateTags(xaction, tagsWithUrls, postId)
    xaction.Commit()
    ctx.Redirect(http.StatusFound, "/"+url)
}

func explodeTags(tagsWithUrls string) []*Tag {
    tags := make([]*Tag, 0)
    for _, t := range strings.Split(tagsWithUrls, ",") {
        t = strings.TrimSpace(t)
        if t == "" {
            continue
        }
        tag, url := t, strings.ToLower(t)
        if strings.Contains(t, ">") {
            arr := strings.Split(t, ">")
            tag, url = arr[0], arr[1]
        }
        tags = append(tags, &Tag{url, tag})
    }
    return tags
}

func updateTags(xaction *sql.Tx, tagsWithUrls string, postId int64) {
    delStmt, _ := xaction.Prepare("delete from tagmap where post_id=?")
    defer delStmt.Close()
    delStmt.Exec(postId)
    for _, t := range explodeTags(tagsWithUrls) {
        tagId, _ := insertOrGetTagId(xaction, t)
        updateTagMap(xaction, postId, tagId)
    }
}

func insertOrGetTagId(xaction *sql.Tx, tag *Tag) (tagId int64, err error) {
    query, _ := xaction.Prepare("select id from tag where url=?")
    defer query.Close()
    err = query.QueryRow(tag.TagUrl).Scan(&tagId)
    switch err {
    case nil:
        return
    case sql.ErrNoRows:
        insertTagSql, _ := xaction.Prepare(`insert into tag
                                            (name, url)
                                            values (?, ?)`)
        defer insertTagSql.Close()
        result, err := insertTagSql.Exec(tag.TagName, tag.TagUrl)
        if err != nil {
            fmt.Println("Failed to insert commenter: " + err.Error())
        }
        return result.LastInsertId()
    default:
        fmt.Printf("err: %s", err.Error())
        return -1, sql.ErrNoRows
    }
    return -1, sql.ErrNoRows
}

func updateTagMap(xaction *sql.Tx, postId int64, tagId int64) {
    stmt, _ := xaction.Prepare(`insert into tagmap
                                (tag_id, post_id)
                                values (?, ?)`)
    defer stmt.Close()
    stmt.Exec(tagId, postId)
}

func upload_image_handler(ctx *web.Context) {
    mr, _ := ctx.Request.MultipartReader()
    files := ""
    part, err := mr.NextPart()
    for err == nil {
        if name := part.FormName(); name != "" {
            if part.FileName() != "" {
                fmt.Printf("filename: %s\n", part.FileName())
                files += fmt.Sprintf("[foo]: /%s", part.FileName())
                handleUpload(ctx.Request, part)
            }
        }
        part, err = mr.NextPart()
    }
    ctx.WriteString(files)
    return
}

func handleUpload(r *http.Request, p *multipart.Part) {
    defer func() {
        if rec := recover(); rec != nil {
            log.Println(rec)
        }
    }()
    lr := &io.LimitedReader{R: p, N: MAX_FILE_SIZE + 1}
    filename := "static/" + p.FileName()
    fo, err := os.Create(filename)
    if err != nil {
        fmt.Printf("err writing %q!, err = %s", filename, err.Error())
    }
    defer fo.Close()
    w := bufio.NewWriter(fo)
    _, err = io.Copy(w, lr)
    if err != nil {
        fmt.Printf("err writing %q!, err = %s", filename, err.Error())
    }
    if err = w.Flush(); err != nil {
        fmt.Printf("err flushing writer for %q!, err = %s", filename, err.Error())
    }
    return
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
        ctx.Abort(http.StatusInternalServerError, "Server Error")
        return
    }
    refUrl := xtractReferer(ctx)
    postId, err := getPostId(xaction, refUrl)
    if err != nil {
        fmt.Println("getPostId failed: " + err.Error())
        ctx.Abort(http.StatusInternalServerError, "Server Error")
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
        ctx.Abort(http.StatusInternalServerError, "Server Error")
    }
    commentId, _ := result.LastInsertId()
    xaction.Commit()
    redir := fmt.Sprintf("/%s#comment-%d", refUrl, commentId)
    ctx.Redirect(http.StatusFound, redir)
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
    web.Post("/submit_post", submit_post_handler)
    web.Post("/upload_images", upload_image_handler)
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
