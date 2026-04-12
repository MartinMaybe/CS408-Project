package main

import (
	"html/template"
	"time"
)

type Page struct {
	Title    string
	Time     string
	Question string
}

var templates = template.Must(template.ParseFiles(
	"templates/partials/header.html",
	"templates/partials/footer.html",
	"templates/landing.html",
	"templates/session.html",
	"templates/statistics.html",
	"templates/results.html",
))

func newPage(title string) *Page {
	question := ""
	if node, ok := nodes[currentSession.CurrentNode]; ok {
		question = node.Question
	}

	return &Page{
		Title:    title,
		Time:     time.Now().Format("2006-01-02 15:04"),
		Question: question,
	}
}
