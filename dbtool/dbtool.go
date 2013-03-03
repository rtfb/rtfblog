package main

import (
    "bytes"
    "database/sql"
    "fmt"
    "net/mail"
    "os"
    "path/filepath"
    "strings"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

func usage() {
    help := []string{
        "Usage:",
        os.Args[0] + " <command> [params...]",
        "",
        "possible commands:",
        "\tinit <../path/to/file.db> <source data>",
        "\t\t-- init clean db with schema.",
        "\t\t   <source data> can be either a directory,",
        "\t\t   or a path to B2Evolution DB dump",
    }
    for _, s := range help {
        println(s)
    }
}

func init_db(fileName string) {
    createTables := []string{
        `create table tag (
            id integer not null primary key,
            name text,
            url text
        )`,
        `create table author (
            id integer not null primary key,
            disp_name text,
            full_name text,
            email text,
            www text
        )`,
        `create table post (
            id integer not null primary key,
            author_id integer not null references author(id) on delete cascade on update cascade,
            title text,
            date long,
            url text,
            body text
        )`,
        `create table tagmap (
            id integer not null primary key,
            tag_id integer not null references tag(id) on delete cascade on update cascade,
            post_id integer not null references post(id) on delete cascade on update cascade
        )`,
        `create table commenter (
            id integer not null primary key,
            name text,
            email text,
            www text,
            ip text
        )`,
        `create table comment (
            id integer not null primary key,
            commenter_id integer not null references commenter(id) on delete cascade on update cascade,
            post_id integer not null references post(id) on delete cascade on update cascade,
            timestamp long,
            body text
        )`,
    }
    os.Remove(fileName)

    db, err := sql.Open("sqlite3", fileName)
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    defer db.Close()
    for _, sql := range createTables {
        _, err := db.Exec(sql)
        if err != nil {
            fmt.Printf("%q: %s\n", err, sql)
            return
        }
    }
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
    stmt, _ := xaction.Prepare("insert into author(id, disp_name, full_name, email, www) values(?, ?, ?, ?, ?)")
    defer stmt.Close()
    stmt.Exec(1, "rtfb", "Vytautas Šaltenis", "vytas@rtfb.lt", "http://rtfb.lt")
    stmt, _ = xaction.Prepare("insert into post(id, author_id, title, date, url, body) values(?, ?, ?, ?, ?, ?)")
    defer stmt.Close()
    stmt.Exec(1, 1, "Labadėna", 123456, "labadena", "Nieko aš čia nerašysiu.")
    imgpost := `This is a post with a figure.

halfimage:

![hi][halfimg]

([Full size][fullimg])

[fullimg]: /no-dox.png
[halfimg]: /no-dox-halfsize.png`
    stmt.Exec(2, 1, "Iliustruotas", 1359308741, "iliustruotas", imgpost)
    stmt, _ = xaction.Prepare("insert into tag(id, name, url) values(?, ?, ?)")
    defer stmt.Close()
    stmt.Exec(1, "Testas", "testas")
    stmt.Exec(2, "Žąsyčiai", "geese")
    stmt, _ = xaction.Prepare("insert into tagmap(id, tag_id, post_id) values(?, ?, ?)")
    defer stmt.Close()
    stmt.Exec(1, 1, 1)
    stmt.Exec(2, 2, 1)
    stmt, _ = xaction.Prepare("insert into commenter(id, name, email, www, ip) values(?, ?, ?, ?, ?)")
    defer stmt.Close()
    stmt.Exec(1, "Vytautas Šaltenis", "Vytautas.Shaltenis@gmail.com", "http://rtfb.lt", "127.0.0.1")
    stmt.Exec(2, "Vardenis Pavardenis", "niekas@niekur.com", "http://delfi.lt", "127.0.0.1")
    stmt, _ = xaction.Prepare("insert into comment(id, commenter_id, post_id, timestamp, body) values(?, ?, ?, ?, ?)")
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
        stmt, _ := xaction.Prepare("insert into post(author_id, title, date, url, body) values(?, ?, ?, ?, ?)")
        defer stmt.Close()
        date, _ := time.Parse("2006-01-02", e.Date)
        result, _ := stmt.Exec(1, e.Title, date.Unix(), e.Url, e.Body)
        postId, _ := result.LastInsertId()
        for _, t := range e.Tags {
            stmt, _ = xaction.Prepare("insert into tag(name, url) values(?, ?)")
            defer stmt.Close()
            result, _ = stmt.Exec(t.TagName, t.TagUrl)
            tagId, _ := result.LastInsertId()
            stmt, _ = xaction.Prepare("insert into tagmap(tag_id, post_id) values(?, ?)")
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

func main() {
    if len(os.Args) < 4 {
        usage()
        return
    }
    cmd := os.Args[1]
    file := os.Args[2]
    srcData := os.Args[3]
    if cmd != "init" {
        fmt.Println("Unknown command %q", cmd)
        usage()
        return
    }
    if !strings.HasSuffix(file, ".db") {
        fmt.Println("File name is supposed to have a .db extension, but was %q", file)
        return
    }
    dbFile, _ := filepath.Abs(file)
    init_db(dbFile)
    srcFile, err := os.Open(srcData)
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
        data, err := readTextEntries(srcData)
        if err != nil {
            println(err.Error())
            return
        }
        populate2(dbFile, data)
    } else {
        fmt.Printf("Import from B2Evo DB not implemented yet\n")
    }
}
