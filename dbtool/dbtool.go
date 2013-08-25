package main

import (
    "../util"
    "bytes"
    "database/sql"
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "net/mail"
    "os"
    "path/filepath"
    "strings"
    "time"

    _ "github.com/lib/pq"
)

func usage() {
    fmt.Fprintf(os.Stderr, "usage: %s [params]\n", filepath.Base(os.Args[0]))
    flag.PrintDefaults()
    os.Exit(2)
}

func init_db(fileName, uname, passwd, fullname, email, www string) {
    db, err := sql.Open("postgres", fileName)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer db.Close()
    stmt, _ := db.Prepare(`insert into author(id, disp_name, salt, passwd, full_name, email, www)
                           values($1, $2, $3, $4, $5, $6, $7)`)
    defer stmt.Close()
    salt, passwdHash, err := util.Encrypt(passwd)
    if err != nil {
        fmt.Printf("Error in util.Encrypt(): %s\n", err)
        return
    }
    stmt.Exec(1, uname, salt, passwdHash, fullname, email, www)
}

func populate(fileName string) {
    db, err := sql.Open("sqlite3", fileName)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer db.Close()
    xaction, err := db.Begin()
    if err != nil {
        fmt.Println(err)
        return
    }
    stmt, _ := xaction.Prepare("insert into post(id, author_id, title, date, url, body) values($1, $2, $3, $4, $5, $6)")
    defer stmt.Close()
    stmt.Exec(1, 1, "Labadėna", 123456, "labadena", "Nieko aš čia nerašysiu.")
    imgpost := `This is a post with a figure.

halfimage:

![hi][halfimg]

([Full size][fullimg])

[fullimg]: /no-dox.png
[halfimg]: /no-dox-halfsize.png`
    stmt.Exec(2, 1, "Iliustruotas", 1359308741, "iliustruotas", imgpost)
    stmt, _ = xaction.Prepare("insert into tag(id, name, url) values($1, $2, $3)")
    defer stmt.Close()
    stmt.Exec(1, "Testas", "testas")
    stmt.Exec(2, "Žąsyčiai", "geese")
    stmt, _ = xaction.Prepare("insert into tagmap(id, tag_id, post_id) values($1, $2, $3)")
    defer stmt.Close()
    stmt.Exec(1, 1, 1)
    stmt.Exec(2, 2, 1)
    stmt, _ = xaction.Prepare("insert into commenter(id, name, email, www, ip) values($1, $2, $3, $4, $5)")
    defer stmt.Close()
    stmt.Exec(1, "Vytautas Šaltenis", "Vytautas.Shaltenis@gmail.com", "http://rtfb.lt", "127.0.0.1")
    stmt.Exec(2, "Vardenis Pavardenis", "niekas@niekur.com", "http://delfi.lt", "127.0.0.1")
    stmt, _ = xaction.Prepare("insert into comment(id, commenter_id, post_id, timestamp, body) values($1, $2, $3, $4, $5)")
    defer stmt.Close()
    stmt.Exec(1, 2, 1, 1356872181, "Nu ir nerašyk, _niekam_ čia neįdomu tavo pisulkos.")
    stmt.Exec(2, 1, 1, 1356879181, "O tu čia tada **nekomentuok** ten kur neparašyta nieko. Eik [ten](http://google.com/)")
    xaction.Commit()
}

func populate2(fileName string, data []*Entry) {
    db, err := sql.Open("sqlite3", fileName)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer db.Close()
    xaction, err := db.Begin()
    if err != nil {
        fmt.Println(err)
        return
    }
    for _, e := range data {
        stmt, _ := xaction.Prepare("insert into post(author_id, title, date, url, body) values($1, $2, $3, $4, $5)")
        defer stmt.Close()
        date, _ := time.Parse("2006-01-02", e.Date)
        result, _ := stmt.Exec(1, e.Title, date.Unix(), e.Url, e.Body)
        postId, _ := result.LastInsertId()
        for _, t := range e.Tags {
            stmt, _ = xaction.Prepare("insert into tag(name, url) values($1, $2)")
            defer stmt.Close()
            result, _ = stmt.Exec(t.TagName, t.TagUrl)
            tagId, _ := result.LastInsertId()
            stmt, _ = xaction.Prepare("insert into tagmap(tag_id, post_id) values($1, $2)")
            defer stmt.Close()
            stmt.Exec(tagId, postId)
        }
    }
    xaction.Commit()
}

type Tag struct {
    TagUrl  string
    TagName string
}

type Entry struct {
    Author string
    Title  string
    Date   string
    Body   string
    Url    string
    Tags   []*Tag
}

func parseTags(tagList string) (tags []*Tag) {
    for _, t := range strings.Split(tagList, ", ") {
        if t == "" {
            continue
        }
        tag := new(Tag)
        tag.TagUrl = strings.ToLower(t)
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
    buf := new(bytes.Buffer)
    buf.ReadFrom(msg.Body)
    entry.Body = buf.String()
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

func readDbConf(path string) (db, uname, passwd, fullname, email, www string) {
    b, err := ioutil.ReadFile(path)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    var config DbConfig
    err = json.Unmarshal(b, &config)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    return config["db_file"], config["uname"], config["passwd"],
        config["fullname"], config["email"], config["www"]
}

func main() {
    dbconf := flag.String("db", "", "Database configuration file (required)")
    srcData := flag.String("src", "", "Can be either a directory, or a path to DB dump configuration file (required)")
    migrateOnly := flag.Bool("notest", false, "Don't populate DB with test data, only migrate what's in -src. (optional)")
    flag.Usage = usage
    flag.Parse()
    if *dbconf == "" || *srcData == "" {
        usage()
        return
    }
    dbFile, uname, passwd, fullname, email, www := readDbConf(*dbconf)
    init_db(dbFile, uname, passwd, fullname, email, www)
    srcFile, err := os.Open(*srcData)
    if err != nil {
        fmt.Println(err)
        return
    }
    defer srcFile.Close()
    fi, err := srcFile.Stat()
    if err != nil {
        fmt.Println(err)
        return
    }
    if fi.IsDir() {
        populate(dbFile)
        data, err := readTextEntries(*srcData)
        if err != nil {
            println(err.Error())
            return
        }
        populate2(dbFile, data)
    } else {
        if !*migrateOnly {
            populate(dbFile)
        }
        importLegacyDb(dbFile, *srcData)
    }
}
