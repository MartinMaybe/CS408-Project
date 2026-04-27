package main

import (
	"math"
	"testing"
)

func TestGetStatisticsSummaryReturnsEmptyStats(t *testing.T) {
	setupTestAppDB(t)

	statistics, err := getStatisticsSummary()
	if err != nil {
		t.Fatalf("getStatisticsSummary returned error: %v", err)
	}

	if statistics.TotalNodes != 7 {
		t.Fatalf("expected 7 total nodes, got %d", statistics.TotalNodes)
	}
	if statistics.TotalDecisions != 0 {
		t.Fatalf("expected 0 total decisions, got %d", statistics.TotalDecisions)
	}
	if statistics.SessionsStarted != 0 {
		t.Fatalf("expected 0 sessions started, got %d", statistics.SessionsStarted)
	}
	if statistics.AverageDepth != 0 {
		t.Fatalf("expected average depth 0, got %f", statistics.AverageDepth)
	}
	if statistics.DeepestPath.HasData {
		t.Fatalf("expected no deepest path data")
	}
	if statistics.YesNoRatio.HasData {
		t.Fatalf("expected no yes/no ratio data")
	}
	if statistics.MostDivisiveQuestion.HasData || statistics.LeastDivisiveQuestion.HasData {
		t.Fatalf("expected no question highlights")
	}
}

func TestGetStatisticsSummaryReturnsGlobalMetrics(t *testing.T) {
	setupTestAppDB(t)

	if _, err := createNode("yesno", "Is it imaginary?", "{}"); err != nil {
		t.Fatalf("createNode returned error: %v", err)
	}

	deepestSessionID := createSessionWithChoices(t, "yes", "yes", "yes")
	createSessionWithChoices(t, "no")

	statistics, err := getStatisticsSummary()
	if err != nil {
		t.Fatalf("getStatisticsSummary returned error: %v", err)
	}

	if statistics.TotalNodes != 8 {
		t.Fatalf("expected 8 total nodes, got %d", statistics.TotalNodes)
	}
	if statistics.TotalDecisions != 4 {
		t.Fatalf("expected 4 total decisions, got %d", statistics.TotalDecisions)
	}
	if statistics.SessionsStarted != 2 {
		t.Fatalf("expected 2 sessions started, got %d", statistics.SessionsStarted)
	}
	if math.Abs(statistics.AverageDepth-2) > 0.001 {
		t.Fatalf("expected average depth 2.0, got %f", statistics.AverageDepth)
	}
	if !statistics.DeepestPath.HasData {
		t.Fatalf("expected deepest path data")
	}
	if statistics.DeepestPath.SessionID != deepestSessionID {
		t.Fatalf("expected deepest session ID %d, got %d", deepestSessionID, statistics.DeepestPath.SessionID)
	}
	if statistics.DeepestPath.Depth != 3 {
		t.Fatalf("expected deepest path depth 3, got %d", statistics.DeepestPath.Depth)
	}
	if statistics.DeepestPath.PathFingerprint == "" {
		t.Fatalf("expected deepest path fingerprint")
	}
	if !statistics.YesNoRatio.HasData {
		t.Fatalf("expected yes/no ratio data")
	}
	if statistics.YesNoRatio.YesCount != 3 || statistics.YesNoRatio.NoCount != 1 {
		t.Fatalf("expected yes/no counts 3/1, got %d/%d", statistics.YesNoRatio.YesCount, statistics.YesNoRatio.NoCount)
	}
	if math.Abs(statistics.YesNoRatio.YesPercent-75) > 0.001 || math.Abs(statistics.YesNoRatio.NoPercent-25) > 0.001 {
		t.Fatalf("unexpected yes/no percentages: %f/%f", statistics.YesNoRatio.YesPercent, statistics.YesNoRatio.NoPercent)
	}
}

func TestGetStatisticsSummaryAppliesMinimumVoteThreshold(t *testing.T) {
	setupTestAppDB(t)

	createSessionWithChoices(t, "yes")
	createSessionWithChoices(t, "yes")
	createSessionWithChoices(t, "no")
	createSessionWithChoices(t, "no")

	statistics, err := getStatisticsSummary()
	if err != nil {
		t.Fatalf("getStatisticsSummary returned error: %v", err)
	}
	if statistics.MostDivisiveQuestion.HasData || statistics.LeastDivisiveQuestion.HasData {
		t.Fatalf("expected no highlights below vote threshold")
	}

	createSessionWithChoices(t, "yes")

	statistics, err = getStatisticsSummary()
	if err != nil {
		t.Fatalf("getStatisticsSummary after threshold returned error: %v", err)
	}
	if !statistics.MostDivisiveQuestion.HasData || !statistics.LeastDivisiveQuestion.HasData {
		t.Fatalf("expected highlights once vote threshold is reached")
	}
	if statistics.MostDivisiveQuestion.NodeID != 1 || statistics.LeastDivisiveQuestion.NodeID != 1 {
		t.Fatalf("expected root node to qualify, got most=%d least=%d", statistics.MostDivisiveQuestion.NodeID, statistics.LeastDivisiveQuestion.NodeID)
	}
}

func TestGetStatisticsSummaryRanksQuestionHighlights(t *testing.T) {
	setupTestAppDB(t)

	for i := 0; i < 3; i++ {
		createSessionWithChoices(t, "yes", "yes")
	}
	for i := 0; i < 3; i++ {
		createSessionWithChoices(t, "yes", "no")
	}
	for i := 0; i < 4; i++ {
		createSessionWithChoices(t, "yes")
	}
	for i := 0; i < 6; i++ {
		createSessionWithChoices(t, "no", "yes")
	}
	for i := 0; i < 4; i++ {
		createSessionWithChoices(t, "no")
	}

	statistics, err := getStatisticsSummary()
	if err != nil {
		t.Fatalf("getStatisticsSummary returned error: %v", err)
	}

	if !statistics.MostDivisiveQuestion.HasData {
		t.Fatalf("expected most divisive question")
	}
	if statistics.MostDivisiveQuestion.NodeID != 1 {
		t.Fatalf("expected root node to be most divisive, got node %d", statistics.MostDivisiveQuestion.NodeID)
	}
	if statistics.MostDivisiveQuestion.YesCount != 10 || statistics.MostDivisiveQuestion.NoCount != 10 {
		t.Fatalf("expected root yes/no counts 10/10, got %d/%d", statistics.MostDivisiveQuestion.YesCount, statistics.MostDivisiveQuestion.NoCount)
	}

	if !statistics.LeastDivisiveQuestion.HasData {
		t.Fatalf("expected least divisive question")
	}
	if statistics.LeastDivisiveQuestion.NodeID != 3 {
		t.Fatalf("expected node 3 to be least divisive, got node %d", statistics.LeastDivisiveQuestion.NodeID)
	}
	if statistics.LeastDivisiveQuestion.YesCount != 6 || statistics.LeastDivisiveQuestion.NoCount != 0 {
		t.Fatalf("expected node 3 yes/no counts 6/0, got %d/%d", statistics.LeastDivisiveQuestion.YesCount, statistics.LeastDivisiveQuestion.NoCount)
	}
}

func createSessionWithChoices(t *testing.T, choices ...string) int {
	t.Helper()

	sessionID, err := createSession()
	if err != nil {
		t.Fatalf("createSession returned error: %v", err)
	}

	currentNodeID := rootNodeID
	for _, choice := range choices {
		portID, err := getPortIDByNodeAndKey(currentNodeID, choice)
		if err != nil {
			t.Fatalf("getPortIDByNodeAndKey(%d, %q) returned error: %v", currentNodeID, choice, err)
		}

		status, err := advanceSessionByPort(sessionID, portID)
		if err != nil {
			t.Fatalf("advanceSessionByPort returned error: %v", err)
		}
		if status == "complete" {
			continue
		}

		sessionRecord, err := getSessionByID(sessionID)
		if err != nil {
			t.Fatalf("getSessionByID returned error: %v", err)
		}
		currentNodeID = sessionRecord.CurrentNodeID
	}

	return sessionID
}
