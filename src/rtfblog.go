package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
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
	textTemplate "text/template"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/gorilla/feeds"
	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	email "github.com/ungerik/go-mail"
)

type SrvConfig map[string]interface{}

var (
	conf   SrvConfig
	data   Data
	logger *log.Logger
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

func produceFeedXML(w http.ResponseWriter, req *http.Request, posts []*Entry) {
	url := AddProtocol(GetHost(req), "http")
	blogTitle := conf.Get("blog_title")
	descr := conf.Get("blog_descr")
	author := conf.Get("author")
	authorEmail := conf.Get("email")
	feed := &feeds.Feed{
		Title:       blogTitle,
		Link:        &feeds.Link{Href: url},
		Description: descr,
		Author:      &feeds.Author{Name: author, Email: authorEmail},
	}
	for _, p := range posts {
		pubDate, err := time.Parse("2006-01-02", p.Date)
		if err != nil {
			logger.Printf("Error parsing date for RSS item %q\n", p.URL)
			logger.Println(err.Error())
			continue
		}
		item := feeds.Item{
			Title:       p.Title,
			Link:        &feeds.Link{Href: url + "/" + p.URL},
			Description: string(p.Body),
			Author:      &feeds.Author{Name: p.Author, Email: authorEmail},
			Created:     pubDate,
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
		return Tmpl("main.html").Execute(w, MkBasicData(ctx, 0, 0))
	}
	if post := data.post(req.URL.Path[1:]); post != nil {
		SetNextTask(-1)
		tmplData := MkBasicData(ctx, 0, 0)
		tmplData["PageTitle"] = post.Title
		tmplData["entry"] = post
		task := *GetTask()
		// Initial task id has to be empty, gets filled by AJAX upon first time
		// it gets shown
		task.ID = ""
		tmplData["CaptchaHtml"] = task
		return Tmpl("post.html").Execute(w, tmplData)
	}
	return PerformStatus(w, req, http.StatusNotFound)
}

func PageNum(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	pgNo, err := strconv.Atoi(req.URL.Query().Get(":pageNo"))
	if err != nil {
		pgNo = 1
		err = nil
	}
	pgNo -= 1
	offset := pgNo * PostsPerPage
	return Tmpl("main.html").Execute(w, MkBasicData(ctx, pgNo, offset))
}

func Admin(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	return Tmpl("admin.html").Execute(w, MkBasicData(ctx, 0, 0))
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
		"Flashes": template.HTML(html),
	})
}

func Logout(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	delete(ctx.Session.Values, "adminlogin")
	http.Redirect(w, req, reverse("home_page"), http.StatusSeeOther)
	return nil
}

func PostsWithTag(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tag := req.URL.Query().Get(":tag")
	heading := fmt.Sprintf(L10n("Posts tagged '%s'"), tag)
	tmplData := MkBasicData(ctx, 0, 0)
	tmplData["PageTitle"] = heading
	tmplData["HeadingText"] = heading + ":"
	tmplData["all_entries"] = data.titlesByTag(tag)
	return Tmpl("archive.html").Execute(w, tmplData)
}

func Archive(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tmplData := MkBasicData(ctx, 0, 0)
	tmplData["PageTitle"] = L10n("Archive")
	tmplData["HeadingText"] = L10n("All posts:")
	tmplData["all_entries"] = data.titles(-1)
	return Tmpl("archive.html").Execute(w, tmplData)
}

func AllComments(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tmplData := MkBasicData(ctx, 0, 0)
	tmplData["all_comments"] = data.allComments()
	return Tmpl("all_comments.html").Execute(w, tmplData)
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
	tmplData["AllTags"] = makeTagList(data.queryAllTags())
	url := strings.TrimRight(req.FormValue("post"), "&")
	if url != "" {
		if post := data.post(url); post != nil {
			tmplData["IsHidden"] = post.Hidden
			tmplData["post"] = post
		}
	} else {
		tmplData["post"] = Entry{}
	}
	return Tmpl("edit_post.html").Execute(w, tmplData)
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
	produceFeedXML(w, req, data.posts(NumFeedItems, 0))
	return nil
}

func Login(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	// TODO: should not be logged in, add check
	uname := req.FormValue("uname")
	a, err := data.author(uname)
	if err == sql.ErrNoRows {
		ctx.Session.AddFlash(L10n("Login failed."))
		return LoginForm(w, req, ctx)
	}
	if err != nil {
		logger.Println(err.Error())
		return err
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
	if action == "delete" && !data.deleteComment(id) {
		logger.Printf("DeleteComment: failed to delete comment for id %q", id)
		return nil
	}
	http.Redirect(w, req, "/"+redir, http.StatusSeeOther)
	return nil
}

func DeletePost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	id := req.FormValue("id")
	if !data.deletePost(id) {
		logger.Printf("DeletePost: failed to delete post for id %q", id)
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
		logger.Printf("ModerateComment: failed to edit comment for id %q", id)
		return nil
	}
	redir := req.FormValue("redirect_to")
	http.Redirect(w, req, fmt.Sprintf("/%s#comment-%s", redir, id), http.StatusSeeOther)
	return nil
}

func SubmitPost(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	tags := req.FormValue("tags")
	url := req.FormValue("url")
	e := Entry{
		EntryLink: EntryLink{
			Title:  req.FormValue("title"),
			URL:    url,
			Hidden: req.FormValue("hidden") == "on",
		},
		Body: template.HTML(req.FormValue("text")),
	}
	postID, idErr := data.postID(url)
	if !data.begin() {
		InternalError(w, req, "SubmitPost, !data.begin()")
		return nil
	}
	if idErr != nil {
		if idErr == sql.ErrNoRows {
			authorID := int64(1) // XXX: it's only me now
			newPostID, err := data.insertPost(authorID, &e)
			if err != nil {
				data.rollback()
				InternalError(w, req, "SubmitPost, !data.insertPost: "+err.Error())
				return err
			}
			postID = newPostID
		} else {
			logger.Println("data.postID() failed: " + idErr.Error())
			data.rollback()
			InternalError(w, req, "SubmitPost, !data.postID: "+idErr.Error())
			return idErr
		}
	} else {
		if !data.updatePost(postID, &e) {
			data.rollback()
			InternalError(w, req, "SubmitPost, !data.updatePost")
			return nil
		}
	}
	err := data.updateTags(explodeTags(tags), postID)
	if err != nil {
		data.rollback()
		return err
	}
	data.commit()
	http.Redirect(w, req, "/"+url, http.StatusSeeOther)
	return nil
}

func explodeTags(tagsStr string) []*Tag {
	var tags []*Tag
	for _, t := range strings.Split(tagsStr, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		tags = append(tags, &Tag{strings.ToLower(t)})
	}
	return tags
}

func UploadImage(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	mr, err := req.MultipartReader()
	if err != nil {
		logger.Println(err.Error())
		return err
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

func CommentHandler(w http.ResponseWriter, req *http.Request, ctx *Context) error {
	refURL := ExtractReferer(req)
	postID, err := data.postID(refURL)
	if err != nil {
		logger.Println("data.postID() failed: " + err.Error())
		InternalError(w, req, "Server Error: "+err.Error())
		return err
	}
	commenter := Commenter{
		Name:    req.FormValue("name"),
		Email:   req.FormValue("email"),
		Website: AddProtocol(req.FormValue("website"), "http"),
		IP:      GetIPAddress(req),
	}
	body := req.FormValue("text")
	commenterID, err := data.commenter(commenter)
	redir := ""
	captchaID := req.FormValue("captcha-id")
	if err == nil {
		// This is a returning commenter, pass his comment through:
		commentURL, err := PublishComment(postID, commenterID, body)
		if err != nil {
			InternalError(w, req, "Server Error: "+err.Error())
			return err
		}
		redir = "/" + refURL + commentURL
	} else if err == sql.ErrNoRows {
		if captchaID == "" {
			lang := DetectLanguageWithTimeout(body)
			log := fmt.Sprintf("Detected language: %q for text %q", lang, body)
			logger.Println(log)
			if lang == "\"lt\"" {
				commentURL, err := PublishCommentWithInsert(postID, commenter, body)
				if err != nil {
					InternalError(w, req, "Server Error: "+err.Error())
					return err
				}
				redir = "/" + refURL + commentURL
			} else {
				WrongCaptchaReply(w, req, "showcaptcha", GetTask())
				return nil
			}
		} else {
			captchaTask := GetTaskByID(captchaID)
			if !CheckCaptcha(captchaTask, req.FormValue("captcha")) {
				WrongCaptchaReply(w, req, "rejected", captchaTask)
				return nil
			}
			commentURL, err := PublishCommentWithInsert(postID, commenter, body)
			if err != nil {
				InternalError(w, req, "Server Error: "+err.Error())
				return err
			}
			redir = "/" + refURL + commentURL
		}
	} else {
		logger.Println("err: " + err.Error())
		WrongCaptchaReply(w, req, "rejected", GetTask())
		return nil
	}
	if conf.Get("notif_send_email") == "true" {
		url := GetHost(req) + redir
		subj, body := mkCommentNotifEmail(commenter, req.FormValue("text"), url, refURL)
		go SendEmail(subj, body)
	}
	RightCaptchaReply(w, redir)
	return nil
}

func mkCommentNotifEmail(commenter Commenter, rawBody, url, postTitle string) (subj, body string) {
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
		Commenter: commenter,
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
		logger.Println("err initing gmail: ", err.Error())
		return
	}
	mess := email.NewBriefMessageFrom(subj, body, gmailSenderAcct, notifee)
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

func initData(_data Data) {
	data = _data
}

func initRoutes(basedir string) {
	Router = pat.New()
	r := Router
	dir := filepath.Join(basedir, conf.Get("staticdir"))
	r.Add("GET", "/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(dir)))).Name("static")
	r.Add("GET", "/login", Handler(LoginForm)).Name("login")
	r.Add("POST", "/login", Handler(Login))
	r.Add("GET", "/logout", Handler(Logout)).Name("logout")
	r.Add("GET", "/admin", checkPerm(Handler(Admin))).Name("admin")
	r.Add("GET", "/page/{pageNo:.*}", Handler(PageNum))
	r.Add("GET", "/tag/{tag:.+}", Handler(PostsWithTag))
	r.Add("GET", "/archive", Handler(Archive)).Name("archive")
	r.Add("GET", "/all_comments", checkPerm(Handler(AllComments))).Name("all_comments")
	r.Add("GET", "/edit_post", checkPerm(Handler(EditPost))).Name("edit_post")
	r.Add("GET", "/load_comments", checkPerm(Handler(LoadComments))).Name("load_comments")
	r.Add("GET", "/feeds/rss.xml", Handler(RssFeed)).Name("rss_feed")
	r.Add("GET", "/favicon.ico", Handler(ServeFavicon)).Name("favicon")
	r.Add("GET", "/comment_submit", Handler(CommentHandler)).Name("comment")
	r.Add("GET", "/delete_comment", checkPerm(Handler(DeleteComment))).Name("delete_comment")
	r.Add("GET", "/delete_post", checkPerm(Handler(DeletePost))).Name("delete_post")
	r.Add("GET", "/robots.txt", Handler(ServeRobots))

	r.Add("POST", "/moderate_comment", checkPerm(Handler(ModerateComment))).Name("moderate_comment")
	r.Add("POST", "/submit_post", checkPerm(Handler(SubmitPost))).Name("submit_post")
	r.Add("POST", "/upload_images", checkPerm(Handler(UploadImage))).Name("upload_image")

	r.Add("GET", "/", Handler(Home)).Name("home_page")
}

func runServer() {
	logger.Print("The server is listening...")
	if err := http.ListenAndServe(os.Getenv("HOST")+conf.Get("port"), Router); err != nil {
		logger.Print("rtfblog server: ", err)
	}
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
		"author":           "Mr. Blog Author",
		"email":            "blog_author@ema.il",
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
		return generatedVersionString
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
	InitL10n("./l10n", "lt-LT")
	logger = MkLogger(conf.Get("log"))
	store = sessions.NewCookieStore([]byte(conf.Get("cookie_secret")))
	db, err := sql.Open("postgres", getDBConnString())
	if err != nil {
		logger.Println("sql: " + err.Error())
		return
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	initData(&DbData{
		db:            db,
		tx:            nil,
		includeHidden: false,
	})
	initRoutes(bindir)
	runServer()
}
