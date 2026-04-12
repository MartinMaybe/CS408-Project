package main

type Page struct {
	Title    string
	Time     string
	Question string
}

type Session struct {
	ID          int
	CurrentNode int
	Decisions   int
}

type Node struct {
	ID       int
	Question string
	YesNext  int
	NoNext   int
}

type SessionPage struct {
	Title       string
	Time        string
	Question    string
	DecisionNum int
}

type SessionResultPage struct {
	Title       string
	Time        string
	Decisions   int
	SummaryText string
}
