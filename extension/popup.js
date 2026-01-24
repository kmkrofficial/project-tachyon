document.addEventListener('DOMContentLoaded', restoreOptions);
document.getElementById('saveBtn').addEventListener('click', saveOptions);

// Tabs Logic
document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
        document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));

        btn.classList.add('active');
        document.getElementById(btn.dataset.tab).classList.add('active');
    });
});

function saveOptions() {
    const mode = document.querySelector('.tab-btn.active').dataset.tab; // 'local' or 'remote'
    const remoteUrl = document.getElementById('remoteUrl').value;
    const apiToken = document.getElementById('apiToken').value;
    const autoGrab = document.getElementById('autoGrab').checked;

    const config = {
        mode,
        remoteUrl: remoteUrl || "http://localhost:45000",
        apiToken,
        autoGrab
    };

    chrome.storage.local.set(config, () => {
        document.getElementById('statusMsg').textContent = 'Settings Saved';
        setTimeout(() => document.getElementById('statusMsg').textContent = '', 2000);

        // Notify Background to reload config
        // chrome.runtime.sendMessage({ type: "CONFIG_UPDATED" });
    });
}

function restoreOptions() {
    chrome.storage.local.get(['mode', 'remoteUrl', 'apiToken', 'autoGrab'], (items) => {
        // Mode
        if (items.mode === 'remote') {
            document.querySelector('[data-tab="remote"]').click();
        } else {
            document.querySelector('[data-tab="local"]').click();
        }

        if (items.remoteUrl) document.getElementById('remoteUrl').value = items.remoteUrl;
        if (items.apiToken) document.getElementById('apiToken').value = items.apiToken;
        document.getElementById('autoGrab').checked = items.autoGrab !== false; // Default true
    });
}
