package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	textTemplate "text/template"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/gorilla/feeds"
	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/rtfb/bark"
	"github.com/rtfb/httputil"
	email "github.com/ungerik/go-mail"
)

type SrvConfig map[string]interface{}

var (
	conf   SrvConfig
	logger *bark.Logger
	genVer string = ""
)

const (
	usage = `rtfblog. A standalone personal blog server.

Usage:
  rtfblog
  rtfblog -h | --help
  rtfblog --version

Options:
  With no arguments it simply runs the server (with either hardcoded config or
  a config it finds in one of locations described in README).
  -h --help     Show this screen.
  --version     Show version.`
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

func listOfPages(numPosts, currPage int) template.HTML {
	list := ""
	numPages := numPosts / PostsPerPage
	if numPosts%PostsPerPage != 0 {
		numPages++
	}
	for p := 0; p < numPages; p++ {
		if p == currPage {
			list += fmt.Sprintf("%d\n", p+1)
		} else {
			list += fmt.Sprintf("<a href=\"/page/%d\">%d</a>\n", p+1, p+1)
		}
	}
	return template.HTML(list)
}

func produceFeedXML(w http.ResponseWriter, req *http.Request, posts []*Entry, ctx *Context) {
	url := httputil.AddProtocol(httputil.GetHost(req), "http")
	blogTitle := conf.Get("blog_title")
	descr := conf.Get("blog_descr")
	author, err := ctx.Db.author()
	logger.LogIf(err)
	feed := &feeds.Feed{
		Title:       blogTitle,
		Link:        &feeds.Link{Href: url},
		Description: descr,
		Author:      &feeds.Author{Name: author.FullName, Email: author.Email},
	}
	for _, p := range posts {
		pubDate, err := time.Parse("2006-01-02", p.Date)
		if logger.LogIff(err, "Error parsing date for RSS item %q\n", p.URL) != nil {
			continue
		}
		item := feeds.Item{
			Title:       p.Title,
			Link:        &feeds.Link{Href: url + "/" + p.URL},
			Description: string(p.Body),
			Author:      &feeds.Author{Name: p.Author, Email: author.Email},
			Created:     pubDate,
		}
		feed.Items = append(feed.Items, &item)
	}
	rss, err := feed.ToRss()
	logger.LogIf(err)
	w.Write([]byte(rss))
}

func Home(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	if req.URL.Path == "/" {
		return Tmpl(ctx, "main.html").Execute(w, MkBasicData(ctx, 0, 0))
	}
	post, err := ctx.Db.post(req.URL.Path[1:], ctx.AdminLogin)
	if err == nil && post != nil {
		ctx.Captcha.SetNextTask(-1)
		tmplData := MkBasicData(ctx, 0, 0)
		tmplData["PageTitle"] = post.Title
		tmplData["entry"] = post
		task := *ctx.Captcha.GetTask()
		// Initial task id has to be empty, gets filled by AJAX upon first time
		// it gets shown
		task.ID = ""
		tmplData["CaptchaHtml"] = task
		return Tmpl(ctx, "post.html").Execute(w, tmplData)
	}
	return PerformStatus(ctx, w, req, http.StatusNotFound)
}

func PageNum(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	pgNo, err := strconv.Atoi(req.URL.Query().Get(":pageNo"))
	if err != nil {
		pgNo = 1
		err = nil
	}
	pgNo -= 1
	offset := pgNo * PostsPerPage
	return Tmpl(ctx, "main.html").Execute(w, MkBasicData(ctx, pgNo, offset))
}

func Admin(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	return Tmpl(ctx, "admin.html").Execute(w, MkBasicData(ctx, 0, 0))
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
	return Tmpl(ctx, "login.html").Execute(w, TmplMap{
		"Flashes": template.HTML(html),
	})
}

func Logout(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	delete(ctx.Session.Values, "adminlogin")
	http.Redirect(w, req, ctx.routeByName("home_page"), http.StatusSeeOther)
	return nil
}

func PostsWithTag(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tag := req.URL.Query().Get(":tag")
	heading := fmt.Sprintf(L10n("Posts tagged '%s'"), tag)
	tmplData := MkBasicData(ctx, 0, 0)
	tmplData["PageTitle"] = heading
	tmplData["HeadingText"] = heading + ":"
	titles, err := ctx.Db.titlesByTag(tag, ctx.AdminLogin)
	if err != nil {
		return err
	}
	tmplData["all_entries"] = titles
	return Tmpl(ctx, "archive.html").Execute(w, tmplData)
}

func Archive(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tmplData := MkBasicData(ctx, 0, 0)
	tmplData["PageTitle"] = L10n("Archive")
	tmplData["HeadingText"] = L10n("All posts:")
	titles, err := ctx.Db.titles(-1, ctx.AdminLogin)
	if err != nil {
		return err
	}
	tmplData["all_entries"] = titles
	return Tmpl(ctx, "archive.html").Execute(w, tmplData)
}

func AllComments(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tmplData := MkBasicData(ctx, 0, 0)
	comm, err := ctx.Db.allComments()
	if err != nil {
		return err
	}
	tmplData["all_comments"] = comm
	return Tmpl(ctx, "all_comments.html").Execute(w, tmplData)
}

func makeTagList(tags []*Tag) []string {
	var strTags []string
	for _, t := range tags {
		strTags = append(strTags, t.Name)
	}
	return strTags
}

func EditPost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tmplData := MkBasicData(ctx, 0, 0)
	tmplData["PageTitle"] = L10n("Edit Post")
	tmplData["IsHidden"] = true // Assume hidden for a new post
	tags, err := ctx.Db.queryAllTags()
	if err != nil {
		return err
	}
	tmplData["AllTags"] = makeTagList(tags)
	url := strings.TrimRight(req.FormValue("post"), "&")
	if url != "" {
		post, err := ctx.Db.post(url, ctx.AdminLogin)
		if err == nil && post != nil {
			tmplData["IsHidden"] = post.Hidden
			tmplData["post"] = post
		}
	} else {
		tmplData["post"] = Entry{}
	}
	return Tmpl(ctx, "edit_post.html").Execute(w, tmplData)
}

func LoadComments(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	url := req.FormValue("post")
	post, err := ctx.Db.post(url, ctx.AdminLogin)
	if err == nil && post != nil {
		b, err := json.Marshal(post)
		if err != nil {
			return logger.LogIf(err)
		}
		w.Write(b)
	}
	return nil
}

func RssFeed(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	posts, err := ctx.Db.posts(NumFeedItems, 0, false)
	if err != nil {
		return logger.LogIf(err)
	}
	produceFeedXML(w, req, posts, ctx)
	return nil
}

func Login(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	// TODO: should not be already logged in, add check
	a, err := ctx.Db.author() // Pick default author
	if err == gorm.RecordNotFound {
		ctx.Session.AddFlash(L10n("Login failed."))
		return LoginForm(w, req, ctx)
	}
	if err != nil {
		return logger.LogIf(err)
	}
	uname := req.FormValue("uname")
	if uname != a.UserName {
		ctx.Session.AddFlash(L10n("Login failed."))
		return LoginForm(w, req, ctx)
	}
	passwd := req.FormValue("passwd")
	req.Form["passwd"] = []string{"***"} // Avoid spilling password to log
	err = cryptoHelper.Decrypt([]byte(a.Passwd), []byte(passwd))
	if err == nil {
		ctx.Session.Values["adminlogin"] = "yes"
		redir := req.FormValue("redirect_to")
		if redir == "login" {
			redir = ""
		}
		http.Redirect(w, req, "/"+redir, http.StatusSeeOther)
	} else {
		ctx.Session.AddFlash(L10n("Login failed."))
		return LoginForm(w, req, ctx)
	}
	return nil
}

func DeleteComment(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	action := req.FormValue("action")
	redir := req.FormValue("redirect_to")
	id := req.FormValue("id")
	if action == "delete" {
		err := ctx.Db.deleteComment(id)
		if err != nil {
			return logger.LogIff(err, "DeleteComment: failed to delete comment for id %q", id)
		}
	}
	http.Redirect(w, req, "/"+redir, http.StatusSeeOther)
	return nil
}

func DeletePost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	id := req.FormValue("id")
	err := ctx.Db.deletePost(id)
	if err != nil {
		return logger.LogIff(err, "DeletePost: failed to delete post for id %q", id)
	}
	http.Redirect(w, req, ctx.routeByName("admin"), http.StatusSeeOther)
	return nil
}

func ModerateComment(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	action := req.FormValue("action")
	text := req.FormValue("edit-comment-text")
	id := req.FormValue("id")
	if action == "edit" {
		err := ctx.Db.updateComment(id, text)
		if err != nil {
			return logger.LogIff(err, "ModerateComment: failed to edit comment for id %q", id)
		}
	}
	redir := req.FormValue("redirect_to")
	http.Redirect(w, req, fmt.Sprintf("/%s#comment-%s", redir, id), http.StatusSeeOther)
	return nil
}

func SubmitPost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	url := req.FormValue("url")
	err := withTransaction(ctx.Db, func(db Data) error {
		postID, err := InsertOrUpdatePost(db, &EntryTable{
			EntryLink: EntryLink{
				Title:  req.FormValue("title"),
				URL:    url,
				Hidden: req.FormValue("hidden") == "on",
			},
			Body: template.HTML(req.FormValue("text")),
		})
		if err != nil {
			return err
		}
		return db.updateTags(explodeTags(req.FormValue("tags")), postID)
	})
	if err == nil {
		http.Redirect(w, req, "/"+url, http.StatusSeeOther)
	}
	return err
}

func explodeTags(tagsStr string) []*Tag {
	var tags []*Tag
	for _, t := range strings.Split(tagsStr, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		tags = append(tags, &Tag{Id: 0, Name: strings.ToLower(t)})
	}
	return tags
}

func UploadImage(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	mr, err := req.MultipartReader()
	if err != nil {
		return logger.LogIf(err)
	}
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
	lr := &io.LimitedReader{R: p, N: MaxFileSize + 1}
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

func prepareCommenter(req *http.Request) *Commenter {
	return &Commenter{
		Name:    req.FormValue("name"),
		Email:   req.FormValue("email"),
		Website: httputil.AddProtocol(req.FormValue("website"), "http"),
		IP:      httputil.GetIPAddress(req),
	}
}

func CommentHandler(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	refURL := httputil.ExtractReferer(req)
	postID, err := ctx.Db.postID(refURL)
	if err != nil {
		return logger.LogIff(err, "ctx.Db.postID('%s') failed", refURL)
	}
	commenter := prepareCommenter(req)
	body := req.FormValue("text")
	commenterID, err := ctx.Db.commenterID(commenter)
	commentURL := ""
	switch err {
	case nil:
		// This is a returning commenter, pass his comment through:
		commentURL, err = PublishComment(ctx.Db, postID, commenterID, body)
	case gorm.RecordNotFound:
		captchaID := req.FormValue("captcha-id")
		if captchaID == "" {
			lang := DetectLanguageWithTimeout(body)
			log := fmt.Sprintf("Detected language: %q for text %q", lang, body)
			logger.Println(log)
			if lang != "\"lt\"" {
				return WrongCaptchaReply(w, req, "showcaptcha", ctx.Captcha.GetTask())
			}
		} else {
			captchaTask := ctx.Captcha.GetTaskByID(captchaID)
			if !CheckCaptcha(captchaTask, req.FormValue("captcha")) {
				return WrongCaptchaReply(w, req, "rejected", captchaTask)
			}
		}
		commentURL, err = PublishCommentAndCommenter(ctx.Db, postID, commenter, body)
	default:
		logger.LogIf(err)
		return WrongCaptchaReply(w, req, "rejected", ctx.Captcha.GetTask())
	}
	if err != nil {
		return err
	}
	redir := "/" + refURL + commentURL
	sendNewCommentNotif(req, redir, commenter)
	return RightCaptchaReply(w, redir)
}

func sendNewCommentNotif(req *http.Request, redir string, commenter *Commenter) {
	if conf.Get("notif_send_email") != "true" {
		return
	}
	refURL := httputil.ExtractReferer(req)
	url := httputil.GetHost(req) + redir
	text := req.FormValue("text")
	subj, body := mkCommentNotifEmail(commenter, text, url, refURL)
	go SendEmail(subj, body)
}

func mkCommentNotifEmail(commenter *Commenter, rawBody, url, postTitle string) (subj, body string) {
	const messageTmpl = `
{{with .Commenter}}
New comment from {{.Name}} <{{.Email}}> ({{.Website}}):
{{end}}

{{.RawBody}}

URL: {{.URL}}
`
	t := textTemplate.Must(textTemplate.New("emailMessage").Parse(messageTmpl))
	var buff bytes.Buffer
	t.Execute(&buff, struct {
		Commenter
		RawBody string
		URL     string
	}{
		Commenter: *commenter,
		RawBody:   rawBody,
		URL:       url,
	})
	subj = fmt.Sprintf("New comment in '%s'", postTitle)
	return subj, buff.String()
}

func SendEmail(subj, body string) {
	gmailSenderAcct := conf.Get("notif_sender_acct")
	gmailSenderPasswd := conf.Get("notif_sender_passwd")
	notifee := conf.Get("email")
	err := email.InitGmail(gmailSenderAcct, gmailSenderPasswd)
	if err != nil {
		logger.LogIff(err, "err initing gmail")
		return
	}
	mess := email.NewBriefMessageFrom(subj, body, gmailSenderAcct, notifee)
	err = mess.Send()
	if err != nil {
		logger.LogIff(err, "err sending email")
		return
	}
}

func initRoutes(gctx *GlobalContext) *pat.Router {
	const (
		G = "GET"
		P = "POST"
	)
	r := gctx.Router
	dir := filepath.Join(gctx.Root, conf.Get("staticdir"))
	mkHandler := func(f HandlerFunc) *Handler {
		return &Handler{h: f, c: gctx, logRq: true}
	}
	mkAdminHandler := func(f HandlerFunc) *Handler {
		return &Handler{
			h: func(w http.ResponseWriter, req *http.Request, ctx *Context) error {
				if !ctx.AdminLogin {
					PerformStatus(ctx, w, req, http.StatusForbidden)
					return nil
				}
				return f(w, req, ctx)
			},
			c:     gctx,
			logRq: true,
		}
	}
	r.Add(G, "/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(dir)))).Name("static")
	r.Add(G, "/login", mkHandler(LoginForm)).Name("login")
	r.Add(P, "/login", mkHandler(Login))
	r.Add(G, "/logout", mkHandler(Logout)).Name("logout")
	r.Add(G, "/admin", mkAdminHandler(Admin)).Name("admin")
	r.Add(G, "/page/{pageNo:.*}", mkHandler(PageNum))
	r.Add(G, "/tag/{tag:.+}", mkHandler(PostsWithTag))
	r.Add(G, "/archive", mkHandler(Archive)).Name("archive")
	r.Add(G, "/all_comments", mkAdminHandler(AllComments)).Name("all_comments")
	r.Add(G, "/edit_post", mkAdminHandler(EditPost)).Name("edit_post")
	r.Add(G, "/load_comments", mkAdminHandler(LoadComments)).Name("load_comments")
	r.Add(G, "/feeds/rss.xml", mkHandler(RssFeed)).Name("rss_feed")
	r.Add(G, "/favicon.ico", &Handler{ServeFavicon, gctx, false}).Name("favicon")
	r.Add(G, "/comment_submit", mkHandler(CommentHandler)).Name("comment")
	r.Add(G, "/delete_comment", mkAdminHandler(DeleteComment)).Name("delete_comment")
	r.Add(G, "/delete_post", mkAdminHandler(DeletePost)).Name("delete_post")
	r.Add(G, "/robots.txt", mkHandler(ServeRobots))

	r.Add(P, "/moderate_comment", mkAdminHandler(ModerateComment)).Name("moderate_comment")
	r.Add(P, "/submit_post", mkAdminHandler(SubmitPost)).Name("submit_post")
	r.Add(P, "/upload_images", mkAdminHandler(UploadImage)).Name("upload_image")

	r.Add(G, "/", mkHandler(Home)).Name("home_page")
	return r
}

func obtainConfiguration(basedir string) SrvConfig {
	hardcodedConf := SrvConfig{
		"database":         "$RTFBLOG_DB_TEST_URL",
		"url":              "localhost",
		"port":             ":8080",
		"staticdir":        "static",
		"notif_send_email": "false",
		"log":              "server.log",
		"cookie_secret":    "dont-forget-to-change-me",
		"language":         "en-US",
	}
	conf := hardcodedConf
	home, err := GetHomeDir()
	if err != nil {
		fmt.Println("Error acquiring user home dir. That can't be good.")
		fmt.Printf("Err = %q", err.Error())
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

func versionString() string {
	ver, err := ioutil.ReadFile("VERSION")
	if err != nil {
		return genVer
	}
	return strings.TrimSpace(string(ver))
}

func getDBConnString() string {
	config := conf.Get("database")
	if config != "" && config[0] == '$' {
		envVar := os.ExpandEnv(config)
		if envVar == "" {
			panic(fmt.Sprintf("Can't find env var %s", config))
		}
		return envVar
	}
	return config
}

func serveAndLogTLS(addr, cert, key string, h http.Handler) {
	logger.LogIf(http.ListenAndServeTLS(addr, cert, key, h))
}

func runForever(handlers *pat.Router) {
	logger.Print("The server is listening...")
	host := os.Getenv("HOST")
	addr := httputil.JoinHostAndPort(host, conf.Get("port"))
	tlsPort := conf.Get("tls_port")
	cert := conf.Get("tls_cert")
	key := conf.Get("tls_key")
	if tlsPort != "" && FileExistsNoErr(cert) && FileExistsNoErr(key) {
		tlsAddr := httputil.JoinHostAndPort(host, tlsPort)
		go serveAndLogTLS(tlsAddr, cert, key, handlers)
	}
	logger.LogIf(http.ListenAndServe(addr, handlers))
}

func main() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	_, err := docopt.Parse(usage, nil, true, versionString(), false)
	if err != nil {
		panic("Can't docopt.Parse!")
	}
	rand.Seed(time.Now().UnixNano())
	bindir := Bindir()
	os.Chdir(bindir)
	conf = obtainConfiguration(bindir)
	InitL10n(bindir, conf.Get("language"))
	logger = bark.AppendFile(conf.Get("log"))
	db := InitDB(getDBConnString())
	defer db.db.Close()
	runForever(initRoutes(&GlobalContext{
		Router: pat.New(),
		Db:     db,
		Root:   bindir,
		Store:  sessions.NewCookieStore([]byte(conf.Get("cookie_secret"))),
	}))
}
