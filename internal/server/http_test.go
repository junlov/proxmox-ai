package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/junlov/proxmox-ai/internal/actions"
	"github.com/junlov/proxmox-ai/internal/config"
	"github.com/junlov/proxmox-ai/internal/policy"
	"github.com/junlov/proxmox-ai/internal/proxmox"
)

type testClient struct {
	calls int32
}

func (c *testClient) Execute(req proxmox.ActionRequest) (proxmox.ActionResult, error) {
	atomic.AddInt32(&c.calls, 1)
	return proxmox.ActionResult{Status: "accepted", Message: "ok"}, nil
}

func newTestServer(client proxmox.Client) *Server {
	cfg := config.Config{
		ListenAddr: ":0",
		Environments: []config.Environment{
			{
				Name:           "home",
				BaseURL:        "https://proxmox.example.com",
				TokenID:        "root@pam!agent",
				TokenSecretEnv: "PVE_TEST_SECRET",
			},
		},
	}
	runner := actions.NewRunner(policy.NewEngine(), client, "")
	srv := New(cfg, runner)
	srv.authToken = "test-token"
	return srv
}

func newAuthedRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Actor-ID", "test-agent")
	return req
}

func TestPlanRejectsUnknownJSONField(t *testing.T) {
	s := newTestServer(&testClient{})
	req := newAuthedRequest(http.MethodPost, "/v1/actions/plan", `{"environment":"home","action":"read_vm","target":"vm/101","unknown":true}`)
	rr := httptest.NewRecorder()

	s.plan(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestApplyRejectsUnknownJSONField(t *testing.T) {
	s := newTestServer(&testClient{})
	req := newAuthedRequest(http.MethodPost, "/v1/actions/apply", `{"environment":"home","action":"start_vm","target":"vm/101","extra":"x"}`)
	rr := httptest.NewRecorder()

	s.apply(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPlanRejectsTrailingJSON(t *testing.T) {
	s := newTestServer(&testClient{})
	req := newAuthedRequest(http.MethodPost, "/v1/actions/plan", `{"environment":"home","action":"read_vm","target":"vm/101"}{"extra":true}`)
	rr := httptest.NewRecorder()

	s.plan(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestPlanAcceptsValidJSON(t *testing.T) {
	s := newTestServer(&testClient{})
	req := newAuthedRequest(http.MethodPost, "/v1/actions/plan", `{"environment":"home","action":"read_vm","target":"vm/101"}`)
	rr := httptest.NewRecorder()

	s.plan(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestApplyIdempotencyReplaysAndPreventsDuplicateExecution(t *testing.T) {
	client := &testClient{}
	s := newTestServer(client)
	body := `{"environment":"home","action":"start_vm","target":"vm/101"}`

	req1 := newAuthedRequest(http.MethodPost, "/v1/actions/apply", body)
	req1.Header.Set("Idempotency-Key", "apply-key-1")
	rr1 := httptest.NewRecorder()
	s.apply(rr1, req1)

	req2 := newAuthedRequest(http.MethodPost, "/v1/actions/apply", body)
	req2.Header.Set("Idempotency-Key", "apply-key-1")
	rr2 := httptest.NewRecorder()
	s.apply(rr2, req2)

	if rr1.Code != http.StatusOK || rr2.Code != http.StatusOK {
		t.Fatalf("expected both responses to be 200, got %d and %d", rr1.Code, rr2.Code)
	}
	if rr1.Body.String() != rr2.Body.String() {
		t.Fatalf("expected identical replay body, got %q and %q", rr1.Body.String(), rr2.Body.String())
	}
	if got := atomic.LoadInt32(&client.calls); got != 1 {
		t.Fatalf("expected single execution call, got %d", got)
	}
}

func TestApplyIdempotencyRejectsDifferentPayloadForSameKey(t *testing.T) {
	client := &testClient{}
	s := newTestServer(client)

	req1 := newAuthedRequest(http.MethodPost, "/v1/actions/apply", `{"environment":"home","action":"start_vm","target":"vm/101"}`)
	req1.Header.Set("Idempotency-Key", "apply-key-2")
	rr1 := httptest.NewRecorder()
	s.apply(rr1, req1)

	req2 := newAuthedRequest(http.MethodPost, "/v1/actions/apply", `{"environment":"home","action":"start_vm","target":"vm/102"}`)
	req2.Header.Set("Idempotency-Key", "apply-key-2")
	rr2 := httptest.NewRecorder()
	s.apply(rr2, req2)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first response to be 200, got %d", rr1.Code)
	}
	if rr2.Code != http.StatusConflict {
		t.Fatalf("expected second response to be 409, got %d (%s)", rr2.Code, rr2.Body.String())
	}
	if got := atomic.LoadInt32(&client.calls); got != 1 {
		t.Fatalf("expected one execution call after conflict response, got %d", got)
	}
}

func TestPlanRequiresBearerAuth(t *testing.T) {
	s := newTestServer(&testClient{})
	req := httptest.NewRequest(http.MethodPost, "/v1/actions/plan", strings.NewReader(`{"environment":"home","action":"read_vm","target":"vm/101"}`))
	rr := httptest.NewRecorder()

	s.plan(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing auth, got %d", rr.Code)
	}
}
