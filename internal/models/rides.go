package models

import (
	"database/sql"
	"errors"
	"time"
)

type Ride struct {
	ID          int
	Title       string
	Description string
	Departure   time.Time
	Created     time.Time
}

type RideModel struct {
	DB *sql.DB
}

func (m *RideModel) Insert(title string, description string, departure time.Time) (int, error) {
	stmt := `INSERT INTO rides (title, description, created, departure)
	VALUES (?, ?, UTC_TIMESTAMP(), ?)`

	result, err := m.DB.Exec(stmt, title, description, departure.UTC())

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()

	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *RideModel) Get(id int) (Ride, error) {
	stmt := `SELECT id, title, description, created, departure
	FROM rides WHERE departure > UTC_TIMESTAMP() AND id = ?`

	row := m.DB.QueryRow(stmt, id)

	var ride Ride

	err := row.Scan(&ride.ID, &ride.Title, &ride.Description, &ride.Created, &ride.Departure)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Ride{}, ErrNoRecord
		default:
			return Ride{}, err

		}
	}
	return ride, nil
}

func (m *RideModel) Latest() ([]Ride, error) {
	stmt := `SELECT id, title, description, created, departure
	FROM rides WHERE departure > UTC_TIMESTAMP() ORDER BY created DESC, id DESC LIMIT 10`

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var rides []Ride

	for rows.Next() {
		var ride Ride

		err = rows.Scan(&ride.ID, &ride.Title, &ride.Description, &ride.Created, &ride.Departure)
		if err != nil {
			return nil, err
		}
		rides = append(rides, ride)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rides, nil
}
