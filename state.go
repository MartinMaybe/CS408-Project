package main

var currentSession = Session{}

var nodes = map[int]Node{
	1: {ID: 1, Question: "Is it an animal?", YesNext: 2, NoNext: 3},
	2: {ID: 2, Question: "Does it have fur?", YesNext: 4, NoNext: 5},
	3: {ID: 3, Question: "Is it a plant?", YesNext: 6, NoNext: 7},
	4: {ID: 4, Question: "Is it a dog?", YesNext: 0, NoNext: 0},
	5: {ID: 5, Question: "Is it a reptile?", YesNext: 0, NoNext: 0},
	6: {ID: 6, Question: "Is it a tree?", YesNext: 0, NoNext: 0},
	7: {ID: 7, Question: "Is it a flower?", YesNext: 0, NoNext: 0},
}
