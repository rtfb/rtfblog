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
	ID   int64
	Name string `gorm:"column:tag"`
}

type TagMap struct {
	ID      int64
	TagID   int64 `gorm:"column:tag_id"`
	EntryID int64 `gorm:"column:post_id"`
}

type Author struct {
	ID       int64
	UserName string `gorm:"column:disp_name"`
	Passwd   string `gorm:"column:passwd"`
	FullName string `gorm:"column:full_name"`
	Email    string `gorm:"column:email"`
	Www      string `gorm:"column:www"`
}

// Commenter and Comment tables have been split up a bit to avoid a couple of
// problems:
// 1. If Comment contains Commenter substruct, gorm complains about duplicate
// 'id' columns and rightfully so. Thus, Commenter's ID is moved to
// CommenterTable.
// 2. If Comment contains Commenter and I try to insert it into a table, gorm
// tries to map Commenter's fields to 'comment' table and fails. Thus,
// CommentTable contains only the fields that map to 'comment' table.

type Commenter struct {
	Name      string
	Email     string
	EmailHash string `sql:"-"`
	Website   string `gorm:"column:www"`
	IP        string `gorm:"column:ip"`
}

type CommenterTable struct {
	ID int64
	Commenter
}

func (t CommenterTable) TableName() string {
	return "commenter"
}

type CommentTable struct {
	CommenterID int64         `gorm:"column:commenter_id"`
	PostID      int64         `gorm:"column:post_id"`
	Body        template.HTML `sql:"-"`
	RawBody     string        `gorm:"column:body"`
	Time        string        `sql:"-"`
	Timestamp   int64         `gorm:"column:timestamp"`
	CommentID   int64         `gorm:"column:id; primary_key:yes"`
}

func (t CommentTable) TableName() string {
	return "comment"
}

type Comment struct {
	Commenter
	CommentTable
}

type CommentWithPostTitle struct {
	Comment
	EntryLink
}

// Note: URL mapping is required, gorm won't be able to map to all-caps URL
// without it. Others are only for consistency.
type EntryLink struct {
	Title  string `gorm:"column:title"`
	URL    string `gorm:"column:url"`
	Hidden bool   `gorm:"column:hidden"`
}

type EntryTable struct {
	EntryLink
	ID       int64
	AuthorID int64         `gorm:"column:author_id"`
	Date     string        `sql:"-"`
	UnixDate int64         `gorm:"column:date"`
	Body     template.HTML `sql:"-"`
	RawBody  string        `gorm:"column:body"`
}

func (e EntryTable) TableName() string {
	return "post"
}

type Entry struct {
	EntryTable
	Author   string     `gorm:"column:disp_name"`
	Tags     []*Tag     `sql:"-"`
	Comments []*Comment `sql:"-"`
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
