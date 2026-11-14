package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"qwq/internal/config"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestConsoleGatewayForwardsHTTPAndWebSocket(t *testing.T) {
	old := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = old })
	config.GlobalConfig.WebUser, config.GlobalConfig.WebPassword = "operator", "test-password"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "operator" || pass != "test-password" {
			t.Error("proxy dropped entrypoint credentials")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/test":
			json.NewEncoder(w).Encode(map[string]string{"method": r.Method, "host": r.Host, "origin": r.Header.Get("Origin")})
		case "/ws/chat":
			upgrader := websocket.Upgrader{}
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			kind, message, err := conn.ReadMessage()
			if err == nil {
				conn.WriteMessage(kind, message)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	_, port, err := net.SplitHostPort(strings.TrimPrefix(upstream.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEB_UI_PORT", port)
	t.Setenv("PORT", "8080")
	gs := NewEnhancedGatewayServer(":8080")
	services := gs.GetServiceStatus()
	if len(services) != 1 || services["web-ui"] == nil || services["web-ui"].URL != upstream.URL {
		t.Fatalf("gateway registered unavailable services or its own port: %+v", services)
	}
	proxy := httptest.NewServer(gs.GetGateway())
	defer proxy.Close()
	for _, tc := range []struct {
		path, password string
		want           int
	}{{"/api/test", "", 401}, {"/api/test", "wrong", 401}, {"/api/unknown", "test-password", 404}, {"/health", "", 200}} {
		r, _ := http.NewRequest(http.MethodGet, proxy.URL+tc.path, nil)
		if tc.password != "" {
			r.SetBasicAuth("operator", tc.password)
		}
		response, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != tc.want {
			t.Errorf("%s: got %d, want %d", tc.path, response.StatusCode, tc.want)
		}
	}
	r, _ := http.NewRequest(http.MethodPost, proxy.URL+"/api/test", nil)
	r.SetBasicAuth("operator", "test-password")
	r.Header.Set("Origin", proxy.URL)
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var echoed map[string]string
	if err := json.NewDecoder(response.Body).Decode(&echoed); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || echoed["method"] != "POST" || echoed["host"] != strings.TrimPrefix(proxy.URL, "http://") || echoed["origin"] != proxy.URL {
		t.Fatalf("write request changed while forwarding: %d %+v", response.StatusCode, echoed)
	}
	headers := http.Header{"Origin": []string{proxy.URL}, "Authorization": []string{"Basic " + base64.StdEncoding.EncodeToString([]byte("operator:test-password"))}}
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(proxy.URL, "http")+"/ws/chat", headers)
	if err != nil {
		t.Fatalf("WebSocket upgrade failed through logging/auth middleware: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	_, message, err := conn.ReadMessage()
	if err != nil || string(message) != "hello" {
		t.Fatalf("WebSocket forwarding failed: %q %v", message, err)
	}
}

func TestGatewayRejectsInvalidUpstreamPort(t *testing.T) {
	t.Setenv("WEB_UI_PORT", "not-a-port")
	if err := NewGatewayServer(":0").Run(context.Background()); err == nil {
		t.Fatal("invalid upstream port was accepted")
	}
}
