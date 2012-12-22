package main

import (
    "github.com/hoisie/web"
    "github.com/lye/mustache"
    "github.com/russross/blackfriday"
    "io/ioutil"
    "log"
    "net/mail"
    "os"
    "path/filepath"
    "strings"
)

type Entry struct {
    Author string
    Title  string
    Date   string
    Body   string
    Url    string
}

var dataset string

func readTextEntry(filename string) (entry *Entry, err error) {
    f, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    msg, err := mail.ReadMessage(f)
    if err != nil {
        return nil, err
    }
    entry = new(Entry)
    entry.Title = msg.Header.Get("subject")
    entry.Author = msg.Header.Get("author")
    entry.Date = msg.Header.Get("isodate")
    base := filepath.Base(filename)
    entry.Url = base[:strings.LastIndex(base, filepath.Ext(filename))]
    b, err := ioutil.ReadAll(msg.Body)
    if err != nil {
        return nil, err
    }
    entry.Body = string(blackfriday.MarkdownCommon(b))
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

func render(ctx *web.Context, tmpl string, data map[string]interface{}) {
    html := mustache.RenderFile("tmpl/"+tmpl+".html.mustache", data)
    ctx.WriteString(html)
}

func handler(ctx *web.Context, path string) {
    posts := loadData(dataset)
    var basicData = map[string]interface{}{
        "PageTitle": "",
        "entries":   posts,
    }
    if path == "" {
        basicData["PageTitle"] = "Velkam"
        render(ctx, "main", basicData)
        return
    } else {
        for _, e := range posts {
            if e.Url == path {
                basicData["PageTitle"] = e.Title
                basicData["entry"] = e
                render(ctx, "post", basicData)
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

func runServer() {
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

func loadData(set string) []*Entry {
    if set == "" {
        return nil
    }
    data, err := readTextEntries(set)
    if err != nil {
        println(err.Error())
        return nil
    }
    return data
}

func main() {
    dataset = "testdata"
    runServer()
}
