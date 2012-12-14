package main

import (
    "github.com/hoisie/mustache"
    "github.com/hoisie/web"
    "github.com/russross/blackfriday"
    "io/ioutil"
    "log"
    "os"
)

func handler(ctx *web.Context, path string) {
    if path == "" {
        ctx.WriteString(mustache.RenderFile("hello.mustache"))
        return
    } else {
        input, err := ioutil.ReadFile(path)
        if err != nil {
            ctx.NotFound("File Not Found\n" + err.Error())
            return
        }
        ctx.WriteString(string(blackfriday.MarkdownCommon(input)))
        return
    }
    ctx.Abort(500, "Server Error")
}

func main() {
    f, err := os.Create("server.log")
    if err != nil {
        println(err.Error())
        return
    }
    logger := log.New(f, "", log.Ldate|log.Ltime)
    web.Get("/(.*)", handler)
    web.SetLogger(logger)
    web.Run(":8080")
}
