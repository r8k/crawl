package api

// module deps
import "bytes"
import "context"
import "testing"
import "net/http"
import "io/ioutil"
import "encoding/json"
import "net/http/httptest"
import "github.com/labstack/echo"
import "github.com/r8k/crawl/crawler"

// TestServer helps in generating
// test servers that can be re-used
type TestServer struct {
	mux     *echo.Echo
	handler *Handler
}

func NewTestServer() *TestServer {
	// create api handler
	handler := &Handler{
		Crawler: crawler.New(),
	}

	e := echo.New()
	handler.Crawler.Logger = e.Logger
	e.Logger.SetOutput(ioutil.Discard)

	return &TestServer{
		mux:     e,
		handler: handler,
	}
}

func (t *TestServer) Close() {
	defer t.handler.Crawler.Close()
	defer t.mux.Shutdown(context.TODO())
}

// test HasContentType
func TestHasContentType(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	mimetype := "application/json"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("Content-Type", mimetype)
	has, err := HasContentType(req, mimetype)

	if err != nil || !has {
		t.Fatalf("expected true, got: %v, err: %v\n", has, err.Error())
	}
}

// test 404 handler
func TestNotFoundHandler(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	server := NewTestServer()
	defer server.Close()
	server.mux.GET("/", echo.NotFoundHandler)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("Got Non-404 response: %d\n", resp.Code)
	}
}

// test 405 handler
func TestMethodNotAllowedHandler(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	server := NewTestServer()
	defer server.Close()
	server.mux.GET("/", echo.MethodNotAllowedHandler)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("Got Non-405 response: %d\n", resp.Code)
	}
}

// test 202 handler
func TestGoodRequestCreateDomain(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	server := NewTestServer()
	defer server.Close()
	server.mux.POST("/", server.handler.CreateDomainHandler)

	domain := &Domain{Domain: "https://cloudflare.com", Depth: 1}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(domain)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", buf)
	req.Header.Add("Content-Type", "application/json")
	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Fatalf("Got Non-202 response: %d\n", resp.Code)
	}
}

// test 400 handler
func TestBadRequest1CreateDomain(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	server := NewTestServer()
	defer server.Close()
	server.mux.POST("/", server.handler.CreateDomainHandler)

	// missing domain
	domain := &Domain{Depth: 1}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(domain)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", buf)
	req.Header.Add("Content-Type", "application/json")
	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Got Non-400 response: %d\n", resp.Code)
	}
}

// test 400 handler
func TestBadRequest2CreateDomain(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	server := NewTestServer()
	defer server.Close()
	server.mux.POST("/", server.handler.CreateDomainHandler)

	// invalid scheme : htt
	domain := &Domain{Domain: "htt://cloudflare.com", Depth: 1}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(domain)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", buf)
	req.Header.Add("Content-Type", "application/json")
	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("Got Non-400 response: %d\n", resp.Code)
	}
}

// test 415 handler
func TestUnSupportedMediaTypeRequestCreateDomain(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	server := NewTestServer()
	defer server.Close()
	server.mux.POST("/", server.handler.CreateDomainHandler)

	domain := &Domain{Domain: "https://cloudflare.com", Depth: 1}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(domain)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", buf)
	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("Got Non-415 response: %d\n", resp.Code)
	}
}

// test 204 handler
func TestGoodRequestGetDomainHandler(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	server := NewTestServer()
	defer server.Close()
	server.mux.POST("/domains", server.handler.CreateDomainHandler)
	server.mux.GET("/domains/:domain", server.handler.GetDomainHandler)

	domain := &Domain{Domain: "https://cloudflare.com", Depth: 1}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(domain)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/domains", buf)
	req.Header.Add("Content-Type", "application/json")
	server.mux.ServeHTTP(resp, req)

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/domains/https%3A%2F%2Fcloudflare.com", nil)
	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("Got Non-204 response: %d\n", resp.Code)
	}
}

// test 404 handler
func TestBadRequestGetDomainHandler(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	server := NewTestServer()
	defer server.Close()
	server.mux.POST("/domains", server.handler.CreateDomainHandler)
	server.mux.GET("/domains/:domain", server.handler.GetDomainHandler)

	domain := &Domain{Domain: "https://cloudflare.com", Depth: 1}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(domain)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/domains", buf)
	req.Header.Add("Content-Type", "application/json")
	server.mux.ServeHTTP(resp, req)

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/domains/https://cloudflare.com", nil)
	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("Got Non-404 response: %d\n", resp.Code)
	}
}

// test 200 handler
func TestGoodRequestGetDomainStatusHandler(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	server := NewTestServer()
	defer server.Close()
	server.mux.POST("/domains", server.handler.CreateDomainHandler)
	server.mux.GET("/domains/:domain/status", server.handler.GetDomainStatusHandler)

	domain := &Domain{Domain: "https://cloudflare.com", Depth: 1}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(domain)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/domains", buf)
	req.Header.Add("Content-Type", "application/json")
	server.mux.ServeHTTP(resp, req)

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/domains/https%3A%2F%2Fcloudflare.com/status", nil)
	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("Got Non-200 response: %d\n", resp.Code)
	}
}

// test 404 handler
func TestBadRequestGetDomainStatusHandler(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	// create test server
	server := NewTestServer()
	defer server.Close()
	server.mux.GET("/domains/:domain/status", server.handler.GetDomainStatusHandler)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/domains/https%3A%2F%2Fcloudflare.com/status", nil)
	server.mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("Got Non-200 response: %d\n", resp.Code)
	}
}
