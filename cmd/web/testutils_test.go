package main

import (
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	"github.com/mos-nabih/lets-ride/internal/models"
)

const testServerBaseURL = "https://testserver"

func newTestApplication(t *testing.T) *application {
	t.Helper()
	templateCache, err := newTemplateCache()
	if err != nil {
		t.Fatal(err)
	}
	sessionManager := scs.New()
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.Cookie.Secure = true
	db := newMockDB(t)
	return &application{
		logger:         slog.New(slog.DiscardHandler),
		rides:          &models.RideModel{DB: db},
		users:          &models.UserModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    form.NewDecoder(),
		sessionManager: sessionManager,
	}
}

type testServer struct{ *httptest.Server }

type testResponse struct {
	status  int
	headers http.Header
	cookies []*http.Cookie
	body    string
}

func newTestServer(t *testing.T, h http.Handler) *testServer {
	t.Helper()
	ts := &testServer{Server: httptest.NewTestServer(t, h)}
	ts.resetClientCookieJar(t)
	ts.Client().CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return ts
}

func (ts *testServer) resetClientCookieJar(t *testing.T) {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	ts.Client().Jar = jar
}

func (ts *testServer) get(t *testing.T, urlPath string) testResponse {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, testServerBaseURL+urlPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	return ts.do(t, req)
}

func (ts *testServer) postForm(t *testing.T, urlPath string, form url.Values) testResponse {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, testServerBaseURL+urlPath, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	return ts.do(t, req)
}

func (ts *testServer) do(t *testing.T, req *http.Request) testResponse {
	t.Helper()
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return testResponse{status: res.StatusCode, headers: res.Header, cookies: res.Cookies(), body: strings.TrimSpace(string(body))}
}

func extractCSRFToken(t *testing.T, body string) string {
	t.Helper()
	matches := regexp.MustCompile(`<input type='hidden' name='csrf_token' value='([^']+)'>`).FindStringSubmatch(body)
	if len(matches) != 2 {
		t.Fatal("no csrf token found in body")
	}
	return html.UnescapeString(matches[1])
}
