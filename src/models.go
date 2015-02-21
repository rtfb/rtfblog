package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
)

type Context struct {
	GlobalContext
	Session    *sessions.Session
	AdminLogin bool
	Captcha    *Deck
}

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
