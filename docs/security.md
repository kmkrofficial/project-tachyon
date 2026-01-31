# Security

This document describes the security features in Tachyon, particularly the hybrid antivirus scanning model.

## Native Antivirus Integration

Tachyon implements a "hybrid security" model that leverages the operating system's native antivirus for file scanning after download completion.

### Platform Support

| Platform | Scanner | Behavior |
|----------|---------|----------|
| Windows | Windows Defender | Triggers `MpCmdRun.exe` to scan the downloaded file |
| Linux | NoOp | Logs warning, scanning skipped |
| macOS | NoOp | Logs warning, scanning skipped |

### How It Works

1. **Download Completion**: When a download finishes successfully, Tachyon triggers a native AV scan.
2. **Windows Defender Scan**: On Windows, the engine executes:
   ```
   MpCmdRun.exe -Scan -ScanType 3 -File <path> -DisableRemediation
   ```
3. **Exit Code Handling**:
   - `0`: File is clean
   - `2`: Threat detected (warning emitted to UI)
   - Other: Scan error (logged as warning)

### Non-Blocking Design

AV scan results are **non-blocking warnings**. If a threat is detected:
- The download is still marked as "completed"
- A warning event (`download:av_warning`) is emitted to the UI
- The file is NOT automatically deleted or quarantined

This design respects user autonomy while providing security visibility.

### Events

```typescript
// Listen for AV warnings in the frontend
EventsOn("download:av_warning", (data) => {
  console.warn("Threat detected:", data.warning);
  // data.id - download ID
  // data.path - file path
  // data.warning - threat description
});
```

### Timeout & Cancellation

- Scan timeout: 60 seconds (for large files)
- Scans are cancelled if the user cancels the download
- Context cancellation is handled gracefully (no error reported)

## ClamAV Integration (Server/Docker Mode)

For server deployments or Docker environments, Tachyon supports ClamAV daemon scanning via TCP.

### Configuration

Set the `CLAMAV_HOST` environment variable to enable ClamAV scanning:

```bash
# Format: hostname:port
export CLAMAV_HOST=localhost:3310

# For Docker
docker run -e CLAMAV_HOST=clamav:3310 tachyon
```

### Protocol

Tachyon uses ClamAV's INSTREAM protocol:
1. Connects to ClamAV daemon via TCP
2. Sends `zINSTREAM\0` command
3. Streams file in chunks with 4-byte big-endian length prefix
4. Sends 4 zero bytes to terminate
5. Reads response: "stream: OK" or "stream: <threat> FOUND"

### Scanner Priority

When determining which scanner to use, Tachyon follows this priority:
1. **ClamAV** - If `CLAMAV_HOST` is set
2. **Windows Defender** - On Windows (if no ClamAV configured)
3. **NoOp** - Linux/Mac without ClamAV (warning logged)
