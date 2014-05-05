package main

import (
    "fmt"
    "net/http"

    "github.com/gorilla/pat"
    "github.com/gorilla/sessions"
)

type Context struct {
    // TODO: add db here
    Session    *sessions.Session
    AdminLogin bool
}

var (
    store  sessions.Store
    Router *pat.Router
)

func NewContext(req *http.Request) (*Context, error) {
    sess, err := store.Get(req, "rtfblog")
    ctx := &Context{
        Session:    sess,
        AdminLogin: sess.Values["adminlogin"] == "yes",
    }
    return ctx, err
}

func MkBasicData(ctx *Context, pageNo, offset int) map[string]interface{} {
    data.hiddenPosts(ctx.AdminLogin)
    numTotalPosts := data.numPosts()
    return map[string]interface{}{
        "PageTitle":       "Velkam",
        "BlogTitle":       conf.Get("blog_title"),
        "BlogSubtitle":    conf.Get("blog_descr"),
        "NeedPagination":  numTotalPosts > PostsPerPage,
        "ListOfPages":     listOfPages(numTotalPosts, pageNo),
        "entries":         data.posts(PostsPerPage, offset),
        "sidebar_entries": data.titles(NumRecentPosts),
        "AdminLogin":      ctx.AdminLogin,
    }
}

func PublishCommentWithInsert(postID int64, ip, name, email, website, body string) (string, error) {
    if !data.begin() {
        return "", nil
    }
    commenterID, err := data.insertCommenter(name, email, website, ip)
    if err != nil {
        logger.Println("data.insertCommenter() failed: " + err.Error())
        data.rollback()
        return "", err
    }
    commentID, err := data.insertComment(commenterID, postID, body)
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
