package models

import (
	"testing"

	"github.com/mos-nabih/lets-ride/internal/assert"
)

func TestUserModelExists(t *testing.T) {
	tests := []struct {
		name   string
		userID int
		want   bool
	}{
		{"Valid ID", 1, true},
		{"Zero ID", 0, false},
		{"Non-existent ID", 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			m := UserModel{DB: db}
			exists, err := m.Exists(tt.userID)
			assert.Equal(t, exists, tt.want)
			assert.Nil(t, err)
		})
	}
}
