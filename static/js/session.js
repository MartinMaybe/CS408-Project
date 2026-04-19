/*
 * Session page controller
 *
 * The server renders only the Bootstrap page shell. This script owns:
 * - session creation and resume behavior
 * - traversal through the existing session API
 * - terminal-branch UI state
 * - checkpoint-4 node creation and dangling-port attachment
 *
 * Architectural note:
 * The active session is considered complete once it selects a dangling port.
 * Node creation does not mutate the session forward. Instead, the browser:
 * 1. remembers which dangling port ended the branch
 * 2. creates a new node
 * 3. attaches that node to the dangling port
 *
 * That keeps checkpoint-4 creation mostly client-side while only adding the
 * minimal backend support needed to attach a port to a node.
 */

(function () {
    "use strict";

    const SESSION_STORAGE_KEY = "public-decision-tree.session_id";
    const COMPLETION_STORAGE_KEY = "public-decision-tree.completion_state";

    const state = {
        sessionId: null,
        session: null,
        node: null,
        isBusy: false,
        isComplete: false,
        pendingPortId: null,
        attachedNodeId: null,
    };

    const dom = {
        sessionMeta: document.getElementById("sessionMeta"),
        sessionAlert: document.getElementById("sessionAlert"),
        loadingSpinner: document.getElementById("loadingSpinner"),
        question: document.getElementById("question"),
        decisionCount: document.getElementById("decisionCount"),
        completionPanel: document.getElementById("completionPanel"),
        resultsPanel: document.getElementById("resultsPanel"),
        resultsSummary: document.getElementById("resultsSummary"),
        journeyPanel: document.getElementById("journeyPanel"),
        journeyLoading: document.getElementById("journeyLoading"),
        journeySteps: document.getElementById("journeySteps"),
        creationPanel: document.getElementById("creationPanel"),
        creationForm: document.getElementById("creationForm"),
        createKind: document.getElementById("createKind"),
        createPrompt: document.getElementById("createPrompt"),
        createSubmitButton: document.getElementById("createSubmitBtn"),
        decisionPanel: document.getElementById("decisionPanel"),
        yesButton: document.getElementById("yesBtn"),
        noButton: document.getElementById("noBtn"),
        yesMeta: document.getElementById("yesMeta"),
        noMeta: document.getElementById("noMeta"),
        restartButton: document.getElementById("restartBtn"),
    };

    document.addEventListener("DOMContentLoaded", initializeSessionPage);

    async function initializeSessionPage() {
        bindEvents();
        renderLoadingState("Loading session...");

        try {
            state.sessionId = await resolveSessionId();
            await refreshSessionState();
            restoreCompletionState();
            renderSessionState();
        } catch (error) {
            renderErrorState(error);
        }
    }

    function bindEvents() {
        dom.yesButton.addEventListener("click", function () {
            void handleDecision("yes");
        });

        dom.noButton.addEventListener("click", function () {
            void handleDecision("no");
        });

        dom.restartButton.addEventListener("click", function () {
            void restartSession();
        });

        dom.creationForm.addEventListener("submit", function (event) {
            event.preventDefault();
            void handleCreationSubmit();
        });
    }

    async function resolveSessionId() {
        const query = new URLSearchParams(window.location.search);
        const forceNewSession = query.get("new") === "1";

        if (forceNewSession) {
            return createAndPersistSession();
        }

        const querySessionId = parsePositiveInteger(query.get("session_id"));
        if (querySessionId !== null) {
            persistSessionId(querySessionId);
            syncSessionIdInURL(querySessionId);
            return querySessionId;
        }

        const storedSessionId = parsePositiveInteger(window.localStorage.getItem(SESSION_STORAGE_KEY));
        if (storedSessionId !== null) {
            syncSessionIdInURL(storedSessionId);
            return storedSessionId;
        }

        return createAndPersistSession();
    }

    async function createAndPersistSession() {
        clearCompletionState();

        const payload = await requestJSON("/api/sessions", {
            method: "POST",
        });

        persistSessionId(payload.session_id);
        syncSessionIdInURL(payload.session_id);
        return payload.session_id;
    }

    async function refreshSessionState() {
        state.session = await fetchSession(state.sessionId);
        state.node = await fetchNode(state.session.current_node_id);
    }

    async function handleDecision(choice) {
        if (state.isBusy || state.sessionId === null || state.node === null || state.isComplete) {
            return;
        }

        const choiceLabel = choice === "yes" ? "Yes" : "No";
        renderBusyDecisionState(choiceLabel);

        try {
            const port = await fetchPort(state.node.id, choice);
            const status = await advanceSession(state.sessionId, port.port_id);

            await refreshSessionState();

            if (status.status === "complete") {
                state.isComplete = true;
                state.pendingPortId = port.port_id;
                state.attachedNodeId = null;
                persistCompletionState();
                showAlert(
                    "This branch currently ends here. Add a follow-up question below to extend it.",
                    "warning"
                );
            } else {
                clearCompletionState();
                clearAlert();
            }

            renderSessionState();
        } catch (error) {
            renderErrorState(error);
        }
    }

    async function handleCreationSubmit() {
        if (state.isBusy || !state.isComplete || state.pendingPortId === null) {
            return;
        }

        const prompt = dom.createPrompt.value.trim();
        const kind = dom.createKind.value.trim();
        if (prompt === "") {
            showAlert("A question prompt is required.", "danger");
            dom.createPrompt.focus();
            return;
        }

        renderBusyCreationState();

        try {
            const createdNode = await createNode(kind, prompt);
            await attachPort(state.pendingPortId, createdNode.node_id);

            state.pendingPortId = null;
            state.attachedNodeId = createdNode.node_id;
            persistCompletionState();

            dom.creationForm.reset();
            dom.createKind.value = "yesno";

            renderSessionState();
            showAlert("Question attached successfully. Start a new session to reach it in the tree.", "success");
        } catch (error) {
            renderErrorState(error);
        }
    }

    async function restartSession() {
        if (state.isBusy) {
            return;
        }

        renderLoadingState("Starting a new session...");

        try {
            state.sessionId = await createAndPersistSession();
            state.isComplete = false;
            state.pendingPortId = null;
            state.attachedNodeId = null;
            await refreshSessionState();
            renderSessionState();
            showAlert("Started a new session.", "success");
        } catch (error) {
            renderErrorState(error);
        }
    }

    function restoreCompletionState() {
        const rawValue = window.localStorage.getItem(COMPLETION_STORAGE_KEY);
        if (!rawValue) {
            state.isComplete = false;
            state.pendingPortId = null;
            state.attachedNodeId = null;
            return;
        }

        try {
            const storedValue = JSON.parse(rawValue);
            if (!storedValue || storedValue.sessionId !== state.sessionId) {
                clearCompletionState();
                return;
            }

            state.isComplete = true;
            state.pendingPortId = parsePositiveInteger(storedValue.pendingPortId);
            state.attachedNodeId = parsePositiveInteger(storedValue.attachedNodeId);
        } catch (_error) {
            clearCompletionState();
        }
    }

    function persistCompletionState() {
        if (!state.isComplete) {
            clearCompletionState();
            return;
        }

        window.localStorage.setItem(
            COMPLETION_STORAGE_KEY,
            JSON.stringify({
                sessionId: state.sessionId,
                pendingPortId: state.pendingPortId,
                attachedNodeId: state.attachedNodeId,
            })
        );
    }

    function clearCompletionState() {
        state.isComplete = false;
        state.pendingPortId = null;
        state.attachedNodeId = null;
        window.localStorage.removeItem(COMPLETION_STORAGE_KEY);
    }

    async function fetchSession(sessionId) {
        return requestJSON("/api/session?session_id=" + encodeURIComponent(sessionId));
    }

    async function fetchNode(nodeId) {
        return requestJSON("/api/node?node_id=" + encodeURIComponent(nodeId));
    }

    async function fetchPort(nodeId, portKey) {
        const query = new URLSearchParams({
            node_id: String(nodeId),
            port_key: portKey,
        });

        return requestJSON("/api/port?" + query.toString());
    }

    async function advanceSession(sessionId, portId) {
        return requestJSON("/api/session", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                session_id: sessionId,
                port_id: portId,
            }),
        });
    }

    async function createNode(kind, prompt) {
        return requestJSON("/api/node", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                kind: kind,
                prompt: prompt,
                json: "{}",
            }),
        });
    }

    async function attachPort(portId, toNodeId) {
        return requestJSON("/api/port", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                port_id: portId,
                to_node_id: toNodeId,
            }),
        });
    }

    async function requestJSON(url, options) {
        const response = await fetch(url, options);
        const contentType = response.headers.get("Content-Type") || "";
        const payload = contentType.includes("application/json")
            ? await response.json()
            : await response.text();

        if (!response.ok) {
            throw new Error(extractErrorMessage(payload, response.statusText));
        }

        return payload;
    }

    async function loadJourneyHistory() {
        try {
            const data = await requestJSON(
                "/api/session/history?session_id=" + encodeURIComponent(state.sessionId)
            );
            renderJourneySteps(data.steps);
        } catch (error) {
            dom.journeyLoading.textContent = "Failed to load session path";
        }
    }

    function renderLoadingState(message) {
        state.isBusy = true;
        dom.loadingSpinner.classList.remove("d-none");
        dom.question.textContent = message;
        dom.decisionCount.textContent = "0 decisions made";
        dom.sessionMeta.textContent = "Preparing session...";
        setCompletionPanel("warning", "This branch currently ends here.");
        dom.resultsPanel.classList.add("d-none");
        dom.journeyPanel.classList.add("d-none")
        dom.creationPanel.classList.add("d-none");
        setDecisionButtonsEnabled(false);
        setCreationControlsEnabled(false);
        setDecisionMeta(
            "Waiting for the session to load.",
            "Waiting for the session to load."
        );
    }

    function renderBusyDecisionState(choiceLabel) {
        state.isBusy = true;
        dom.loadingSpinner.classList.remove("d-none");
        dom.question.textContent = "Submitting " + choiceLabel + "...";
        setDecisionButtonsEnabled(false);
        setCreationControlsEnabled(false);
        setDecisionMeta(
            "Please wait while the next card loads.",
            "Please wait while the next card loads."
        );
    }

    function renderBusyCreationState() {
        state.isBusy = true;
        dom.loadingSpinner.classList.remove("d-none");
        setDecisionButtonsEnabled(false);
        setCreationControlsEnabled(false);
    }

    function renderSessionState() {
        state.isBusy = false;
        dom.loadingSpinner.classList.add("d-none");
        dom.question.textContent = state.node ? state.node.prompt : "Unable to load node.";
        dom.decisionCount.textContent = formatDecisionCount(state.session ? state.session.path_length : 0);
        dom.sessionMeta.textContent = buildSessionMeta();

        if (state.isComplete) {
            renderCompletionState();
            return;
        }

        renderActiveTraversalState();
    }

    function renderActiveTraversalState() {
        dom.resultsPanel.classList.add("d-none");
        dom.creationPanel.classList.add("d-none");
        dom.completionPanel.classList.add("d-none");
        dom.decisionPanel.classList.remove("opacity-50");
        setDecisionButtonsEnabled(true);
        setCreationControlsEnabled(false);
        setDecisionMeta(
            'Select to follow the "No" branch.',
            'Select to follow the "Yes" branch.'
        );
    }

    function renderCompletionState() {
        dom.resultsPanel.classList.remove("d-none");
        dom.completionPanel.classList.remove("d-none");
        dom.journeyPanel.classList.remove("d-none");
        void loadJourneyHistory();
        dom.decisionPanel.classList.add("opacity-50");
        setDecisionButtonsEnabled(false);

        if (state.attachedNodeId !== null) {
            setCompletionPanel("success", "This branch has been extended successfully.");
            dom.resultsSummary.textContent = buildExtendedResultsSummary();
            dom.creationPanel.classList.add("d-none");
            setCreationControlsEnabled(false);
            setDecisionMeta(
                "This completed session has already been extended.",
                "This completed session has already been extended."
            );
            return;
        }

        setCompletionPanel("warning", "This branch currently ends here. Add a new question below to extend it.");
        dom.resultsSummary.textContent = buildCompletedResultsSummary();
        dom.creationPanel.classList.remove("d-none");
        setCreationControlsEnabled(true);
        setDecisionMeta(
            "This path has ended.",
            "This path has ended."
        );
    }

    function renderJourneySteps(steps) {
        if (!steps || steps.length === 0) {
            dom.journeyLoading.textContent = "No steps recorded for this session";
            return;
        }

        dom.journeySteps.innerHTML = "";

        steps.forEach(function (step) {
            const li = document.createElement("li");
            li.className = "mb-2";
            li.innerHTML = 
                `<span class="fw-semibold">` + step.node_prompt + "</span>" +
                `<span class="text-muted ms-2">→ ` +  step.port_key + "</span>";
            dom.journeySteps.appendChild(li);
        });

        dom.journeyLoading.style.display = "none";
        dom.journeySteps.style.display = "block";
    }

    function renderErrorState(error) {
        state.isBusy = false;
        dom.loadingSpinner.classList.add("d-none");
        dom.question.textContent = "Unable to load the session.";
        dom.sessionMeta.textContent = "Session unavailable";
        dom.decisionCount.textContent = "0 decisions made";
        dom.resultsPanel.classList.add("d-none");
        dom.creationPanel.classList.add("d-none");
        dom.completionPanel.classList.add("d-none");
        setDecisionButtonsEnabled(false);
        setCreationControlsEnabled(false);
        setDecisionMeta(
            "Fix the error and try again.",
            "Fix the error and try again."
        );
        showAlert(error instanceof Error ? error.message : String(error), "danger");
    }

    function buildSessionMeta() {
        if (!state.session || !state.node) {
            return "Preparing session...";
        }

        return "Session #" + state.session.id + " • Node #" + state.node.id + " • " + formatDecisionCount(state.session.path_length);
    }

    function buildCompletedResultsSummary() {
        const pathLength = state.session ? state.session.path_length : 0;
        const prompt = state.node ? state.node.prompt : "Unknown question";

        return "You completed " + formatDecisionCount(pathLength) + ". The branch ended after \"" + prompt + "\" and is ready for a new question.";
    }

    function buildExtendedResultsSummary() {
        const pathLength = state.session ? state.session.path_length : 0;
        const prompt = state.node ? state.node.prompt : "Unknown question";

        return "You completed " + formatDecisionCount(pathLength) + ". The branch that ended after \"" + prompt + "\" has now been extended with a new question.";
    }

    function setCompletionPanel(tone, message) {
        dom.completionPanel.textContent = message;
        dom.completionPanel.className = "alert alert-" + tone;
    }

    function showAlert(message, tone) {
        dom.sessionAlert.textContent = message;
        dom.sessionAlert.className = "alert alert-" + tone;
        dom.sessionAlert.classList.remove("d-none");
    }

    function clearAlert() {
        dom.sessionAlert.textContent = "";
        dom.sessionAlert.className = "alert d-none";
    }

    function setDecisionButtonsEnabled(isEnabled) {
        dom.yesButton.disabled = !isEnabled;
        dom.noButton.disabled = !isEnabled;
    }

    function setCreationControlsEnabled(isEnabled) {
        dom.createKind.disabled = !isEnabled;
        dom.createPrompt.disabled = !isEnabled;
        dom.createSubmitButton.disabled = !isEnabled;
    }

    function setDecisionMeta(noText, yesText) {
        dom.noMeta.textContent = noText;
        dom.yesMeta.textContent = yesText;
    }

    function formatDecisionCount(count) {
        const suffix = count === 1 ? "decision" : "decisions";
        return count + " " + suffix;
    }

    function persistSessionId(sessionId) {
        window.localStorage.setItem(SESSION_STORAGE_KEY, String(sessionId));
    }

    function syncSessionIdInURL(sessionId) {
        const url = new URL(window.location.href);
        url.searchParams.set("session_id", String(sessionId));
        url.searchParams.delete("new");
        window.history.replaceState({}, "", url);
    }

    function parsePositiveInteger(value) {
        if (value === null || value === "") {
            return null;
        }

        const parsedValue = Number.parseInt(value, 10);
        if (!Number.isInteger(parsedValue) || parsedValue <= 0) {
            return null;
        }

        return parsedValue;
    }

    function extractErrorMessage(payload, fallbackMessage) {
        if (typeof payload === "string" && payload.trim() !== "") {
            return payload;
        }

        if (payload && typeof payload === "object" && typeof payload.error === "string") {
            return payload.error;
        }

        return fallbackMessage || "Unexpected request failure.";
    }
})();
