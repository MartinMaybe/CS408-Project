package main

import (
	"html/template"
	"time"
)

type Page struct {
	Title string
	Time  string
}

var templates = template.Must(template.ParseFiles(
	"templates/partials/header.html",
	"templates/partials/footer.html",
	"templates/landing.html",
	"templates/session.html",
	"templates/statistics.html",
))

func newPage(title string) *Page {
	return &Page{
		Title: title,
		Time:  time.Now().Format("2006-01-02 15:04"),
	}
}
