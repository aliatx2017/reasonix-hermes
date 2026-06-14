# Windows Sandbox Support — Design Document

> Investigation and recommendation for adding OS-level sandbox confinement to
> Windows, making `[sandbox].mode = "enforce"` functional on all three platforms.

> **Research validation** (Carlucci et al., 2026, 2606.13474): The "Loss of Control
> Risk" paper finds that operational variability erodes safeguards over time and
> that the constrain/audit/reverse/halt taxonomy is the correct risk model for
> internal agent deployment. Our three-platform sandbox (macOS Seatbelt, Linux
> bubblewrap, Windows AppContainer) directly implements this taxonomy: constrain
> (read-only root), audit (checkpoint snapshots), reverse (undo/rewind), halt
> (permission gating).

## 1. Current State

| Platform | Tool | Implementation | `Available()` |
|----------|------|---------------|---------------|
| macOS | `sandbox-exec` (built-in) | SBPL profile: read-all, write-allow roots, deny network | `exec.LookPath("sandbox-exec")` |
| Linux | `bwrap` (bubblewrap) | bind-mounts + unshare-net | `exec.LookPath("bwrap")` |
| Windows | **none** | Falls through `!darwin` build tag → `seatbelt_other.go` | Always `false` |

The sandbox interface is clean:
- `Spec{Mode, WriteRoots, Network, RemoteURL, RemoteToken, Shell}` — configuration
- `Command(spec, sh, command) → ([]string, bool)` — returns argv; bool = wrapped
- `Available() → bool` — whether the local tool exists on PATH
- `Run(spec, command) → (string, bool, error)` — remote execution only

The contract: **read everything, write only to WriteRoots+temp+caches, network optional**.

## 2. Windows Sandbox Options Evaluated

### 2.1 Job Objects
- **What**: Kernel object for managing process groups. Resource limits only (CPU, memory, priority).
- **Verdict**: ❌ No filesystem or network isolation. Useful as a process-lifecycle addition, not a sandbox.

### 2.2 Hyper-V Containers
- **What**: Docker-style container with dedicated kernel. Requires Hyper-V enabled, Docker runtime.
- **Verdict**: ❌ Too heavy. Seconds of startup overhead per command. Not suitable for per-bash-call sandboxing.

### 2.3 Windows Sandbox
- **What**: Lightweight VM-based desktop. Configured via `.wsb` XML. Built into Win10/11 Pro+.
- **Verdict**: ❌ GUI-only, ~10s startup, requires Hyper-V, unavailable on Home editions. Not headless/programmatic.

### 2.4 AppContainer (LowBox) ✅
- **What**: Process isolation built into Windows 8+. Process runs with an AppContainer SID — can read freely, write only to granted locations, network controlled by capability SIDs.
- **Verdict**: ✅ Best match. Same semantic model as macOS Seatbelt (default-deny writes, allow-list roots, network toggle). No virtualization, no external dependencies, sub-millisecond startup.

## 3. AppContainer Technical Details

### 3.1 How It Works

An AppContainer process receives a restricted token containing:
- An AppContainer SID (unique per app identity)
- Capability SIDs (e.g. `internetClient` for network access)
- Low integrity level (always)

The Windows security subsystem then denies access to resources not explicitly granted to that SID — producing the same "read anywhere, write only to granted paths" model.

### 3.2 Win32 API Surface

| API | DLL | Purpose |
|-----|-----|---------|
| `CreateAppContainerProfile` | `userenv` | Create/register an AppContainer profile → returns SID |
| `DeleteAppContainerProfile` | `userenv` | Remove the profile on cleanup |
| `DeriveCapabilitySidsFromName` | `api-ms-win-security-cpwl-l1-1-0` | Convert `"internetClient"` → capability SID |
| `CreateProcess` + `EXTENDED_STARTUPINFO_PRESENT` | `kernel32` | Launch process with AppContainer token |
| `InitializeProcThreadAttributeList` | `kernel32` | Build attribute list for `STARTUPINFOEX` |
| `UpdateProcThreadAttribute` | `kernel32` | Set `PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES` |
| `SetNamedSecurityInfo` | `advapi32` | Grant write access to WriteRoots for the AppContainer SID |

### 3.3 Go Support

`golang.org/x/sys/windows` v0.31.0 (already in `go.mod`) provides:
- ✅ `CreateProcess`, `STARTUPINFOEX`, `ProcThreadAttributeList`
- ✅ `InitializeProcThreadAttributeList`, `UpdateProcThreadAttribute`, `DeleteProcThreadAttributeList`
- ✅ `EXTENDED_STARTUPINFO_PRESENT` constant

**Missing** (must define ourselves):
- `PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES` = `0x0002000D`
- `SECURITY_CAPABILITIES` struct
- `SID_AND_ATTRIBUTES` struct
- `CreateAppContainerProfile`, `DeleteAppContainerProfile`, `DeriveCapabilitySidsFromName` syscalls

### 3.4 Write Confinement

By default, an AppContainer process can write only to:
- Its virtualized profile directory (`%LOCALAPPDATA%\Packages\<SID>\...`)
- Directories explicitly granted via ACL

For the coding agent, WriteRoots (workspace, temp dirs, toolchain caches) must be granted via `SetNamedSecurityInfo` — adding an ALLOW ACE for the AppContainer SID on each WriteRoot directory.

### 3.5 Shell

On Windows the bash tool resolves to `cmd.exe /c` or `powershell -Command` (see `shell.go`). The AppContainer wraps whichever shell is resolved — the sandbox layer is shell-agnostic.

## 4. Implementation Plan

### 4.1 Files

```
internal/sandbox/appcontainer_windows.go   (~350 lines)
internal/sandbox/appcontainer_windows_test.go (~100 lines)
internal/sandbox/seatbelt_other.go          (modify: restrict to !darwin,!windows)
```

### 4.2 New Types & Constants (~60 lines)

```go
//go:build windows

const (
    PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES = 0x0002000D
)

type SID_AND_ATTRIBUTES struct {
    Sid        *windows.SID
    Attributes uint32
}

type SECURITY_CAPABILITIES struct {
    AppContainerSid *windows.SID
    Capabilities    *SID_AND_ATTRIBUTES
    CapabilityCount uint32
    Reserved        uint32
}

// syscall wrappers
//sys createAppContainerProfile(...)  = userenv.CreateAppContainerProfile
//sys deleteAppContainerProfile(...)  = userenv.DeleteAppContainerProfile
//sys deriveCapabilitySidsFromName(...) = api-ms-win-security-cpwl-l1-1-0.DeriveCapabilitySidsFromName
```

### 4.3 Core Logic (~200 lines)

```go
func Command(spec Spec, sh Shell, command string) ([]string, bool) {
    if !spec.enforce() {
        return sh.argv(command), false
    }
    // AppContainer always available on Windows 8+
    return appContainerArgv(spec, sh, command), true
}

func Available() bool {
    // Win8+ always supports AppContainer; no external tool needed
    return true
}

func appContainerArgv(spec Spec, sh Shell, command string) []string {
    // 1. Get or create AppContainer profile SID ("ReasonixSandbox")
    // 2. Build capability SIDs (internetClient if spec.Network)
    // 3. Grant WriteRoots ACLs to the AppContainer SID
    // 4. Build SECURITY_CAPABILITIES
    // 5. Build STARTUPINFOEX with ProcThreadAttributeList
    // 6. Call CreateProcess with EXTENDED_STARTUPINFO_PRESENT
    // 7. Return argv
}
```

### 4.4 WriteRoot ACL Granting

For each directory in `writeAllowDirs(spec.WriteRoots)`:
```go
func grantWriteAccess(dir string, appContainerSid *windows.SID) error {
    // Get existing DACL
    // Add ALLOW ACE: GENERIC_WRITE | GENERIC_READ | GENERIC_EXECUTE for appContainerSid
    // SetNamedSecurityInfo(dir, SE_FILE_OBJECT, DACL_SECURITY_INFORMATION, ...)
}
```

### 4.5 Cleanup

AppContainer profiles are persistent. We can:
- **Keep**: Profile reuse across sessions is cheap (single `CreateAppContainerProfile` call, cached)
- **Delete on session close**: `deleteAppContainerProfile` on controller Close()
- **Lazy**: Profile creation is idempotent — second call with same name returns existing SID

## 5. Complexity & Risk Assessment

| Factor | Rating | Notes |
|--------|--------|-------|
| **Lines of new code** | ~350 Go + ~60 syscall defs | Moderate |
| **Win32 API surface** | 7 functions, 3 structs | Well-documented, stable since Win8 |
| **golang.org/x/sys gaps** | 1 constant, 2 structs, 3 syscalls | Must define manually; no upstream dependency |
| **WriteRoot ACL management** | ~50 lines | `SetNamedSecurityInfo` is the trickiest part |
| **Testing** | Hard to test cross-platform (needs Windows host) | CI needs Windows runner |
| **Backward compat** | Zero risk — new file, gated by `//go:build windows` | Existing paths unchanged |
| **Fallback** | If any part fails, fall back to unwrapped command | Graceful degradation |

## 6. Alternative: Minimal Approach

If the full AppContainer implementation is too complex for an initial pass:

### 6.1 Job Object for Process Lifecycle Only
Add a Windows Job Object that:
- Kills all child processes when the parent dies
- Does NOT provide filesystem/network isolation

~30 lines, lower risk. Filesystem/network isolation deferred to remote sandbox.

### 6.2 Documentation-Only
Update docs to recommend `mode = "remote"` for Windows users, document the gap clearly. Zero code changes.

## 7. Recommendation

**Implement the full AppContainer approach** (`appcontainer_windows.go`). Rationale:

1. The existing sandbox is one of Reasonix's strongest security features — leaving Windows as the only platform without it is a significant gap
2. AppContainer is the canonical Windows answer to this problem (Microsoft's own recommendation for legacy app sandboxing)
3. The interface already exists — we're adding one more platform implementation behind the same `Command()` signature
4. The `golang.org/x/sys/windows` package already has 80% of the needed APIs; the remaining 20% is straightforward syscall definitions
5. ~350 lines is a reasonable scope for one focused file

### Implementation Order

| Phase | Scope | Status |
|-------|-------|--------|
| 1. Research & design | This document | ✅ Done |
| 2. Win32 types + syscalls | `appcontainer_windows.go` types + `CreateAppContainerProfile`/`DeleteAppContainerProfile` | Next |
| 3. Core Command() impl | `CreateProcess` + `SECURITY_CAPABILITIES` + network caps | Next |
| 4. WriteRoot ACLs | `SetNamedSecurityInfo` grants per WriteRoot | Next |
| 5. Tests | Windows CI runner, basic confinement tests | After core |
| 6. Docs | Update `docs/SPEC.md` §8.4, `HERMES-GUIDE.md` §16.x | After tests |
