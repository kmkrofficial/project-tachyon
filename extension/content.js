// TDM Content Script — Video Grab Panel
// Injects a floating download button on pages where media is detected.
// Communicates with background.js via chrome.runtime messaging.

(function () {
    "use strict";

    // Prevent double-injection
    if (window.__tdmContentLoaded) return;
    window.__tdmContentLoaded = true;

    let panel = null;
    let mediaList = [];
    let panelExpanded = false;

    // ─── Floating Button ──────────────────────────────────────────────────────

    function createPanel() {
        if (panel) return;

        panel = document.createElement("div");
        panel.id = "tdm-grab-panel";
        panel.innerHTML = `
            <div id="tdm-grab-btn" title="TDM - Download Media">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
                    <polyline points="7 10 12 15 17 10"/>
                    <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
                <span id="tdm-grab-count">0</span>
            </div>
            <div id="tdm-grab-dropdown" class="tdm-hidden">
                <div id="tdm-grab-header">
                    <span>Captured Videos</span>
                    <button id="tdm-grab-download-all" title="Download All">⬇ All</button>
                </div>
                <div id="tdm-grab-list"></div>
            </div>
        `;

        const style = document.createElement("style");
        style.textContent = `
            #tdm-grab-panel {
                position: fixed;
                top: 80px;
                right: 20px;
                z-index: 2147483647;
                font-family: 'Segoe UI', system-ui, sans-serif;
                font-size: 13px;
                color: #eee;
            }

            #tdm-grab-btn {
                width: 48px;
                height: 48px;
                background: linear-gradient(135deg, #3b82f6, #1d4ed8);
                border-radius: 50%;
                display: flex;
                align-items: center;
                justify-content: center;
                cursor: pointer;
                box-shadow: 0 4px 15px rgba(59, 130, 246, 0.4);
                transition: all 0.2s;
                position: relative;
                margin-left: auto;
            }

            #tdm-grab-btn:hover {
                transform: scale(1.1);
                box-shadow: 0 6px 20px rgba(59, 130, 246, 0.6);
            }

            #tdm-grab-count {
                position: absolute;
                top: -4px;
                right: -4px;
                background: #ef4444;
                color: white;
                font-size: 11px;
                font-weight: 700;
                width: 20px;
                height: 20px;
                border-radius: 50%;
                display: flex;
                align-items: center;
                justify-content: center;
                box-shadow: 0 2px 4px rgba(0,0,0,0.3);
            }

            #tdm-grab-dropdown {
                margin-top: 8px;
                background: #1a1a1a;
                border: 1px solid #333;
                border-radius: 10px;
                width: 340px;
                max-height: 420px;
                overflow: hidden;
                box-shadow: 0 8px 30px rgba(0,0,0,0.5);
                display: flex;
                flex-direction: column;
            }

            #tdm-grab-dropdown.tdm-hidden {
                display: none;
            }

            #tdm-grab-header {
                padding: 12px 16px;
                display: flex;
                align-items: center;
                justify-content: space-between;
                border-bottom: 1px solid #333;
                font-weight: 600;
                font-size: 14px;
                background: #222;
                border-radius: 10px 10px 0 0;
            }

            #tdm-grab-download-all {
                background: #3b82f6;
                color: white;
                border: none;
                padding: 6px 14px;
                border-radius: 6px;
                cursor: pointer;
                font-size: 12px;
                font-weight: 600;
                transition: background 0.2s;
            }

            #tdm-grab-download-all:hover {
                background: #2563eb;
            }

            #tdm-grab-list {
                overflow-y: auto;
                max-height: 360px;
                padding: 8px 0;
            }

            .tdm-media-item {
                display: flex;
                align-items: center;
                padding: 10px 16px;
                gap: 10px;
                cursor: pointer;
                transition: background 0.15s;
                border-bottom: 1px solid #222;
            }

            .tdm-media-item:last-child {
                border-bottom: none;
            }

            .tdm-media-item:hover {
                background: #262626;
            }

            .tdm-media-icon {
                width: 32px;
                height: 32px;
                border-radius: 6px;
                display: flex;
                align-items: center;
                justify-content: center;
                font-size: 16px;
                flex-shrink: 0;
            }

            .tdm-media-icon.video { background: rgba(239, 68, 68, 0.15); color: #ef4444; }
            .tdm-media-icon.stream { background: rgba(34, 197, 94, 0.15); color: #22c55e; }

            .tdm-media-info {
                flex: 1;
                min-width: 0;
            }

            .tdm-media-name {
                font-size: 12px;
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
                color: #eee;
            }

            .tdm-media-meta {
                font-size: 11px;
                color: #888;
                display: flex;
                gap: 8px;
                margin-top: 2px;
            }

            .tdm-media-meta .quality {
                color: #3b82f6;
                font-weight: 600;
            }

            .tdm-media-dl {
                width: 28px;
                height: 28px;
                background: rgba(59, 130, 246, 0.15);
                color: #3b82f6;
                border: none;
                border-radius: 6px;
                cursor: pointer;
                display: flex;
                align-items: center;
                justify-content: center;
                font-size: 14px;
                flex-shrink: 0;
                transition: all 0.15s;
            }

            .tdm-media-dl:hover {
                background: #3b82f6;
                color: white;
            }

            #tdm-grab-list::-webkit-scrollbar {
                width: 6px;
            }

            #tdm-grab-list::-webkit-scrollbar-track {
                background: transparent;
            }

            #tdm-grab-list::-webkit-scrollbar-thumb {
                background: #444;
                border-radius: 3px;
            }
        `;

        document.documentElement.appendChild(style);
        document.documentElement.appendChild(panel);

        // Toggle dropdown on button click
        document.getElementById("tdm-grab-btn").addEventListener("click", (e) => {
            e.stopPropagation();
            panelExpanded = !panelExpanded;
            document.getElementById("tdm-grab-dropdown").classList.toggle("tdm-hidden", !panelExpanded);
        });

        // Download All
        document.getElementById("tdm-grab-download-all").addEventListener("click", (e) => {
            e.stopPropagation();
            if (mediaList.length === 0) return;
            chrome.runtime.sendMessage({
                type: "DOWNLOAD_ALL_MEDIA",
                mediaItems: mediaList
            });
        });

        // Close dropdown when clicking outside
        document.addEventListener("click", (e) => {
            if (panel && !panel.contains(e.target)) {
                panelExpanded = false;
                document.getElementById("tdm-grab-dropdown")?.classList.add("tdm-hidden");
            }
        });

        // Make panel draggable
        makeDraggable(panel, document.getElementById("tdm-grab-btn"));
    }

    function makeDraggable(element, handle) {
        let isDragging = false;
        let startX, startY, origX, origY;

        handle.addEventListener("mousedown", (e) => {
            isDragging = true;
            startX = e.clientX;
            startY = e.clientY;
            const rect = element.getBoundingClientRect();
            origX = rect.left;
            origY = rect.top;
            e.preventDefault();
        });

        document.addEventListener("mousemove", (e) => {
            if (!isDragging) return;
            const dx = e.clientX - startX;
            const dy = e.clientY - startY;
            // Only start drag after 5px threshold
            if (Math.abs(dx) < 5 && Math.abs(dy) < 5) return;
            element.style.right = "auto";
            element.style.left = (origX + dx) + "px";
            element.style.top = (origY + dy) + "px";
        });

        document.addEventListener("mouseup", () => {
            isDragging = false;
        });
    }

    // ─── Rendering ────────────────────────────────────────────────────────────

    function updatePanel() {
        if (!panel) createPanel();

        const countEl = document.getElementById("tdm-grab-count");
        const listEl = document.getElementById("tdm-grab-list");

        countEl.textContent = mediaList.length;
        countEl.style.display = mediaList.length > 0 ? "flex" : "none";

        if (mediaList.length === 0) {
            panel.style.display = "none";
            return;
        }

        panel.style.display = "block";

        listEl.innerHTML = "";
        for (const item of mediaList) {
            const row = document.createElement("div");
            row.className = "tdm-media-item";

            const icon = item.type === "stream" ? "📡" : "🎬";
            const iconClass = item.type === "stream" ? "stream" : "video";
            const size = item.size ? formatSize(item.size) : "";
            const quality = item.quality || item.resolution || "";

            row.innerHTML = `
                <div class="tdm-media-icon ${iconClass}">${icon}</div>
                <div class="tdm-media-info">
                    <div class="tdm-media-name" title="${escapeHtml(item.filename)}">${escapeHtml(item.filename)}</div>
                    <div class="tdm-media-meta">
                        ${quality ? `<span class="quality">${escapeHtml(quality)}</span>` : ""}
                        ${size ? `<span>${size}</span>` : ""}
                        <span>${item.type}</span>
                    </div>
                </div>
                <button class="tdm-media-dl" title="Download with TDM">⬇</button>
            `;

            // Download single item
            row.querySelector(".tdm-media-dl").addEventListener("click", (e) => {
                e.stopPropagation();
                chrome.runtime.sendMessage({
                    type: "DOWNLOAD_MEDIA",
                    mediaItem: item
                });
            });

            // If it's a stream manifest, offer to resolve quality options
            if (item.type === "stream") {
                row.addEventListener("click", () => resolveAndExpand(item, row));
            }

            listEl.appendChild(row);
        }
    }

    async function resolveAndExpand(streamItem, rowEl) {
        // Ask background to resolve manifest into individual streams
        chrome.runtime.sendMessage(
            { type: "RESOLVE_STREAMS", url: streamItem.url, pageUrl: streamItem.tabUrl },
            (response) => {
                if (!response?.streams?.variants?.length) return;

                const variants = response.streams.variants;
                // Add resolved variants as separate items (deduplicated)
                for (const v of variants) {
                    const exists = mediaList.some(m => m.url === v.url);
                    if (!exists) {
                        mediaList.push({
                            url: v.url,
                            type: "video",
                            contentType: v.content_type || streamItem.contentType,
                            size: v.size || 0,
                            filename: v.filename || streamItem.filename,
                            quality: v.quality || "",
                            resolution: v.resolution || "",
                            tabUrl: streamItem.tabUrl,
                            requestHeaders: streamItem.requestHeaders,
                            timestamp: Date.now()
                        });
                    }
                }
                updatePanel();
            }
        );
    }

    // ─── Helpers ──────────────────────────────────────────────────────────────

    function formatSize(bytes) {
        if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + " GB";
        if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + " MB";
        if (bytes >= 1024) return (bytes / 1024).toFixed(0) + " KB";
        return bytes + " B";
    }

    function escapeHtml(str) {
        const div = document.createElement("div");
        div.textContent = str;
        return div.innerHTML;
    }

    // ─── Listen for captured media from background ────────────────────────────

    chrome.runtime.onMessage.addListener((message) => {
        if (message.type === "MEDIA_CAPTURED" && message.media) {
            const existing = mediaList.findIndex(m => m.url === message.media.url);
            if (existing !== -1) {
                mediaList[existing] = message.media;
            } else {
                mediaList.push(message.media);
            }
            updatePanel();
        }
    });

    // ─── Also scan existing <video> elements on page load ─────────

    function scanDOMMedia() {
        // Skip YouTube pages — handled by dedicated YouTube extractor below
        if (isYouTubePage()) return;

        const elements = document.querySelectorAll("video[src], video source[src]");
        for (const el of elements) {
            const src = el.src || el.getAttribute("src");
            if (!src || src.startsWith("blob:") || src.startsWith("data:")) continue;

            const existing = mediaList.some(m => m.url === src);
            if (existing) continue;

            mediaList.push({
                url: src,
                type: "video",
                contentType: "",
                size: 0,
                filename: src.split("/").pop().split("?")[0] || "video",
                quality: "",
                resolution: "",
                tabUrl: location.href,
                requestHeaders: {},
                timestamp: Date.now()
            });
        }

        if (mediaList.length > 0) {
            updatePanel();
        }
    }

    // Scan after DOM is ready
    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", scanDOMMedia);
    } else {
        scanDOMMedia();
    }

    // Watch for dynamically added media elements
    const observer = new MutationObserver((mutations) => {
        for (const mutation of mutations) {
            for (const node of mutation.addedNodes) {
                if (node.nodeType !== 1) continue;
                if (node.tagName === "VIDEO" ||
                    node.querySelector?.("video")) {
                    scanDOMMedia();
                    return;
                }
            }
        }
    });

    observer.observe(document.documentElement, {
        childList: true,
        subtree: true
    });

    // ─── YouTube Multi-Resolution Extraction ──────────────────────────────────

    function isYouTubePage() {
        return location.hostname === "www.youtube.com" &&
               location.pathname === "/watch" &&
               new URLSearchParams(location.search).has("v");
    }

    function extractYouTubeFormats() {
        if (!isYouTubePage()) return;

        // Inject script into page context to access YouTube's internal player API
        const script = document.createElement("script");
        script.textContent = `(function(){
            try {
                // Strategy 1: ytInitialPlayerResponse (works on fresh page load)
                var pr = window.ytInitialPlayerResponse;

                // Strategy 2: Player element API (works after SPA navigation)
                if ((!pr || !pr.streamingData) && document.querySelector("ytd-player")) {
                    try {
                        var player = document.querySelector("ytd-player");
                        if (player.player_) {
                            pr = player.player_.getPlayerResponse();
                        }
                    } catch(e) {}
                }

                // Strategy 3: ytplayer.config (older approach)
                if ((!pr || !pr.streamingData) && window.ytplayer && window.ytplayer.config) {
                    try {
                        var args = window.ytplayer.config.args;
                        if (args && args.raw_player_response) {
                            pr = args.raw_player_response;
                        }
                    } catch(e) {}
                }

                if (!pr || !pr.streamingData) {
                    window.postMessage({ type: "__TDM_YT_FORMATS__", data: { error: "no_streaming_data", title: "", formats: [], adaptiveFormats: [] } }, "*");
                    return;
                }

                var sd = pr.streamingData;
                var title = "";
                try { title = pr.videoDetails.title || ""; } catch(e) {}

                var out = { title: title, formats: [], adaptiveFormats: [] };

                function extractFormat(f) {
                    // Direct URL available
                    if (f.url) {
                        return {
                            itag: f.itag, url: f.url, mimeType: f.mimeType || "",
                            bitrate: f.bitrate || 0, width: f.width || 0, height: f.height || 0,
                            contentLength: f.contentLength || "0", qualityLabel: f.qualityLabel || "",
                            fps: f.fps || 0
                        };
                    }
                    return null; // signatureCipher URLs not supported
                }

                (sd.formats || []).forEach(function(f) {
                    var extracted = extractFormat(f);
                    if (extracted) out.formats.push(extracted);
                });

                (sd.adaptiveFormats || []).forEach(function(f) {
                    var extracted = extractFormat(f);
                    if (extracted) out.adaptiveFormats.push(extracted);
                });

                // Report count for debugging
                out.totalFormats = (sd.formats || []).length;
                out.totalAdaptive = (sd.adaptiveFormats || []).length;
                out.extractedFormats = out.formats.length;
                out.extractedAdaptive = out.adaptiveFormats.length;

                window.postMessage({ type: "__TDM_YT_FORMATS__", data: out }, "*");
            } catch(e) {
                window.postMessage({ type: "__TDM_YT_FORMATS__", data: { error: e.message, title: "", formats: [], adaptiveFormats: [] } }, "*");
            }
        })();`;
        document.documentElement.appendChild(script);
        script.remove();
    }

    // Listen for YouTube format data posted from MAIN world script
    window.addEventListener("message", (event) => {
        if (event.source !== window) return;
        if (event.data?.type !== "__TDM_YT_FORMATS__") return;

        const payload = event.data.data;
        if (!payload) return;

        // Log extraction results for debugging
        console.log("[TDM] YouTube format extraction:", {
            error: payload.error || "none",
            title: payload.title,
            totalFormats: payload.totalFormats,
            totalAdaptive: payload.totalAdaptive,
            extractedFormats: payload.extractedFormats,
            extractedAdaptive: payload.extractedAdaptive
        });

        if (payload.error) {
            console.warn("[TDM] YouTube extraction failed:", payload.error);
            return;
        }

        const allFormats = [
            ...(payload.formats || []),
            ...(payload.adaptiveFormats || [])
        ];

        // Keep only video formats (skip audio-only)
        const videoFormats = allFormats.filter(f =>
            f.url && f.mimeType && f.mimeType.startsWith("video/")
        );

        if (videoFormats.length === 0) {
            console.warn("[TDM] No downloadable YouTube video formats found (all may use signatureCipher)");
            return;
        }

        const pageTitle = (payload.title || document.title || "")
            .replace(/\s*[-|]\s*YouTube$/i, "").trim() || "YouTube Video";
        // Sanitize for filesystem — remove chars illegal on Windows
        const safeTitle = pageTitle.replace(/[\\/:*?"<>|]/g, "_");

        // Replace all media with the extracted YouTube formats
        mediaList = [];

        for (const f of videoFormats) {
            const ext = f.mimeType.includes("webm") ? "webm" : "mp4";
            const label = f.qualityLabel || (f.height ? f.height + "p" : "unknown");
            const isMuxed = (payload.formats || []).some(mf => mf.itag === f.itag);
            const suffix = isMuxed ? "" : " (video only)";

            mediaList.push({
                url: f.url,
                type: "video",
                contentType: f.mimeType.split(";")[0],
                size: parseInt(f.contentLength) || 0,
                filename: `${safeTitle} [${label}].${ext}`,
                quality: label + suffix,
                resolution: f.width && f.height ? `${f.width}\u00d7${f.height}` : "",
                tabUrl: location.href,
                requestHeaders: {},
                timestamp: Date.now(),
                itag: f.itag,
                fps: f.fps || 0
            });
        }

        // Sort: muxed first, then by height descending
        mediaList.sort((a, b) => {
            const aMuxed = a.quality.includes("video only") ? 1 : 0;
            const bMuxed = b.quality.includes("video only") ? 1 : 0;
            if (aMuxed !== bMuxed) return aMuxed - bMuxed;
            const hA = parseInt(a.quality) || 0;
            const hB = parseInt(b.quality) || 0;
            return hB - hA;
        });

        updatePanel();

        // Notify background to replace captured media for this tab
        chrome.runtime.sendMessage({
            type: "YOUTUBE_FORMATS",
            tabUrl: location.href,
            formats: mediaList
        }).catch(() => {});
    });

    // Run YouTube extraction on initial load
    if (isYouTubePage()) {
        // Delay to ensure ytInitialPlayerResponse is populated
        setTimeout(extractYouTubeFormats, 2000);
    }

    // YouTube is a SPA — detect navigation between videos
    window.addEventListener("yt-navigate-finish", () => {
        if (isYouTubePage()) {
            mediaList = [];
            updatePanel();
            setTimeout(extractYouTubeFormats, 2000);
        }
    });
})();
