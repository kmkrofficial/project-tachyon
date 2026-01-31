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

## Future: ClamAV Integration

Task 1.2 will add ClamAV TCP client support for server/Docker deployments. Configure via `CLAMAV_HOST` environment variable.
