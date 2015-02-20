package main

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gorilla/sessions"
	"github.com/nicksnyder/go-i18n/i18n"
)

type Context struct {
	GlobalContext
	Session    *sessions.Session
	AdminLogin bool
	Captcha    *Deck
}

const (
	l10n = "l10n"
)

var (
	L10n i18n.TranslateFunc
)

func NewContext(req *http.Request, gctx *GlobalContext) (*Context, error) {
	sess, err := gctx.Store.Get(req, "rtfblog")
	ctx := &Context{
		GlobalContext: *gctx,
		Session:       sess,
		AdminLogin:    sess.Values["adminlogin"] == "yes",
		Captcha:       deck,
	}
	return ctx, err
}

// Loads translation files and inits L10n func that retrieves the translations.
// l10nDir is a name of a directory with translations.
// userLocale specifies a locale preferred by the user (a preference or accept
// header or language cookie).
func InitL10n(root, userLocale string) {
	l10nDir := filepath.Join(root, l10n)
	i18n.MustLoadTranslationFile(filepath.Join(l10nDir, "en-US.all.json"))
	i18n.MustLoadTranslationFile(filepath.Join(l10nDir, "lt-LT.all.json"))
	defaultLocale := "en-US" // known valid locale
	L10n = i18n.MustTfunc(userLocale, defaultLocale)
	AddTemplateFunc("L10n", L10n)
}

func MkBasicData(ctx *Context, pageNo, offset int) TmplMap {
	ctx.Db.hiddenPosts(ctx.AdminLogin)
	numTotalPosts := ctx.Db.numPosts()
	return TmplMap{
		"PageTitle":       L10n("Welcome"),
		"BlogTitle":       conf.Get("blog_title"),
		"BlogSubtitle":    conf.Get("blog_descr"),
		"NeedPagination":  numTotalPosts > PostsPerPage,
		"ListOfPages":     listOfPages(numTotalPosts, pageNo),
		"entries":         ctx.Db.posts(PostsPerPage, offset),
		"sidebar_entries": ctx.Db.titles(NumRecentPosts),
		"AdminLogin":      ctx.AdminLogin,
	}
}

func PublishCommentWithInsert(db Data, postID int64, commenter Commenter, rawBody string) (string, error) {
	if db.begin() != nil {
		return "", nil
	}
	commenterID, err := db.insertCommenter(commenter)
	if err != nil {
		db.rollback()
		return "", logger.LogIff(err, "db.insertCommenter() failed")
	}
	commentID, err := db.insertComment(commenterID, postID, rawBody)
	if err != nil {
		db.rollback()
		return "", logger.LogIff(err, "db.insertComment() failed")
	}
	db.commit()
	return fmt.Sprintf("#comment-%d", commentID), nil
}

func PublishComment(db Data, postID, commenterID int64, body string) (string, error) {
	if db.begin() != nil {
		return "", nil
	}
	commentID, err := db.insertComment(commenterID, postID, body)
	if err != nil {
		db.rollback()
		return "", logger.LogIff(err, "db.insertComment() failed")
	}
	db.commit()
	return fmt.Sprintf("#comment-%d", commentID), nil
}
