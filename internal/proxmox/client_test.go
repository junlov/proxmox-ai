package proxmox

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/junlov/proxmox-ai/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newMockClient(t *testing.T, tokenSecret string, fn roundTripFunc) *APIClient {
	t.Helper()
	return &APIClient{
		envs: map[string]apiEnvironment{
			"home": {
				baseURL:     "https://proxmox.example.com",
				tokenID:     "root@pam!agent",
				tokenSecret: tokenSecret,
			},
		},
		httpClient: &http.Client{
			Transport: fn,
			Timeout:   3 * time.Second,
		},
		readRetries: 3,
	}
}

func TestExecuteDryRunSkipsHTTPCall(t *testing.T) {
	var calls int32
	client := newMockClient(t, "test-secret", func(r *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":"ok"}`)),
			Header:     make(http.Header),
		}, nil
	})

	t.Setenv("PVE_TEST_SECRET", "test-secret")
	result, err := client.Execute(ActionRequest{
		Environment: "home",
		Action:      ActionStartVM,
		Target:      "node1/100",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("expected zero HTTP calls for dry-run, got %d", got)
	}
	if result.Status != "planned" {
		t.Fatalf("unexpected status: %q", result.Status)
	}
}

func TestExecuteStartVMSendsAuthAndEndpoint(t *testing.T) {
	var gotPath, gotMethod, gotAuth string
	client := newMockClient(t, "super-secret", func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":"UPID:node1:0001"}`)),
			Header:     make(http.Header),
		}, nil
	})

	t.Setenv("PVE_TEST_SECRET", "super-secret")
	result, err := client.Execute(ActionRequest{
		Environment: "home",
		Action:      ActionStartVM,
		Target:      "node1/101",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if gotPath != "/api2/json/nodes/node1/qemu/101/status/start" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("unexpected method: %q", gotMethod)
	}
	if gotAuth != "PVEAPIToken=root@pam!agent=super-secret" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if result.Message != "UPID:node1:0001" {
		t.Fatalf("unexpected message: %q", result.Message)
	}
}

func TestExecuteStartVMSupportsVMTargetWithNodeParam(t *testing.T) {
	var gotPath string
	client := newMockClient(t, "super-secret", func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":"UPID:node1:0001"}`)),
			Header:     make(http.Header),
		}, nil
	})

	result, err := client.Execute(ActionRequest{
		Environment: "home",
		Action:      ActionStartVM,
		Target:      "vm/101",
		Params: map[string]any{
			"node": "node1",
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if gotPath != "/api2/json/nodes/node1/qemu/101/status/start" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
	if result.Message != "UPID:node1:0001" {
		t.Fatalf("unexpected message: %q", result.Message)
	}
}

func TestExecuteReadVMRetriesOnTransientErrors(t *testing.T) {
	var calls int32
	client := newMockClient(t, "retry-secret", func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api2/json/nodes/node1/qemu/200/status/current" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %q", r.Method)
		}

		call := atomic.AddInt32(&calls, 1)
		if call < 3 {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader(`{"errors":"temporary upstream failure"}`)),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":{"status":"running"}}`)),
			Header:     make(http.Header),
		}, nil
	})

	t.Setenv("PVE_TEST_SECRET", "retry-secret")
	result, err := client.Execute(ActionRequest{
		Environment: "home",
		Action:      ActionReadVM,
		Target:      "node1/200",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 calls with retries, got %d", got)
	}
	if result.Status != "ok" {
		t.Fatalf("unexpected status: %q", result.Status)
	}
}

func TestExecuteReadInventoryReturnsOnlyRunningWhenRequested(t *testing.T) {
	var gotPath, gotMethod string
	client := newMockClient(t, "inventory-secret", func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"vmid":100,"name":"web","type":"qemu","status":"running"},{"vmid":200,"name":"batch","type":"lxc","status":"stopped"},{"vmid":300,"name":"api","type":"lxc","status":"running"}]}`)),
			Header:     make(http.Header),
		}, nil
	})

	t.Setenv("PVE_TEST_SECRET", "inventory-secret")
	result, err := client.Execute(ActionRequest{
		Environment: "home",
		Action:      ActionReadInventory,
		Target:      "inventory/running",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if gotPath != "/api2/json/cluster/resources" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("unexpected method: %q", gotMethod)
	}
	if result.Status != "ok" {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	items, ok := result.Data.([]any)
	if !ok {
		t.Fatalf("expected []any data, got %T", result.Data)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 running resources, got %d", len(items))
	}
}

func TestExecuteCloneVMSendsCloneEndpoint(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	client := newMockClient(t, "clone-secret", func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":"UPID:node1:9999"}`)),
			Header:     make(http.Header),
		}, nil
	})

	result, err := client.Execute(ActionRequest{
		Environment: "home",
		Action:      ActionCloneVM,
		Target:      "vm/103",
		Params: map[string]any{
			"node":     "node1",
			"newid":    104,
			"name":     "ubuntu-clone-104",
			"snapname": "baseline",
			"full":     false,
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if gotPath != "/api2/json/nodes/node1/qemu/103/clone" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("unexpected method: %q", gotMethod)
	}
	if !strings.Contains(gotBody, "newid=104") {
		t.Fatalf("expected body to include newid, got %q", gotBody)
	}
	if !strings.Contains(gotBody, "snapname=baseline") {
		t.Fatalf("expected body to include snapname, got %q", gotBody)
	}
	if !strings.Contains(gotBody, "full=0") {
		t.Fatalf("expected body to include full=0, got %q", gotBody)
	}
	if result.Message != "UPID:node1:9999" {
		t.Fatalf("unexpected message: %q", result.Message)
	}
}

func TestNewAPIClientTLSVerificationEnabled(t *testing.T) {
	client, err := NewAPIClient(nil)
	if err != nil {
		t.Fatalf("NewAPIClient returned error: %v", err)
	}
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.httpClient.Transport)
	}
	if transport.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig should be set")
	}
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify must remain false")
	}
	if transport.TLSClientConfig.MinVersion < 0x0303 {
		t.Fatalf("unexpected MinVersion: %#x", transport.TLSClientConfig.MinVersion)
	}
}

func TestNewAPIClientFailsWhenTokenSecretMissing(t *testing.T) {
	t.Setenv("PVE_TEST_SECRET", "")
	_, err := NewAPIClient([]config.Environment{{
		Name:           "home",
		BaseURL:        "https://proxmox.example.com",
		TokenID:        "root@pam!agent",
		TokenSecretEnv: "PVE_TEST_SECRET",
	}})
	if err == nil {
		t.Fatal("expected constructor error when token secret env var is missing")
	}
	if err.Error() != `missing token secret env var "PVE_TEST_SECRET" for environment "home"` {
		t.Fatalf("unexpected error: %v", err)
	}
}
