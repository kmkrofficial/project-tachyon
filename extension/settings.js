document.addEventListener("DOMContentLoaded", init);

async function init() {
    document.getElementById("backBtn").addEventListener("click", () => {
        // If opened as a popup tab, close it; otherwise go back
        window.close();
    });

    document.getElementById("saveBtn").addEventListener("click", saveSettings);

    await loadSettings();
}

async function loadSettings() {
    return new Promise((resolve) => {
        chrome.runtime.sendMessage({ type: "GET_SETTINGS" }, (settings) => {
            if (!settings) { resolve(); return; }

            document.getElementById("collisionAction").value = settings.collisionAction || "ask";
            document.getElementById("notifyDownloadStarted").checked = settings.notifyDownloadStarted !== false;
            document.getElementById("notifyDownloadComplete").checked = settings.notifyDownloadComplete !== false;
            document.getElementById("notifyDownloadFailed").checked = settings.notifyDownloadFailed !== false;
            document.getElementById("notifyCollisionDetected").checked = settings.notifyCollisionDetected !== false;
            document.getElementById("notifyMediaCaptured").checked = !!settings.notifyMediaCaptured;
            document.getElementById("notifyConnectionLost").checked = settings.notifyConnectionLost !== false;

            resolve();
        });
    });
}

function saveSettings() {
    const settings = {
        collisionAction: document.getElementById("collisionAction").value,
        notifyDownloadStarted: document.getElementById("notifyDownloadStarted").checked,
        notifyDownloadComplete: document.getElementById("notifyDownloadComplete").checked,
        notifyDownloadFailed: document.getElementById("notifyDownloadFailed").checked,
        notifyCollisionDetected: document.getElementById("notifyCollisionDetected").checked,
        notifyMediaCaptured: document.getElementById("notifyMediaCaptured").checked,
        notifyConnectionLost: document.getElementById("notifyConnectionLost").checked
    };

    chrome.runtime.sendMessage({ type: "SAVE_SETTINGS", settings }, (resp) => {
        if (resp?.ok) {
            const msg = document.getElementById("statusMsg");
            msg.textContent = "Settings saved";
            setTimeout(() => { msg.textContent = ""; }, 2000);
        }
    });
}
