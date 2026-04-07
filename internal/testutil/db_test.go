package testutil

import (
	"context"
	"testing"

	dbsqlite "github.com/beacon-stack/prism/internal/db/generated/sqlite"
)

func TestNewTestDB_ReturnsWorkingQueries(t *testing.T) {
	q := NewTestDB(t)

	now := "2026-01-01T00:00:00Z"
	params := dbsqlite.CreateQualityProfileParams{
		ID:             "test-profile-id",
		Name:           "HD Quality",
		CutoffJson:     `{"id":4,"name":"HDTV-720p"}`,
		QualitiesJson:  `[{"id":4,"name":"HDTV-720p","allowed":true}]`,
		UpgradeAllowed: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	created, err := q.CreateQualityProfile(context.Background(), params)
	if err != nil {
		t.Fatalf("CreateQualityProfile() error = %v", err)
	}
	if created.ID != params.ID {
		t.Errorf("created.ID = %q, want %q", created.ID, params.ID)
	}
	if created.Name != params.Name {
		t.Errorf("created.Name = %q, want %q", created.Name, params.Name)
	}

	fetched, err := q.GetQualityProfile(context.Background(), params.ID)
	if err != nil {
		t.Fatalf("GetQualityProfile() error = %v", err)
	}
	if fetched.ID != params.ID {
		t.Errorf("fetched.ID = %q, want %q", fetched.ID, params.ID)
	}
	if fetched.Name != params.Name {
		t.Errorf("fetched.Name = %q, want %q", fetched.Name, params.Name)
	}
	if fetched.UpgradeAllowed != 1 {
		t.Errorf("fetched.UpgradeAllowed = %d, want 1", fetched.UpgradeAllowed)
	}
}

func TestNewTestDBWithSQL_IsolatedFromNewTestDB(t *testing.T) {
	// Each helper call creates a fully independent in-memory database.
	q1 := NewTestDB(t)
	q2, _ := NewTestDBWithSQL(t)

	now := "2026-01-01T00:00:00Z"
	params := dbsqlite.CreateQualityProfileParams{
		ID:            "isolation-check",
		Name:          "Test Profile",
		CutoffJson:    `{}`,
		QualitiesJson: `[]`,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Insert into q1 only.
	if _, err := q1.CreateQualityProfile(context.Background(), params); err != nil {
		t.Fatalf("CreateQualityProfile on q1: %v", err)
	}

	// q2 must not see q1's data — they share no state.
	profiles, err := q2.ListQualityProfiles(context.Background())
	if err != nil {
		t.Fatalf("ListQualityProfiles on q2: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("q2 has %d profiles, want 0 (databases should be isolated)", len(profiles))
	}
}
