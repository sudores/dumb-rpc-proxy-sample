package proxy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/sudores/twt-test-task/pkg/proxy"
)

// newTestProxy creates a Proxy pointed at the given upstream URL.
func newTestProxy(t *testing.T, upstreamURL string) http.Handler {
	t.Helper()
	nop := zerolog.Nop()
	p, err := proxy.New(proxy.Config{
		UpstreamURL: upstreamURL,
		Timeout:     5 * time.Second,
	}, &nop)
	if err != nil {
		t.Fatalf("proxy.New: %v", err)
	}
	return p.Handler()
}

// fakeUpstream starts a test HTTP server that returns the given response for every POST.
func fakeUpstream(t *testing.T, statusCode int, response string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = strings.NewReader(response).WriteTo(w)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// --- health check ---

func TestHealthCheck(t *testing.T) {
	h := newTestProxy(t, "http://localhost")
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", http.NoBody)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("want status=ok, got %q", body["status"])
	}
}

// --- method enforcement ---

func TestOnlyPOSTAllowed(t *testing.T) {
	h := newTestProxy(t, "http://localhost")

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequestWithContext(context.Background(), method, "/", http.NoBody)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /: want 405, got %d", method, w.Code)
		}
	}
}

// --- single RPC request ---

func TestSingleRPCRequest(t *testing.T) {
	upstream := fakeUpstream(t, http.StatusOK, `{"jsonrpc":"2.0","id":1,"result":"0x1"}`)
	h := newTestProxy(t, upstream.URL)

	body := `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp proxy.RPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %+v", resp.Error)
	}
}

// --- batch RPC request ---

func TestBatchRPCRequest(t *testing.T) {
	upstreamResp := `[{"jsonrpc":"2.0","id":1,"result":"0x1"},{"jsonrpc":"2.0","id":2,"result":"0x2"}]`
	upstream := fakeUpstream(t, http.StatusOK, upstreamResp)
	h := newTestProxy(t, upstream.URL)

	body := `[{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]},` +
		`{"jsonrpc":"2.0","id":2,"method":"eth_chainId","params":[]}]`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp []proxy.RPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("want 2 responses, got %d", len(resp))
	}
}

// --- upstream returns an RPC-level error ---

func TestUpstreamRPCError(t *testing.T) {
	upstreamResp := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}`
	upstream := fakeUpstream(t, http.StatusOK, upstreamResp)
	h := newTestProxy(t, upstream.URL)

	body := `{"jsonrpc":"2.0","id":1,"method":"eth_nonExistent","params":[]}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp proxy.RPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected RPC error, got nil")
	}
	if resp.Error.Code != -32601 {
		t.Fatalf("want code -32601, got %d", resp.Error.Code)
	}
}

// --- upstream is unreachable ---

func TestUpstreamUnavailable(t *testing.T) {
	h := newTestProxy(t, "http://127.0.0.1:1")

	body := `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp proxy.RPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected RPC error for unavailable upstream, got nil")
	}
}

// --- upstream non-200 HTTP status is passed through ---

func TestUpstreamHTTPErrorPassthrough(t *testing.T) {
	upstream := fakeUpstream(t, http.StatusTooManyRequests, `{"error":"rate limited"}`)
	h := newTestProxy(t, upstream.URL)

	body := `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", w.Code)
	}
}

// --- integration test (skipped with -short) ---

func TestIntegration_EthBlockNumber(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	nop := zerolog.Nop()
	p, err := proxy.New(proxy.Config{
		UpstreamURL: "https://polygon.drpc.org",
		Timeout:     10 * time.Second,
	}, &nop)
	if err != nil {
		t.Fatalf("proxy.New: %v", err)
	}
	h := p.Handler()

	body := `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp proxy.RPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("RPC error: %+v", resp.Error)
	}
	if string(resp.Result) == "" || string(resp.Result) == "null" {
		t.Fatal("expected non-empty result")
	}
	t.Logf("current Polygon block: %s", resp.Result)
}
