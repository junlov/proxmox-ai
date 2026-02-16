package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/junlov/proxmox-ai/internal/config"
)

type ActionType string

const (
	ActionReadVM       ActionType = "read_vm"
	ActionStartVM      ActionType = "start_vm"
	ActionStopVM       ActionType = "stop_vm"
	ActionSnapshotVM   ActionType = "snapshot_vm"
	ActionMigrateVM    ActionType = "migrate_vm"
	ActionDeleteVM     ActionType = "delete_vm"
	ActionStorageEdit  ActionType = "storage_edit"
	ActionFirewallEdit ActionType = "firewall_edit"
)

type ActionRequest struct {
	Environment    string         `json:"environment"`
	Action         ActionType     `json:"action"`
	Target         string         `json:"target"`
	Params         map[string]any `json:"params"`
	DryRun         bool           `json:"dry_run"`
	ApprovedBy     string         `json:"approved_by,omitempty"`
	ApprovalTicket string         `json:"approval_ticket,omitempty"`
	Reason         string         `json:"reason,omitempty"`
	ExpiresAt      string         `json:"expires_at,omitempty"`
	Actor          string         `json:"-"`
}

type ActionResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Client interface {
	Execute(req ActionRequest) (ActionResult, error)
}

const (
	defaultHTTPTimeout = 15 * time.Second
	defaultReadRetries = 3
)

type APIError struct {
	StatusCode int
	Method     string
	Endpoint   string
	Message    string
}

func (e *APIError) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("proxmox API error (%s %s): %s", e.Method, e.Endpoint, e.Message)
	}
	return fmt.Sprintf("proxmox API error (%s %s) status %d: %s", e.Method, e.Endpoint, e.StatusCode, e.Message)
}

type apiEnvironment struct {
	baseURL     string
	tokenID     string
	tokenSecret string
}

type APIClient struct {
	envs        map[string]apiEnvironment
	httpClient  *http.Client
	readRetries int
}

func NewAPIClient(environments []config.Environment) (*APIClient, error) {
	envs := make(map[string]apiEnvironment, len(environments))
	for _, env := range environments {
		tokenSecret := strings.TrimSpace(os.Getenv(env.TokenSecretEnv))
		if tokenSecret == "" {
			return nil, fmt.Errorf("missing token secret env var %q for environment %q", env.TokenSecretEnv, env.Name)
		}
		envs[env.Name] = apiEnvironment{
			baseURL:     strings.TrimRight(env.BaseURL, "/"),
			tokenID:     env.TokenID,
			tokenSecret: tokenSecret,
		}
	}
	return &APIClient{
		envs:        envs,
		httpClient:  newHTTPClient(defaultHTTPTimeout),
		readRetries: defaultReadRetries,
	}, nil
}

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: false,
			},
			ForceAttemptHTTP2: true,
		},
	}
}

func BuildTokenAuthHeader(tokenID, tokenSecret string) string {
	return fmt.Sprintf("PVEAPIToken=%s=%s", tokenID, tokenSecret)
}

func (c *APIClient) Execute(req ActionRequest) (ActionResult, error) {
	if req.DryRun {
		return ActionResult{Status: "planned", Message: "dry-run only; no Proxmox API call made"}, nil
	}

	env, ok := c.envs[req.Environment]
	if !ok {
		return ActionResult{}, fmt.Errorf("unknown environment %q", req.Environment)
	}

	method, endpoint, params, err := requestSpec(req)
	if err != nil {
		return ActionResult{}, err
	}

	body := encodeParams(params)
	respBody, err := c.performRequest(env, method, endpoint, body)
	if err != nil {
		return ActionResult{}, err
	}

	var envelope struct {
		Data any `json:"data"`
	}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &envelope); err != nil {
			return ActionResult{}, fmt.Errorf("decode proxmox response: %w", err)
		}
	}

	status := "accepted"
	message := "request accepted by Proxmox API"
	if req.Action == ActionReadVM {
		status = "ok"
		message = "vm state retrieved from Proxmox API"
	}
	if taskID, ok := envelope.Data.(string); ok && taskID != "" {
		message = taskID
	}

	return ActionResult{Status: status, Message: message}, nil
}

func requestSpec(req ActionRequest) (method string, endpoint string, params map[string]any, err error) {
	switch req.Action {
	case ActionReadVM:
		node, vmid, err := parseVMTarget(req.Target)
		if err != nil {
			return "", "", nil, err
		}
		return http.MethodGet, fmt.Sprintf("/api2/json/nodes/%s/qemu/%s/status/current", node, vmid), nil, nil
	case ActionStartVM:
		node, vmid, err := parseVMTarget(req.Target)
		if err != nil {
			return "", "", nil, err
		}
		return http.MethodPost, fmt.Sprintf("/api2/json/nodes/%s/qemu/%s/status/start", node, vmid), req.Params, nil
	case ActionStopVM:
		node, vmid, err := parseVMTarget(req.Target)
		if err != nil {
			return "", "", nil, err
		}
		return http.MethodPost, fmt.Sprintf("/api2/json/nodes/%s/qemu/%s/status/stop", node, vmid), req.Params, nil
	case ActionSnapshotVM:
		node, vmid, err := parseVMTarget(req.Target)
		if err != nil {
			return "", "", nil, err
		}
		return http.MethodPost, fmt.Sprintf("/api2/json/nodes/%s/qemu/%s/snapshot", node, vmid), req.Params, nil
	case ActionMigrateVM:
		node, vmid, err := parseVMTarget(req.Target)
		if err != nil {
			return "", "", nil, err
		}
		return http.MethodPost, fmt.Sprintf("/api2/json/nodes/%s/qemu/%s/migrate", node, vmid), req.Params, nil
	case ActionDeleteVM:
		node, vmid, err := parseVMTarget(req.Target)
		if err != nil {
			return "", "", nil, err
		}
		return http.MethodDelete, fmt.Sprintf("/api2/json/nodes/%s/qemu/%s", node, vmid), req.Params, nil
	case ActionStorageEdit:
		endpoint, method, params, err := customEndpointSpec(req.Params, http.MethodPut)
		return method, endpoint, params, err
	case ActionFirewallEdit:
		endpoint, method, params, err := customEndpointSpec(req.Params, http.MethodPost)
		return method, endpoint, params, err
	default:
		return "", "", nil, fmt.Errorf("unsupported action %q", req.Action)
	}
}

func customEndpointSpec(params map[string]any, defaultMethod string) (endpoint string, method string, body map[string]any, err error) {
	if params == nil {
		return "", "", nil, errors.New("params are required for this action")
	}
	rawEndpoint, ok := params["endpoint"].(string)
	if !ok || strings.TrimSpace(rawEndpoint) == "" {
		return "", "", nil, errors.New(`params.endpoint is required for this action`)
	}
	method = defaultMethod
	if rawMethod, ok := params["method"].(string); ok && strings.TrimSpace(rawMethod) != "" {
		method = strings.ToUpper(strings.TrimSpace(rawMethod))
	}
	if !strings.HasPrefix(rawEndpoint, "/api2/json/") {
		return "", "", nil, fmt.Errorf("invalid endpoint %q", rawEndpoint)
	}

	body = make(map[string]any, len(params))
	for k, v := range params {
		if k == "endpoint" || k == "method" {
			continue
		}
		body[k] = v
	}
	return rawEndpoint, method, body, nil
}

func parseVMTarget(target string) (node string, vmid string, err error) {
	parts := strings.Split(strings.TrimSpace(target), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid VM target %q; expected node/vmid", target)
	}
	return parts[0], parts[1], nil
}

func encodeParams(params map[string]any) io.Reader {
	if len(params) == 0 {
		return nil
	}
	values := url.Values{}
	for k, v := range params {
		if k == "" {
			continue
		}
		switch typed := v.(type) {
		case string:
			values.Set(k, typed)
		case bool:
			values.Set(k, strconv.FormatBool(typed))
		case int:
			values.Set(k, strconv.Itoa(typed))
		case int64:
			values.Set(k, strconv.FormatInt(typed, 10))
		case float64:
			values.Set(k, strconv.FormatFloat(typed, 'f', -1, 64))
		default:
			values.Set(k, fmt.Sprint(typed))
		}
	}
	return strings.NewReader(values.Encode())
}

func (c *APIClient) performRequest(env apiEnvironment, method, endpoint string, body io.Reader) ([]byte, error) {
	attempts := 1
	if method == http.MethodGet {
		attempts = c.readRetries
	}
	if attempts < 1 {
		attempts = 1
	}

	fullURL := env.baseURL + endpoint
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequestWithContext(context.Background(), method, fullURL, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", BuildTokenAuthHeader(env.tokenID, env.tokenSecret))
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < attempts {
				continue
			}
			return nil, &APIError{
				Method:   method,
				Endpoint: endpoint,
				Message:  err.Error(),
			}
		}

		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}
		if method == http.MethodGet && attempt < attempts && (resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusGatewayTimeout) {
			continue
		}

		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Method:     method,
			Endpoint:   endpoint,
			Message:    extractErrorMessage(respBody),
		}
	}
	return nil, &APIError{
		Method:   method,
		Endpoint: endpoint,
		Message:  "request failed after retries",
	}
}

func extractErrorMessage(respBody []byte) string {
	if len(respBody) == 0 {
		return "empty error response"
	}
	var envelope struct {
		Errors any    `json:"errors"`
		Data   any    `json:"data"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return strings.TrimSpace(string(respBody))
	}
	switch {
	case envelope.Error != "":
		return envelope.Error
	case envelope.Errors != nil:
		return fmt.Sprint(envelope.Errors)
	case envelope.Data != nil:
		return fmt.Sprint(envelope.Data)
	default:
		return strings.TrimSpace(string(respBody))
	}
}
