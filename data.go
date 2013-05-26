package main

import (
    "fmt"
    "strings"
)

const (
    MAX_FILE_SIZE    = 50 * 1024 * 1024 // bytes
    POSTS_PER_PAGE   = 5
    NUM_FEED_ITEMS   = 3
    NUM_RECENT_POSTS = 10
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
    RawBody   string
    Time      string
    CommentId string
}

type Entry struct {
    Author   string
    Title    string
    Date     string
    Body     string
    RawBody  string
    Url      string
    Tags     []*Tag
    Comments []*Comment
}

func (e *Entry) HasTags() bool {
    return len(e.Tags) > 0
}

func (e *Entry) HasComments() bool {
    return len(e.Comments) > 0
}

func (e *Entry) NumComments() int {
    return len(e.Comments)
}

func (e *Entry) TagsStr() string {
    parts := make([]string, 0)
    for _, t := range e.Tags {
        part := fmt.Sprintf(`<a href="/tag/%s">%s</a>`, t.TagUrl, t.TagName)
        parts = append(parts, part)
    }
    return strings.Join(parts, ", ")
}

func (e *Entry) TagsWithUrls() string {
    parts := make([]string, 0)
    for _, t := range e.Tags {
        part := fmt.Sprintf("%s", t.TagName)
        if t.TagUrl != t.TagName {
            part = fmt.Sprintf("%s>%s", t.TagName, t.TagUrl)
        }
        parts = append(parts, part)
    }
    return strings.Join(parts, ", ")
}
