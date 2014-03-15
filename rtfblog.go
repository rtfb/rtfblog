package main

import (
    "bufio"
    "database/sql"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "math/rand"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "github.com/gorilla/feeds"
    "github.com/gorilla/pat"
    "github.com/gorilla/sessions"
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
        return SrvConfig{}
    }
    err = json.Unmarshal(b, &config)
    if err != nil {
        println(err.Error())
        return SrvConfig{}
    }
    return
}

func render(w http.ResponseWriter, tmpl string, data map[string]interface{}) {
    html := mustache.RenderFile(fmt.Sprintf("tmpl/%s.html.mustache", tmpl), data)
    w.Write([]byte(html))
}

func xtractReferer(req *http.Request) string {
    referers := req.Header["Referer"]
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

func produceFeedXml(w http.ResponseWriter, posts []*Entry) {
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
    w.Write([]byte(rss))
}

func Home(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    if req.URL.Path == "/" {
        render(w, "main", MkBasicData(ctx, 0, 0))
        return nil
    }
    path := req.URL.Path[1:]
    if path == "robots.txt" {
        http.ServeFile(w, req, filepath.Join(conf.Get("staticdir"), "robots.txt"))
        return nil
    }
    if post := data.post(path); post != nil {
        SetNextTask(-1)
        tmplData := MkBasicData(ctx, 0, 0)
        tmplData["PageTitle"] = post.Title
        tmplData["entry"] = post
        tmplData["CaptchaHtml"] = CaptchaHtml(GetTask())
        render(w, "post", tmplData)
    } else {
        return PerformStatus(w, req, http.StatusNotFound)
    }
    return nil
}

func PageNum(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    pgNo, err := strconv.Atoi(req.URL.Query().Get(":pageNo"))
    if err != nil {
        pgNo = 1
        err = nil
    }
    offset := (pgNo - 1) * POSTS_PER_PAGE
    render(w, "main", MkBasicData(ctx, pgNo, offset))
    return nil
}

func Admin(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    tmplData := MkBasicData(ctx, 0, 0)
    tmplData["PageTitle"] = "Admin Console"
    render(w, "admin", tmplData)
    return nil
}

func LoginForm(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    flashes := ctx.Session.Flashes()
    html := ""
    // TODO: extract that to separate flashes template
    format := `<p><strong style="color: red">
%s
</strong></p>`
    if len(flashes) > 0 {
        for _, f := range flashes {
            html = html + fmt.Sprintf(format, f)
        }
    }
    return Tmpl("login.html").Execute(w, map[string]interface{}{
        "Flashes": html,
    })
}

func Logout(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    delete(ctx.Session.Values, "adminlogin")
    http.Redirect(w, req, reverse("home_page"), http.StatusSeeOther)
    return nil
}

func PostsWithTag(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    tag := req.URL.Query().Get(":tag")
    heading := fmt.Sprintf("Posts tagged '%s'", tag)
    tmplData := MkBasicData(ctx, 0, 0)
    tmplData["PageTitle"] = heading
    tmplData["HeadingText"] = heading + ":"
    tmplData["all_entries"] = data.titlesByTag(tag)
    render(w, "archive", tmplData)
    return nil
}

func Archive(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    tmplData := MkBasicData(ctx, 0, 0)
    tmplData["PageTitle"] = "Archive"
    tmplData["HeadingText"] = "All posts:"
    tmplData["all_entries"] = data.titles(-1)
    render(w, "archive", tmplData)
    return nil
}

func AllComments(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    tmplData := MkBasicData(ctx, 0, 0)
    tmplData["PageTitle"] = "All Comments"
    tmplData["all_comments"] = data.allComments()
    render(w, "all_comments", tmplData)
    return nil
}

func EditPost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    tmplData := MkBasicData(ctx, 0, 0)
    tmplData["PageTitle"] = "Edit Post"
    tmplData["IsHidden"] = true // Assume hidden for a new post
    url := strings.TrimRight(req.FormValue("post"), "&")
    if url != "" {
        if post := data.post(url); post != nil {
            tmplData["IsHidden"] = post.Hidden
            tmplData["post"] = post
        }
    } else {
        tmplData["post"] = Entry{}
    }
    render(w, "edit_post", tmplData)
    return nil
}

func LoadComments(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    url := req.FormValue("post")
    if post := data.post(url); post != nil {
        b, err := json.Marshal(post)
        if err != nil {
            logger.Println(err.Error())
            return err
        }
        w.Write(b)
    }
    return nil
}

func RssFeed(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    data.hiddenPosts(false)
    produceFeedXml(w, data.posts(NUM_FEED_ITEMS, 0))
    return nil
}

func Login(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    // TODO: should not be logged in, add check
    uname := req.FormValue("uname")
    a, err := data.author(uname)
    if err == sql.ErrNoRows {
        ctx.Session.AddFlash("Login failed.")
        return LoginForm(w, req, ctx)
    }
    if err != nil {
        logger.Println(err.Error())
        return err
    }
    passwd := req.FormValue("passwd")
    req.Form["passwd"] = []string{"***"} // Avoid spilling password to log
    err = Decrypt([]byte(a.Passwd), []byte(passwd))
    if err == nil {
        ctx.Session.Values["adminlogin"] = "yes"
        redir := req.FormValue("redirect_to")
        if redir == "login" {
            redir = ""
        }
        http.Redirect(w, req, "/"+redir, http.StatusSeeOther)
    } else {
        ctx.Session.AddFlash("Login failed.")
        return LoginForm(w, req, ctx)
    }
    return nil
}

func DeleteComment(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    action := req.FormValue("action")
    redir := req.FormValue("redirect_to")
    id := req.FormValue("id")
    if action == "delete" && !data.deleteComment(id) {
        // TODO: log nothing to del
        return nil
    }
    http.Redirect(w, req, "/"+redir, http.StatusSeeOther)
    return nil
}

func DeletePost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    if !data.deletePost(req.FormValue("id")) {
        // TODO: log nothing to del
        return nil
    }
    http.Redirect(w, req, reverse("admin"), http.StatusSeeOther)
    return nil
}

func ModerateComment(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    action := req.FormValue("action")
    text := req.FormValue("edit-comment-text")
    id := req.FormValue("id")
    if action == "edit" && !data.updateComment(id, text) {
        // TODO: log error editing
        return nil
    }
    redir := req.FormValue("redirect_to")
    http.Redirect(w, req, fmt.Sprintf("/%s#comment-%s", redir, id), http.StatusSeeOther)
    return nil
}

func SubmitPost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    tagsWithUrls := req.FormValue("tags")
    url := req.FormValue("url")
    e := Entry{
        EntryLink: EntryLink{
            Title:  req.FormValue("title"),
            Url:    url,
            Hidden: req.FormValue("hidden") == "on",
        },
        Body: req.FormValue("text"),
    }
    postId, idErr := data.postId(url)
    if !data.begin() {
        InternalError(w, req, "SubmitPost, !data.begin()")
        return nil
    }
    if idErr != nil {
        if idErr == sql.ErrNoRows {
            authorId := int64(1) // XXX: it's only me now
            newPostId, err := data.insertPost(authorId, &e)
            if err != nil {
                data.rollback()
                InternalError(w, req, "SubmitPost, !data.insertPost: "+err.Error())
                return err
            }
            postId = newPostId
        } else {
            logger.Println("data.postId() failed: " + idErr.Error())
            data.rollback()
            InternalError(w, req, "SubmitPost, !data.postId: "+idErr.Error())
            return idErr
        }
    } else {
        if !data.updatePost(postId, &e) {
            data.rollback()
            InternalError(w, req, "SubmitPost, !data.updatePost")
            return nil
        }
    }
    data.updateTags(explodeTags(tagsWithUrls), postId)
    data.commit()
    http.Redirect(w, req, "/"+url, http.StatusSeeOther)
    return nil
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

func UploadImage(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    mr, _ := req.MultipartReader()
    files := ""
    part, err := mr.NextPart()
    for err == nil {
        if name := part.FormName(); name != "" {
            if part.FileName() != "" {
                files += fmt.Sprintf("[foo]: /%s", part.FileName())
                handleUpload(req, part)
            }
        }
        part, err = mr.NextPart()
    }
    w.Write([]byte(files))
    return nil
}

func handleUpload(r *http.Request, p *multipart.Part) {
    defer func() {
        if rec := recover(); rec != nil {
            logger.Println(rec)
        }
    }()
    lr := &io.LimitedReader{R: p, N: MAX_FILE_SIZE + 1}
    filename := filepath.Join(conf.Get("staticdir"), p.FileName())
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

func detectLanguageWithTimeout(text string) string {
    c := make(chan string, 1)
    go func() {
        c <- DetectLanguage(text)
    }()
    select {
    case lang := <-c:
        return lang
    case <-time.After(1500 * time.Millisecond):
        return "timedout"
    }
}

func CommentHandler(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    refUrl := xtractReferer(req)
    postId, err := data.postId(refUrl)
    if err != nil {
        logger.Println("data.postId() failed: " + err.Error())
        InternalError(w, req, "Server Error: "+err.Error())
        return err
    }
    ip := req.RemoteAddr
    name := req.FormValue("name")
    email := req.FormValue("email")
    website := req.FormValue("website")
    body := req.FormValue("text")
    commenterId, err := data.commenter(name, email, website, ip)
    redir := ""
    captchaId := req.FormValue("captcha-id")
    if err == nil {
        // This is a returning commenter, pass his comment through:
        commentUrl, err := PublishComment(postId, commenterId, body)
        if err != nil {
            InternalError(w, req, "Server Error: "+err.Error())
            return err
        }
        redir = "/" + refUrl + commentUrl
    } else if err == sql.ErrNoRows {
        if captchaId == "" {
            lang := detectLanguageWithTimeout(body)
            log := fmt.Sprintf("Detected language: %q for text %q", lang, body)
            logger.Println(log)
            if lang == "\"lt\"" {
                commentUrl, err := PublishCommentWithInsert(postId, req.RemoteAddr, name, email, website, body)
                if err != nil {
                    InternalError(w, req, "Server Error: "+err.Error())
                    return err
                }
                redir = "/" + refUrl + commentUrl
            } else {
                WrongCaptchaReply(w, req, "showcaptcha", GetTask())
                return nil
            }
        } else {
            captchaTask := GetTaskById(captchaId)
            if !CheckCaptcha(captchaTask, req.FormValue("captcha")) {
                WrongCaptchaReply(w, req, "rejected", captchaTask)
                return nil
            } else {
                commentUrl, err := PublishCommentWithInsert(postId, req.RemoteAddr, name, email, website, body)
                if err != nil {
                    InternalError(w, req, "Server Error: "+err.Error())
                    return err
                }
                redir = "/" + refUrl + commentUrl
            }
        }
    } else {
        logger.Println("err: " + err.Error())
        WrongCaptchaReply(w, req, "rejected", GetTask())
        return nil
    }
    url := conf.Get("url") + conf.Get("port") + redir
    if conf.Get("notif_send_email") == "true" {
        go SendEmail(name, email, website, req.FormValue("text"), url, refUrl)
    }
    RightCaptchaReply(w, redir)
    return nil
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

func ServeFavicon(w http.ResponseWriter, req *http.Request, ctx *Context) error {
    http.ServeFile(w, req, conf.Get("favicon"))
    return nil
}

func runServer(_data Data) {
    Router = pat.New()
    data = _data
    r := Router
    basedir, _ := filepath.Split(fullPathToBinary())
    dir := filepath.Join(basedir, conf.Get("staticdir"))
    r.Add("GET", "/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(dir)))).Name("static")
    r.Add("GET", "/login", Handler(LoginForm)).Name("login")
    r.Add("POST", "/login", Handler(Login))
    r.Add("GET", "/logout", Handler(Logout)).Name("logout")
    r.Add("GET", "/admin", checkPerm(Handler(Admin))).Name("admin")
    r.Add("GET", "/page/{pageNo:.*}", Handler(PageNum))
    r.Add("GET", "/tag/{tag:[0-9a-zA-Z]+}", Handler(PostsWithTag))
    r.Add("GET", "/archive", Handler(Archive)).Name("archive")
    r.Add("GET", "/all_comments", checkPerm(Handler(AllComments))).Name("all_comments")
    r.Add("GET", "/edit_post", checkPerm(Handler(EditPost))).Name("edit_post")
    r.Add("GET", "/load_comments", checkPerm(Handler(LoadComments))).Name("load_comments")
    r.Add("GET", "/feeds/rss.xml", Handler(RssFeed)).Name("rss_feed")
    r.Add("GET", "/favicon.ico", Handler(ServeFavicon)).Name("favicon")
    r.Add("GET", "/comment_submit", Handler(CommentHandler)).Name("comment")
    r.Add("GET", "/delete_comment", checkPerm(Handler(DeleteComment))).Name("delete_comment")
    r.Add("GET", "/delete_post", checkPerm(Handler(DeletePost))).Name("delete_post")

    r.Add("POST", "/moderate_comment", checkPerm(Handler(ModerateComment))).Name("moderate_comment")
    r.Add("POST", "/submit_post", checkPerm(Handler(SubmitPost))).Name("submit_post")
    r.Add("POST", "/upload_images", checkPerm(Handler(UploadImage))).Name("upload_image")

    r.Add("GET", "/", Handler(Home)).Name("home_page")

    logger.Print("The server is listening...")
    if err := http.ListenAndServe(os.Getenv("HOST")+conf.Get("port"), r); err != nil {
        logger.Print("rtfblog server: ", err)
    }
}

func fullPathToBinary() string {
    if filepath.IsAbs(os.Args[0]) {
        return os.Args[0]
    }
    cwd, err := os.Getwd()
    if err != nil {
        return filepath.Clean(os.Args[0])
    }
    return filepath.Join(cwd, os.Args[0])
}

func obtainConfiguration(basedir string) SrvConfig {
    hardcodedConf := SrvConfig{
        "database":         "user=tstusr dbname=tstdb password=tstpwd",
        "url":              "localhost",
        "port":             ":8080",
        "staticdir":        "static",
        "notif_send_email": "false",
        "log":              "server.log",
        "cookie_secret":    "dont-forget-to-change-me",
        "author":           "Mr. Blog Author",
        "email":            "blog_author@ema.il",
    }
    conf := hardcodedConf
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
    //runtime.GOMAXPROCS(runtime.NumCPU())
    rand.Seed(time.Now().UnixNano())
    basedir, _ := filepath.Split(fullPathToBinary())
    os.Chdir(basedir)
    conf = obtainConfiguration(basedir)
    logger = MkLogger(conf.Get("log"))
    store = sessions.NewCookieStore([]byte(conf.Get("cookie_secret")))
    db, err := sql.Open("postgres", conf.Get("database"))
    if err != nil {
        logger.Println("sql: " + err.Error())
        return
    }
    defer db.Close()
    runServer(&DbData{db, nil, false})
}
