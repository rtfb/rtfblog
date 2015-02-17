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
	Id   int64
	Name string `gorm:"column:tag"`
}

type TagMap struct {
	Id      int64
	TagID   int64 `gorm:"column:tag_id"`
	EntryID int64 `gorm:"column:post_id"`
}

type Author struct {
	Id       int64
	UserName string `gorm:"column:disp_name"`
	Passwd   string `gorm:"column:passwd"`
	FullName string `gorm:"column:full_name"`
	Email    string `gorm:"column:email"`
	Www      string `gorm:"column:www"`
}

type Commenter struct {
	Name      string
	Email     string
	EmailHash string
	Website   string
	IP        string
}

type Comment struct {
	Commenter
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
		format := `<a href="/tag/%s">%s</a>`
		url := t.Name
		title := Capitalize(t.Name)
		part := fmt.Sprintf(format, url, title)
		parts = append(parts, part)
	}
	return template.HTML(strings.Join(parts, ", "))
}

func (e Entry) TagsList() string {
	var parts []string
	for _, t := range e.Tags {
		parts = append(parts, t.Name)
	}
	return strings.Join(parts, ", ")
}

func (t TagMap) TableName() string {
	return "tagmap"
}
