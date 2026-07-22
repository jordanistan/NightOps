(() => {
  "use strict";

  const cacheKey = "nightops-companion-missions-v1";
  const missions = document.querySelector("#missions");
  const detail = document.querySelector("#detail");
  const notice = document.querySelector("#notice");
  const connectionText = document.querySelector("#connection-text");
  const connectionDot = document.querySelector("#connection-dot");
  const authValueInput = document.querySelector("#auth-value");
  let syncEnabled = false;
  let authValue = "";

  const escapeHTML = (value) => String(value ?? "").replace(/[&<>'"]/g, (character) => ({"&":"&amp;","<":"&lt;",">":"&gt;","'":"&#39;",'"':"&quot;"}[character]));
  const formatTime = (value) => value ? new Date(value).toLocaleString() : "Not scheduled";

  function showNotice(message) {
    notice.textContent = message;
    notice.hidden = !message;
  }

  function renderMissions(items) {
    if (!items.length) {
      missions.innerHTML = '<p class="empty">No missions are recorded locally yet.</p>';
      return;
    }
    missions.innerHTML = items.map((mission) => `<article class="mission-card">
      <div class="status-label">${escapeHTML(mission.status || "UNKNOWN")}</div>
      <h2>${escapeHTML(mission.name || "Unnamed mission")}</h2>
      <p>${escapeHTML(mission.launch_site_name || "Launch site unavailable")}</p>
      <p>${escapeHTML(formatTime(mission.planned_start))}</p>
      <button data-mission="${escapeHTML(mission.id)}">Open mission</button>
    </article>`).join("");
    missions.querySelectorAll("[data-mission]").forEach((button) => button.addEventListener("click", () => loadDetail(button.dataset.mission)));
  }

  async function fetchJSON(path, options) {
    const request = options || {};
    request.headers = Object.assign({}, request.headers || {});
    if (authValue) request.headers.Authorization = `Bearer ${authValue}`;
    const response = await fetch(path, request);
    const value = await response.json();
    if (!response.ok) throw new Error(value.error || response.statusText);
    return value;
  }

  async function load() {
    try {
      const status = await fetchJSON("/api/v1/status");
      syncEnabled = Boolean(status.sync_enabled);
      connectionDot.className = "dot ready";
      connectionText.textContent = "LOCAL API ONLINE · OFFLINE-FIRST";
      const payload = await fetchJSON("/api/v1/missions");
      localStorage.setItem(cacheKey, JSON.stringify(payload.missions || []));
      renderMissions(payload.missions || []);
      showNotice("");
    } catch (error) {
      connectionDot.className = "dot";
      connectionText.textContent = "OFFLINE · SHOWING CACHED ARCHIVE";
      try {
        renderMissions(JSON.parse(localStorage.getItem(cacheKey) || "[]"));
        showNotice("The local API is unavailable. Cached mission records are read-only until it returns.");
      } catch (_) {
        renderMissions([]);
        showNotice("The local API is unavailable and no cached mission records exist.");
      }
    }
  }

  async function loadDetail(id) {
    try {
      const mission = await fetchJSON(`/api/v1/missions/${encodeURIComponent(id)}`);
      detail.innerHTML = `<h2>${escapeHTML(mission.name)}</h2><dl>
        <dt>Status</dt><dd>${escapeHTML(mission.status)}</dd>
        <dt>Launch site</dt><dd>${escapeHTML(mission.launch_site_name || "Unavailable")}</dd>
        <dt>Time zone</dt><dd>${escapeHTML(mission.timezone || "Unavailable")}</dd>
        <dt>Planned start</dt><dd>${escapeHTML(formatTime(mission.planned_start))}</dd>
        <dt>Planned end</dt><dd>${escapeHTML(formatTime(mission.planned_end))}</dd>
        <dt>Recorded</dt><dd>${escapeHTML(formatTime(mission.created_at))}</dd>
      </dl>`;
      detail.hidden = false;
      detail.scrollIntoView({behavior: "smooth", block: "start"});
    } catch (error) {
      showNotice(`Mission detail unavailable: ${error.message}`);
    }
  }

  async function exportSync() {
    if (!syncEnabled) { showNotice("Sync is disabled in the NightOps configuration."); return; }
    try {
      const bundle = await fetchJSON("/api/v1/sync");
      const blob = new Blob([JSON.stringify(bundle, null, 2)], {type: "application/json"});
      const link = document.createElement("a");
      link.href = URL.createObjectURL(blob);
      link.download = "nightops-sync.json";
      link.click();
      URL.revokeObjectURL(link.href);
      showNotice("");
    } catch (error) { showNotice(`Sync export unavailable: ${error.message}`); }
  }

  async function importSync(event) {
    const file = event.target.files[0];
    event.target.value = "";
    if (!file) return;
    if (!syncEnabled) { showNotice("Sync is disabled in the NightOps configuration."); return; }
    try {
      const bundle = JSON.parse(await file.text());
      const report = await fetchJSON("/api/v1/sync", {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(bundle)});
      showNotice(`Sync merged: ${report.added || 0} added · ${report.updated || 0} updated · ${report.skipped || 0} skipped.`);
      await load();
    } catch (error) { showNotice(`Sync import unavailable: ${error.message}`); }
  }

  document.querySelector("#refresh").addEventListener("click", load);
  document.querySelector("#auth-form").addEventListener("submit", (event) => {
    event.preventDefault();
    authValue = authValueInput.value;
    load();
  });
  document.querySelector("#export-sync").addEventListener("click", exportSync);
  document.querySelector("#import-sync").addEventListener("change", importSync);
  if ("serviceWorker" in navigator) navigator.serviceWorker.register("sw.js").catch(() => {});
  load();
})();
