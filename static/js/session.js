/*
 * Session page controller
 *
 * This file owns the browser-side flow for the one-page session experience.
 * The server only renders the shell; everything else is requested from the API.
 *
 * Main responsibilities:
 * - decide whether to create a new session or resume an existing one
 * - keep the current session ID in the URL and localStorage
 * - fetch session/node data from the API
 * - resolve yes/no selections into port IDs
 * - submit decisions and redraw the card without a full page reload
 * - keep the Bootstrap shell in sync with loading, error, and completion states
 */

(function () {
    "use strict";

    const STORAGE_KEY = "public-decision-tree.session_id";

    const state = {
        sessionId: null,
        session: null,
        node: null,
        isBusy: false,
        isComplete: false,
    };

    const dom = {
        sessionMeta: document.getElementById("sessionMeta"),
        sessionAlert: document.getElementById("sessionAlert"),
        loadingSpinner: document.getElementById("loadingSpinner"),
        question: document.getElementById("question"),
        decisionCount: document.getElementById("decisionCount"),
        completionPanel: document.getElementById("completionPanel"),
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

        const storedSessionId = parsePositiveInteger(window.localStorage.getItem(STORAGE_KEY));
        if (storedSessionId !== null) {
            syncSessionIdInURL(storedSessionId);
            return storedSessionId;
        }

        return createAndPersistSession();
    }

    async function createAndPersistSession() {
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
        if (state.isBusy || state.sessionId === null || state.node === null) {
            return;
        }

        const choiceLabel = choice === "yes" ? "Yes" : "No";
        renderBusyDecisionState(choiceLabel);

        try {
            const port = await fetchPort(state.node.id, choice);
            const status = await advanceSession(state.sessionId, port.port_id);

            state.isComplete = status.status === "complete";
            await refreshSessionState();
            renderSessionState();

            if (state.isComplete) {
                showAlert(
                    "This branch currently ends here. The next question can be attached here in a later checkpoint.",
                    "warning"
                );
            } else {
                clearAlert();
            }
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
            await refreshSessionState();
            renderSessionState();
            showAlert("Started a new session.", "success");
        } catch (error) {
            renderErrorState(error);
        }
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

    function renderLoadingState(message) {
        state.isBusy = true;
        dom.loadingSpinner.classList.remove("d-none");
        dom.question.textContent = message;
        dom.decisionCount.textContent = "0 decisions made";
        dom.sessionMeta.textContent = "Preparing session...";
        dom.completionPanel.classList.add("d-none");
        setDecisionButtonsEnabled(false);
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
        setDecisionMeta(
            "Please wait while the next card loads.",
            "Please wait while the next card loads."
        );
    }

    function renderSessionState() {
        state.isBusy = false;
        dom.loadingSpinner.classList.add("d-none");
        dom.question.textContent = state.node ? state.node.prompt : "Unable to load node.";
        dom.decisionCount.textContent = formatDecisionCount(state.session ? state.session.path_length : 0);
        dom.sessionMeta.textContent = buildSessionMeta();
        dom.completionPanel.classList.toggle("d-none", !state.isComplete);
        dom.decisionPanel.classList.toggle("opacity-50", state.isComplete);

        if (state.isComplete) {
            setDecisionButtonsEnabled(false);
            setDecisionMeta(
                "This path has ended.",
                "This path has ended."
            );
            return;
        }

        setDecisionButtonsEnabled(true);
        setDecisionMeta(
            'Select to follow the "No" branch.',
            'Select to follow the "Yes" branch.'
        );
    }

    function renderErrorState(error) {
        state.isBusy = false;
        dom.loadingSpinner.classList.add("d-none");
        dom.question.textContent = "Unable to load the session.";
        dom.sessionMeta.textContent = "Session unavailable";
        dom.decisionCount.textContent = "0 decisions made";
        dom.completionPanel.classList.add("d-none");
        setDecisionButtonsEnabled(false);
        setDecisionMeta(
            "Fix the error and try again.",
            "Fix the error and try again."
        );
        showAlert(error instanceof Error ? error.message : String(error), "danger");
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

    function setDecisionMeta(noText, yesText) {
        dom.noMeta.textContent = noText;
        dom.yesMeta.textContent = yesText;
    }

    function buildSessionMeta() {
        if (!state.session || !state.node) {
            return "Preparing session...";
        }

        return "Session #" + state.session.id + " • Node #" + state.node.id;
    }

    function formatDecisionCount(count) {
        const suffix = count === 1 ? "decision" : "decisions";
        return count + " " + suffix + " made";
    }

    function persistSessionId(sessionId) {
        window.localStorage.setItem(STORAGE_KEY, String(sessionId));
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
