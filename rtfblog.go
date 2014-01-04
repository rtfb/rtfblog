package main

import (
    "bufio"
    "bytes"
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
    _ "github.com/lib/pq"
    "github.com/lye/mustache"
    email "github.com/ungerik/go-mail"
)

type SrvConfig map[string]interface{}

var (
    conf   SrvConfig
    data   Data
    logger *log.Logger
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
        logger.Println(err.Error())
    }
    ctx.WriteString(rss)
}

func getPostByUrl(ctx *web.Context, data Data, url string) *Entry {
    if post := data.post(url); post != nil {
        return post
    }
    html := mustache.RenderFile("tmpl/404.html.mustache", map[string]interface{}{})
    ctx.NotFound(html)
    return nil
}

func handler(ctx *web.Context, path string) {
    value, found := ctx.GetSecureCookie("adminlogin")
    adminLogin := found && value == "yesplease"
    data.hiddenPosts(adminLogin)
    numTotalPosts := data.numPosts()
    var basicData = map[string]interface{}{
        "PageTitle":       "",
        "BlogTitle":       conf.Get("blog_title"),
        "BlogSubtitle":    conf.Get("blog_descr"),
        "NeedPagination":  numTotalPosts > POSTS_PER_PAGE,
        "ListOfPages":     listOfPages(numTotalPosts, 0),
        "entries":         data.posts(POSTS_PER_PAGE, 0),
        "sidebar_entries": data.titles(NUM_RECENT_POSTS),
    }
    basicData["AdminLogin"] = adminLogin
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
    if strings.HasPrefix(path, "tag/") {
        tag := path[4:]
        heading := "Posts tagged '" + tag + "'"
        basicData["PageTitle"] = heading
        basicData["HeadingText"] = heading + ":"
        basicData["all_entries"] = data.titlesByTag(tag)
        render(ctx, "archive", basicData)
        return
    }
    switch path {
    case "":
        basicData["PageTitle"] = "Velkam"
        render(ctx, "main", basicData)
        return
    case "admin":
        if !adminLogin {
            ctx.Abort(http.StatusForbidden, "Verboten")
            return
        }
        basicData["PageTitle"] = "Admin Console"
        render(ctx, "admin", basicData)
        return
    case "archive":
        basicData["PageTitle"] = "Archive"
        basicData["HeadingText"] = "All posts:"
        basicData["all_entries"] = data.titles(-1)
        render(ctx, "archive", basicData)
        return
    case "all_comments":
        if !adminLogin {
            ctx.Abort(http.StatusForbidden, "Verboten")
            return
        }
        basicData["PageTitle"] = "All Comments"
        basicData["all_comments"] = data.allComments()
        render(ctx, "all_comments", basicData)
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
        if !adminLogin {
            ctx.Abort(http.StatusForbidden, "Verboten")
            return
        }
        basicData["PageTitle"] = "Edit Post"
        basicData["IsHidden"] = true // Assume hidden for a new post
        url := ctx.Params["post"]
        if url != "" {
            if post := data.post(url); post != nil {
                basicData["IsHidden"] = post.Hidden
                basicData["post"] = post
            }
        } else {
            basicData["post"] = Entry{}
        }
        render(ctx, "edit_post", basicData)
        return
    case "load_comments":
        if !adminLogin {
            ctx.Abort(http.StatusForbidden, "Verboten")
            return
        }
        if post := getPostByUrl(ctx, data, ctx.Params["post"]); post != nil {
            b, err := json.Marshal(post)
            if err != nil {
                logger.Println(err.Error())
                return
            }
            ctx.WriteString(string(b))
        }
        return
    case "feeds/rss.xml":
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
        logger.Println(err.Error())
        ctx.Redirect(http.StatusFound, "/login")
        return
    }
    passwd := ctx.Request.Form["passwd"][0]
    hash := SaltAndPepper(a.Salt, passwd)
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
    ctx.Params["passwd"] = "***"
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

func delete_post_handler(ctx *web.Context) {
    if !data.deletePost(ctx.Params["id"]) {
        return
    }
    ctx.Redirect(http.StatusFound, "/admin")
}

func moderate_comment_handler(ctx *web.Context) {
    action := ctx.Params["action"]
    text := ctx.Params["edit-comment-text"]
    id := ctx.Params["id"]
    if action == "edit" && !data.updateComment(id, text) {
        return
    }
    redir := ctx.Params["redirect_to"]
    ctx.Redirect(http.StatusFound, fmt.Sprintf("/%s#comment-%s", redir, id))
}

func submit_post_handler(ctx *web.Context) {
    tagsWithUrls := ctx.Params["tags"]
    url := ctx.Params["url"]
    e := Entry{
        EntryLink: EntryLink{
            Title:  ctx.Params["title"],
            Url:    url,
            Hidden: ctx.Params["hidden"] == "on",
        },
        Body: ctx.Params["text"],
    }
    postId, idErr := data.postId(url)
    if !data.begin() {
        return
    }
    if idErr != nil {
        if idErr == sql.ErrNoRows {
            authorId := int64(1) // XXX: it's only me now
            newPostId, err := data.insertPost(authorId, &e)
            if err != nil {
                ctx.Abort(http.StatusInternalServerError, "Server Error")
                data.rollback()
                return
            }
            postId = newPostId
        } else {
            logger.Println("data.postId() failed: " + idErr.Error())
            ctx.Abort(http.StatusInternalServerError, "Server Error")
            data.rollback()
            return
        }
    } else {
        if !data.updatePost(postId, &e) {
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
            logger.Println(rec)
        }
    }()
    lr := &io.LimitedReader{R: p, N: MAX_FILE_SIZE + 1}
    filename := "static/" + p.FileName()
    fo, err := os.Create(filename)
    if err != nil {
        logger.Printf("err writing %q!, err = %s\n", filename, err.Error())
    }
    defer fo.Close()
    w := bufio.NewWriter(fo)
    _, err = io.Copy(w, lr)
    if err != nil {
        logger.Printf("err writing %q!, err = %s\n", filename, err.Error())
    }
    if err = w.Flush(); err != nil {
        logger.Printf("err flushing writer for %q!, err = %s\n", filename, err.Error())
    }
    return
}

func wrongCaptchaReply(ctx *web.Context, status string) {
    var response = map[string]interface{}{
        "status":  status,
        "name":    ctx.Params["name"],
        "email":   ctx.Params["email"],
        "website": ctx.Params["website"],
        "body":    ctx.Params["text"],
    }
    b, err := json.Marshal(response)
    if err != nil {
        logger.Println(err.Error())
        return
    }
    ctx.WriteString(string(b))
}

func rightCaptchaReply(ctx *web.Context, redir string) {
    var response = map[string]interface{}{
        "status": "accepted",
        "redir":  redir,
    }
    b, err := json.Marshal(response)
    if err != nil {
        logger.Println(err.Error())
        return
    }
    ctx.WriteString(string(b))
}

func detectLanguage(text string) string {
    var rq = map[string]string{
        "document": text,
    }
    b, err := json.Marshal(rq)
    if err != nil {
        logger.Println(err.Error())
        return ""
    }
    url := "https://services.open.xerox.com/RestOp/LanguageIdentifier/GetLanguageForString"
    client := &http.Client{}
    req, err := http.NewRequest("POST", url, bytes.NewReader(b))
    // XXX: the docs say I need to specify Content-Length, but in practice I
    // see that it works without it:
    //req.Header.Add("Content-Length", fmt.Sprintf("%d", len(string(b))))
    req.Header.Add("Content-Type", "application/json; charset=utf-8")
    resp, err := client.Do(req)
    if err != nil {
        logger.Println(err.Error())
        return ""
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        logger.Println(err.Error())
        return ""
    }
    return string(body)
}

func detectLanguageWithTimeout(text string) string {
    c := make(chan string, 1)
    go func() {
        c <- detectLanguage(text)
    }()
    select {
    case lang := <-c:
        return lang
    case <-time.After(1500 * time.Millisecond):
        return "timedout"
    }
}

func publishCommentWithInsert(ctx *web.Context, postId int64, refUrl string) string {
    if !data.begin() {
        return ""
    }
    ip := ctx.Request.RemoteAddr
    name := ctx.Params["name"]
    email := ctx.Params["email"]
    website := ctx.Params["website"]
    commenterId, err := data.insertCommenter(name, email, website, ip)
    if err != nil {
        logger.Println("data.insertCommenter() failed: " + err.Error())
        ctx.Abort(http.StatusInternalServerError, "Server Error")
        data.rollback()
        return ""
    }
    body := ctx.Params["text"]
    commentId, err := data.insertComment(commenterId, postId, body)
    if err != nil {
        ctx.Abort(http.StatusInternalServerError, "Server Error")
        data.rollback()
        return ""
    }
    data.commit()
    return fmt.Sprintf("#comment-%d", commentId)
}

func publishComment(ctx *web.Context, postId, commenterId int64, refUrl string) string {
    if !data.begin() {
        return ""
    }
    body := ctx.Params["text"]
    commentId, err := data.insertComment(commenterId, postId, body)
    if err != nil {
        ctx.Abort(http.StatusInternalServerError, "Server Error")
        data.rollback()
        return ""
    }
    data.commit()
    return fmt.Sprintf("#comment-%d", commentId)
}

func comment_handler(ctx *web.Context) {
    refUrl := xtractReferer(ctx)
    postId, err := data.postId(refUrl)
    if err != nil {
        logger.Println("data.postId() failed: " + err.Error())
        ctx.Abort(http.StatusInternalServerError, "Server Error")
        return
    }
    ip := ctx.Request.RemoteAddr
    name := ctx.Params["name"]
    email := ctx.Params["email"]
    website := ctx.Params["website"]
    commenterId, err := data.commenter(name, email, website, ip)
    redir := ""
    if err == nil {
        // This is a returning commenter, pass his comment through:
        redir = "/" + refUrl + publishComment(ctx, postId, commenterId, refUrl)
    } else if err == sql.ErrNoRows {
        body := ctx.Params["text"]
        lang := detectLanguageWithTimeout(body)
        log := fmt.Sprintf("Detected language: %q for text %q", lang, body)
        logger.Println(log)
        if lang == "\"lt\"" {
            redir = "/" + refUrl + publishCommentWithInsert(ctx, postId, refUrl)
        } else {
            captcha := ctx.Params["captcha"]
            if captcha == "" {
                wrongCaptchaReply(ctx, "showcaptcha")
                return
            }
            if captcha != "dvylika" {
                wrongCaptchaReply(ctx, "rejected")
                return
            } else {
                redir = "/" + refUrl + publishCommentWithInsert(ctx, postId, refUrl)
            }
        }
    } else {
        logger.Println("err: " + err.Error())
        wrongCaptchaReply(ctx, "rejected")
        return
    }
    url := conf.Get("url") + conf.Get("port") + redir
    if conf.Get("notif_send_email") == "true" {
        go SendEmail(ctx.Params["name"], ctx.Params["email"],
            ctx.Params["website"], ctx.Params["text"], url, refUrl)
    }
    rightCaptchaReply(ctx, redir)
    return
}

func SendEmail(author, mail, www, comment, url, postTitle string) {
    gmailSenderAcct := conf.Get("notif_sender_acct")
    gmailSenderPasswd := conf.Get("notif_sender_passwd")
    notifee := conf.Get("email")
    err := email.InitGmail(gmailSenderAcct, gmailSenderPasswd)
    if err != nil {
        logger.Println("err initing gmail: ", err.Error())
        return
    }
    format := "\n\nNew comment from %s <%s> (%s):\n\n%s\n\nURL: %s"
    message := fmt.Sprintf(format, author, mail, www, comment, url)
    subj := fmt.Sprintf("New comment in '%s'", postTitle)
    mess := email.NewBriefMessageFrom(subj, message, gmailSenderAcct, notifee)
    err = mess.Send()
    if err != nil {
        logger.Println("err sending email: ", err.Error())
        return
    }
}

func serve_favicon(ctx *web.Context) {
    http.ServeFile(ctx, ctx.Request, conf.Get("favicon"))
}

func checkAdmin(handler func(ctx *web.Context)) func(ctx *web.Context) {
    return func(ctx *web.Context) {
        value, found := ctx.GetSecureCookie("adminlogin")
        adminLogin := found && value == "yesplease"
        if !adminLogin {
            ctx.Abort(http.StatusForbidden, "Verboten")
            return
        }
        handler(ctx)
    }
}

func runServer(_data Data) {
    data = _data
    web.Get("/comment_submit", comment_handler)
    web.Post("/login_submit", login_handler)
    web.Get("/delete_comment", checkAdmin(delete_comment_handler))
    web.Get("/delete_post", checkAdmin(delete_post_handler))
    web.Post("/moderate_comment", checkAdmin(moderate_comment_handler))
    web.Post("/submit_post", checkAdmin(submit_post_handler))
    web.Post("/upload_images", checkAdmin(upload_image_handler))
    web.Get("/favicon.ico", serve_favicon)
    web.Get("/(.*)", handler)
    web.SetLogger(logger)
    web.Config.StaticDir = conf.Get("staticdir")
    web.Config.CookieSecret = conf.Get("cookie_secret")
    web.Run(conf.Get("port"))
}

func obtainConfiguration() SrvConfig {
    hardcodedConf := SrvConfig{}
    conf := hardcodedConf
    basedir, _ := filepath.Split(filepath.Clean(os.Args[0]))
    home, err := GetHomeDir()
    if err != nil {
        fmt.Println("Error acquiring user home dir. That can't be good.")
        fmt.Println("Err = %q", err.Error())
    }
    // Read the most generic config first, then more specific, each latter will
    // override the former values:
    confPaths := []string{
        "/etc/rtfblogrc",
        filepath.Join(home, ".rtfblogrc"),
        filepath.Join(basedir, ".rtfblogrc"),
        filepath.Join(basedir, "server.conf"),
    }
    for _, p := range confPaths {
        exists, err := FileExists(p)
        if err != nil {
            fmt.Printf("Can't check %q for existence, skipping...", p)
            continue
        }
        if exists {
            for k, v := range loadConfig(p) {
                conf[k] = v
            }
        }
    }
    return conf
}

func main() {
    conf = obtainConfiguration()
    logger = MkLogger(conf.Get("log"))
    db, err := sql.Open("postgres", conf.Get("database"))
    if err != nil {
        logger.Println("sql: " + err.Error())
        return
    }
    defer db.Close()
    runServer(&DbData{db, nil, false})
}
