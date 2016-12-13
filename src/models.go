package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
)

type Context struct {
	globalContext
	Session    *sessions.Session
	AdminLogin bool
	Captcha    *Deck
}

func NewContext(req *http.Request, gctx *globalContext) (*Context, error) {
	sess, err := gctx.Store.Get(req, "rtfblog")
	logger.LogIf(err)
	ctx := &Context{
		globalContext: *gctx,
		Session:       sess,
		AdminLogin:    sess.Values["adminlogin"] == "yes",
		Captcha:       deck,
	}
	return ctx, nil
}

func MkFlashes(ctx *Context) template.HTML {
	flashes := ctx.Session.Flashes()
	html := ""
	// TODO: extract that to separate flashes template
	format := `<div id="flash-%d" class="flash-box">
<p>%s</p>
<svg onclick="removeElt('flash-%d');">
<circle cx="12" cy="12" r="11" stroke-width="0" fill="white" fill-opacity="0" />
<path stroke="black" stroke-width="4" fill="none" d="M6.25,6.25,17.75,17.75" />
<path stroke="black" stroke-width="4" fill="none" d="M6.25,17.75,17.75,6.25" />
</svg>
</div>`
	for i, f := range flashes {
		html += fmt.Sprintf(format, i, f, i)
	}
	return template.HTML(`<div class="six columns">` + html + "</div>")
}

func MkBasicData(ctx *Context, pageNo, offset int) tmplMap {
	numTotalPosts, err := ctx.Db.numPosts(ctx.AdminLogin)
	logger.LogIf(err)
	titles, err := ctx.Db.titles(NumRecentPosts, ctx.AdminLogin)
	logger.LogIf(err)
	posts, err := ctx.Db.posts(PostsPerPage, offset, ctx.AdminLogin)
	logger.LogIf(err)
	return tmplMap{
		"PageTitle":       L10n("Welcome"),
		"BlogTitle":       conf.Interface.BlogTitle,
		"BlogSubtitle":    conf.Interface.BlogDescr,
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
	author, err := db.author()
	if err != nil {
		return -1, err
	}
	postID, idErr := db.postID(post.URL)
	if idErr != nil {
		if idErr == gorm.ErrRecordNotFound {
			post.AuthorID = author.ID
			newPostID, err := db.insertPost(post)
			if err != nil {
				return -1, err
			}
			postID = newPostID
		} else {
			return -1, logger.LogIff(idErr, "db.postID() failed")
		}
	} else {
		post.ID = postID
		post.AuthorID = author.ID
		updErr := db.updatePost(post)
		if updErr != nil {
			return -1, updErr
		}
	}
	return postID, nil
}

func InsertOrUpdateAuthor(db Data, newAuthor *Author) (id int64, err error) {
	author, err := db.author() // Pick default author
	id = author.ID
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			id, err = db.insertAuthor(newAuthor)
		}
	} else {
		newAuthor.ID = id
		err = db.updateAuthor(newAuthor)
	}
	return id, logger.LogIff(err, "Failed to insert author")
}
