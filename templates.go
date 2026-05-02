// This file defines shared UI structures and template configuration.
//
// This file is responsible for:
//   - Defining base page data (title, timestamp)
//   - Initializing and parsing HTML templates
//   - Providing helper functions for page data
//
// Templates are parsed at startup and reused at handlers
// for rendering UI pages.
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
