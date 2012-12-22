package main

import (
    "github.com/lye/mustache"
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
    Url   string
}

var posts []*Entry

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
    base := filepath.Base(filename)
    entry.Url = base[:strings.LastIndex(base, filepath.Ext(filename))]
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

func render(ctx *web.Context, tmpl string, title string, key string, data interface{}) {
    html := mustache.RenderFile("tmpl/"+tmpl+".html.mustache",
        map[string]interface{}{
            "PageTitle": title,
            "entries":   posts,
            key:         data,
        })
    ctx.WriteString(html)
}

func handler(ctx *web.Context, path string) {
    if path == "" {
        render(ctx, "main", "Velkam", "", nil)
        return
    } else {
        for _, e := range posts {
            if e.Url == path {
                render(ctx, "post", e.Title, "entry", e)
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
    data, err := readTextEntries(set)
    if err != nil {
        println(err.Error())
        return nil
    }
    return data
}

func main() {
    posts = loadData("testdata")
    runServer()
}
