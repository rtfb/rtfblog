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
	"github.com/howeyc/gopass"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/rtfb/bark"
	"github.com/rtfb/httputil"
	email "github.com/ungerik/go-mail"
)

var (
	logger *bark.Logger
	genVer string
)

const (
	usage = `rtfblog. A standalone personal blog server.

Usage:
  rtfblog
  rtfblog --adduser <username> <email> <web> <display name>
  rtfblog -h | --help
  rtfblog --version

Options:
  With no arguments it simply runs the server (with either hardcoded config or
  a config it finds in one of locations described in README).
  -h --help     Show this screen.
  --version     Show version.`
	defaultCookieSecret = "dont-forget-to-change-me"
)

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
	blogTitle := conf.Interface.BlogTitle
	descr := conf.Interface.BlogDescr
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
		_, err := ctx.Db.author() // Pick default author
		if err == gorm.RecordNotFound {
			// Author was not configured yet, so pretend this is an admin
			// session and show the Edit Author form:
			ctx.Session.Values["adminlogin"] = "yes"
			return EditAuthorForm(w, req, ctx)
		}
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
	pgNo--
	offset := pgNo * PostsPerPage
	return Tmpl(ctx, "main.html").Execute(w, MkBasicData(ctx, pgNo, offset))
}

func Admin(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	if conf.Server.CookieSecret == defaultCookieSecret {
		ctx.Session.AddFlash(L10n("You are using default cookie secret, consider changing."))
	}
	return Tmpl(ctx, "admin.html").Execute(w, MkBasicData(ctx, 0, 0))
}

func LoginForm(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	return Tmpl(ctx, "login.html").Execute(w, MkBasicData(ctx, 0, 0))
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
	filename := filepath.Join(conf.Server.StaticDir, p.FileName())
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
	if !conf.Notifications.SendEmail {
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
	gmailSenderAcct := conf.Notifications.SenderAcct
	gmailSenderPasswd := conf.Notifications.SenderPasswd
	notifee := conf.Notifications.AdminEmail
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

func EditAuthorForm(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tmplData := MkBasicData(ctx, 0, 0)
	author, err := ctx.Db.author()
	if err != nil && err != gorm.RecordNotFound {
		return err
	}
	if err == gorm.RecordNotFound {
		author.UserName = req.FormValue("username")
		author.FullName = req.FormValue("display_name")
		author.Email = req.FormValue("email")
		author.Www = req.FormValue("www")
	}
	tmplData["PageTitle"] = L10n("Edit Author")
	tmplData["author"] = author
	tmplData["EditExistingAuthor"] = err != gorm.RecordNotFound
	return Tmpl(ctx, "edit_author.html").Execute(w, tmplData)
}

func SubmitAuthor(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	username := req.FormValue("username")
	displayname := req.FormValue("display_name")
	email := req.FormValue("email")
	www := req.FormValue("www")
	a, err := ctx.Db.author()
	if err != nil && err != gorm.RecordNotFound {
		return err
	}
	if err != gorm.RecordNotFound {
		oldPasswd := req.FormValue("old_password")
		req.Form["old_password"] = []string{"***"} // Avoid spilling password to log
		err = cryptoHelper.Decrypt([]byte(a.Passwd), []byte(oldPasswd))
		if err != nil {
			ctx.Session.AddFlash(L10n("Incorrect password."))
			return EditAuthorForm(w, req, ctx)
		}
	}
	passwd := req.FormValue("password")
	passwd2 := req.FormValue("confirm_password")
	req.Form["password"] = []string{"***"}         // Avoid spilling password to log
	req.Form["confirm_password"] = []string{"***"} // Avoid spilling password to log
	if passwd != passwd2 {
		ctx.Session.AddFlash(L10n("Passwords should match."))
		return EditAuthorForm(w, req, ctx)
	}
	crypt, err := EncryptBcrypt([]byte(passwd))
	if err != nil {
		return err
	}
	err = withTransaction(ctx.Db, func(db Data) error {
		_, err := InsertOrUpdateAuthor(db, &Author{
			UserName: username,
			FullName: displayname,
			Email:    email,
			Www:      www,
			Passwd:   crypt,
		})
		return err
	})
	if err == nil {
		http.Redirect(w, req, "/", http.StatusSeeOther)
	}
	return err
}

func initRoutes(gctx *GlobalContext) *pat.Router {
	const (
		G = "GET"
		P = "POST"
	)
	r := gctx.Router
	dir := filepath.Join(gctx.Root, conf.Server.StaticDir)
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
	r.Add(G, "/edit_author", mkAdminHandler(EditAuthorForm)).Name("edit_author")

	r.Add(P, "/moderate_comment", mkAdminHandler(ModerateComment)).Name("moderate_comment")
	r.Add(P, "/submit_post", mkAdminHandler(SubmitPost)).Name("submit_post")
	r.Add(P, "/submit_author", mkAdminHandler(SubmitAuthor)).Name("submit_author")
	r.Add(P, "/upload_images", mkAdminHandler(UploadImage)).Name("upload_image")

	r.Add(G, "/", mkHandler(Home)).Name("home_page")
	return r
}

func versionString() string {
	ver, err := ioutil.ReadFile("VERSION")
	if err != nil {
		return genVer
	}
	return strings.TrimSpace(string(ver))
}

func getDBConnString() string {
	config := conf.Server.DBConn
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
	addr := httputil.JoinHostAndPort(host, conf.Server.Port)
	tlsPort := conf.Server.TLSPort
	cert := conf.Server.TLSCert
	key := conf.Server.TLSKey
	if tlsPort != "" && FileExistsNoErr(cert) && FileExistsNoErr(key) {
		tlsAddr := httputil.JoinHostAndPort(host, tlsPort)
		go serveAndLogTLS(tlsAddr, cert, key, handlers)
	}
	logger.LogIf(http.ListenAndServe(addr, handlers))
}

func promptPasswd(username string) (string, error) {
	fmt.Printf(L10n("Type password for user %s: "), username)
	passwd := gopass.GetPasswd()
	fmt.Printf(L10n("Confirm password: "))
	passwd2 := gopass.GetPasswd()
	if string(passwd2) != string(passwd) {
		return "", fmt.Errorf("passwords do not match")
	}
	crypt, err := EncryptBcrypt(passwd)
	return string(crypt), err
}

func main() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	args, err := docopt.Parse(usage, nil, true, versionString(), false)
	if err != nil {
		panic("Can't docopt.Parse!")
	}
	rand.Seed(time.Now().UnixNano())
	bindir := Bindir()
	os.Chdir(bindir)
	conf = readConfigs(bindir)
	InitL10n(bindir, conf.Interface.Language)
	logger = bark.AppendFile(conf.Server.Log)
	db := InitDB(getDBConnString())
	defer db.db.Close()
	if args["--adduser"].(bool) {
		_, err = db.author()
		if err != gorm.RecordNotFound {
			fmt.Println(L10n("Author already added, can't add another, exiting"))
			return
		}
		passwd, err := promptPasswd(args["<username>"].(string))
		if err != nil {
			fmt.Printf(L10n("Error: %s\n"), err.Error())
			return
		}
		err = withTransaction(db, func(db Data) error {
			_, err := db.insertAuthor(&Author{
				UserName: args["<username>"].(string),
				Passwd:   passwd,
				FullName: args["<display name>"].(string),
				Email:    args["<email>"].(string),
				Www:      args["<web>"].(string),
			})
			return err
		})
		if err != nil {
			fmt.Printf(L10n("Failed to add user: %s\n"), err.Error())
		}
		return
	}
	runForever(initRoutes(&GlobalContext{
		Router: pat.New(),
		Db:     db,
		Root:   bindir,
		Store:  sessions.NewCookieStore([]byte(conf.Server.CookieSecret)),
	}))
}
