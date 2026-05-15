package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	contentTypeHTML = "text/html; charset=utf-8"
	hdrCT          = "Content-Type"
)

// callbackResult holds the authorization code received from the OAuth provider.
type callbackResult struct {
	Code  string
	State string
	Error string
}

// callbackServer is a one-shot local HTTP server that listens for the OAuth callback.
type callbackServer struct {
	listener net.Listener
	server   *http.Server
	result   chan callbackResult
	once     sync.Once
}

// newCallbackServer creates a callback server bound to 127.0.0.1 on a random port.
func newCallbackServer() (*callbackServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("binding local server: %w", err)
	}

	s := &callbackServer{
		listener: listener,
		result:   make(chan callbackResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s, nil
}

// Port returns the port the server is listening on.
func (s *callbackServer) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// Start begins serving in a goroutine. The server auto-shuts-down after the first callback.
func (s *callbackServer) Start() {
	go func() {
		if err := s.server.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			s.once.Do(func() {
				s.result <- callbackResult{Error: fmt.Sprintf("server error: %v", err)}
			})
		}
	}()
}

// WaitForCallback blocks until the OAuth callback is received or the context expires.
func (s *callbackServer) WaitForCallback(ctx context.Context) (callbackResult, error) {
	select {
	case result := <-s.result:
		// Gracefully shut down the server
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
		return result, nil
	case <-ctx.Done():
		_ = s.server.Shutdown(context.Background())
		return callbackResult{}, fmt.Errorf("timed out waiting for authentication (press Ctrl+C to cancel)")
	}
}

// handleCallback processes the OAuth redirect. It extracts the authorization code,
// responds with a success page, and signals the result channel.
func (s *callbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Check for error from OAuth provider
	if errParam := query.Get("error"); errParam != "" {
		errDesc := query.Get("error_description")
		if errDesc == "" {
			errDesc = errParam
		}
		s.once.Do(func() {
			s.result <- callbackResult{Error: errDesc}
		})

		w.Header().Set(hdrCT, contentTypeHTML)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, authErrorPage, errDesc)
		return
	}

	code := query.Get("code")
	state := query.Get("state")

	if code == "" {
		s.once.Do(func() {
			s.result <- callbackResult{Error: "no authorization code in callback"}
		})
		w.Header().Set(hdrCT, contentTypeHTML)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, authErrorPage, "no authorization code received")
		return
	}

	s.once.Do(func() {
		s.result <- callbackResult{Code: code, State: state}
	})

	w.Header().Set(hdrCT, contentTypeHTML)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, authSuccessPage)
}

// authSuccessPage is returned to the browser after successful authentication.
const authSuccessPage = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>VecTrade CLI — Authenticated</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background: #0F172A;
      color: #F1F5F9;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
      padding: 24px;
    }
    .card {
      text-align: center;
      max-width: 420px;
    }
    .icon {
      width: 64px;
      height: 64px;
      border-radius: 16px;
      background: rgba(16,185,129,0.15);
      display: flex;
      align-items: center;
      justify-content: center;
      margin: 0 auto 24px;
      font-size: 28px;
    }
    h1 { font-size: 24px; font-weight: 700; margin-bottom: 8px; }
    p { font-size: 14px; color: #94A3B8; line-height: 1.6; }
    .hint {
      margin-top: 20px;
      padding: 12px 16px;
      background: #1E293B;
      border-radius: 8px;
      font-family: "SF Mono", "Fira Code", monospace;
      font-size: 13px;
      color: #60A5FA;
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon">✓</div>
    <h1>Authentication Successful</h1>
    <p>You can close this tab and return to your terminal.</p>
    <div class="hint">vectrade auth status</div>
  </div>
  <script>setTimeout(function(){ window.close(); }, 3000);</script>
</body>
</html>`

// authErrorPage is returned when authentication fails.
const authErrorPage = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>VecTrade CLI — Authentication Failed</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background: #0F172A;
      color: #F1F5F9;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
      padding: 24px;
    }
    .card {
      text-align: center;
      max-width: 420px;
    }
    .icon {
      width: 64px;
      height: 64px;
      border-radius: 16px;
      background: rgba(239,68,68,0.15);
      display: flex;
      align-items: center;
      justify-content: center;
      margin: 0 auto 24px;
      font-size: 28px;
    }
    h1 { font-size: 24px; font-weight: 700; margin-bottom: 8px; }
    p { font-size: 14px; color: #94A3B8; line-height: 1.6; }
    .hint {
      margin-top: 20px;
      padding: 12px 16px;
      background: #1E293B;
      border-radius: 8px;
      font-family: "SF Mono", "Fira Code", monospace;
      font-size: 13px;
      color: #60A5FA;
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon">✗</div>
    <h1>Authentication Failed</h1>
    <p>%s</p>
    <div class="hint">vectrade auth login --provider google</div>
  </div>
</body>
</html>`
