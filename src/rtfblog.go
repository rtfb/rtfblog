package rtfblog

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
	_ "github.com/mattn/go-sqlite3"
	"github.com/rtfb/bark"
	email "github.com/rtfb/go-mail"
	"github.com/rtfb/gopass"
	"github.com/rtfb/httputil"
	"github.com/rtfb/rtfblog/src/assets"
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

func (s *server) produceFeedXML(w http.ResponseWriter, req *http.Request, posts []*Entry, ctx *Context) {
	url := httputil.AddProtocol(httputil.GetHost(req), "http")
	blogTitle := s.conf.Interface.BlogTitle
	descr := s.conf.Interface.BlogDescr
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

func (s *server) home(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	if req.URL.Path == "/" {
		_, err := ctx.Db.author() // Pick default author
		if err == gorm.ErrRecordNotFound {
			// Author was not configured yet, so pretend this is an admin
			// session and show the Edit Author form:
			ctx.Session.Values["adminlogin"] = "yes"
			return s.editAuthorForm(w, req, ctx)
		}
		return tmpl(ctx, "main.html").Execute(w, MkBasicData(ctx, 0, 0, s.conf))
	}
	post, err := ctx.Db.post(req.URL.Path[1:], ctx.AdminLogin)
	if err == nil && post != nil {
		ctx.Captcha.SetNextTask(-1)
		tmplData := MkBasicData(ctx, 0, 0, s.conf)
		tmplData["PageTitle"] = post.Title
		tmplData["entry"] = post
		task := *ctx.Captcha.NextTask()
		// Initial task id has to be empty, gets filled by AJAX upon first time
		// it gets shown
		task.ID = ""
		tmplData["CaptchaHtml"] = task
		return tmpl(ctx, "post.html").Execute(w, tmplData)
	}
	return performStatus(ctx, w, req, http.StatusNotFound)
}

func (s *server) pageNum(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	pgNo, err := strconv.Atoi(req.URL.Query().Get(":pageNo"))
	if err != nil {
		pgNo = 1
	}
	pgNo--
	offset := pgNo * PostsPerPage
	return tmpl(ctx, "main.html").Execute(w, MkBasicData(ctx, pgNo, offset, s.conf))
}

func (s *server) admin(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	if s.conf.Server.CookieSecret == defaultCookieSecret {
		ctx.Session.AddFlash(L10n("You are using default cookie secret, consider changing."))
	}
	return tmpl(ctx, "admin.html").Execute(w, MkBasicData(ctx, 0, 0, s.conf))
}

func (s *server) loginForm(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	return tmpl(ctx, "login.html").Execute(w, MkBasicData(ctx, 0, 0, s.conf))
}

func logout(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	delete(ctx.Session.Values, "adminlogin")
	http.Redirect(w, req, ctx.routeByName("home_page"), http.StatusSeeOther)
	return nil
}

func (s *server) postsWithTag(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tag := req.URL.Query().Get(":tag")
	heading := fmt.Sprintf(L10n("Posts tagged '%s'"), tag)
	tmplData := MkBasicData(ctx, 0, 0, s.conf)
	tmplData["PageTitle"] = heading
	tmplData["HeadingText"] = heading + ":"
	titles, err := ctx.Db.titlesByTag(tag, ctx.AdminLogin)
	if err != nil {
		return err
	}
	tmplData["all_entries"] = titles
	return tmpl(ctx, "archive.html").Execute(w, tmplData)
}

func (s *server) archive(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tmplData := MkBasicData(ctx, 0, 0, s.conf)
	tmplData["PageTitle"] = L10n("Archive")
	tmplData["HeadingText"] = L10n("All posts:")
	titles, err := ctx.Db.titles(-1, ctx.AdminLogin)
	if err != nil {
		return err
	}
	tmplData["all_entries"] = titles
	return tmpl(ctx, "archive.html").Execute(w, tmplData)
}

func (s *server) allComments(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tmplData := MkBasicData(ctx, 0, 0, s.conf)
	comm, err := ctx.Db.allComments()
	if err != nil {
		return err
	}
	tmplData["all_comments"] = comm
	return tmpl(ctx, "all_comments.html").Execute(w, tmplData)
}

func makeTagList(tags []*Tag) []string {
	var strTags []string
	for _, t := range tags {
		strTags = append(strTags, t.Name)
	}
	return strTags
}

func (s *server) editPost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tmplData := MkBasicData(ctx, 0, 0, s.conf)
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
	return tmpl(ctx, "edit_post.html").Execute(w, tmplData)
}

func loadComments(w http.ResponseWriter, req *http.Request, ctx *Context) error {
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

func (s *server) rssFeed(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	posts, err := ctx.Db.posts(NumFeedItems, 0, false)
	if err != nil {
		return logger.LogIf(err)
	}
	s.produceFeedXML(w, req, posts, ctx)
	return nil
}

func (s *server) login(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	// TODO: should not be already logged in, add check
	a, err := ctx.Db.author() // Pick default author
	if err == gorm.ErrRecordNotFound {
		ctx.Session.AddFlash(L10n("Login failed."))
		return s.loginForm(w, req, ctx)
	}
	if err != nil {
		return logger.LogIf(err)
	}
	uname := req.FormValue("uname")
	if uname != a.UserName {
		ctx.Session.AddFlash(L10n("Login failed."))
		return s.loginForm(w, req, ctx)
	}
	passwd := req.FormValue("passwd")
	req.Form["passwd"] = []string{"***"} // Avoid spilling password to log
	err = s.cryptoHelper.Decrypt([]byte(a.Passwd), []byte(passwd))
	if err == nil {
		ctx.Session.Values["adminlogin"] = "yes"
		redir := req.FormValue("redirect_to")
		if redir == "login" {
			redir = ""
		}
		http.Redirect(w, req, "/"+redir, http.StatusSeeOther)
	} else {
		ctx.Session.AddFlash(L10n("Login failed."))
		return s.loginForm(w, req, ctx)
	}
	return nil
}

func deleteComment(w http.ResponseWriter, req *http.Request, ctx *Context) error {
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

func deletePost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	id := req.FormValue("id")
	err := ctx.Db.deletePost(id)
	if err != nil {
		return logger.LogIff(err, "DeletePost: failed to delete post for id %q", id)
	}
	http.Redirect(w, req, ctx.routeByName("admin"), http.StatusSeeOther)
	return nil
}

func moderateComment(w http.ResponseWriter, req *http.Request, ctx *Context) error {
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

func submitPost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	url := req.FormValue("url")
	err := withTransaction(ctx.Db, func(db Data) error {
		postID, err := InsertOrUpdatePost(db, &EntryTable{
			EntryLink: EntryLink{
				Title:  req.FormValue("title"),
				URL:    url,
				Hidden: req.FormValue("hidden") == "on",
			},
			RawBody: req.FormValue("text"),
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
		tags = append(tags, &Tag{ID: 0, Name: strings.ToLower(t)})
	}
	return tags
}

func (s *server) uploadImage(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	mr, err := req.MultipartReader()
	if err != nil {
		return logger.LogIf(err)
	}
	files := ""
	part, err := mr.NextPart()
	for err == nil {
		if name := part.FormName(); name != "" {
			if part.FileName() != "" {
				files += fmt.Sprintf("[foo]: /static/%s", part.FileName())
				s.handleUpload(req, part, ctx.assets.Root)
			}
		}
		part, err = mr.NextPart()
	}
	w.Write([]byte(files))
	return nil
}

func (s *server) handleUpload(r *http.Request, p *multipart.Part, root string) {
	defer func() {
		if rec := recover(); rec != nil {
			logger.Println(rec)
		}
	}()
	lr := &io.LimitedReader{R: p, N: MaxFileSize + 1}
	filename := filepath.Join(root, s.conf.Server.StaticDir, p.FileName())
	logger.Printf("attempt to upload %s to %s\n", p.FileName(), filename)
	fo, err := os.Create(filename)
	if err != nil {
		logger.Printf("err writing %q!, err = %s\n", filename, err.Error())
		return
	}
	defer fo.Close()
	w := bufio.NewWriter(fo)
	nwritten, err := io.Copy(w, lr)
	if err != nil {
		logger.Printf("err writing %q!, err = %s\n", filename, err.Error())
	}
	if err = w.Flush(); err != nil {
		logger.Printf("err flushing writer for %q!, err = %s\n", filename, err.Error())
	}
	logger.Printf("ok, num bytes written to %s: %d\n", filename, nwritten)
}

func prepareCommenter(req *http.Request) *Commenter {
	return &Commenter{
		Name:    req.FormValue("name"),
		Email:   req.FormValue("email"),
		Website: httputil.AddProtocol(req.FormValue("website"), "http"),
		IP:      httputil.GetIPAddress(req),
	}
}

func captchaNewCommenter(w http.ResponseWriter, req *http.Request, ctx *Context) bool {
	body := req.FormValue("text")
	captchaID := req.FormValue("captcha-id")
	if captchaID == "" {
		lang := DetectLanguageWithTimeout(body)
		log := fmt.Sprintf("Detected language: %q for text %q", lang, body)
		logger.Println(log)
		if lang != `"lt"` {
			WrongCaptchaReply(w, req, "showcaptcha", ctx.Captcha.NextTask())
			return false
		}
	} else {
		captchaTask := ctx.Captcha.GetTask(captchaID)
		if !CheckCaptcha(captchaTask, req.FormValue("captcha")) {
			WrongCaptchaReply(w, req, "rejected", captchaTask)
			return false
		}
	}
	return true
}

func (s *server) commentHandler(w http.ResponseWriter, req *http.Request, ctx *Context) error {
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
	case gorm.ErrRecordNotFound:
		if !captchaNewCommenter(w, req, ctx) {
			return nil
		}
		commentURL, err = PublishCommentAndCommenter(ctx.Db, postID, commenter, body)
	default:
		logger.LogIf(err)
		return WrongCaptchaReply(w, req, "rejected", ctx.Captcha.NextTask())
	}
	if err != nil {
		return err
	}
	redir := "/" + refURL + commentURL
	s.sendNewCommentNotif(req, redir, commenter)
	return RightCaptchaReply(w, redir)
}

func (s *server) sendNewCommentNotif(req *http.Request, redir string, commenter *Commenter) {
	if !s.conf.Notifications.SendEmail {
		return
	}
	refURL := httputil.ExtractReferer(req)
	url := httputil.GetHost(req) + redir
	text := req.FormValue("text")
	subj, body := mkCommentNotifEmail(commenter, text, url, refURL)
	go s.sendEmail(subj, body)
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

func (s *server) sendEmail(subj, body string) {
	gmailSenderAcct := s.conf.Notifications.SenderAcct
	gmailSenderPasswd := s.conf.Notifications.SenderPasswd
	notifee := s.conf.Notifications.AdminEmail
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

func (s *server) editAuthorForm(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tmplData := MkBasicData(ctx, 0, 0, s.conf)
	author, err := ctx.Db.author()
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		author.UserName = req.FormValue("username")
		author.FullName = req.FormValue("display_name")
		author.Email = req.FormValue("email")
		author.Www = req.FormValue("www")
	}
	tmplData["PageTitle"] = L10n("Edit Author")
	tmplData["author"] = author
	tmplData["EditExistingAuthor"] = err != gorm.ErrRecordNotFound
	return tmpl(ctx, "edit_author.html").Execute(w, tmplData)
}

func (s *server) submitAuthor(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	username := req.FormValue("username")
	displayname := req.FormValue("display_name")
	email := req.FormValue("email")
	www := req.FormValue("www")
	a, err := ctx.Db.author()
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err != gorm.ErrRecordNotFound {
		oldPasswd := req.FormValue("old_password")
		req.Form["old_password"] = []string{"***"} // Avoid spilling password to log
		err = s.cryptoHelper.Decrypt([]byte(a.Passwd), []byte(oldPasswd))
		if err != nil {
			ctx.Session.AddFlash(L10n("Incorrect password."))
			return s.editAuthorForm(w, req, ctx)
		}
	}
	passwd := req.FormValue("password")
	passwd2 := req.FormValue("confirm_password")
	req.Form["password"] = []string{"***"}         // Avoid spilling password to log
	req.Form["confirm_password"] = []string{"***"} // Avoid spilling password to log
	if passwd != passwd2 {
		ctx.Session.AddFlash(L10n("Passwords should match."))
		return s.editAuthorForm(w, req, ctx)
	}
	crypt, err := encryptBcrypt([]byte(passwd))
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

func (s *server) initRoutes() *pat.Router {
	const (
		G = "GET"
		P = "POST"
	)
	r := s.gctx.Router
	mkHandler := func(f handlerFunc) *handler {
		return &handler{h: f, c: &s.gctx, logRq: true}
	}
	mkAdminHandler := func(f handlerFunc) *handler {
		return &handler{
			h: func(w http.ResponseWriter, req *http.Request, ctx *Context) error {
				if !ctx.AdminLogin {
					performStatus(ctx, w, req, http.StatusForbidden)
					return nil
				}
				return f(w, req, ctx)
			},
			c:     &s.gctx,
			logRq: true,
		}
	}
	r.Add(G, "/static/", http.FileServer(s.gctx.assets)).Name("static")
	r.Add(G, "/login", mkHandler(s.loginForm)).Name("login")
	r.Add(P, "/login", mkHandler(s.login))
	r.Add(G, "/logout", mkHandler(logout)).Name("logout")
	r.Add(G, "/admin", mkAdminHandler(s.admin)).Name("admin")
	r.Add(G, "/page/{pageNo:.*}", mkHandler(s.pageNum))
	r.Add(G, "/tag/{tag:.+}", mkHandler(s.postsWithTag))
	r.Add(G, "/archive", mkHandler(s.archive)).Name("archive")
	r.Add(G, "/all_comments", mkAdminHandler(s.allComments)).Name("all_comments")
	r.Add(G, "/edit_post", mkAdminHandler(s.editPost)).Name("edit_post")
	r.Add(G, "/load_comments", mkAdminHandler(loadComments)).Name("load_comments")
	r.Add(G, "/feeds/rss.xml", mkHandler(s.rssFeed)).Name("rss_feed")
	r.Add(G, "/favicon.ico", &handler{s.serveFavicon, &s.gctx, false}).Name("favicon")
	r.Add(G, "/comment_submit", mkHandler(s.commentHandler)).Name("comment")
	r.Add(G, "/delete_comment", mkAdminHandler(deleteComment)).Name("delete_comment")
	r.Add(G, "/delete_post", mkAdminHandler(deletePost)).Name("delete_post")
	r.Add(G, "/robots.txt", mkHandler(s.serveRobots))
	r.Add(G, "/edit_author", mkAdminHandler(s.editAuthorForm)).Name("edit_author")

	r.Add(P, "/moderate_comment", mkAdminHandler(moderateComment)).Name("moderate_comment")
	r.Add(P, "/submit_post", mkAdminHandler(submitPost)).Name("submit_post")
	r.Add(P, "/submit_author", mkAdminHandler(s.submitAuthor)).Name("submit_author")
	r.Add(P, "/upload_images", mkAdminHandler(s.uploadImage)).Name("upload_image")

	r.Add(G, "/", mkHandler(s.home)).Name("home_page")
	return r
}

func versionString() string {
	ver, err := ioutil.ReadFile("VERSION")
	if err != nil {
		return genVer
	}
	return strings.TrimSpace(string(ver))
}

func serveAndLogTLS(addr, cert, key string, h http.Handler) {
	logger.LogIf(http.ListenAndServeTLS(addr, cert, key, h))
}

func (s *server) runForever(handlers *pat.Router) {
	logger.Print("The server is listening...")
	host := os.Getenv("HOST")
	addr := httputil.JoinHostAndPort(host, s.conf.Server.Port)
	tlsPort := s.conf.Server.TLSPort
	cert := s.conf.Server.TLSCert
	key := s.conf.Server.TLSKey
	if tlsPort != "" && assets.FileExistsNoErr(cert) && assets.FileExistsNoErr(key) {
		tlsAddr := httputil.JoinHostAndPort(host, tlsPort)
		go serveAndLogTLS(tlsAddr, cert, key, handlers)
	}
	logger.LogIf(http.ListenAndServe(addr, handlers))
}

func promptPasswd(username string) (string, error) {
	fmt.Printf(L10n("Type password for user %s: "), username)
	passwd, err := gopass.GetPasswd()
	if err != nil {
		return "", err
	}
	fmt.Printf(L10n("Confirm password: "))
	passwd2, err := gopass.GetPasswd()
	if err != nil {
		return "", err
	}
	if string(passwd2) != string(passwd) {
		return "", fmt.Errorf("passwords do not match")
	}
	crypt, err := encryptBcrypt(passwd)
	return string(crypt), err
}

func insertUser(db *DbData, args map[string]interface{}) {
	_, err := db.author()
	if err != gorm.ErrRecordNotFound {
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

func Main() {
	args, err := docopt.Parse(usage, nil, true, versionString(), false)
	if err != nil {
		panic("Can't docopt.Parse!")
	}
	rand.Seed(time.Now().UnixNano())
	assets := assets.NewAssetBin(bindir())
	conf := readConfigs(assets.FSOnly())
	InitL10n(assets, conf.Interface.Language)
	logger = bark.AppendFile(conf.Server.Log)
	db := InitDB(conf, bindir())
	defer db.db.Close()
	s := newServer(new(BcryptHelper), globalContext{
		Router: pat.New(),
		Db:     db,
		assets: assets,
		Store:  sessions.NewCookieStore([]byte(conf.Server.CookieSecret)),
	}, conf)
	if args["--adduser"].(bool) {
		insertUser(db, args)
		return
	}
	s.runForever(s.initRoutes())
}
