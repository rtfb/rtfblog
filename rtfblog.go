package main

import (
    "crypto/md5"
    "database/sql"
    "fmt"
    "github.com/hoisie/web"
    "github.com/lye/mustache"
    _ "github.com/mattn/go-sqlite3"
    "github.com/russross/blackfriday"
    "io/ioutil"
    "log"
    "net/mail"
    "os"
    "path"
    "path/filepath"
    "strings"
    "time"
)

type Tag struct {
    TagUrl  string
    TagName string
}

type Comment struct {
    Name      string
    Email     string
    EmailHash string
    Website   string
    Ip        string
    Body      string
    Time      string
}

type Entry struct {
    Author   string
    Title    string
    Date     string
    Body     string
    Url      string
    Tags     []*Tag
    Comments []*Comment
}

var dataset string
var dbName string

func (e *Entry) HasTags() bool {
    if len(e.Tags) > 0 {
        return true
    }
    return false
}

func (e *Entry) HasComments() bool {
    return len(e.Comments) > 0
}

func (e *Entry) TagsStr() string {
    parts := make([]string, 0)
    for _, t := range e.Tags {
        part := fmt.Sprintf(`<a href="/tag/%s">%s</a>`, t.TagUrl, t.TagName)
        parts = append(parts, part)
    }
    return strings.Join(parts, ", ")
}

func parseTags(tagList string) (tags []*Tag) {
    for _, t := range strings.Split(tagList, ", ") {
        if t == "" {
            continue
        }
        tag := new(Tag)
        tag.TagUrl = "/tag/" + strings.ToLower(t)
        tag.TagName = t
        tags = append(tags, tag)
    }
    return
}

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
    entry.Tags = parseTags(msg.Header.Get("tags"))
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

func readDb(dbName string) (entries []*Entry, err error) {
    db, err := sql.Open("sqlite3", dbName)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer db.Close()
    rows, err := db.Query(`select a.disp_name, p.id, p.title, p.date,
                                  p.body, p.url
                           from author as a, post as p
                           where a.id=p.author_id`)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer rows.Close()
    for rows.Next() {
        entry := new(Entry)
        var id int
        var unixDate int64
        rows.Scan(&entry.Author, &id, &entry.Title, &unixDate, &entry.Body, &entry.Url)
        entry.Date = time.Unix(unixDate, 0).Format("2006-01-02")
        entry.Tags = queryTags(db, id)
        entry.Comments = queryComments(db, id)
        entries = append(entries, entry)
    }
    return
}

func queryTags(db *sql.DB, postId int) []*Tag {
    stmt, err := db.Prepare(`select t.name, t.url
                             from tag as t, tagmap as tm
                             where t.id = tm.tag_id
                                   and tm.post_id = ?`)
    if err != nil {
        fmt.Println(err.Error())
        return nil
    }
    defer stmt.Close()
    rows, err := stmt.Query(postId)
    if err != nil {
        fmt.Println(err.Error())
        return nil
    }
    defer rows.Close()
    tags := make([]*Tag, 0)
    for rows.Next() {
        tag := new(Tag)
        rows.Scan(&tag.TagName, &tag.TagUrl)
        tags = append(tags, tag)
    }
    return tags
}

func queryComments(db *sql.DB, postId int) []*Comment {
    stmt, err := db.Prepare(`select a.name, a.email, a.www, a.ip,
                                    c.timestamp, c.body
                             from commenter as a, comment as c
                             where a.id = c.commenter_id
                                   and c.post_id = ?`)
    if err != nil {
        fmt.Println(err.Error())
        return nil
    }
    defer stmt.Close()
    data, err := stmt.Query(postId)
    if err != nil {
        fmt.Println(err.Error())
        return nil
    }
    defer data.Close()
    comments := make([]*Comment, 0)
    for data.Next() {
        comment := new(Comment)
        var unixDate int64
        var body string
        data.Scan(&comment.Name, &comment.Email, &comment.Website, &comment.Ip,
            &unixDate, &body)
        hash := md5.New()
        hash.Write([]byte(strings.ToLower(comment.Email)))
        comment.EmailHash = fmt.Sprintf("%x", hash.Sum(nil))
        comment.Time = time.Unix(unixDate, 0).Format("2006-01-02 15:04")
        comment.Body = string(blackfriday.MarkdownCommon([]byte(body)))
        comments = append(comments, comment)
    }
    return comments
}

func render(ctx *web.Context, tmpl string, data map[string]interface{}) {
    html := mustache.RenderFile("tmpl/"+tmpl+".html.mustache", data)
    ctx.WriteString(html)
}

func handler(ctx *web.Context, path string) {
    posts := loadData(dataset, dbName)
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

func loadData(set string, db string) []*Entry {
    if set == "" {
        return nil
    }
    data, err := readTextEntries(set)
    if err != nil {
        println(err.Error())
        return nil
    }
    if dbName == "" {
        return data
    }
    d2, err2 := readDb(path.Join(set, dbName))
    if err2 != nil {
        println(err2.Error())
        return nil
    }
    return append(data, d2...)
}

func main() {
    dataset = "testdata"
    dbName = "foo.db"
    runServer()
}
