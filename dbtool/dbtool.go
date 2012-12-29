package main

import (
    "database/sql"
    "fmt"
    _ "github.com/mattn/go-sqlite3"
    "os"
    "strings"
)

func usage() {
    help := []string{
        "Usage:",
        os.Args[0] + " <command> [params...]",
        "",
        "possible commands:",
        "\tinit <file.db> -- init clean db with schema",
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
    xaction, err := db.Begin()
    if err != nil {
        fmt.Println(err)
        return
    }
    stmt, err := xaction.Prepare("insert into author(id, disp_name, full_name, email, www) values(?, ?, ?, ?, ?)")
    defer stmt.Close()
    stmt.Exec(1, "rtfb", "Vytautas Šaltenis", "vytas@rtfb.lt", "http://rtfb.lt")
    stmt, _ = xaction.Prepare("insert into post(id, author_id, title, date, url, body) values(?, ?, ?, ?, ?, ?)")
    defer stmt.Close()
    stmt.Exec(1, 1, "Labadėna", 123456, "labadena", "Nieko aš čia nerašysiu.")
    stmt, _ = xaction.Prepare("insert into tag(id, name, url) values(?, ?, ?)")
    defer stmt.Close()
    stmt.Exec(1, "Testas", "testas")
    stmt.Exec(2, "Žąsyčiai", "geese")
    stmt, _ = xaction.Prepare("insert into tagmap(id, tag_id, post_id) values(?, ?, ?)")
    defer stmt.Close()
    stmt.Exec(1, 1, 1)
    stmt.Exec(2, 2, 1)
    xaction.Commit()
}

func main() {
    if len(os.Args) < 3 {
        usage()
        return
    }
    cmd := os.Args[1]
    file := os.Args[2]
    if cmd != "init" {
        fmt.Println("Unknown command %q", cmd)
        usage()
        return
    }
    if !strings.HasSuffix(file, ".db") {
        fmt.Println("File name is supposed to have a .db extensios, but was %q", file)
        return
    }
    init_db(file)
    populate(file)
}
