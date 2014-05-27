package main

import (
    "fmt"
    "html/template"
    "strings"
)

const (
    MaxFileSize    = 50 * 1024 * 1024 // bytes
    PostsPerPage   = 5
    NumFeedItems   = 3
    NumRecentPosts = 10
)

type Tag struct {
    TagURL  string
    TagName string
}

type Author struct {
    UserName string
    Passwd   string
    FullName string
    Email    string
    Www      string
}

type Comment struct {
    Name      string
    Email     string
    EmailHash string
    Website   string
    IP        string
    Body      template.HTML
    RawBody   string
    Time      string
    CommentID string
}

type CommentWithPostTitle struct {
    Comment
    EntryLink
}

type EntryLink struct {
    Title  string
    URL    string
    Hidden bool
}

type Entry struct {
    EntryLink
    Author   string
    Date     string
    Body     template.HTML
    RawBody  string
    Tags     []*Tag
    Comments []*Comment
}

func (e Entry) HasTags() bool {
    return len(e.Tags) > 0
}

func (e Entry) HasComments() bool {
    return len(e.Comments) > 0
}

func (e Entry) NumCommentsStr() string {
    return L10n("{{.Count}} comments", len(e.Comments))
}

func (e Entry) TagsStr() template.HTML {
    var parts []string
    for _, t := range e.Tags {
        part := fmt.Sprintf(`<a href="/tag/%s">%s</a>`, t.TagURL, t.TagName)
        parts = append(parts, part)
    }
    return template.HTML(strings.Join(parts, ", "))
}

func (e Entry) TagsWithUrls() string {
    var parts []string
    for _, t := range e.Tags {
        part := fmt.Sprintf("%s", t.TagName)
        if t.TagURL != t.TagName {
            part = fmt.Sprintf("%s>%s", t.TagName, t.TagURL)
        }
        parts = append(parts, part)
    }
    return strings.Join(parts, ", ")
}