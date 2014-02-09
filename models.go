package main

import (
    //"fmt"
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
        AdminLogin: false,
    }
    return ctx, err
}

func MkBasicData(ctx *Context, pageNo, offset int) map[string]interface{} {
    adminLogin := ctx.Session.Values["adminlogin"] == "yes"
    data.hiddenPosts(adminLogin)
    numTotalPosts := data.numPosts()
    return map[string]interface{}{
        "PageTitle":       "Velkam",
        "BlogTitle":       conf.Get("blog_title"),
        "BlogSubtitle":    conf.Get("blog_descr"),
        "NeedPagination":  numTotalPosts > POSTS_PER_PAGE,
        "ListOfPages":     listOfPages(numTotalPosts, pageNo),
        "entries":         data.posts(POSTS_PER_PAGE, offset),
        "sidebar_entries": data.titles(NUM_RECENT_POSTS),
        "AdminLogin":      adminLogin,
    }
}
