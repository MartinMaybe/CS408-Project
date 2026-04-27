package main

import (
	"database/sql"
	"fmt"
	"math"
)

const statisticsMinimumVoteThreshold = 5

type statisticsSummary struct {
	TotalNodes            int
	TotalDecisions        int
	SessionsStarted       int
	AverageDepth          float64
	DeepestPath           deepestPathStat
	YesNoRatio            yesNoRatioStat
	MostDivisiveQuestion  questionHighlightStat
	LeastDivisiveQuestion questionHighlightStat
	MinimumVoteThreshold  int
}

type deepestPathStat struct {
	HasData         bool
	SessionID       int
	Depth           int
	PathFingerprint string
}

type yesNoRatioStat struct {
	HasData    bool
	YesCount   int
	NoCount    int
	TotalCount int
	YesPercent float64
	NoPercent  float64
}

type questionHighlightStat struct {
	HasData    bool
	NodeID     int
	Prompt     string
	YesCount   int
	NoCount    int
	TotalVotes int
	YesPercent float64
	NoPercent  float64
	Score      float64
	Distance   float64
}

type statisticsPageData struct {
	Title      string
	Time       string
	Statistics statisticsSummary
}

func newStatisticsPageData(page *Page, statistics statisticsSummary) statisticsPageData {
	return statisticsPageData{
		Title:      page.Title,
		Time:       page.Time,
		Statistics: statistics,
	}
}

func getStatisticsSummary() (statisticsSummary, error) {
	var summary statisticsSummary
	summary.MinimumVoteThreshold = statisticsMinimumVoteThreshold

	totalNodes, err := countRows("nodes")
	if err != nil {
		return statisticsSummary{}, err
	}
	summary.TotalNodes = totalNodes

	totalDecisions, err := countRows("session_history")
	if err != nil {
		return statisticsSummary{}, err
	}
	summary.TotalDecisions = totalDecisions

	sessionsStarted, averageDepth, err := getSessionDepthStats()
	if err != nil {
		return statisticsSummary{}, err
	}
	summary.SessionsStarted = sessionsStarted
	summary.AverageDepth = averageDepth

	deepestPath, err := getDeepestPathStat()
	if err != nil {
		return statisticsSummary{}, err
	}
	summary.DeepestPath = deepestPath

	yesNoRatio, err := getYesNoRatioStat()
	if err != nil {
		return statisticsSummary{}, err
	}
	summary.YesNoRatio = yesNoRatio

	mostDivisive, leastDivisive, err := getQuestionHighlightStats()
	if err != nil {
		return statisticsSummary{}, err
	}
	summary.MostDivisiveQuestion = mostDivisive
	summary.LeastDivisiveQuestion = leastDivisive

	return summary, nil
}

func countRows(tableName string) (int, error) {
	var count int
	err := appDB.QueryRow("SELECT COUNT(*) FROM " + tableName).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count %s: %w", tableName, err)
	}

	return count, nil
}

func getSessionDepthStats() (int, float64, error) {
	var sessionsStarted int
	var averageDepth float64
	err := appDB.QueryRow(`
		SELECT COUNT(*), COALESCE(AVG(path_length), 0)
		FROM sessions`,
	).Scan(&sessionsStarted, &averageDepth)
	if err != nil {
		return 0, 0, fmt.Errorf("query session depth stats: %w", err)
	}

	return sessionsStarted, averageDepth, nil
}

func getDeepestPathStat() (deepestPathStat, error) {
	var stat deepestPathStat
	err := appDB.QueryRow(`
		SELECT id, path_length, path_fingerprint
		FROM sessions
		ORDER BY path_length DESC, id ASC
		LIMIT 1`,
	).Scan(&stat.SessionID, &stat.Depth, &stat.PathFingerprint)
	if err == sql.ErrNoRows {
		return deepestPathStat{}, nil
	}
	if err != nil {
		return deepestPathStat{}, fmt.Errorf("query deepest path: %w", err)
	}

	stat.HasData = true
	return stat, nil
}

func getYesNoRatioStat() (yesNoRatioStat, error) {
	var stat yesNoRatioStat
	err := appDB.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN p.port_key = 'yes' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN p.port_key = 'no' THEN 1 ELSE 0 END), 0)
		FROM session_history h
		JOIN ports p ON p.id = h.port_id
		WHERE p.port_key IN ('yes', 'no')`,
	).Scan(&stat.YesCount, &stat.NoCount)
	if err != nil {
		return yesNoRatioStat{}, fmt.Errorf("query yes/no ratio: %w", err)
	}

	stat.TotalCount = stat.YesCount + stat.NoCount
	stat.HasData = stat.TotalCount > 0
	if stat.HasData {
		stat.YesPercent = percentage(stat.YesCount, stat.TotalCount)
		stat.NoPercent = percentage(stat.NoCount, stat.TotalCount)
	}

	return stat, nil
}

func getQuestionHighlightStats() (questionHighlightStat, questionHighlightStat, error) {
	rows, err := appDB.Query(`
		SELECT
			n.id,
			n.prompt,
			COALESCE(SUM(CASE WHEN p.port_key = 'yes' AND h.id IS NOT NULL THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN p.port_key = 'no' AND h.id IS NOT NULL THEN 1 ELSE 0 END), 0)
		FROM nodes n
		JOIN ports p ON p.from_node_id = n.id AND p.port_key IN ('yes', 'no')
		LEFT JOIN session_history h ON h.port_id = p.id
		WHERE n.kind = 'yesno'
		GROUP BY n.id, n.prompt
		HAVING COALESCE(SUM(CASE WHEN h.id IS NOT NULL THEN 1 ELSE 0 END), 0) >= ?
		ORDER BY n.id`,
		statisticsMinimumVoteThreshold,
	)
	if err != nil {
		return questionHighlightStat{}, questionHighlightStat{}, fmt.Errorf("query question highlights: %w", err)
	}
	defer rows.Close()

	var mostDivisive questionHighlightStat
	var leastDivisive questionHighlightStat
	for rows.Next() {
		var stat questionHighlightStat
		if err := rows.Scan(&stat.NodeID, &stat.Prompt, &stat.YesCount, &stat.NoCount); err != nil {
			return questionHighlightStat{}, questionHighlightStat{}, fmt.Errorf("scan question highlight: %w", err)
		}

		stat.TotalVotes = stat.YesCount + stat.NoCount
		stat.HasData = true
		stat.YesPercent = percentage(stat.YesCount, stat.TotalVotes)
		stat.NoPercent = percentage(stat.NoCount, stat.TotalVotes)
		stat.Distance = math.Abs(float64(stat.YesCount-stat.NoCount)) / float64(stat.TotalVotes)
		stat.Score = stat.Distance + 1/math.Sqrt(float64(stat.TotalVotes))

		if isBetterMostDivisive(stat, mostDivisive) {
			mostDivisive = stat
		}
		if isBetterLeastDivisive(stat, leastDivisive) {
			leastDivisive = stat
		}
	}
	if err := rows.Err(); err != nil {
		return questionHighlightStat{}, questionHighlightStat{}, fmt.Errorf("iterate question highlights: %w", err)
	}

	return mostDivisive, leastDivisive, nil
}

func isBetterMostDivisive(candidate questionHighlightStat, current questionHighlightStat) bool {
	if !current.HasData {
		return true
	}
	if candidate.Score != current.Score {
		return candidate.Score < current.Score
	}
	if candidate.TotalVotes != current.TotalVotes {
		return candidate.TotalVotes > current.TotalVotes
	}

	return candidate.NodeID < current.NodeID
}

func isBetterLeastDivisive(candidate questionHighlightStat, current questionHighlightStat) bool {
	if !current.HasData {
		return true
	}
	if candidate.Distance != current.Distance {
		return candidate.Distance > current.Distance
	}
	if candidate.TotalVotes != current.TotalVotes {
		return candidate.TotalVotes > current.TotalVotes
	}

	return candidate.NodeID < current.NodeID
}

func percentage(count int, total int) float64 {
	if total == 0 {
		return 0
	}

	return float64(count) / float64(total) * 100
}
