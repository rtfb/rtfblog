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
        pgNo, err := strconv.Atoi(strings.Replace(path, "page/", "", -1))
        if err != nil {
            pgNo = 1
        }
        offset := (pgNo - 1) * POSTS_PER_PAGE
        if offset > 0 {
            basicData["entries"] = data.posts(POSTS_PER_PAGE, offset)
        }
        basicData["PageTitle"] = "Velkam"
        basicData["ListOfPages"] = listOfPages(numTotalPosts, pgNo-1)
        render(ctx, "main", basicData)
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
        url := ctx.Params["post"]
        if url != "" {
            if post := data.post(url); post != nil {
                basicData["Title"] = post.Title
                basicData["Url"] = post.Url
                basicData["TagsWithUrls"] = post.TagsWithUrls()
                basicData["RawBody"] = post.RawBody
            }
        }
        render(ctx, "edit_post", basicData)
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

func login_handler(ctx *web.Context) {
    uname := ctx.Params["uname"]
    a, err := data.author(uname)
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
    hash := util.SaltAndPepper(a.Salt, passwd)
    if hash == a.Passwd {
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
    if action == "delete" && !data.deleteComment(id) {
        return
    }
    ctx.Redirect(http.StatusFound, "/"+redir)
}

func moderate_comment_handler(ctx *web.Context) {
    action := ctx.Params["action"]
    redir := ctx.Params["redirect_to"]
    text := ctx.Params["text"]
    id := ctx.Params["id"]
    if action == "edit" && !data.updateComment(id, text) {
        return
    }
    ctx.Redirect(http.StatusFound, "/"+redir)
}

func submit_post_handler(ctx *web.Context) {
    title := ctx.Params["title"]
    url := ctx.Params["url"]
    tagsWithUrls := ctx.Params["tags"]
    text := ctx.Params["text"]
    postId, idErr := data.postId(url)
    if !data.begin() {
        return
    }
    if idErr != nil {
        if idErr == sql.ErrNoRows {
            authorId := int64(1) // XXX: it's only me now
            newPostId, err := data.insertPost(authorId, title, url, text)
            if err != nil {
                ctx.Abort(http.StatusInternalServerError, "Server Error")
                data.rollback()
                return
            }
            postId = newPostId
        } else {
            fmt.Println("data.postId() failed: " + idErr.Error())
            ctx.Abort(http.StatusInternalServerError, "Server Error")
            data.rollback()
            return
        }
    } else {
        if !data.updatePost(postId, title, url, text) {
            ctx.Abort(http.StatusInternalServerError, "Server Error")
            data.rollback()
            return
        }
    }
    data.updateTags(explodeTags(tagsWithUrls), postId)
    data.commit()
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
    refUrl := xtractReferer(ctx)
    postId, err := data.postId(refUrl)
    if err != nil {
        fmt.Println("data.postId() failed: " + err.Error())
        ctx.Abort(http.StatusInternalServerError, "Server Error")
        return
    }
    if !data.begin() {
        return
    }
    name := ctx.Params["name"]
    email := ctx.Params["email"]
    website := ctx.Params["website"]
    ip := ctx.Request.RemoteAddr
    commenterId, err := data.selOrInsCommenter(name, email, website, ip)
    if err != nil {
        fmt.Println("data.selOrInsCommenter() failed: " + err.Error())
        ctx.Abort(http.StatusInternalServerError, "Server Error")
        data.rollback()
        return
    }
    body := ctx.Params["text"]
    commentId, err := data.insertComment(commenterId, postId, body)
    if err != nil {
        ctx.Abort(http.StatusInternalServerError, "Server Error")
        data.rollback()
        return
    }
    data.commit()
    redir := fmt.Sprintf("/%s#comment-%d", refUrl, commentId)
    url := conf.Get("url") + conf.Get("port") + redir
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

func main() {
    root, _ := filepath.Split(filepath.Clean(os.Args[0]))
    conf = loadConfig(filepath.Join(root, "server.conf"))
    db, err := sql.Open("sqlite3", conf.Get("database"))
    if err != nil {
        fmt.Println("sql: " + err.Error())
        return
    }
    defer db.Close()
    runServer(&DbData{db, nil})
}
