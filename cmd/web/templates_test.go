package main

import (
	"testing"
	"time"

	"github.com/mos-nabih/lets-ride/internal/assert"
)

func TestDepartureDate(t *testing.T) {
	tm := time.Date(2024, 3, 17, 10, 15, 0, 0, time.FixedZone("CET", 60*60))
	want := tm.In(time.Local).Format("02 Jan 2006 at 15:04 MST")
	assert.Equal(t, departureDate(tm), want)
}
