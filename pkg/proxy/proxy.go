package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
)

// RPCRequest represents a single JSON-RPC 2.0 request.
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RPCResponse represents a single JSON-RPC 2.0 response.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Config holds proxy configuration.
type Config struct {
	UpstreamURL string
	Timeout     time.Duration
}

// Proxy forwards JSON-RPC requests to an upstream endpoint.
type Proxy struct {
	upstream   *url.URL
	httpClient *http.Client
	logger     *zerolog.Logger
}

// New creates a new Proxy from the given config.
func New(cfg Config, logger *zerolog.Logger) (*Proxy, error) {
	u, err := url.Parse(cfg.UpstreamURL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL: %w", err)
	}

	return &Proxy{
		upstream: u,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger,
	}, nil
}

// Handler returns an http.Handler that serves the proxy and health check.
func (p *Proxy) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", p.handleHealth)
	mux.HandleFunc("/", p.handleRPC)
	return mux
}

func (p *Proxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (p *Proxy) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10 MB limit
	if err != nil {
		p.writeRPCError(w, nil, -32700, "failed to read request body")
		return
	}
	defer r.Body.Close()

	start := time.Now()
	method := p.peekMethod(body)

	resp, statusCode, err := p.forward(r.Context(), body)
	latency := time.Since(start)

	if err != nil {
		p.logger.Error().Str("method", method).Dur("latency", latency).Err(err).Msg("upstream error")
		p.writeRPCError(w, nil, -32603, "upstream error: "+err.Error())
		return
	}

	p.logger.Info().Str("method", method).Int("status", statusCode).Dur("latency", latency).Msg("proxied")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(resp)
}

// forward sends body to the upstream and returns the raw response bytes + status code.
func (p *Proxy) forward(ctx context.Context, body []byte) (data []byte, statusCode int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.upstream.String(), bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return data, resp.StatusCode, nil
}

// peekMethod extracts the RPC method name for logging without failing on batch requests.
func (p *Proxy) peekMethod(body []byte) string {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return ""
	}

	if body[0] == '[' {
		var batch []RPCRequest
		if err := json.Unmarshal(body, &batch); err == nil && len(batch) > 0 {
			if len(batch) == 1 {
				return batch[0].Method
			}
			return fmt.Sprintf("batch(%d)", len(batch))
		}
		return "batch"
	}

	var req RPCRequest
	if err := json.Unmarshal(body, &req); err == nil {
		return req.Method
	}

	return ""
}

func (p *Proxy) writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	resp := RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors still return HTTP 200
	_ = json.NewEncoder(w).Encode(resp)
}
