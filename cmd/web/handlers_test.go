package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mos-nabih/lets-ride/internal/assert"
)

func TestHome(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())

	res := ts.get(t, "/")
	assert.Equal(t, res.status, http.StatusOK)
	assert.True(t, strings.Contains(res.body, "<title>Ride board · Let’s Ride</title>"))
	assert.True(t, strings.Contains(res.body, "<a href='/ride/view/1'>Lakeside loop</a>"))
	assert.True(t, strings.Contains(res.body, "<time datetime='2099-03-17T10:15:00Z'>"))

	res = ts.get(t, "/static/css/main.css")
	assert.Equal(t, res.status, http.StatusOK)
	assert.True(t, strings.HasPrefix(res.headers.Get("Content-Type"), "text/css"))
}

func TestRideView(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	tests := []struct {
		name       string
		urlPath    string
		wantStatus int
		wantBody   string
	}{
		{"Valid ID", "/ride/view/1", http.StatusOK, "Meet at the north entrance."},
		{"Non-existent ID", "/ride/view/2", http.StatusNotFound, "404 page not found"},
		{"Negative ID", "/ride/view/-1", http.StatusNotFound, "404 page not found"},
		{"Zero ID", "/ride/view/0", http.StatusNotFound, "404 page not found"},
		{"Decimal ID", "/ride/view/1.23", http.StatusNotFound, "404 page not found"},
		{"String ID", "/ride/view/foo", http.StatusNotFound, "404 page not found"},
		{"Empty ID", "/ride/view/", http.StatusNotFound, "404 page not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts.resetClientCookieJar(t)
			res := ts.get(t, tt.urlPath)
			assert.Equal(t, res.status, tt.wantStatus)
			assert.True(t, strings.Contains(res.body, tt.wantBody))
		})
	}
}

func TestUserSignup(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	tests := []struct {
		name              string
		userName          string
		userEmail         string
		userPassword      string
		useValidCSRFToken bool
		wantStatus        int
		wantBody          string
	}{
		{"Valid submission", "Bob", "bob@example.com", "validPa$$word", true, http.StatusSeeOther, ""},
		{"Invalid CSRF token", "Bob", "bob@example.com", "validPa$$word", false, http.StatusBadRequest, "Bad Request"},
		{"Empty name", "", "bob@example.com", "validPa$$word", true, http.StatusUnprocessableEntity, "This field cannot be blank"},
		{"Empty email", "Bob", "", "validPa$$word", true, http.StatusUnprocessableEntity, "This field cannot be blank"},
		{"Empty password", "Bob", "bob@example.com", "", true, http.StatusUnprocessableEntity, "This field cannot be blank"},
		{"Invalid email", "Bob", "bob@example.", "validPa$$word", true, http.StatusUnprocessableEntity, "This field must be a valid email address"},
		{"Short password", "Bob", "bob@example.com", "pa$$", true, http.StatusUnprocessableEntity, "This field must be at least 8 characters long"},
		{"Long password", "Bob", "bob@example.com", strings.Repeat("a", 73), true, http.StatusUnprocessableEntity, "This field must not be more than 72 bytes long"},
		{"Duplicate email", "Bob", "dupe@example.com", "validPa$$word", true, http.StatusUnprocessableEntity, "Email address is already in use"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts.resetClientCookieJar(t)
			res := ts.get(t, "/user/signup")
			assert.Equal(t, res.status, http.StatusOK)
			form := url.Values{"name": {tt.userName}, "email": {tt.userEmail}, "password": {tt.userPassword}}
			if tt.useValidCSRFToken {
				form.Set("csrf_token", extractCSRFToken(t, res.body))
			}
			res = ts.postForm(t, "/user/signup", form)
			assert.Equal(t, res.status, tt.wantStatus)
			assert.True(t, strings.Contains(res.body, tt.wantBody))
			if tt.wantStatus == http.StatusUnprocessableEntity {
				assert.True(t, strings.Contains(res.body, "<form action='/user/signup' method='POST' class='page-form' novalidate>"))
			}
			if tt.wantStatus == http.StatusSeeOther {
				assert.Equal(t, res.headers.Get("Location"), "/user/login")
				res = ts.get(t, "/user/login")
				assert.True(t, strings.Contains(res.body, "Your signup was successful. Please log in."))
			}
		})
	}
}

func TestRideCreate(t *testing.T) {
	app := newTestApplication(t)
	ts := newTestServer(t, app.routes())
	const createPath = "/ride/create"

	res := ts.get(t, createPath)
	assert.Equal(t, res.status, http.StatusSeeOther)
	assert.Equal(t, res.headers.Get("Location"), "/user/login")

	res = ts.get(t, "/user/login")
	login := url.Values{
		"email":      {mockUserEmail},
		"password":   {mockUserPassword},
		"csrf_token": {extractCSRFToken(t, res.body)},
	}
	res = ts.postForm(t, "/user/login", login)
	assert.Equal(t, res.status, http.StatusSeeOther)
	assert.Equal(t, res.headers.Get("Location"), createPath)

	res = ts.get(t, createPath)
	assert.Equal(t, res.status, http.StatusOK)
	form := url.Values{
		"title":       {"Weekend loop"},
		"description": {"Meet by the bridge."},
		"csrf_token":  {extractCSRFToken(t, res.body)},
	}

	for _, tt := range []struct {
		name      string
		departure string
		message   string
	}{
		{"Invalid time", "invalid", "Enter a valid departure date and time"},
		{"Past time", "2020-01-01T12:00", "Departure must be in the future"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			form.Set("departure", tt.departure)
			res := ts.postForm(t, createPath, form)
			assert.Equal(t, res.status, http.StatusUnprocessableEntity)
			assert.True(t, strings.Contains(res.body, tt.message))
			assert.True(t, strings.Contains(res.body, "Weekend loop"))
		})
	}

	form.Set("departure", time.Now().AddDate(0, 0, 1).Format("2006-01-02T15:04"))
	res = ts.postForm(t, createPath, form)
	assert.Equal(t, res.status, http.StatusSeeOther)
	assert.Equal(t, res.headers.Get("Location"), "/ride/view/3")
}
