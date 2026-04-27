package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type stressConfig struct {
	BaseURL          string
	SessionCount     int
	MaxDepth         int
	Seed             int64
	Timeout          time.Duration
	Delay            time.Duration
	CreateOnTerminal bool
	Verbose          bool
	Progress         bool
}

type stressRunner struct {
	config           stressConfig
	client           *http.Client
	random           *rand.Rand
	sessionsCreated  int
	decisionsMade    int
	nodesCreated     int
	portsAttached    int
	terminalBranches int
	startedAt        time.Time
	terminalRows     int
	terminalColumns  int
	progressActive   bool
	logLines         []string
	lastRenderAt     time.Time
}

func runStressTest(args []string) error {
	config, err := parseStressConfig(args)
	if err != nil {
		return err
	}

	runner := stressRunner{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		random: rand.New(rand.NewSource(config.Seed)),
	}

	return runner.run()
}

func parseStressConfig(args []string) (stressConfig, error) {
	config := stressConfig{
		BaseURL:          "http://localhost:8080",
		SessionCount:     100,
		MaxDepth:         25,
		Seed:             time.Now().UnixNano(),
		Timeout:          10 * time.Second,
		Delay:            0,
		CreateOnTerminal: true,
		Verbose:          true,
		Progress:         true,
	}

	flags := flag.NewFlagSet("stress", flag.ContinueOnError)
	flags.StringVar(&config.BaseURL, "base-url", config.BaseURL, "base URL of a running Public Decision Tree server")
	flags.IntVar(&config.SessionCount, "sessions", config.SessionCount, "number of sessions to create")
	flags.IntVar(&config.MaxDepth, "max-depth", config.MaxDepth, "maximum decisions per generated session")
	flags.Int64Var(&config.Seed, "seed", config.Seed, "random seed for repeatable stress runs")
	flags.DurationVar(&config.Timeout, "timeout", config.Timeout, "HTTP client timeout")
	flags.DurationVar(&config.Delay, "delay", config.Delay, "delay between API operations")
	flags.BoolVar(&config.CreateOnTerminal, "create-on-terminal", config.CreateOnTerminal, "create and attach a generated yes/no node when a session reaches a dangling branch")
	flags.BoolVar(&config.Verbose, "verbose", config.Verbose, "print every API operation")
	flags.BoolVar(&config.Progress, "progress", config.Progress, "redraw a live progress bar at the bottom of the terminal output")

	if err := flags.Parse(args); err != nil {
		return stressConfig{}, err
	}

	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if config.BaseURL == "" {
		return stressConfig{}, fmt.Errorf("base-url is required")
	}
	if config.SessionCount <= 0 {
		return stressConfig{}, fmt.Errorf("sessions must be greater than zero")
	}
	if config.MaxDepth <= 0 {
		return stressConfig{}, fmt.Errorf("max-depth must be greater than zero")
	}
	if config.Timeout <= 0 {
		return stressConfig{}, fmt.Errorf("timeout must be greater than zero")
	}
	if config.Delay < 0 {
		return stressConfig{}, fmt.Errorf("delay cannot be negative")
	}

	return config, nil
}

func (runner *stressRunner) run() error {
	runner.startedAt = time.Now()
	runner.startProgress()
	defer runner.stopProgress()

	runner.logf(
		"Starting stress test base_url=%s sessions=%d max_depth=%d seed=%d create_on_terminal=%t verbose=%t",
		runner.config.BaseURL,
		runner.config.SessionCount,
		runner.config.MaxDepth,
		runner.config.Seed,
		runner.config.CreateOnTerminal,
		runner.config.Verbose,
	)

	for sessionNumber := 1; sessionNumber <= runner.config.SessionCount; sessionNumber++ {
		if err := runner.runSession(sessionNumber); err != nil {
			return err
		}
		runner.renderTerminal(false)
	}

	runner.stopProgress()
	fmt.Printf(
		"Stress test complete sessions=%d decisions=%d terminal_branches=%d nodes_created=%d ports_attached=%d",
		runner.sessionsCreated,
		runner.decisionsMade,
		runner.terminalBranches,
		runner.nodesCreated,
		runner.portsAttached,
	)
	fmt.Println()

	return nil
}

func (runner *stressRunner) runSession(sessionNumber int) error {
	createdSession, err := runner.createSession()
	if err != nil {
		return fmt.Errorf("session %d create: %w", sessionNumber, err)
	}

	runner.sessionsCreated++
	runner.logf("SESSION %d created id=%d", sessionNumber, createdSession.SessionID)

	for step := 0; step < runner.config.MaxDepth; step++ {
		session, err := runner.getSession(createdSession.SessionID)
		if err != nil {
			return fmt.Errorf("session %d fetch state: %w", createdSession.SessionID, err)
		}

		node, err := runner.getNode(session.CurrentNodeID)
		if err != nil {
			return fmt.Errorf("session %d fetch node %d: %w", createdSession.SessionID, session.CurrentNodeID, err)
		}

		choice, yesProbability := runner.weightedChoice(node.Prompt, createdSession.SessionID, step)
		port, err := runner.getPort(node.ID, choice)
		if err != nil {
			return fmt.Errorf("session %d fetch %s port for node %d: %w", createdSession.SessionID, choice, node.ID, err)
		}

		runner.logf(
			"SESSION %d step=%d node_id=%d prompt=%q choice=%s p_yes=%.2f port_id=%d",
			createdSession.SessionID,
			step,
			node.ID,
			node.Prompt,
			choice,
			yesProbability,
			port.PortID,
		)

		status, err := runner.advanceSession(createdSession.SessionID, port.PortID)
		if err != nil {
			return fmt.Errorf("session %d advance by port %d: %w", createdSession.SessionID, port.PortID, err)
		}

		runner.decisionsMade++
		runner.logf("SESSION %d step=%d advance_status=%s", createdSession.SessionID, step, status.Status)

		if status.Status == "complete" {
			runner.terminalBranches++
			if runner.config.CreateOnTerminal {
				if err := runner.extendTerminalBranch(createdSession.SessionID, node.ID, port.PortID, step); err != nil {
					return err
				}
			}
			return nil
		}

		runner.sleep()
	}

	runner.logf(
		"SESSION %d stopped_after_max_depth=%d",
		createdSession.SessionID,
		runner.config.MaxDepth,
	)
	return nil
}

func (runner *stressRunner) extendTerminalBranch(sessionID int, nodeID int, portID int, step int) error {
	prompt := runner.randomQuestion(sessionID, nodeID, step)
	createdNode, err := runner.createNode(prompt)
	if err != nil {
		return fmt.Errorf("session %d create node at terminal port %d: %w", sessionID, portID, err)
	}

	runner.nodesCreated++
	runner.logf(
		"SESSION %d terminal_port_id=%d created_node_id=%d prompt=%q",
		sessionID,
		portID,
		createdNode.NodeID,
		prompt,
	)

	if err := runner.attachPort(portID, createdNode.NodeID); err != nil {
		return fmt.Errorf("session %d attach terminal port %d to node %d: %w", sessionID, portID, createdNode.NodeID, err)
	}

	runner.portsAttached++
	runner.logf("SESSION %d attached_port_id=%d to_node_id=%d", sessionID, portID, createdNode.NodeID)
	runner.sleep()
	return nil
}

func (runner *stressRunner) createSession() (CreateSessionResponse, error) {
	var response CreateSessionResponse
	err := runner.postJSON("/api/sessions", nil, &response)
	return response, err
}

func (runner *stressRunner) getSession(sessionID int) (SessionResponse, error) {
	var response SessionResponse
	err := runner.getJSON("/api/session?session_id="+url.QueryEscape(fmt.Sprint(sessionID)), &response)
	return response, err
}

func (runner *stressRunner) getNode(nodeID int) (NodeResponse, error) {
	var response NodeResponse
	err := runner.getJSON("/api/node?node_id="+url.QueryEscape(fmt.Sprint(nodeID)), &response)
	return response, err
}

func (runner *stressRunner) getPort(nodeID int, portKey string) (PortLookupResponse, error) {
	query := url.Values{}
	query.Set("node_id", fmt.Sprint(nodeID))
	query.Set("port_key", portKey)

	var response PortLookupResponse
	err := runner.getJSON("/api/port?"+query.Encode(), &response)
	return response, err
}

func (runner *stressRunner) advanceSession(sessionID int, portID int) (SessionStatusResponse, error) {
	request := SessionAdvanceRequest{
		SessionID: sessionID,
		PortID:    portID,
	}

	var response SessionStatusResponse
	err := runner.postJSON("/api/session", request, &response)
	return response, err
}

func (runner *stressRunner) createNode(prompt string) (CreateNodeResponse, error) {
	request := CreateNodeRequest{
		Kind:   "yesno",
		Prompt: prompt,
		JSON:   "{}",
	}

	var response CreateNodeResponse
	err := runner.postJSON("/api/node", request, &response)
	return response, err
}

func (runner *stressRunner) attachPort(portID int, toNodeID int) error {
	request := PortAttachRequest{
		PortID:   portID,
		ToNodeID: toNodeID,
	}

	var response PortStatusResponse
	return runner.postJSON("/api/port", request, &response)
}

func (runner *stressRunner) getJSON(path string, target interface{}) error {
	return runner.requestJSON(http.MethodGet, path, nil, target)
}

func (runner *stressRunner) postJSON(path string, payload interface{}, target interface{}) error {
	return runner.requestJSON(http.MethodPost, path, payload, target)
}

func (runner *stressRunner) requestJSON(method string, path string, payload interface{}, target interface{}) error {
	var body io.Reader
	if payload != nil {
		requestBody, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(requestBody)
	}

	request, err := http.NewRequest(method, runner.config.BaseURL+path, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	runner.logf("HTTP %s %s", method, request.URL.String())
	response, err := runner.client.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	if target == nil {
		return nil
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	runner.logf("HTTP %s %s status=%d body=%s", method, path, response.StatusCode, strings.TrimSpace(string(responseBody)))
	return nil
}

func (runner *stressRunner) weightedChoice(prompt string, sessionID int, step int) (string, float64) {
	normalizedPrompt := strings.ToLower(prompt)
	yesProbability := 0.5

	yesCues := []string{
		"alive",
		"animal",
		"artificial",
		"brightly colored",
		"common",
		"device",
		"easy",
		"electric",
		"emergency",
		"found",
		"home",
		"indoors",
		"metal",
		"notice",
		"outdoors",
		"people",
		"safe",
		"tool",
		"use",
		"useful",
		"vehicle",
		"water",
	}
	noCues := []string{
		"ancient",
		"dangerous",
		"difficult",
		"fictional",
		"fragile",
		"hard",
		"heavier",
		"less common",
		"older",
		"rare",
		"secret",
		"spaceship",
		"under a bed",
		"unopened",
	}

	for _, cue := range yesCues {
		if strings.Contains(normalizedPrompt, cue) {
			yesProbability += 0.035
		}
	}
	for _, cue := range noCues {
		if strings.Contains(normalizedPrompt, cue) {
			yesProbability -= 0.045
		}
	}

	hour := time.Now().Hour()
	if hour >= 6 && hour <= 11 {
		yesProbability += 0.04
	}
	if hour >= 22 || hour <= 3 {
		yesProbability -= 0.05
	}
	if time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday {
		yesProbability += 0.025
	}

	// Session and step nudges make long runs less uniform while staying repeatable for a seed.
	if sessionID%5 == 0 {
		yesProbability += 0.04
	}
	if sessionID%7 == 0 {
		yesProbability -= 0.05
	}
	if step >= 4 {
		yesProbability += 0.015
	}
	if step >= 10 {
		yesProbability -= 0.03
	}

	yesProbability += runner.random.Float64()*0.18 - 0.09
	yesProbability = clampFloat(yesProbability, 0.08, 0.92)

	if runner.random.Float64() < yesProbability {
		return "yes", yesProbability
	}

	return "no", yesProbability
}

func (runner *stressRunner) randomQuestion(sessionID int, nodeID int, step int) string {
	subjects := []string{
		"a bicycle",
		"a campfire",
		"a carbonated drink",
		"a charging cable",
		"a city bus",
		"a constellation",
		"a crowded airport",
		"a cryptic email",
		"a desert animal",
		"a family recipe",
		"a forgotten password",
		"a garden hose",
		"a glass bottle",
		"a haunted rumor",
		"a handwritten note",
		"a houseplant",
		"a keychain",
		"a kitchen appliance",
		"a library book",
		"a lost wallet",
		"a mountain trail",
		"a musical instrument",
		"a neighborhood park",
		"a paper airplane",
		"a public statue",
		"a rainstorm",
		"a recipe",
		"a robot",
		"a school backpack",
		"a secret tunnel",
		"a shared calendar",
		"a small island",
		"a snow shovel",
		"a space probe",
		"a suspicious button",
		"a train station",
		"a video game character",
		"a weather balloon",
		"an ancient coin",
		"an automatic door",
		"an elevator",
		"an emergency exit",
		"an empty notebook",
		"an ocean creature",
		"an old photograph",
		"an online account",
		"an unopened package",
		"the answer",
		"the artifact",
		"the building",
		"the creature",
		"the decision",
		"the doorway",
		"the evidence",
		"the invention",
		"the landmark",
		"the machine",
		"the map",
		"the mystery",
		"the object",
		"the path",
		"the signal",
		"the sky",
		"the story",
		"the tool",
		"the vehicle",
		"this idea",
		"this place",
		"this situation",
		"your favorite snack",
		"your phone battery",
	}
	traits := []string{
		"alive",
		"allowed on an airplane",
		"artificial",
		"bigger than a refrigerator",
		"brightly colored",
		"built for speed",
		"commonly found indoors",
		"covered by insurance",
		"dangerous without training",
		"designed for children",
		"easy to carry",
		"easy to lose",
		"expensive to replace",
		"fictional",
		"fragile",
		"found in most homes",
		"hard to clean",
		"hard to move alone",
		"heavier than a person",
		"larger than a backpack",
		"made before 1980",
		"made mostly of glass",
		"made mostly of metal",
		"made mostly of plastic",
		"mentioned in movies",
		"older than the internet",
		"part of a hobby",
		"pleasant to smell",
		"powered by electricity",
		"quiet enough for a library",
		"rare in daily life",
		"safe to touch",
		"secretly useful",
		"smaller than a shoebox",
		"something people can eat",
		"something people can wear",
		"surprisingly expensive",
		"stored outside",
		"usually found outdoors",
		"used by more than one person",
		"used every day",
		"used every week",
		"visible at night",
		"waterproof",
		"younger than a teenager",
	}
	actions := []string{
		"be repaired with simple tools",
		"be safely ignored",
		"belong in a museum",
		"change color over time",
		"confuse someone from 200 years ago",
		"double as a gift",
		"fit inside a car",
		"float in water",
		"help during a road trip",
		"hide inside a closet",
		"make a loud noise",
		"move without being pushed",
		"start an argument",
		"survive a week outside",
		"teach someone a skill",
		"travel faster than a bicycle",
		"work during a power outage",
	}
	locations := []string{
		"at a beach",
		"at a grocery store",
		"at a museum",
		"at a school",
		"at a summer camp",
		"behind a locked door",
		"in a basement",
		"in a coffee shop",
		"in a forest",
		"in a hospital",
		"in a kitchen",
		"in a rainy parking lot",
		"in a spaceship",
		"in an office",
		"inside a backpack",
		"near a river",
		"on a farm",
		"on a sidewalk",
		"on public transportation",
		"under a bed",
	}
	categories := []string{
		"animal",
		"appliance",
		"building",
		"collection",
		"device",
		"food",
		"game",
		"landmark",
		"machine",
		"natural object",
		"person-made object",
		"plant",
		"tool",
		"toy",
		"vehicle",
		"warning sign",
	}
	comparisons := []string{
		"a chair",
		"a flashlight",
		"a house cat",
		"a microwave",
		"a phone",
		"a piano",
		"a suitcase",
		"a washing machine",
		"an apple",
		"an elephant",
		"an instruction manual",
		"an umbrella",
	}
	groups := []string{
		"a parent",
		"a tourist",
		"a tired student",
		"a mechanic",
		"a librarian",
		"a chef",
		"a firefighter",
		"a five-year-old",
		"most adults",
		"your neighbor",
	}
	timeframes := []string{
		"in the next hour",
		"before sunrise",
		"during a storm",
		"on a weekend",
		"after midnight",
		"during a long trip",
		"before the end of the day",
	}
	templates := []string{
		"Is %s %s?",
		"Can %s %s?",
		"Would %s usually be found %s?",
		"Is %s more like %s than %s?",
		"Would most people describe %s as a %s?",
		"Could %s be useful %s?",
		"Is %s less common than %s?",
		"Would %s trust %s?",
		"Would %s notice %s %s?",
		"Could %s become a problem %s?",
		"Would %s be happier with %s than %s?",
		"Is %s something %s would understand quickly?",
		"Would %s be difficult to explain to a child?",
		"Is %s something people would notice immediately?",
		"Could %s become important during an emergency?",
	}

	subject := subjects[runner.random.Intn(len(subjects))]
	template := templates[runner.random.Intn(len(templates))]
	question := ""
	switch template {
	case "Is %s %s?":
		question = fmt.Sprintf(template, subject, traits[runner.random.Intn(len(traits))])
	case "Can %s %s?":
		question = fmt.Sprintf(template, subject, actions[runner.random.Intn(len(actions))])
	case "Would %s usually be found %s?":
		question = fmt.Sprintf(template, subject, locations[runner.random.Intn(len(locations))])
	case "Is %s more like %s than %s?":
		first := comparisons[runner.random.Intn(len(comparisons))]
		second := comparisons[runner.random.Intn(len(comparisons))]
		for second == first {
			second = comparisons[runner.random.Intn(len(comparisons))]
		}
		question = fmt.Sprintf(template, subject, first, second)
	case "Would most people describe %s as a %s?":
		question = fmt.Sprintf(template, subject, categories[runner.random.Intn(len(categories))])
	case "Could %s be useful %s?":
		question = fmt.Sprintf(template, subject, locations[runner.random.Intn(len(locations))])
	case "Is %s less common than %s?":
		question = fmt.Sprintf(template, subject, comparisons[runner.random.Intn(len(comparisons))])
	case "Would %s trust %s?":
		question = fmt.Sprintf(template, groups[runner.random.Intn(len(groups))], subject)
	case "Would %s notice %s %s?":
		question = fmt.Sprintf(template, groups[runner.random.Intn(len(groups))], subject, timeframes[runner.random.Intn(len(timeframes))])
	case "Could %s become a problem %s?":
		question = fmt.Sprintf(template, subject, timeframes[runner.random.Intn(len(timeframes))])
	case "Would %s be happier with %s than %s?":
		question = fmt.Sprintf(template, groups[runner.random.Intn(len(groups))], subject, comparisons[runner.random.Intn(len(comparisons))])
	case "Is %s something %s would understand quickly?":
		question = fmt.Sprintf(template, subject, groups[runner.random.Intn(len(groups))])
	case "Would %s be difficult to explain to a child?":
		question = fmt.Sprintf(template, subject)
	case "Is %s something people would notice immediately?":
		question = fmt.Sprintf(template, subject)
	case "Could %s become important during an emergency?":
		question = fmt.Sprintf(template, subject)
	}

	suffix := runner.random.Intn(100000)
	return fmt.Sprintf("%s [stress s%d n%d d%d r%d]", question, sessionID, nodeID, step, suffix)
}

func (runner *stressRunner) sleep() {
	if runner.config.Delay > 0 {
		time.Sleep(runner.config.Delay)
	}
}

func (runner *stressRunner) logf(format string, args ...interface{}) {
	if runner.config.Verbose {
		runner.appendLog("%s %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
	}
}

func (runner *stressRunner) startProgress() {
	if !runner.config.Progress {
		return
	}

	runner.terminalRows = readPositiveEnvInt("LINES", 24)
	runner.terminalColumns = readPositiveEnvInt("COLUMNS", 100)
	if runner.terminalRows < 5 {
		runner.terminalRows = 5
	}
	if runner.terminalColumns < 40 {
		runner.terminalColumns = 40
	}

	runner.progressActive = true
	fmt.Fprint(os.Stdout, "\033[?25l\033[2J\033[H")
	runner.renderTerminal(true)
}

func (runner *stressRunner) stopProgress() {
	if !runner.progressActive {
		return
	}

	runner.renderTerminal(true)
	fmt.Fprintf(os.Stdout, "\033[%d;1H\033[2K\033[?25h", runner.terminalRows)
	runner.progressActive = false
}

func (runner *stressRunner) appendLog(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	if !runner.config.Progress {
		fmt.Println(line)
		return
	}

	runner.logLines = append(runner.logLines, line)

	maxLines := runner.terminalRows - 2
	if maxLines < 1 {
		maxLines = 1
	}
	if len(runner.logLines) > maxLines {
		runner.logLines = runner.logLines[len(runner.logLines)-maxLines:]
	}

	runner.renderTerminal(false)
}

func (runner *stressRunner) renderTerminal(force bool) {
	if !runner.config.Progress || runner.config.SessionCount <= 0 {
		return
	}
	if !force && !runner.lastRenderAt.IsZero() && time.Since(runner.lastRenderAt) < 50*time.Millisecond {
		return
	}
	runner.lastRenderAt = time.Now()

	if !runner.progressActive {
		fmt.Fprint(os.Stdout, runner.buildProgressLine())
		return
	}

	logRows := runner.terminalRows - 2
	if logRows < 1 {
		logRows = 1
	}

	for row := 1; row <= logRows; row++ {
		fmt.Fprintf(os.Stdout, "\033[%d;1H\033[2K", row)
		logIndex := len(runner.logLines) - logRows + row - 1
		if logIndex >= 0 && logIndex < len(runner.logLines) {
			fmt.Fprint(os.Stdout, truncateTerminalLine(runner.logLines[logIndex], runner.terminalColumns))
		}
	}

	fmt.Fprintf(os.Stdout, "\033[%d;1H\033[2K%s", runner.terminalRows, runner.buildProgressLine())
	fmt.Fprintf(os.Stdout, "\033[%d;1H", runner.terminalRows-1)
}

func (runner *stressRunner) buildProgressLine() string {
	completed := runner.sessionsCreated
	if completed > runner.config.SessionCount {
		completed = runner.config.SessionCount
	}

	width := 32
	percent := float64(completed) / float64(runner.config.SessionCount)
	filled := int(percent * float64(width))
	if filled > width {
		filled = width
	}

	elapsed := time.Since(runner.startedAt).Round(time.Second)
	rate := 0.0
	if elapsed.Seconds() > 0 {
		rate = float64(runner.sessionsCreated) / elapsed.Seconds()
	}

	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	line := fmt.Sprintf(
		"[%s] %3.0f%% sessions=%d/%d decisions=%d nodes=%d terminals=%d elapsed=%s rate=%.1f/s",
		bar,
		percent*100,
		runner.sessionsCreated,
		runner.config.SessionCount,
		runner.decisionsMade,
		runner.nodesCreated,
		runner.terminalBranches,
		elapsed,
		rate,
	)
	return truncateTerminalLine(line, runner.terminalColumns)
}

func readPositiveEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsedValue, err := strconv.Atoi(value)
	if err != nil || parsedValue <= 0 {
		return fallback
	}

	return parsedValue
}

func truncateTerminalLine(line string, width int) string {
	if width <= 1 {
		return ""
	}
	if len(line) <= width-1 {
		return line
	}

	if width <= 4 {
		return line[:width-1]
	}

	return line[:width-4] + "..."
}

func clampFloat(value float64, minimum float64, maximum float64) float64 {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}

	return value
}
