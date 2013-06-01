package main

import (
    "./util"
    "bufio"
    "database/sql"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "github.com/gorilla/feeds"
    "github.com/hoisie/web"
    "github.com/lye/mustache"
    _ "github.com/mattn/go-sqlite3"
    email "github.com/ungerik/go-mail"
)

type SrvConfig map[string]interface{}

var (
    db   *sql.DB
    conf SrvConfig
    data Data
)

func (c *SrvConfig) Get(key string) string {
    val, ok := (*c)[key].(string)
    if !ok {
        return ""
    }
    return val
}

func loadConfig(path string) (config SrvConfig) {
    b, err := ioutil.ReadFile(path)
    if err != nil {
        println("readconf: " + err.Error())
        return SrvConfig{}
    }
    err = json.Unmarshal(b, &config)
    if err != nil {
        println(err.Error())
        return SrvConfig{}
    }
    return
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
    numPages := numPosts / POSTS_PER_PAGE
    if numPosts%POSTS_PER_PAGE != 0 {
        numPages += 1
    }
    for p := 0; p < numPages; p++ {
        if p == currPage {
            list += fmt.Sprintf("%d\n", p+1)
        } else {
            list += fmt.Sprintf("<a href=\"/page/%d\">%d</a>\n", p+1, p+1)
        }
    }
    return
}

func renderPage(ctx *web.Context, path string, tmplData map[string]interface{}, data Data) {
    pgNo, err := strconv.Atoi(strings.Replace(path, "page/", "", -1))
    if err != nil {
        pgNo = 1
    }
    offset := (pgNo - 1) * POSTS_PER_PAGE
    tmplData["PageTitle"] = "Velkam"
    tmplData["entries"] = data.posts(POSTS_PER_PAGE, offset)
    tmplData["ListOfPages"] = listOfPages(data.numPosts(), pgNo-1)
    render(ctx, "main", tmplData)
}

func produceFeedXml(ctx *web.Context, posts []*Entry) {
    url := conf.Get("url") + conf.Get("port")
    blogTitle := conf.Get("blog_title")
    descr := conf.Get("blog_descr")
    author := conf.Get("author")
    authorEmail := conf.Get("email")
    now := time.Now()
    feed := &feeds.Feed{
        Title:       blogTitle,
        Link:        &feeds.Link{Href: url},
        Description: descr,
        Author:      &feeds.Author{author, authorEmail},
        Created:     now,
    }
    for _, p := range posts {
        item := feeds.Item{
            Title:       p.Title,
            Link:        &feeds.Link{Href: p.Url},
            Description: p.Body,
            Author:      &feeds.Author{p.Author, authorEmail},
            Created:     now,
        }
        feed.Items = append(feed.Items, &item)
    }
    rss, err := feed.ToRss()
    if err != nil {
        fmt.Println(err.Error())
    }
    ctx.WriteString(rss)
}

func getPostByUrl(ctx *web.Context, data Data, url string) *Entry {
    if post := data.post(url); post != nil {
        return post
    }
    ctx.NotFound("Page not found: " + url)
    return nil
}

func handler(ctx *web.Context, path string) {
    numTotalPosts := data.numPosts()
    var basicData = map[string]interface{}{
        "PageTitle":       "",
        "NeedPagination":  numTotalPosts > POSTS_PER_PAGE,
        "ListOfPages":     listOfPages(numTotalPosts, 0),
        "entries":         data.posts(POSTS_PER_PAGE, 0),
        "sidebar_entries": data.titles(NUM_RECENT_POSTS),
    }
    value, found := ctx.GetSecureCookie("adminlogin")
    basicData["AdminLogin"] = found && value == "yesplease"
    if strings.HasPrefix(path, "page/") {
        renderPage(ctx, path, basicData, data)
        return
    }
    switch path {
    case "":
        basicData["PageTitle"] = "Velkam"
        render(ctx, "main", basicData)
        return
    case "admin":
        basicData["PageTitle"] = "Admin Console"
        render(ctx, "admin", basicData)
        return
    case "archive":
        basicData["PageTitle"] = "Archive"
        basicData["all_entries"] = data.titles(-1)
        render(ctx, "archive", basicData)
        return
    case "login":
        referer := xtractReferer(ctx)
        if referer == "login" {
            basicData["LoginFailed"] = true
        } else {
            basicData["RedirectTo"] = referer
        }
        basicData["PageTitle"] = "Login"
        render(ctx, "login", basicData)
        return
    case "logout":
        ctx.SetSecureCookie("adminlogin", "", 0)
        ctx.Redirect(http.StatusFound, "/"+xtractReferer(ctx))
        return
    case "edit_post":
        basicData["PageTitle"] = "Edit Post"
        if post := getPostByUrl(ctx, data, ctx.Params["post"]); post != nil {
            basicData["Title"] = post.Title
            basicData["Url"] = post.Url
            basicData["TagsWithUrls"] = post.TagsWithUrls()
            basicData["RawBody"] = post.RawBody
            render(ctx, "edit_post", basicData)
        }
        return
    case "load_comments":
        if post := getPostByUrl(ctx, data, ctx.Params["post"]); post != nil {
            b, err := json.Marshal(post)
            if err != nil {
                fmt.Println(err.Error())
                return
            }
            ctx.WriteString(string(b))
        }
        return
    case "feed.xml":
        produceFeedXml(ctx, data.posts(NUM_FEED_ITEMS, 0))
        return
    default:
        if post := getPostByUrl(ctx, data, path); post != nil {
            basicData["PageTitle"] = post.Title
            basicData["entry"] = post
            render(ctx, "post", basicData)
        }
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
    row := db.QueryRow(`select salt, passwd, full_name, email, www
                        from author where disp_name=?`, uname)
    var salt, passwdHash, fullName, email, www string
    err := row.Scan(&salt, &passwdHash, &fullName, &email, &www)
    if err == sql.ErrNoRows {
        ctx.Redirect(http.StatusFound, "/login")
        return
    }
    if err != nil {
        fmt.Println(err.Error())
        ctx.Redirect(http.StatusFound, "/login")
        return
    }
    passwd := ctx.Request.Form["passwd"][0]
    hash := util.SaltAndPepper(salt, passwd)
    if hash == passwdHash {
        ctx.SetSecureCookie("adminlogin", "yesplease", 3600*24)
        redir := ctx.Params["redirect_to"]
        if redir == "login" {
            redir = ""
        }
        ctx.Redirect(http.StatusFound, "/"+redir)
    } else {
        ctx.Redirect(http.StatusFound, "/login")
    }
}

func delete_comment_handler(ctx *web.Context) {
    action := ctx.Params["action"]
    redir := ctx.Params["redirect_to"]
    id := ctx.Params["id"]
    if action == "delete" {
        _, err := db.Exec(`delete from comment where id=?`, id)
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
        _, err := db.Exec(`update comment set body=? where id=?`, text, id)
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
    url := conf.Get("url") + conf.Get("port") + redir
    name := ctx.Params["name"]
    email := ctx.Params["email"]
    website := ctx.Params["website"]
    go SendEmail(name, email, website, body, url, refUrl)
    ctx.Redirect(http.StatusFound, redir)
}

func SendEmail(author, mail, www, comment, url, postTitle string) {
    gmailSenderAcct := conf.Get("notif_sender_acct")
    gmailSenderPasswd := conf.Get("notif_sender_passwd")
    notifee := conf.Get("email")
    err := email.InitGmail(gmailSenderAcct, gmailSenderPasswd)
    if err != nil {
        println("err initing gmail: ", err.Error())
        return
    }
    format := "\n\nNew comment from %s <%s> (%s):\n\n%s\n\nURL: %s"
    message := fmt.Sprintf(format, author, mail, www, comment, url)
    subj := fmt.Sprintf("New comment in '%s'", postTitle)
    mess := email.NewBriefMessageFrom(subj, message, gmailSenderAcct, notifee)
    err = mess.Send()
    if err != nil {
        println("err sending email: ", err.Error())
        return
    }
}

func serve_favicon(ctx *web.Context) {
    http.ServeFile(ctx, ctx.Request, conf.Get("favicon"))
}

func runServer(_data Data) {
    data = _data
    f, err := os.Create(conf.Get("log"))
    if err != nil {
        println("create log: " + err.Error())
        return
    }
    logger := log.New(f, "", log.Ldate|log.Ltime)
    web.Post("/comment_submit", comment_handler)
    web.Post("/login_submit", login_handler)
    web.Get("/delete_comment", delete_comment_handler)
    web.Post("/moderate_comment", moderate_comment_handler)
    web.Post("/submit_post", submit_post_handler)
    web.Post("/upload_images", upload_image_handler)
    web.Get("/favicon.ico", serve_favicon)
    web.Get("/(.*)", handler)
    web.SetLogger(logger)
    web.Config.StaticDir = conf.Get("staticdir")
    web.Config.CookieSecret = conf.Get("cookie_secret")
    web.Run(conf.Get("port"))
}

func openDb(dbFile string) *sql.DB {
    db, err := sql.Open("sqlite3", dbFile)
    if err != nil {
        fmt.Println("sql: " + err.Error())
        return nil
    }
    return db
}

func main() {
    root, _ := filepath.Split(filepath.Clean(os.Args[0]))
    conf = loadConfig(filepath.Join(root, "server.conf"))
    db = openDb(conf.Get("database"))
    defer db.Close()
    runServer(&DbData{})
}
