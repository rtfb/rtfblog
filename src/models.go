package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
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
	numTotalPosts := ctx.Db.numPosts(ctx.AdminLogin)
	titles, err := ctx.Db.titles(NumRecentPosts, ctx.AdminLogin)
	logger.LogIf(err)
	return TmplMap{
		"PageTitle":       L10n("Welcome"),
		"BlogTitle":       conf.Get("blog_title"),
		"BlogSubtitle":    conf.Get("blog_descr"),
		"NeedPagination":  numTotalPosts > PostsPerPage,
		"ListOfPages":     listOfPages(numTotalPosts, pageNo),
		"entries":         ctx.Db.posts(PostsPerPage, offset, ctx.AdminLogin),
		"sidebar_entries": titles,
		"AdminLogin":      ctx.AdminLogin,
	}
}

func PublishCommentWithInsert(db Data, postID int64, commenter Commenter, rawBody string) (string, error) {
	if db.begin() != nil {
		return "", nil
	}
	defer db.rollback()
	commenterID, err := db.insertCommenter(commenter)
	if err != nil {
		return "", logger.LogIff(err, "db.insertCommenter() failed")
	}
	commentID, err := db.insertComment(commenterID, postID, rawBody)
	if err != nil {
		return "", logger.LogIff(err, "db.insertComment() failed")
	}
	db.commit()
	return fmt.Sprintf("#comment-%d", commentID), nil
}

func PublishComment(db Data, postID, commenterID int64, body string) (string, error) {
	if db.begin() != nil {
		return "", nil
	}
	defer db.rollback()
	commentID, err := db.insertComment(commenterID, postID, body)
	if err != nil {
		return "", logger.LogIff(err, "db.insertComment() failed")
	}
	db.commit()
	return fmt.Sprintf("#comment-%d", commentID), nil
}

func InsertOrUpdatePost(db Data, post *EntryTable) (id int64, err error) {
	postID, idErr := db.postID(post.URL)
	if idErr != nil {
		if idErr == gorm.RecordNotFound {
			post.AuthorID = int64(1) // XXX: it's only me now
			newPostID, err := db.insertPost(post)
			if err != nil {
				return -1, err
			}
			postID = newPostID
		} else {
			return -1, logger.LogIff(idErr, "db.postID() failed")
		}
	} else {
		post.Id = postID
		updErr := db.updatePost(post)
		if updErr != nil {
			return -1, updErr
		}
	}
	return postID, nil
}
