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
    const config = await chrome.storage.local.get(["serverUrl", "apiToken"]);
    const serverUrl = config.serverUrl || DEFAULT_SERVER_URL;
    const apiToken = config.apiToken || DEFAULT_TOKEN;

    try {
        // Get Cookies (Best Effort)
        const cookies = await chrome.cookies.getAll({ url: url });
        const cookieString = cookies.map(c => `${c.name}=${c.value}`).join("; ");

        // Determine User Agent (Roughly)
        const userAgent = navigator.userAgent;

        const payload = {
            url: url,
            cookies: cookieString,
            userAgent: userAgent,
            referer: url // Simplification
        };

        const response = await fetch(`${serverUrl}/api/v1/download`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "X-Tachyon-Token": apiToken
            },
            body: JSON.stringify(payload)
        });

        if (response.ok) {
            chrome.action.setBadgeText({ text: "OK" });
            chrome.action.setBadgeBackgroundColor({ color: "#00AA00" });
            setTimeout(() => chrome.action.setBadgeText({ text: "" }), 2000);
        } else {
            console.error("Tachyon Server Error:", response.status);
            chrome.action.setBadgeText({ text: "ERR" });
            chrome.action.setBadgeBackgroundColor({ color: "#AA0000" });
        }
    } catch (err) {
        console.error("Failed to connect to Tachyon:", err);
        chrome.action.setBadgeText({ text: "OFF" });
        chrome.action.setBadgeBackgroundColor({ color: "#555555" });
    }
}

// Media Sniffer
chrome.webRequest.onHeadersReceived.addListener(
    (details) => {
        if (details.tabId === -1) return; // Ignore background requests

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
