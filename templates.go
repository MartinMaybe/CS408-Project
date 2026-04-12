package main

import (
	"html/template"
	"time"
)

var templates = template.Must(template.ParseFiles(
	"templates/landing.html",
	"templates/session.html",
	"templates/statistics.html",
	"templates/results.html",
))

func newPage() *Page {
	question := ""
	if node, ok := nodes[currentSession.CurrentNode]; ok {
		question = node.Question
	}

	return &Page{
		Title:    "Public Decision Tree",
		Time:     time.Now().Format("2006-01-02 15:04"),
		Question: question,
	}
}
