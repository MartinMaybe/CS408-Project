package main

import (
	"html/template"
	"log"
	"net/http"
)

type Page struct {
	Title string
	Body  string
}

var templates = template.Must(template.ParseFiles("hello.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, "hello.html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("helloHandler was called")

	p := &Page{
		Title: "Hello",
		Body:  "World",
	}
	renderTemplate(w, "hello.html", p)
}

func main() {
	http.HandleFunc("/", helloHandler)
	log.Println("Starting server on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
