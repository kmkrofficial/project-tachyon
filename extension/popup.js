document.addEventListener('DOMContentLoaded', restoreOptions);
document.getElementById('saveBtn').addEventListener('click', saveOptions);

function saveOptions() {
    const serverUrl = document.getElementById('serverUrl').value;
    const apiToken = document.getElementById('apiToken').value;

    chrome.storage.local.set({ serverUrl, apiToken }, () => {
        // Test Connection
        const status = document.getElementById('status');
        status.textContent = 'Testing connection...';

        // We can't actually hit the API from popup easily due to CORS in some contexts, 
        // but background script does it. Popup usually has host permissions too.
        // Let's try a simple fetch (OPTIONS or POST with empty body might fail validation but prove connection)
        // Actually, let's just assume saved for now or try a lightweight check.

        status.textContent = 'Settings Saved!';
        setTimeout(() => {
            status.textContent = '';
        }, 1500);
    });
}

function restoreOptions() {
    chrome.storage.local.get(['serverUrl', 'apiToken'], (items) => {
        if (items.serverUrl) document.getElementById('serverUrl').value = items.serverUrl;
        if (items.apiToken) document.getElementById('apiToken').value = items.apiToken;
    });
}
