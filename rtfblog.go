package main

import (
    "github.com/hoisie/web"
    "github.com/russross/blackfriday"
    "io/ioutil"
    "log"
    "os"
)

func hello(val string) string {
    input, err := ioutil.ReadFile("testdata/foo.md")
    if err != nil {
        println(err.Error())
        return "err"
    }
    return string(blackfriday.MarkdownCommon(input))
}

func main() {
    f, err := os.Create("server.log")
    if err != nil {
        println(err.Error())
        return
    }
    logger := log.New(f, "", log.Ldate|log.Ltime)
    web.Get("/(.*)", hello)
    web.SetLogger(logger)
    web.Run(":8080")
}
