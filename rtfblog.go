package main

import (
    "github.com/hoisie/mustache"
    "github.com/hoisie/web"
    "github.com/russross/blackfriday"
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "strings"
)

type Entry struct {
    Title string
    Body  string
}

func readTextEntry(filename string) (entry *Entry, err error) {
    f, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    b, err := ioutil.ReadAll(f)
    if err != nil {
        return nil, err
    }

    lines := strings.Split(string(b), "\n")
    entry = new(Entry)
    entry.Title = lines[0]
    text := strings.Join(lines[2:], "\n")
    entry.Body = string(blackfriday.MarkdownCommon([]byte(text)))
    return
}

func readTextEntries(root string) (entries []*Entry, err error) {
    filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if strings.ToLower(filepath.Ext(path)) != ".txt" {
            return nil
        }
        entry, _ := readTextEntry(path)
        if entry == nil {
            return nil
        }
        entries = append(entries, entry)
        return nil
    })
    return
}

func handler(ctx *web.Context, path string) {
    if path == "" {
        data, _ := readTextEntries("testdata")
        html := mustache.RenderFile("tmpl/main.html.mustache",
            map[string]interface{}{
                "entries": data})
        ctx.WriteString(html)
        return
    } else {
        data, _ := readTextEntries("testdata")
        for _, e := range data {
            if e.Url == path {
                html := mustache.RenderFile("tmpl/post.html.mustache",
                    map[string]interface{}{
                        "entry": e,
                        "entries": data})
                ctx.WriteString(html)
                return
            }
        }
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
