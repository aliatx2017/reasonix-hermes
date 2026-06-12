package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"reasonix/internal/netclient"
)

// RemoteRequest is the JSON payload sent to a remote sandbox API.
type RemoteRequest struct {
	Command []string `json:"command"`
	Env     []string `json:"env,omitempty"`
	WorkDir string   `json:"workdir,omitempty"`
	Network bool     `json:"network"`
	Timeout int      `json:"timeout_sec,omitempty"` // 0 = no timeout
}

// RemoteResponse is the JSON response from a remote sandbox API.
type RemoteResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	TimedOut bool   `json:"timed_out"`
	Error    string `json:"error,omitempty"`
}

// commandRemote sends a shell command to a remote sandbox API for isolated
// execution and returns the combined output. It is the remote-analog of
// running locally through sandbox-exec or bubblewrap.
func commandRemote(spec Spec, command string) (string, error) {
	url := spec.RemoteURL
	if url == "" {
		return "", fmt.Errorf("remote sandbox URL is not configured")
	}

	token := spec.RemoteToken
	if token == "" {
		token = os.Getenv("REMOTE_SANDBOX_TOKEN")
	}

	req := RemoteRequest{
		Command: []string{"sh", "-c", command},
		Env:     os.Environ(),
		WorkDir: func() string {
			wd, _ := os.Getwd()
			return wd
		}(),
		Network: spec.Network,
		Timeout: 300, // 5-minute default for remote
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal remote request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build remote request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	// Use a bounded reader to prevent oversized responses.
	client := netclient.DefaultClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("remote sandbox request: %w", err)
	}
	defer resp.Body.Close()

	var rawBody bytes.Buffer
	limitReader := io.LimitReader(resp.Body, 10*1024*1024) // 10 MB cap
	if _, err := rawBody.ReadFrom(limitReader); err != nil {
		return "", fmt.Errorf("read remote response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("remote sandbox returned %d: %s", resp.StatusCode, rawBody.String())
	}

	var remoteResp RemoteResponse
	if err := json.Unmarshal(rawBody.Bytes(), &remoteResp); err != nil {
		return "", fmt.Errorf("decode remote response: %w", err)
	}

	if remoteResp.Error != "" {
		return remoteResp.Stdout, fmt.Errorf("remote sandbox error: %s", remoteResp.Error)
	}

	output := remoteResp.Stdout
	if remoteResp.Stderr != "" {
		output += "\n" + remoteResp.Stderr
	}

	if remoteResp.ExitCode != 0 {
		return output, fmt.Errorf("exit status %d", remoteResp.ExitCode)
	}

	return output, nil
}
