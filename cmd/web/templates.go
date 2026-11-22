package main

import (
	"html/template"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/mos-nabih/lets-ride/assets"
	"github.com/mos-nabih/lets-ride/internal/models"
)

type templateData struct {
	Ride            models.Ride
	Rides           []models.Ride
	CurrentYear     int
	Form            any
	Flash           string
	IsAuthenticated bool
	CSRFToken       string
}

func departureDate(t time.Time) string {
	return t.Local().Format("02 Jan 2006 at 15:04 MST")
}

var functions = template.FuncMap{
	"departureDate": departureDate,
	"boardTimeZone": func() string { return time.Local.String() },
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := fs.Glob(assets.Files, "html/pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		patterns := []string{
			"html/base.tmpl",
			"html/partials/*.tmpl",
			page,
		}

		ts, err := template.New(name).Funcs(functions).ParseFS(assets.Files, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
