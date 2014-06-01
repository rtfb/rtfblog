package main

import (
    "fmt"
    "net/http"
    "path/filepath"

    "github.com/gorilla/pat"
    "github.com/gorilla/sessions"
    "github.com/nicksnyder/go-i18n/i18n"
)

type Context struct {
    // TODO: add db here
    Session    *sessions.Session
    AdminLogin bool
}

var (
    store  sessions.Store
    Router *pat.Router
    L10n   i18n.TranslateFunc
)

func NewContext(req *http.Request) (*Context, error) {
    sess, err := store.Get(req, "rtfblog")
    ctx := &Context{
        Session:    sess,
        AdminLogin: sess.Values["adminlogin"] == "yes",
    }
    return ctx, err
}

// Loads translation files and inits L10n func that retrieves the translations.
// l10nDir is a name of a directory with translations.
// userLocale specifies a locale preferred by the user (a preference or accept
// header or language cookie).
func InitL10n(l10nDir, userLocale string) {
    i18n.MustLoadTranslationFile(filepath.Join(l10nDir, "en-US.all.json"))
    i18n.MustLoadTranslationFile(filepath.Join(l10nDir, "lt-LT.all.json"))
    defaultLocale := "en-US" // known valid locale
    L10n = i18n.MustTfunc(userLocale, defaultLocale)
    // Also assign L10n to a list of template funcs:
    funcs["L10n"] = L10n
}

func MkBasicData(ctx *Context, pageNo, offset int) map[string]interface{} {
    data.hiddenPosts(ctx.AdminLogin)
    numTotalPosts := data.numPosts()
    return map[string]interface{}{
        "PageTitle":       L10n("Welcome"),
        "BlogTitle":       conf.Get("blog_title"),
        "BlogSubtitle":    conf.Get("blog_descr"),
        "NeedPagination":  numTotalPosts > PostsPerPage,
        "ListOfPages":     listOfPages(numTotalPosts, pageNo),
        "entries":         data.posts(PostsPerPage, offset),
        "sidebar_entries": data.titles(NumRecentPosts),
        "AdminLogin":      ctx.AdminLogin,
    }
}

func PublishCommentWithInsert(postID int64, commenter Commenter, rawBody string) (string, error) {
    if !data.begin() {
        return "", nil
    }
    commenterID, err := data.insertCommenter(commenter.Name, commenter.Email, commenter.Website, commenter.IP)
    if err != nil {
        logger.Println("data.insertCommenter() failed: " + err.Error())
        data.rollback()
        return "", err
    }
    commentID, err := data.insertComment(commenterID, postID, rawBody)
    if err != nil {
        data.rollback()
        return "", err
    }
    data.commit()
    return fmt.Sprintf("#comment-%d", commentID), nil
}

func PublishComment(postID, commenterID int64, body string) (string, error) {
    if !data.begin() {
        return "", nil
    }
    commentID, err := data.insertComment(commenterID, postID, body)
    if err != nil {
        data.rollback()
        return "", err
    }
    data.commit()
    return fmt.Sprintf("#comment-%d", commentID), nil
}
