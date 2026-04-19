(function () {
    "use strict";

    const dom = {
        sessionIdInput: document.getElementById("sessionIdInput"),
        sessionSearchBtn: document.getElementById("sessionSearchBtn"),
        searchError: document.getElementById("searchError"),
        searchResult: document.getElementById("searchResult"),
        searchMeta: document.getElementById("searchMeta"),
        searchSteps: document.getElementById("searchSteps"),
    };

    document.addEventListener("DOMContentLoaded", function () {
        dom.sessionSearchBtn.addEventListener("click", function () {
            void searchLookup();
        });

        dom.sessionIdInput.addEventListener("keydown", function (event) {
            if (event.key === "Enter") {
                void searchLookup();
            }
        });
    });

    async function searchLookup() {
        const rawValue = dom.sessionIdInput.value.trim();
        const sessionId = parseInt(rawValue, 10);

        if (!rawValue || isNaN(sessionId) || sessionId <= 0) {
            showError("Please enter a valid session ID.");
            return;
        }

        clearResults();
        dom.sessionSearchBtn.disabled = true;
        dom.sessionSearchBtn.textContent = "Loading...";

        try {
            const [session, history] = await Promise.all([
                fetchJSON("/api/session?session_id=" + sessionId),
                fetchJSON("/api/session/history?session_id=" + sessionId),
            ]);

            renderResult(session, history.steps);
        } catch (error) {
            showError(error.message || "Session not found.");
        } finally {
            dom.sessionSearchBtn.disabled = false;
            dom.sessionSearchBtn.textContent = "Search";
        }
    }

    function renderResult(session, steps) {
        const decisionWord = session.path_length === 1 ? "decision" : "decisions";
        dom.searchMeta.textContent =
            "Session #" + session.id + " — " +
            session.path_length + " " + decisionWord + " made.";

        dom.searchSteps.innerHTML = "";

        if (!steps || steps.length === 0) {
            dom.searchMeta.textContent += " No path recorded yet.";
            dom.searchResult.classList.remove("d-none");
            return;
        }

        steps.forEach(function (step) {
            const li = document.createElement("li");
            li.className = "mb-2";
            li.innerHTML =
                '<span class="fw-semibold">' + step.node_prompt + "</span>" +
                '<span class="text-muted ms-2">→ ' + step.port_key + "</span>";
            dom.searchSteps.appendChild(li);
        });

        dom.searchResult.classList.remove("d-none");
    }

    function showError(message) {
        dom.searchError.textContent = message;
        dom.searchError.classList.remove("d-none");
        dom.searchResult.classList.add("d-none");
    }

    function clearResults() {
        dom.searchError.classList.add("d-none");
        dom.searchError.textContent = "";
        dom.searchResult.classList.add("d-none");
        dom.searchSteps.innerHTML = "";
        dom.searchMeta.textContent = "";
    }

    async function fetchJSON(url) {
        const res = await fetch(url);
        const contentType = res.headers.get("Content-Type") || "";
        const payload = contentType.includes("application/json")
            ? await res.json()
            : await res.text();

        if (!res.ok) {
            throw new Error(typeof payload === "string" ? payload : "Request failed.");
        }

        return payload;
    }

})();