document.addEventListener("DOMContentLoaded", init);

let currentTabId = null;

async function init() {
    restoreOptions();
    updateSaveBtnVisibility();

    document.getElementById("saveBtn").addEventListener("click", saveOptions);
    document.getElementById("recheckBtn").addEventListener("click", recheckConnection);
    document.getElementById("downloadAllBtn").addEventListener("click", downloadAllMedia);
    document.getElementById("settingsBtn").addEventListener("click", openSettings);

    // Toggle tabs
    document.querySelectorAll(".tab-btn").forEach(btn => {
        btn.addEventListener("click", () => {
            document.querySelectorAll(".tab-btn").forEach(b => b.classList.remove("active"));
            document.querySelectorAll(".tab-content").forEach(c => c.classList.remove("active"));
            btn.classList.add("active");
            document.getElementById(btn.dataset.tab).classList.add("active");
            updateSaveBtnVisibility();
        });
    });

    // Live toggle — persist immediately
    document.getElementById("interceptEnabled").addEventListener("change", (e) => {
        chrome.storage.local.set({ interceptEnabled: e.target.checked });
    });
    document.getElementById("autoGrab").addEventListener("change", (e) => {
        chrome.storage.local.set({ autoGrab: e.target.checked });
    });

    // Listen for status changes from background
    chrome.runtime.onMessage.addListener((message) => {
        if (message.type === "TDM_STATUS") {
            updateConnectionUI(message.connected);
        }
        if (message.type === "MEDIA_CAPTURED") {
            loadCapturedMedia();
        }
    });

    // Get current status from background
    chrome.runtime.sendMessage({ type: "GET_STATUS" }, (response) => {
        if (response) {
            updateConnectionUI(response.connected);
            document.getElementById("interceptEnabled").checked = response.interceptEnabled !== false;
        }
    });

    // Get current tab and load captured media
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (tab) {
        currentTabId = tab.id;
        loadCapturedMedia();
    }
}

// ─── Connection UI ────────────────────────────────────────────────────────────

function updateConnectionUI(connected) {
    const banner = document.getElementById("connectionBanner");
    const text = document.getElementById("connectionText");
    const dot = document.getElementById("statusIcon");
    const label = document.getElementById("statusLabel");

    if (connected) {
        banner.className = "connection-banner connected";
        text.textContent = "TDM is running";
        dot.className = "status-dot connected";
        label.textContent = "Connected";
    } else {
        banner.className = "connection-banner disconnected";
        text.textContent = "TDM not detected - downloads will use browser";
        dot.className = "status-dot disconnected";
        label.textContent = "Offline";
    }
}

async function recheckConnection() {
    const btn = document.getElementById("recheckBtn");
    btn.textContent = "Checking...";
    btn.disabled = true;

    chrome.runtime.sendMessage({ type: "CHECK_HEALTH" }, (response) => {
        updateConnectionUI(response?.connected ?? false);
        btn.textContent = "Re-check TDM Connection";
        btn.disabled = false;
    });
}

function updateSaveBtnVisibility() {
    const activeTab = document.querySelector(".tab-btn.active")?.dataset.tab;
    document.getElementById("saveBtn").style.display = activeTab === "remote" ? "block" : "none";
}

function openSettings() {
    chrome.tabs.create({ url: chrome.runtime.getURL("settings.html") });
}

// ─── Captured Media ───────────────────────────────────────────────────────────

function loadCapturedMedia() {
    if (!currentTabId) return;

    chrome.runtime.sendMessage({ type: "GET_CAPTURED_MEDIA", tabId: currentTabId }, (response) => {
        const media = response?.media || [];
        renderMediaList(media);
    });
}

function renderMediaList(media) {
    const listEl = document.getElementById("mediaList");
    const countEl = document.getElementById("mediaCount");
    const dlAllBtn = document.getElementById("downloadAllBtn");

    countEl.textContent = media.length;
    dlAllBtn.disabled = media.length === 0;

    if (media.length === 0) {
        listEl.innerHTML = '<div class="media-empty">No videos detected on this page</div>';
        return;
    }

    listEl.innerHTML = "";
    for (const item of media) {
        const row = document.createElement("div");
        row.className = "media-item";

        const icon = item.type === "stream" ? "📡" : "🎬";
        const iconClass = item.type === "stream" ? "stream" : "video";
        const size = item.size ? formatSize(item.size) : "";
        const quality = item.quality || item.resolution || "";

        row.innerHTML = `
            <div class="media-icon ${iconClass}">${icon}</div>
            <div class="media-info">
                <div class="media-name" title="${escapeAttr(item.filename)}">${escapeHtml(item.filename)}</div>
                <div class="media-meta">
                    ${quality ? `<span class="quality">${escapeHtml(quality)}</span>` : ""}
                    ${size ? `<span>${size}</span>` : ""}
                    <span>${item.type}</span>
                </div>
            </div>
            <button class="media-dl-btn" title="Download with TDM">⬇</button>
        `;

        row.querySelector(".media-dl-btn").addEventListener("click", () => {
            chrome.runtime.sendMessage({ type: "DOWNLOAD_MEDIA", mediaItem: item });
        });

        listEl.appendChild(row);
    }
}

function downloadAllMedia() {
    if (!currentTabId) return;

    chrome.runtime.sendMessage({ type: "GET_CAPTURED_MEDIA", tabId: currentTabId }, (response) => {
        const media = response?.media || [];
        if (media.length > 0) {
            chrome.runtime.sendMessage({ type: "DOWNLOAD_ALL_MEDIA", mediaItems: media });
        }
    });
}

// ─── Settings ─────────────────────────────────────────────────────────────────

function saveOptions() {
    const mode = document.querySelector(".tab-btn.active").dataset.tab;
    const remoteUrl = document.getElementById("remoteUrl").value;
    const apiToken = document.getElementById("apiToken").value;
    const autoGrab = document.getElementById("autoGrab").checked;
    const interceptEnabled = document.getElementById("interceptEnabled").checked;

    const config = {
        mode,
        remoteUrl: remoteUrl || "http://localhost:4444",
        apiToken,
        autoGrab,
        interceptEnabled
    };

    chrome.storage.local.set(config, () => {
        document.getElementById("statusMsg").textContent = "Settings Saved";
        setTimeout(() => document.getElementById("statusMsg").textContent = "", 2000);

        chrome.runtime.sendMessage({ type: "CHECK_HEALTH" }, (response) => {
            updateConnectionUI(response?.connected ?? false);
        });
    });
}

function restoreOptions() {
    chrome.storage.local.get(
        ["mode", "remoteUrl", "apiToken", "autoGrab", "interceptEnabled"],
        (items) => {
            if (items.mode === "remote") {
                document.querySelector('[data-tab="remote"]').click();
            } else {
                document.querySelector('[data-tab="local"]').click();
            }

            if (items.remoteUrl) document.getElementById("remoteUrl").value = items.remoteUrl;
            if (items.apiToken) document.getElementById("apiToken").value = items.apiToken;
            document.getElementById("autoGrab").checked = items.autoGrab !== false;
            document.getElementById("interceptEnabled").checked = items.interceptEnabled !== false;
            updateSaveBtnVisibility();
        }
    );
}

// ─── Util ─────────────────────────────────────────────────────────────────────

function formatSize(bytes) {
    if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + " GB";
    if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + " MB";
    if (bytes >= 1024) return (bytes / 1024).toFixed(0) + " KB";
    return bytes + " B";
}

function escapeHtml(str) {
    const div = document.createElement("div");
    div.textContent = str || "";
    return div.innerHTML;
}

function escapeAttr(str) {
    return (str || "").replace(/"/g, "&quot;").replace(/</g, "&lt;");
}
