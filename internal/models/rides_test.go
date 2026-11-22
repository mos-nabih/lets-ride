package models

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/mos-nabih/lets-ride/internal/assert"
)

func TestRideModelLatest(t *testing.T) {
	db := newTestDB(t)
	rides := RideModel{DB: db}

	var dbNow time.Time
	if err := db.QueryRowContext(t.Context(), "SELECT UTC_TIMESTAMP()").Scan(&dbNow); err != nil {
		t.Fatal(err)
	}
	departure := dbNow.Add(time.Hour)
	var upcomingIDs []int
	for i := range 11 {
		id, err := rides.Insert(fmt.Sprintf("Ride %d", i), "Meet by the bridge.", departure)
		if err != nil {
			t.Fatal(err)
		}
		upcomingIDs = append(upcomingIDs, id)
	}
	expiredID, err := rides.Insert("Departed ride", "Meet by the bridge.", dbNow.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	// Make the newest ID the oldest post to distinguish post order from ID order.
	if _, err := db.ExecContext(t.Context(), "UPDATE rides SET created = ? WHERE id = ?", dbNow.Add(-time.Hour), upcomingIDs[10]); err != nil {
		t.Fatal(err)
	}

	latest, err := rides.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 10 {
		t.Fatalf("got %d upcoming rides, want 10", len(latest))
	}
	for i, ride := range latest {
		assert.Equal(t, ride.ID, upcomingIDs[9-i])
	}

	ride, err := rides.Get(upcomingIDs[10])
	assert.Nil(t, err)
	assert.True(t, ride.Departure.Equal(departure))
	_, err = rides.Get(expiredID)
	assert.True(t, errors.Is(err, ErrNoRecord))
}
