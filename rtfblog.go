package main

import (
    "github.com/hoisie/mustache"
    "github.com/hoisie/web"
    "github.com/russross/blackfriday"
    "io/ioutil"
    "log"
    "os"
)

type Entry struct {
    Title    string
    Body     string
}

func handler(ctx *web.Context, path string) {
    if path == "" {
        var data = []Entry {
            {"Title1", "Body 1"},
            {"Title2", "Body 2"},
        }
        html := mustache.RenderFile("hello.mustache",
            map[string]interface{}{
                "entries": data})
        ctx.WriteString(html)
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
    web.Config.StaticDir = "static"
    web.Run(":8080")
}
