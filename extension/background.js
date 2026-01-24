// Tachyon Background Script

const DEFAULT_SERVER_URL = "http://localhost:45000";
const DEFAULT_TOKEN = "tachyon-dev-token";

// Initialize Context Menu
chrome.runtime.onInstalled.addListener(() => {
    chrome.contextMenus.create({
        id: "download-with-tachyon",
        title: "Download with Tachyon",
        contexts: ["link", "selection", "page", "video", "audio"]
    });
});

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
    if (info.menuItemId === "download-with-tachyon") {
        let url = info.linkUrl || info.srcUrl || info.selectionText || info.pageUrl;
        if (url) {
            await sendToTachyon(url);
        }
    }
});

async function sendToTachyon(url) {
    // Get Config
    const config = await chrome.storage.local.get(["mode", "remoteUrl", "apiToken", "autoGrab"]);

    // Auto-Grab Check (If triggered automatically by sniffer) -- wait, this fn is called by Context Menu too.
    // We assume context menu click = FORCE download.
    // If called by sniffer, it handles logic.

    let serverUrl = "http://localhost:45000";
    let token = "";

    if (config.mode === 'remote') {
        serverUrl = config.remoteUrl || serverUrl;
        token = config.apiToken || "";
    }

    try {
        const cookies = await chrome.cookies.getAll({ url: url });
        const cookieString = cookies.map(c => `${c.name}=${c.value}`).join("; ");
        const userAgent = navigator.userAgent;

        const payload = {
            url: url,
            cookies: cookieString,
            userAgent: userAgent,
            referer: url
        };

        const response = await fetch(`${serverUrl}/api/v1/download`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "X-Tachyon-Token": token
            },
            body: JSON.stringify(payload)
        });

        if (response.ok) {
            chrome.action.setBadgeText({ text: "OK" });
            chrome.action.setBadgeBackgroundColor({ color: "#22c55e" });
            setTimeout(() => chrome.action.setBadgeText({ text: "" }), 2000);
        } else {
            console.error("Tachyon Server Error:", response.status);
            chrome.action.setBadgeText({ text: "ERR" });
            chrome.action.setBadgeBackgroundColor({ color: "#ef4444" });
        }
    } catch (err) {
        console.error("Failed to connect to Tachyon:", err);
        chrome.action.setBadgeText({ text: "OFF" });
        chrome.action.setBadgeBackgroundColor({ color: "#555" });
    }
}

// Media Sniffer
chrome.webRequest.onHeadersReceived.addListener(
    async (details) => {
        if (details.tabId === -1) return;

        // Check if Auto-Grab enabled
        const config = await chrome.storage.local.get("autoGrab");
        if (config.autoGrab === false) return; // Default true if undefined, but explicit false disables

        const contentTypeHeader = details.responseHeaders.find(h => h.name.toLowerCase() === 'content-type');
        if (!contentTypeHeader) return;

        const type = contentTypeHeader.value.toLowerCase();
        const isVideo = type.includes('video/') || type.includes('application/x-mpegurl') || type.includes('application/vnd.apple.mpegurl');

        // Size check (optional, e.g. > 1MB)
        const contentLengthHeader = details.responseHeaders.find(h => h.name.toLowerCase() === 'content-length');
        const size = contentLengthHeader ? parseInt(contentLengthHeader.value) : 0;

        if (isVideo && size > 1024 * 1024) { // > 1MB
            // Highlight Icon to indicate capture available
            chrome.action.setBadgeText({ text: "VID", tabId: details.tabId });
            chrome.action.setBadgeBackgroundColor({ color: "#FF6600", tabId: details.tabId });

            // Store the stream URL for popup or automatic capture?
            // For now, let's just log it or notify context menu logic.
            console.log("Tachyon Sniffed:", details.url);
        }
    },
    { urls: ["<all_urls>"] },
    ["responseHeaders"]
);
