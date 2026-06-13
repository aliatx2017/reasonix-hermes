//go:build windows

package sandbox

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// --- Missing constants and types (not yet in golang.org/x/sys/windows) ---

const (
	// PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES attaches an AppContainer token
	// to a process created via CreateProcess + EXTENDED_STARTUPINFO_PRESENT.
	PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES = 0x0002000D

	// SE_FILE_OBJECT is the object type for filesystem objects passed to
	// SetNamedSecurityInfo.
	SE_FILE_OBJECT = 1

	// DACL_SECURITY_INFORMATION requests that SetNamedSecurityInfo update the
	// discretionary ACL.
	DACL_SECURITY_INFORMATION = 4

	// internetClient / internetClientServer capability names for network access.
	capInternetClient       = "internetClient"
	capInternetClientServer = "internetClientServer"

	// AppContainer profile name — shared across sessions.
	appContainerName = "ReasonixSandbox"
)

// SECURITY_CAPABILITIES describes the AppContainer token for a process.
type SECURITY_CAPABILITIES struct {
	AppContainerSid *windows.SID
	Capabilities    *windows.SIDAndAttributes
	CapabilityCount uint32
	Reserved        uint32
}

// --- DLLs for missing syscalls ---

var (
	modUserenv                       = windows.NewLazySystemDLL("userenv.dll")
	modCPWL                          = windows.NewLazySystemDLL("api-ms-win-security-cpwl-l1-1-0.dll")
	procCreateAppContainerProfile    = modUserenv.NewProc("CreateAppContainerProfile")
	procDeleteAppContainerProfile    = modUserenv.NewProc("DeleteAppContainerProfile")
	procDeriveCapabilitySidsFromName = modCPWL.NewProc("DeriveCapabilitySidsFromName")
)

// createAppContainerProfile creates or opens an AppContainer profile and returns
// its SID. The call is idempotent — if the profile already exists, the existing
// SID is returned.
func createAppContainerProfile(name string, displayName string) (*windows.SID, error) {
	var psid *windows.SID
	pname, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	pdisplay, err := windows.UTF16PtrFromString(displayName)
	if err != nil {
		return nil, err
	}
	r0, _, _ := syscall.SyscallN(
		procCreateAppContainerProfile.Addr(),
		uintptr(unsafe.Pointer(pname)),
		uintptr(unsafe.Pointer(pdisplay)),
		uintptr(unsafe.Pointer(pdisplay)), // description = display name
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&psid)),
	)
	if r0 != 0 {
		return nil, fmt.Errorf("CreateAppContainerProfile: %w", windows.Errno(r0))
	}
	return psid, nil
}

// deleteAppContainerProfile removes an AppContainer profile. Safe to call on a
// non-existent profile (returns success).
func deleteAppContainerProfile(sid *windows.SID) error {
	r0, _, _ := syscall.SyscallN(
		procDeleteAppContainerProfile.Addr(),
		uintptr(unsafe.Pointer(sid)),
	)
	if r0 != 0 {
		return fmt.Errorf("DeleteAppContainerProfile: %w", windows.Errno(r0))
	}
	return nil
}

// deriveCapabilitySidsFromName converts a capability name (e.g. "internetClient")
// to its capability SID group.
func deriveCapabilitySidsFromName(name string) ([]windows.SIDAndAttributes, error) {
	pname, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	var capGroupSids *windows.SIDAndAttributes
	var capCount uint32
	r0, _, _ := syscall.SyscallN(
		procDeriveCapabilitySidsFromName.Addr(),
		uintptr(unsafe.Pointer(pname)),
		uintptr(unsafe.Pointer(&capGroupSids)),
		uintptr(unsafe.Pointer(&capCount)),
	)
	if r0 != 0 {
		return nil, fmt.Errorf("DeriveCapabilitySidsFromName(%s): %w", name, windows.Errno(r0))
	}
	// Convert the returned array into a Go slice.
	out := unsafe.Slice(capGroupSids, capCount)
	result := make([]windows.SIDAndAttributes, capCount)
	copy(result, out)
	return result, nil
}

// --- AppContainer singleton ---

var (
	appContainerOnce sync.Once
	appContainerSid  *windows.SID
	appContainerErr  error
)

// getAppContainerSid returns the cached AppContainer profile SID, creating the
// profile on first call. The profile is never deleted — it is reused across
// sessions (CreateAppContainerProfile is idempotent).
func getAppContainerSid() (*windows.SID, error) {
	appContainerOnce.Do(func() {
		appContainerSid, appContainerErr = createAppContainerProfile(
			appContainerName,
			"Reasonix Hermes Sandbox",
		)
	})
	return appContainerSid, appContainerErr
}

// --- Capability cache ---

var (
	netCapsOnce sync.Once
	netCapSids  []windows.SIDAndAttributes
	netCapsErr  error
)

// getNetworkCapabilities returns the internetClient + internetClientServer
// capability SIDs, derived once on first call.
func getNetworkCapabilities() ([]windows.SIDAndAttributes, error) {
	netCapsOnce.Do(func() {
		for _, capName := range []string{capInternetClient, capInternetClientServer} {
			sa, err := deriveCapabilitySidsFromName(capName)
			if err != nil {
				netCapsErr = err
				return
			}
			netCapSids = append(netCapSids, sa...)
		}
	})
	return netCapSids, netCapsErr
}

// --- WriteRoot ACL granting ---

var (
	aclsMu      sync.Mutex
	aclsGranted = map[string]bool{}
)

// grantWriteAccess grants GENERIC_WRITE | GENERIC_READ | GENERIC_EXECUTE to the
// AppContainer SID on the given directory by building a new DACL and calling
// SetNamedSecurityInfo. Already-granted directories (tracked in aclsGranted) are
// skipped. On error the error is returned — the caller may log and continue.
func grantWriteAccess(dir string, appSid *windows.SID) error {
	aclsMu.Lock()
	if aclsGranted[dir] {
		aclsMu.Unlock()
		return nil
	}
	aclsGranted[dir] = true
	aclsMu.Unlock()

	const desiredAccess = windows.GENERIC_WRITE | windows.GENERIC_READ | windows.GENERIC_EXECUTE

	ea := windows.EXPLICIT_ACCESS{
		AccessPermissions: desiredAccess,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       windows.OBJECT_INHERIT_ACE | windows.CONTAINER_INHERIT_ACE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_UNKNOWN,
			TrusteeValue: windows.TrusteeValueFromSID(appSid),
		},
	}

	dacl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{ea}, nil)
	if err != nil {
		return fmt.Errorf("grantWriteAccess ACLFromEntries(%s): %w", dir, err)
	}

	return windows.SetNamedSecurityInfo(
		dir,
		SE_FILE_OBJECT,
		DACL_SECURITY_INFORMATION,
		nil, nil,
		dacl, nil,
	)
}

// --- Command implementation ---

// Command returns the argv to run `command` through sh. On Windows, the
// AppContainer sandbox requires programmatic CreateProcess (it cannot be
// expressed as a wrapper binary in argv). The bash tool detects this via
// IsAppContainer() and uses ExecAppContainer instead of exec.Cmd.
func Command(spec Spec, sh Shell, command string) ([]string, bool) {
	if spec.Mode != "enforce" {
		return sh.argv(command), false
	}

	// Pre-grant write access to all WriteRoot directories. Best-effort: if ACL
	// granting fails, the command may still work if the directories are inside
	// the AppContainer's virtualized profile.
	if sid, err := getAppContainerSid(); err == nil {
		for _, dir := range writeAllowDirs(spec.WriteRoots) {
			_ = grantWriteAccess(dir, sid)
		}
	}

	// The command runs unwrapped via exec.Cmd. The bash tool detects
	// IsAppContainer() and calls ExecAppContainer for true sandboxing.
	return sh.argv(command), false
}

// Available reports whether the AppContainer sandbox is available. On Windows
// 8+, AppContainer is always available — no external tool needed.
func Available() bool {
	_, err := getAppContainerSid()
	return err == nil
}

// IsAppContainer reports whether the Windows AppContainer sandbox is the
// active enforcement backend. Always true on Windows (AppContainer is
// built into the OS from Win8+).
func IsAppContainer() bool { return true }

// --- AppContainer process launcher ---

// ExecAppContainer launches `command` through `sh` inside an AppContainer and
// returns combined stdout+stderr. This is called by the bash tool instead of
// exec.Cmd on Windows when sandbox enforcement is active.
func ExecAppContainer(spec Spec, sh Shell, command string, env []string, dir string) (string, error) {
	appSid, err := getAppContainerSid()
	if err != nil {
		return "", fmt.Errorf("appcontainer: get SID: %w", err)
	}

	// Grant write access to WriteRoot directories.
	for _, d := range writeAllowDirs(spec.WriteRoots) {
		_ = grantWriteAccess(d, appSid)
	}

	// Build capability SIDs.
	var caps []windows.SIDAndAttributes
	if spec.Network {
		caps, err = getNetworkCapabilities()
		if err != nil {
			return "", fmt.Errorf("appcontainer: get network caps: %w", err)
		}
	}

	sc := SECURITY_CAPABILITIES{
		AppContainerSid: appSid,
	}
	if len(caps) > 0 {
		sc.Capabilities = &caps[0]
		sc.CapabilityCount = uint32(len(caps))
	}

	// Build argv: shell path + command args.
	argv := sh.argv(command)

	// Create pipes for stdout and stderr.
	var sa windows.SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 1

	var stdoutRead, stdoutWrite windows.Handle
	if err := windows.CreatePipe(&stdoutRead, &stdoutWrite, &sa, 0); err != nil {
		return "", fmt.Errorf("appcontainer: CreatePipe(stdout): %w", err)
	}
	defer windows.CloseHandle(stdoutRead)

	var stderrRead, stderrWrite windows.Handle
	if err := windows.CreatePipe(&stderrRead, &stderrWrite, &sa, 0); err != nil {
		windows.CloseHandle(stdoutWrite)
		return "", fmt.Errorf("appcontainer: CreatePipe(stderr): %w", err)
	}
	defer windows.CloseHandle(stderrRead)

	var si windows.StartupInfoEx
	si.StartupInfo.Cb = uint32(unsafe.Sizeof(si))
	si.StartupInfo.Flags = windows.STARTF_USESTDHANDLES
	si.StartupInfo.StdOutput = stdoutWrite
	si.StartupInfo.StdErr = stderrWrite

	// Build attribute list for SECURITY_CAPABILITIES.
	al, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		windows.CloseHandle(stdoutWrite)
		windows.CloseHandle(stderrWrite)
		return "", fmt.Errorf("appcontainer: NewProcThreadAttributeList: %w", err)
	}
	defer al.Delete()

	if err := al.Update(
		PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES,
		unsafe.Pointer(&sc),
		unsafe.Sizeof(sc),
	); err != nil {
		windows.CloseHandle(stdoutWrite)
		windows.CloseHandle(stderrWrite)
		return "", fmt.Errorf("appcontainer: Update attribute: %w", err)
	}
	si.ProcThreadAttributeList = al.List()

	// Build command line and environment strings.
	cmdLine := windows.StringToUTF16Ptr(makeCmdLine(argv))
	var exeName *uint16
	if len(argv) > 0 {
		exeName, _ = windows.UTF16PtrFromString(argv[0])
	}
	var envBlock *uint16
	if len(env) > 0 {
		envBlock = windows.StringToUTF16Ptr(makeEnvBlock(env))
	}
	var curDir *uint16
	if dir != "" {
		curDir, _ = windows.UTF16PtrFromString(dir)
	}

	var pi windows.ProcessInformation
	err = windows.CreateProcess(
		exeName,
		cmdLine,
		nil, nil,
		true, // inherit handles (for the pipes)
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT,
		envBlock,
		curDir,
		&si.StartupInfo,
		&pi,
	)
	windows.CloseHandle(stdoutWrite)
	windows.CloseHandle(stderrWrite)

	if err != nil {
		return "", fmt.Errorf("appcontainer: CreateProcess: %w", err)
	}
	defer windows.CloseHandle(pi.Process)
	defer windows.CloseHandle(pi.Thread)

	// Read stdout and stderr.
	var outBuf, errBuf []byte
	buf := make([]byte, 4096)
	for {
		var n uint32
		if e := windows.ReadFile(stdoutRead, buf, &n, nil); e != nil || n == 0 {
			break
		}
		outBuf = append(outBuf, buf[:n]...)
	}
	for {
		var n uint32
		if e := windows.ReadFile(stderrRead, buf, &n, nil); e != nil || n == 0 {
			break
		}
		errBuf = append(errBuf, buf[:n]...)
	}

	// Wait for process completion.
	windows.WaitForSingleObject(pi.Process, windows.INFINITE)

	var exitCode uint32
	windows.GetExitCodeProcess(pi.Process, &exitCode)

	result := string(outBuf)
	if len(errBuf) > 0 {
		if len(result) > 0 {
			result += "\n"
		}
		result += string(errBuf)
	}

	if exitCode != 0 {
		return result, fmt.Errorf("exit status %d", exitCode)
	}

	return result, nil
}

// makeCmdLine joins argv into a Windows command line, quoting arguments that
// contain spaces or special characters.
func makeCmdLine(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	var cmdLine string
	for i, arg := range argv {
		if i > 0 {
			cmdLine += " "
		}
		cmdLine += windows.EscapeArg(arg)
	}
	return cmdLine
}

// makeEnvBlock joins env entries with null bytes and adds a trailing null,
// producing the format CreateProcess expects.
func makeEnvBlock(env []string) string {
	var block string
	for _, e := range env {
		block += e + "\x00"
	}
	block += "\x00"
	return block
}
