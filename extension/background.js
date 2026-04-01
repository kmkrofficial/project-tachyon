// TDM Background Script — Service Worker
// Detects TDM availability, intercepts downloads, sniffs video streams.

const DEFAULT_SERVER_URL = "http://localhost:4444";
const HEALTH_CHECK_ALARM = "tdm-health-check";
const HEALTH_CHECK_INTERVAL_MIN = 1;
const HEALTH_TIMEOUT_MS = 3000;

// File extensions that TDM should intercept
const INTERCEPT_EXTENSIONS = new Set([
    "zip", "rar", "7z", "tar", "gz", "bz2", "xz", "iso",
    "exe", "msi", "dmg", "deb", "rpm", "appimage",
    "mp4", "mkv", "avi", "mov", "wmv", "flv", "webm",
    "mp3", "flac", "wav", "aac", "ogg", "wma",
    "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx",
    "apk", "ipa", "bin", "img", "torrent"
]);

const MIN_INTERCEPT_SIZE = 512;

// ─── Media Content Types ──────────────────────────────────────────────────────

// Minimum media size to capture (100KB - skip thumbnails/previews)
const MIN_MEDIA_SIZE = 100 * 1024;

// ─── State ────────────────────────────────────────────────────────────────────

let tdmConnected = false;
let interceptEnabled = true;

// Settings defaults
const DEFAULT_SETTINGS = {
    collisionAction: "ask",         // "ask", "always_download", "rename"
    notifyDownloadStarted: true,
    notifyDownloadComplete: true,
    notifyDownloadFailed: true,
    notifyCollisionDetected: true,
    notifyMediaCaptured: false,
    notifyConnectionLost: true
};

// Per-tab captured media: Map<tabId, MediaItem[]>
const capturedMedia = new Map();

// ─── MediaItem Structure ──────────────────────────────────────────────────────
// { url, type, contentType, size, filename, quality, resolution, tabUrl, requestHeaders, timestamp }

// ─── Lifecycle ────────────────────────────────────────────────────────────────

chrome.runtime.onInstalled.addListener(async () => {
    chrome.contextMenus.create({
        id: "download-with-tachyon",
        title: "Download with TDM",
        contexts: ["link", "selection", "page", "video", "audio"]
    });
    chrome.contextMenus.create({
        id: "download-video-tachyon",
        title: "Download Video with TDM",
        contexts: ["video", "audio"]
    });

    const config = await chrome.storage.local.get(["interceptEnabled"]);
    if (config.interceptEnabled === undefined) {
        await chrome.storage.local.set({ interceptEnabled: true });
    }

    chrome.alarms.create(HEALTH_CHECK_ALARM, { periodInMinutes: HEALTH_CHECK_INTERVAL_MIN });
    await checkTDMHealth();
});

chrome.runtime.onStartup.addListener(async () => {
    chrome.alarms.create(HEALTH_CHECK_ALARM, { periodInMinutes: HEALTH_CHECK_INTERVAL_MIN });
    await loadConfig();
    await checkTDMHealth();
});

chrome.alarms.onAlarm.addListener(async (alarm) => {
    if (alarm.name === HEALTH_CHECK_ALARM) {
        await checkTDMHealth();
    }
});

// Clean up when tabs close
chrome.tabs.onRemoved.addListener((tabId) => {
    capturedMedia.delete(tabId);
});

// ─── Health Check ─────────────────────────────────────────────────────────────

async function checkTDMHealth() {
    const config = await chrome.storage.local.get(["mode", "remoteUrl"]);
    const serverUrl = config.mode === "remote"
        ? (config.remoteUrl || DEFAULT_SERVER_URL)
        : DEFAULT_SERVER_URL;

    try {
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), HEALTH_TIMEOUT_MS);

        const response = await fetch(`${serverUrl}/v1/health`, {
            method: "GET",
            signal: controller.signal
        });
        clearTimeout(timeoutId);

        if (response.ok) {
            const data = await response.json();
            if (data.status === "ok") {
                setConnected(true);
                return;
            }
        }
        setConnected(false);
    } catch {
        setConnected(false);
    }
}

function setConnected(connected) {
    const changed = tdmConnected !== connected;
    tdmConnected = connected;

    if (connected) {
        chrome.action.setIcon({ path: {
            "16": "icons/icon16.png",
            "48": "icons/icon48.png",
            "128": "icons/icon128.png"
        }});
        chrome.action.setBadgeText({ text: "" });
        chrome.action.setTitle({ title: "TDM - Connected" });
    } else {
        chrome.action.setBadgeText({ text: "OFF" });
        chrome.action.setBadgeBackgroundColor({ color: "#555" });
        chrome.action.setTitle({ title: "TDM - Not Connected" });
    }

    if (changed) {
        chrome.runtime.sendMessage({ type: "TDM_STATUS", connected }).catch(() => {});
        if (!connected) notifyConnectionLost();
    }
}

// ─── Config ───────────────────────────────────────────────────────────────────

async function loadConfig() {
    const config = await chrome.storage.local.get(["interceptEnabled"]);
    interceptEnabled = config.interceptEnabled !== false;
}

chrome.storage.onChanged.addListener((changes) => {
    if (changes.interceptEnabled) {
        interceptEnabled = changes.interceptEnabled.newValue !== false;
    }
});

// ─── Settings Loader ──────────────────────────────────────────────────────────

async function getSettings() {
    const stored = await chrome.storage.local.get(Object.keys(DEFAULT_SETTINGS));
    return { ...DEFAULT_SETTINGS, ...stored };
}

// ─── OS Notifications ─────────────────────────────────────────────────────────

function notifyDownloadStarted(filename) {
    getSettings().then(settings => {
        if (!settings.notifyDownloadStarted) return;
        chrome.notifications.create("dl-start-" + Date.now(), {
            type: "basic",
            iconUrl: "icons/icon128.png",
            title: "TDM - Download Started",
            message: filename || "Download initiated",
            priority: 1
        });
    });
}

function notifyDownloadComplete(filename) {
    getSettings().then(settings => {
        if (!settings.notifyDownloadComplete) return;
        chrome.notifications.create("dl-done-" + Date.now(), {
            type: "basic",
            iconUrl: "icons/icon128.png",
            title: "TDM - Download Complete",
            message: (filename || "File") + " finished downloading",
            priority: 2
        });
    });
}

function notifyDownloadFailed(filename, reason) {
    getSettings().then(settings => {
        if (!settings.notifyDownloadFailed) return;
        chrome.notifications.create("dl-fail-" + Date.now(), {
            type: "basic",
            iconUrl: "icons/icon128.png",
            title: "TDM - Download Failed",
            message: (filename || "File") + " - " + (reason || "Unknown error"),
            priority: 2
        });
    });
}

function notifyMediaCaptured(filename, count) {
    getSettings().then(settings => {
        if (!settings.notifyMediaCaptured) return;
        chrome.notifications.create("media-" + Date.now(), {
            type: "basic",
            iconUrl: "icons/icon128.png",
            title: "TDM - Video Detected",
            message: (filename || "Video") + " (" + count + " total on page)",
            priority: 0
        });
    });
}

function notifyConnectionLost() {
    getSettings().then(settings => {
        if (!settings.notifyConnectionLost) return;
        chrome.notifications.create("conn-lost", {
            type: "basic",
            iconUrl: "icons/icon128.png",
            title: "TDM - Connection Lost",
            message: "TDM is no longer reachable. Downloads will use the browser.",
            priority: 1
        });
    });
}

// ─── Collision Check ──────────────────────────────────────────────────────────

async function checkCollision(url, filename) {
    const config = await chrome.storage.local.get(["mode", "remoteUrl"]);
    const serverUrl = config.mode === "remote"
        ? (config.remoteUrl || DEFAULT_SERVER_URL)
        : DEFAULT_SERVER_URL;

    try {
        const response = await fetch(`${serverUrl}/v1/browser/check`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ url, filename: filename || "" })
        });
        if (response.ok) return await response.json();
    } catch { /* server unreachable */ }
    return { status: "clear" };
}

// Prompt the user via a notification with buttons about a collision
async function promptCollision(collisionInfo, url, filename, extraHeaders) {
    const settings = await getSettings();
    if (!settings.notifyCollisionDetected) {
        // Notification disabled — always download silently
        forceSendToTachyon(url, filename, extraHeaders);
        return;
    }

    const notifId = "collision-" + Date.now();
    const lines = [];

    if (collisionInfo.status === "downloading") {
        const pct = Math.round(collisionInfo.progress || 0);
        lines.push(`"${collisionInfo.filename}" is currently downloading (${pct}%).`);
    } else if (collisionInfo.status === "completed") {
        lines.push(`"${collisionInfo.filename}" was already downloaded.`);
    } else {
        lines.push(`"${collisionInfo.filename}" already exists on disk.`);
    }

    // Store collision context so button click handler can act on it
    pendingCollisions.set(notifId, { url, filename, extraHeaders, collisionInfo });

    chrome.notifications.create(notifId, {
        type: "basic",
        iconUrl: "icons/icon128.png",
        title: "TDM - File Already Exists",
        message: lines.join(" ") + "\nClick to open download options.",
        requireInteraction: true,
        priority: 2,
        buttons: [
            { title: "Download Anyway" },
            { title: "Cancel" }
        ]
    });
}

const pendingCollisions = new Map();

chrome.notifications.onButtonClicked.addListener((notifId, btnIdx) => {
    const ctx = pendingCollisions.get(notifId);
    if (!ctx) return;

    pendingCollisions.delete(notifId);
    chrome.notifications.clear(notifId);

    if (btnIdx === 0) {
        // Download anyway — skip collision check
        forceSendToTachyon(ctx.url, ctx.filename, ctx.extraHeaders);
    }
    // btnIdx === 1 → Cancel, do nothing
});

chrome.notifications.onClicked.addListener((notifId) => {
    if (pendingCollisions.has(notifId)) {
        // Open the TDM popup/settings for manual decision
        chrome.notifications.clear(notifId);
        pendingCollisions.delete(notifId);
    }
});

// ─── Download Interception ────────────────────────────────────────────────────

chrome.downloads.onCreated.addListener(async (downloadItem) => {
    if (!tdmConnected || !interceptEnabled) return;

    const url = downloadItem.finalUrl || downloadItem.url;
    if (!url || url.startsWith("blob:") || url.startsWith("data:")) return;

    const shouldIntercept = shouldInterceptUrl(url, downloadItem.fileSize);
    if (!shouldIntercept) return;

    try {
        await chrome.downloads.cancel(downloadItem.id);
        chrome.downloads.erase({ id: downloadItem.id });
    } catch {
        return;
    }

    await sendToTachyon(url, downloadItem.filename);
});

function shouldInterceptUrl(url, fileSize) {
    try {
        const pathname = new URL(url).pathname;
        const ext = pathname.split(".").pop().toLowerCase().split("?")[0];
        if (INTERCEPT_EXTENSIONS.has(ext)) return true;
    } catch { /* skip */ }

    if (fileSize && fileSize > MIN_INTERCEPT_SIZE) return true;
    return false;
}

// ─── Context Menu ─────────────────────────────────────────────────────────────

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
    if (info.menuItemId === "download-with-tachyon") {
        const url = info.linkUrl || info.srcUrl || info.selectionText || info.pageUrl;
        if (url) await sendToTachyon(url);
    }
    if (info.menuItemId === "download-video-tachyon") {
        const url = info.srcUrl || info.linkUrl;
        if (url) await sendToTachyon(url);
    }
});

// ─── Send to TDM ─────────────────────────────────────────────────────────────

async function sendToTachyon(url, filename, extraHeaders) {
    const settings = await getSettings();

    // Collision check based on settings
    if (settings.collisionAction === "ask") {
        const collision = await checkCollision(url, filename);
        if (collision.status !== "clear") {
            promptCollision(collision, url, filename, extraHeaders);
            return;
        }
    }
    // "always_download" and "rename" both proceed directly —
    // the backend handles rename via FindAvailablePathExcluding

    await forceSendToTachyon(url, filename, extraHeaders);
}

// Force-send without collision check (used after user confirms or for always-download)
async function forceSendToTachyon(url, filename, extraHeaders) {
    const config = await chrome.storage.local.get(["mode", "remoteUrl", "apiToken"]);

    let serverUrl = DEFAULT_SERVER_URL;
    let token = "";

    if (config.mode === "remote") {
        serverUrl = config.remoteUrl || serverUrl;
        token = config.apiToken || "";
    }

    try {
        const cookies = await chrome.cookies.getAll({ url });
        const cookieString = cookies.map(c => `${c.name}=${c.value}`).join("; ");

        const payload = {
            url,
            cookies: cookieString,
            userAgent: navigator.userAgent,
            referer: url,
            filename: filename || ""
        };

        // Merge extra headers (e.g. Range, Origin captured from sniffer)
        if (extraHeaders) {
            payload.extra_headers = extraHeaders;
        }

        const response = await fetch(`${serverUrl}/v1/browser/trigger`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "X-Tachyon-Token": token
            },
            body: JSON.stringify(payload)
        });

        if (response.ok) {
            showBadge("OK", "#22c55e", 2000);
            notifyDownloadStarted(filename || extractFilenameFromUrl(url));
        } else {
            console.error("TDM Server Error:", response.status);
            showBadge("ERR", "#ef4444", 3000);
            notifyDownloadFailed(filename || extractFilenameFromUrl(url), "Server returned " + response.status);
        }
    } catch (err) {
        console.error("Failed to connect to TDM:", err);
        setConnected(false);
        showBadge("OFF", "#555", 3000);
        notifyDownloadFailed(filename || extractFilenameFromUrl(url), "Connection failed");
    }
}

function extractFilenameFromUrl(url) {
    try {
        const pathname = new URL(url).pathname;
        const segments = pathname.split("/").filter(Boolean);
        return segments.length > 0 ? decodeURIComponent(segments[segments.length - 1]) : url;
    } catch { return url; }
}

// Send a grabbed stream to TDM for resolution/format extraction
async function sendStreamToTachyon(mediaItem) {
    const config = await chrome.storage.local.get(["mode", "remoteUrl", "apiToken"]);

    let serverUrl = DEFAULT_SERVER_URL;
    let token = "";

    if (config.mode === "remote") {
        serverUrl = config.remoteUrl || serverUrl;
        token = config.apiToken || "";
    }

    try {
        // For YouTube/googlevideo URLs, get cookies from youtube.com domain
        let cookieDomain = mediaItem.url;
        try {
            const urlHost = new URL(mediaItem.url).hostname;
            if (urlHost.endsWith("googlevideo.com")) {
                cookieDomain = "https://www.youtube.com";
            }
        } catch {}
        const cookies = await chrome.cookies.getAll({ url: cookieDomain });
        const cookieString = cookies.map(c => `${c.name}=${c.value}`).join("; ");

        const payload = {
            url: mediaItem.url,
            page_url: mediaItem.tabUrl || "",
            cookies: cookieString,
            user_agent: navigator.userAgent,
            referer: mediaItem.tabUrl || mediaItem.url,
            filename: mediaItem.filename || "",
            content_type: mediaItem.contentType || "",
            size: mediaItem.size || 0,
            quality: mediaItem.quality || "",
            request_headers: mediaItem.requestHeaders || {}
        };

        const response = await fetch(`${serverUrl}/v1/grab/download`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "X-Tachyon-Token": token
            },
            body: JSON.stringify(payload)
        });

        if (response.ok) {
            showBadge("OK", "#22c55e", 2000);
            notifyDownloadStarted(mediaItem.filename || extractFilenameFromUrl(mediaItem.url));
            return true;
        } else {
            console.error("TDM Grab Error:", response.status);
            showBadge("ERR", "#ef4444", 3000);
            notifyDownloadFailed(mediaItem.filename || extractFilenameFromUrl(mediaItem.url), "Server returned " + response.status);
            return false;
        }
    } catch (err) {
        console.error("Failed to send stream to TDM:", err);
        setConnected(false);
        notifyDownloadFailed(mediaItem.filename || extractFilenameFromUrl(mediaItem.url), "Connection failed");
        return false;
    }
}

// Resolve streams from an HLS/DASH manifest URL via TDM backend
async function resolveStreams(manifestUrl, pageUrl) {
    const config = await chrome.storage.local.get(["mode", "remoteUrl", "apiToken"]);

    let serverUrl = DEFAULT_SERVER_URL;
    let token = "";

    if (config.mode === "remote") {
        serverUrl = config.remoteUrl || serverUrl;
        token = config.apiToken || "";
    }

    try {
        const cookies = await chrome.cookies.getAll({ url: manifestUrl });
        const cookieString = cookies.map(c => `${c.name}=${c.value}`).join("; ");

        const response = await fetch(`${serverUrl}/v1/grab/resolve`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "X-Tachyon-Token": token
            },
            body: JSON.stringify({
                url: manifestUrl,
                page_url: pageUrl || "",
                cookies: cookieString,
                user_agent: navigator.userAgent,
                referer: pageUrl || manifestUrl
            })
        });

        if (response.ok) {
            return await response.json();
        }
    } catch (err) {
        console.error("Stream resolve failed:", err);
    }
    return null;
}

function showBadge(text, color, durationMs) {
    chrome.action.setBadgeText({ text });
    chrome.action.setBadgeBackgroundColor({ color });
    setTimeout(() => {
        if (tdmConnected) {
            chrome.action.setBadgeText({ text: "" });
        }
    }, durationMs);
}

// ─── Advanced Media Sniffer ───────────────────────────────────────────────────

chrome.webRequest.onBeforeSendHeaders.addListener(
    (details) => {
        // Cache request headers so we can forward them to TDM when downloading
        if (details.tabId === -1) return;
        const cached = requestHeaderCache.get(details.requestId);
        if (!cached) {
            const headers = {};
            for (const h of details.requestHeaders || []) {
                headers[h.name] = h.value || "";
            }
            requestHeaderCache.set(details.requestId, {
                headers,
                tabId: details.tabId,
                url: details.url
            });
            // Evict old entries (keep last 500)
            if (requestHeaderCache.size > 500) {
                const oldest = requestHeaderCache.keys().next().value;
                requestHeaderCache.delete(oldest);
            }
        }
    },
    { urls: ["<all_urls>"] },
    ["requestHeaders"]
);

const requestHeaderCache = new Map();

chrome.webRequest.onHeadersReceived.addListener(
    async (details) => {
        if (details.tabId === -1) return;

        const config = await chrome.storage.local.get("autoGrab");
        if (config.autoGrab === false) return;

        const respHeaders = {};
        let contentType = "";
        let contentLength = 0;
        let contentDisposition = "";

        for (const h of details.responseHeaders || []) {
            const name = h.name.toLowerCase();
            respHeaders[name] = h.value || "";
            if (name === "content-type") contentType = (h.value || "").toLowerCase();
            if (name === "content-length") contentLength = parseInt(h.value) || 0;
            if (name === "content-disposition") contentDisposition = h.value || "";
        }

        // Early reject — skip non-media content types entirely
        if (isNonMediaContentType(contentType)) return;

        // Classify the response
        const mediaType = classifyMedia(details.url, contentType, contentLength, contentDisposition);
        if (!mediaType) return;

        // Get request headers we cached
        const cachedReq = requestHeaderCache.get(details.requestId);
        requestHeaderCache.delete(details.requestId);

        // Build media item
        // Get tab info (URL + title) for filename and dedup
        let tabUrl = "";
        let tabTitle = "";
        try {
            const tab = await chrome.tabs.get(details.tabId);
            tabUrl = tab.url || "";
            tabTitle = tab.title || "";
        } catch { /* tab may be gone */ }

        // For YouTube videoplayback URLs, use tab title as filename
        let filename = extractFilename(details.url, contentDisposition, contentType);
        if (isYouTubeVideo(details.url) && tabTitle) {
            // Strip " - YouTube" suffix and sanitize for filesystem
            const clean = tabTitle.replace(/\s*[-|]\s*YouTube$/i, "").replace(/[\\/:*?"<>|]/g, "_").trim();
            if (clean) {
                const ext = contentType.includes("webm") ? "webm" : "mp4";
                filename = clean + "." + ext;
            }
        }

        const mediaItem = {
            url: details.url,
            type: mediaType.type,          // "video", "stream"
            contentType,
            size: contentLength,
            filename,
            quality: mediaType.quality || "",
            resolution: mediaType.resolution || "",
            tabUrl,
            requestHeaders: cachedReq?.headers || {},
            timestamp: Date.now()
        };

        // Add to captured list for this tab (YouTube URLs are normalized for dedup)
        addCapturedMedia(details.tabId, mediaItem);

        // Update badge
        const count = (capturedMedia.get(details.tabId) || []).length;
        chrome.action.setBadgeText({ text: String(count), tabId: details.tabId });
        chrome.action.setBadgeBackgroundColor({ color: "#FF6600", tabId: details.tabId });

        // OS notification for captured video
        notifyMediaCaptured(mediaItem.filename, count);

        // Notify content script / popup about new media
        chrome.runtime.sendMessage({
            type: "MEDIA_CAPTURED",
            tabId: details.tabId,
            media: mediaItem,
            totalCount: count
        }).catch(() => {});

        // Also notify the content script in the tab
        chrome.tabs.sendMessage(details.tabId, {
            type: "MEDIA_CAPTURED",
            media: mediaItem,
            totalCount: count
        }).catch(() => {});
    },
    { urls: ["<all_urls>"] },
    ["responseHeaders"]
);

// ─── Media Classification ─────────────────────────────────────────────────────

// Reject content types that are never video
function isNonMediaContentType(ct) {
    if (!ct) return true;  // No content-type = unknown, reject
    return ct.startsWith("image/") ||
           ct.startsWith("text/") ||
           ct.startsWith("font/") ||
           ct.startsWith("application/javascript") ||
           ct.startsWith("application/json") ||
           ct.startsWith("application/wasm") ||
           ct.startsWith("audio/");  // video-only sniffer
}

function classifyMedia(url, contentType, size, contentDisposition) {
    const urlLower = url.toLowerCase();

    // 1. YouTube / Google Video — highest priority
    if (isYouTubeVideo(url)) {
        const quality = guessQualityFromYouTube(url);
        return { type: "video", quality: quality.label || "YouTube", resolution: quality.resolution };
    }

    // 2. HLS manifests
    if (contentType.includes("mpegurl") ||
        contentType.includes("x-mpegurl") ||
        urlLower.includes(".m3u8")) {
        return { type: "stream", quality: "HLS" };
    }

    // 3. DASH manifests
    if (contentType.includes("dash+xml") ||
        urlLower.includes(".mpd")) {
        return { type: "stream", quality: "DASH" };
    }

    // 4. Direct video by content-type — require non-zero size to skip 0KB tracker responses
    if (contentType.startsWith("video/") && size > MIN_MEDIA_SIZE) {
        const quality = guessQualityFromUrl(url);
        return { type: "video", quality: quality.label, resolution: quality.resolution };
    }

    // 5. Large octet-stream that looks like a video stream (strict URL patterns)
    if (contentType === "application/octet-stream" && size > MIN_MEDIA_SIZE && looksLikeVideo(url)) {
        return { type: "video", quality: guessQualityFromUrl(url).label || "unknown" };
    }

    // No file-extension or content-disposition guessing — too noisy
    return null; // Not video — skip
}

function guessQualityFromUrl(url) {
    const urlLower = url.toLowerCase();
    // Common resolution patterns in URLs
    const patterns = [
        { match: /2160|4k|uhd/i, label: "4K", resolution: "3840×2160" },
        { match: /1440|2k|qhd/i, label: "1440p", resolution: "2560×1440" },
        { match: /1080|fhd/i, label: "1080p", resolution: "1920×1080" },
        { match: /720|hd/i, label: "720p", resolution: "1280×720" },
        { match: /480|sd/i, label: "480p", resolution: "854×480" },
        { match: /360/i, label: "360p", resolution: "640×360" },
        { match: /240/i, label: "240p", resolution: "426×240" },
        { match: /144/i, label: "144p", resolution: "256×144" },
    ];

    for (const p of patterns) {
        if (p.match.test(urlLower)) {
            return { label: p.label, resolution: p.resolution };
        }
    }
    return { label: "unknown", resolution: "" };
}

// YouTube-specific quality extraction from itag or URL params
function guessQualityFromYouTube(url) {
    try {
        const u = new URL(url);
        // YouTube itag → quality mapping (common video itags)
        const itagMap = {
            "18": { label: "360p", resolution: "640×360" },
            "22": { label: "720p", resolution: "1280×720" },
            "37": { label: "1080p", resolution: "1920×1080" },
            "137": { label: "1080p", resolution: "1920×1080" },
            "248": { label: "1080p", resolution: "1920×1080" },
            "136": { label: "720p", resolution: "1280×720" },
            "247": { label: "720p", resolution: "1280×720" },
            "135": { label: "480p", resolution: "854×480" },
            "244": { label: "480p", resolution: "854×480" },
            "134": { label: "360p", resolution: "640×360" },
            "243": { label: "360p", resolution: "640×360" },
            "133": { label: "240p", resolution: "426×240" },
            "242": { label: "240p", resolution: "426×240" },
            "160": { label: "144p", resolution: "256×144" },
            "278": { label: "144p", resolution: "256×144" },
            "264": { label: "1440p", resolution: "2560×1440" },
            "271": { label: "1440p", resolution: "2560×1440" },
            "313": { label: "4K", resolution: "3840×2160" },
            "315": { label: "4K", resolution: "3840×2160" },
        };
        const itag = u.searchParams.get("itag");
        if (itag && itagMap[itag]) return itagMap[itag];

        // Fallback: check quality param
        const quality = u.searchParams.get("quality");
        if (quality === "hd1080") return { label: "1080p", resolution: "1920×1080" };
        if (quality === "hd720") return { label: "720p", resolution: "1280×720" };
        if (quality === "large") return { label: "480p", resolution: "854×480" };
        if (quality === "medium") return { label: "360p", resolution: "640×360" };
        if (quality === "small") return { label: "240p", resolution: "426×240" };
    } catch { /* skip */ }
    return guessQualityFromUrl(url);
}

function isYouTubeVideo(url) {
    try {
        const u = new URL(url);
        const host = u.hostname;
        // googlevideo.com hosts the actual video data
        if (host.endsWith("googlevideo.com") && u.pathname.includes("/videoplayback")) {
            // Skip SABR (Server ABR) URLs — these are POST-based chunked streams
            // that cannot be downloaded with a simple GET request
            if (u.searchParams.get("sabr") === "1") return false;
            // Skip audio-only streams via mime param
            const mime = u.searchParams.get("mime");
            if (mime && mime.startsWith("audio/")) return false;
            return true;
        }
    } catch { /* skip */ }
    return false;
}

// Normalize YouTube videoplayback URL for deduplication
// Strips range, rn, rbuf, and sequence params so all chunks of the same video match
function normalizeVideoUrl(url) {
    if (!isYouTubeVideo(url)) return url;
    try {
        const u = new URL(url);
        u.searchParams.delete("range");
        u.searchParams.delete("rn");
        u.searchParams.delete("rbuf");
        u.searchParams.delete("sq");
        u.searchParams.delete("lmt");
        u.searchParams.delete("alr");
        u.searchParams.delete("cpn");
        u.searchParams.delete("clen");
        u.searchParams.delete("dur");
        return u.toString();
    } catch { return url; }
}

function looksLikeVideo(url) {
    // Strict patterns — must clearly indicate video content
    const patterns = [
        /videoplayback/i,
        /googlevideo\.com/i,
        /\.m4s(?:\?|$)/i,
        /\.m4v(?:\?|$)/i,
        /seg-\d+-/i,
        /frag\(\d+\)/i,
        /\/video\/[\w-]+\.(?:mp4|webm|ts)/i,
        /\/hls\/.*\.ts(?:\?|$)/i
    ];
    return patterns.some(p => p.test(url));
}

function getExtFromUrl(url) {
    try {
        const pathname = new URL(url).pathname;
        const lastPart = pathname.split("/").pop();
        const ext = lastPart.split(".").pop().toLowerCase();
        if (ext.length <= 5) return ext;
    } catch { /* skip */ }
    return "";
}

function extractFilename(url, contentDisposition, contentType) {
    // 1. From Content-Disposition
    if (contentDisposition) {
        const fn = extractFilenameFromDisposition(contentDisposition);
        if (fn) return fn;
    }

    // 2. From URL path
    try {
        const pathname = new URL(url).pathname;
        const segments = pathname.split("/").filter(Boolean);
        if (segments.length > 0) {
            const last = decodeURIComponent(segments[segments.length - 1]);
            if (last.includes(".") && last.length < 200) return last;
        }
    } catch { /* skip */ }

    // 3. Generate from content type
    const extMap = {
        "video/mp4": "video.mp4",
        "video/webm": "video.webm",
        "video/x-matroska": "video.mkv",
        "application/x-mpegurl": "stream.m3u8",
        "application/dash+xml": "stream.mpd"
    };
    return extMap[contentType] || "video_" + Date.now();
}

function extractFilenameFromDisposition(header) {
    if (!header) return "";
    // filename*= (RFC 5987)
    const star = header.match(/filename\*\s*=\s*[\w-]*'[\w-]*'([^;\s]+)/i);
    if (star) return decodeURIComponent(star[1]);
    // filename=
    const plain = header.match(/filename\s*=\s*"?([^";\n]+)"?/i);
    if (plain) return plain[1].trim();
    return "";
}

// ─── Captured Media Store ─────────────────────────────────────────────────────

function addCapturedMedia(tabId, mediaItem) {
    if (!capturedMedia.has(tabId)) {
        capturedMedia.set(tabId, []);
    }
    const list = capturedMedia.get(tabId);

    // Normalize URL for dedup (collapses YouTube chunks into one entry)
    const normalizedUrl = normalizeVideoUrl(mediaItem.url);

    // Deduplicate by normalized URL (keep latest, accumulate size)
    const existing = list.findIndex(m => normalizeVideoUrl(m.url) === normalizedUrl);
    if (existing !== -1) {
        // Keep the higher-quality entry or update size
        const prev = list[existing];
        if (mediaItem.size > prev.size) {
            mediaItem.size = mediaItem.size; // keep new size
        } else {
            mediaItem.size = prev.size;
        }
        list[existing] = mediaItem;
    } else {
        list.push(mediaItem);
    }

    // Cap at 50 per tab
    if (list.length > 50) {
        list.shift();
    }
}

// ─── Message Handler ──────────────────────────────────────────────────────────

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.type === "GET_STATUS") {
        sendResponse({ connected: tdmConnected, interceptEnabled });
        return false;
    }

    if (message.type === "CHECK_HEALTH") {
        checkTDMHealth().then(() => {
            sendResponse({ connected: tdmConnected });
        });
        return true;
    }

    if (message.type === "GET_CAPTURED_MEDIA") {
        const tabId = message.tabId;
        const media = capturedMedia.get(tabId) || [];
        sendResponse({ media });
        return false;
    }

    if (message.type === "YOUTUBE_FORMATS") {
        // Content script extracted all YouTube video formats — replace sniffer entries
        const tabId = sender?.tab?.id;
        if (tabId && message.formats?.length > 0) {
            capturedMedia.set(tabId, message.formats);
            const count = message.formats.length;
            chrome.action.setBadgeText({ text: String(count), tabId });
            chrome.action.setBadgeBackgroundColor({ color: "#FF6600", tabId });
        }
        return false;
    }

    if (message.type === "DOWNLOAD_MEDIA") {
        const item = message.mediaItem;
        sendStreamToTachyon(item).then(ok => {
            sendResponse({ success: ok });
        });
        return true;
    }

    if (message.type === "DOWNLOAD_ALL_MEDIA") {
        const items = message.mediaItems || [];
        Promise.all(items.map(item => sendStreamToTachyon(item))).then(results => {
            sendResponse({ success: results.every(Boolean) });
        });
        return true;
    }

    if (message.type === "RESOLVE_STREAMS") {
        resolveStreams(message.url, message.pageUrl).then(data => {
            sendResponse({ streams: data });
        });
        return true;
    }

    if (message.type === "GET_SETTINGS") {
        getSettings().then(settings => sendResponse(settings));
        return true;
    }

    if (message.type === "SAVE_SETTINGS") {
        const keys = Object.keys(DEFAULT_SETTINGS);
        const toSave = {};
        for (const k of keys) {
            if (message.settings[k] !== undefined) toSave[k] = message.settings[k];
        }
        chrome.storage.local.set(toSave, () => sendResponse({ ok: true }));
        return true;
    }

    if (message.type === "CHECK_COLLISION") {
        checkCollision(message.url, message.filename).then(result => {
            sendResponse(result);
        });
        return true;
    }

    return false;
});

// Initial load
loadConfig().then(() => checkTDMHealth());
