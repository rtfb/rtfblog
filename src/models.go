package main

import (
	"fmt"
	"html/template"
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

func MkFlashes(ctx *Context) template.HTML {
	flashes := ctx.Session.Flashes()
	html := ""
	// TODO: extract that to separate flashes template
	format := `<p><strong style="color: red">
%s
</strong></p>`
	for _, f := range flashes {
		html = html + fmt.Sprintf(format, f)
	}
	return template.HTML(html)
}

func MkBasicData(ctx *Context, pageNo, offset int) TmplMap {
	numTotalPosts, err := ctx.Db.numPosts(ctx.AdminLogin)
	logger.LogIf(err)
	titles, err := ctx.Db.titles(NumRecentPosts, ctx.AdminLogin)
	logger.LogIf(err)
	posts, err := ctx.Db.posts(PostsPerPage, offset, ctx.AdminLogin)
	logger.LogIf(err)
	return TmplMap{
		"PageTitle":       L10n("Welcome"),
		"BlogTitle":       conf.Get("blog_title"),
		"BlogSubtitle":    conf.Get("blog_descr"),
		"NeedPagination":  numTotalPosts > PostsPerPage,
		"ListOfPages":     listOfPages(numTotalPosts, pageNo),
		"entries":         posts,
		"sidebar_entries": titles,
		"AdminLogin":      ctx.AdminLogin,
		"Version":         versionString(),
		"Flashes":         MkFlashes(ctx),
	}
}

func withTransaction(db Data, fn func(db Data) error) error {
	txErr := db.begin()
	if txErr != nil {
		return txErr
	}
	err := fn(db)
	if err != nil {
		db.rollback()
		return err
	}
	db.commit()
	return nil
}

func PublishCommentAndCommenter(db Data, postID int64, commenter *Commenter, rawBody string) (string, error) {
	var commentID int64
	err := withTransaction(db, func(db Data) error {
		commenterID, err := db.insertCommenter(commenter)
		if err != nil {
			return logger.LogIff(err, "db.insertCommenter() failed")
		}
		commentID, err = db.insertComment(commenterID, postID, rawBody)
		if err != nil {
			return logger.LogIff(err, "db.insertComment() failed")
		}
		return nil
	})
	return fmt.Sprintf("#comment-%d", commentID), err
}

func PublishComment(db Data, postID, commenterID int64, body string) (string, error) {
	var commentID int64
	err := withTransaction(db, func(db Data) error {
		var insErr error
		commentID, insErr = db.insertComment(commenterID, postID, body)
		return logger.LogIff(insErr, "db.insertComment() failed")
	})
	return fmt.Sprintf("#comment-%d", commentID), err
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
