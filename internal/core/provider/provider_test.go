package provider

import (
	"context"
	"database/sql"
	"testing"

	db "github.com/beacon-stack/prism/internal/db/generated"
)

type fakeStore struct {
	rows map[string]string
}

func newFakeStore() *fakeStore { return &fakeStore{rows: make(map[string]string)} }

func (f *fakeStore) GetSetting(_ context.Context, key string) (string, error) {
	v, ok := f.rows[key]
	if !ok {
		return "", sql.ErrNoRows
	}
	return v, nil
}

func (f *fakeStore) SetSetting(_ context.Context, arg db.SetSettingParams) error {
	f.rows[arg.Key] = arg.Value
	return nil
}

func (f *fakeStore) DeleteSetting(_ context.Context, key string) error {
	delete(f.rows, key)
	return nil
}

func TestResolver_EffectiveKey_Override_WinsOverDefault(t *testing.T) {
	store := newFakeStore()
	r := NewResolver(store)

	if err := r.SetOverride(context.Background(), TMDB, "my-override"); err != nil {
		t.Fatal(err)
	}

	key, source, err := r.EffectiveKey(context.Background(), TMDB)
	if err != nil {
		t.Fatal(err)
	}
	if source != SourceOverride {
		t.Errorf("source = %q; want %q", source, SourceOverride)
	}
	if key != "my-override" {
		t.Errorf("key = %q; want my-override", key)
	}
}

func TestResolver_ClearOverride_RevertsToDefault(t *testing.T) {
	r := NewResolver(newFakeStore())
	ctx := context.Background()
	if err := r.SetOverride(ctx, TMDB, "will-be-cleared"); err != nil {
		t.Fatal(err)
	}
	if err := r.ClearOverride(ctx, TMDB); err != nil {
		t.Fatal(err)
	}
	has, err := r.HasOverride(ctx, TMDB)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected override to be absent after clear")
	}
}

func TestResolver_TraktWorks(t *testing.T) {
	r := NewResolver(newFakeStore())
	ctx := context.Background()
	if err := r.SetOverride(ctx, Trakt, "trakt-override"); err != nil {
		t.Fatal(err)
	}
	key, source, err := r.EffectiveKey(ctx, Trakt)
	if err != nil {
		t.Fatal(err)
	}
	if source != SourceOverride {
		t.Errorf("source = %q; want %q", source, SourceOverride)
	}
	if key != "trakt-override" {
		t.Errorf("key = %q; want trakt-override", key)
	}
}

func TestResolver_SetOverride_TrimsWhitespace(t *testing.T) {
	r := NewResolver(newFakeStore())
	ctx := context.Background()
	if err := r.SetOverride(ctx, TMDB, "  key  \n"); err != nil {
		t.Fatal(err)
	}
	key, _, _ := r.EffectiveKey(ctx, TMDB)
	if key != "key" {
		t.Errorf("key = %q; want 'key' (whitespace trimmed)", key)
	}
}

func TestResolver_SetOverride_RejectsEmpty(t *testing.T) {
	r := NewResolver(newFakeStore())
	if err := r.SetOverride(context.Background(), TMDB, "   "); err == nil {
		t.Error("expected error when override value is empty after trim")
	}
}

func TestResolver_UnknownProvider_Rejected(t *testing.T) {
	r := NewResolver(newFakeStore())
	ctx := context.Background()
	if _, _, err := r.EffectiveKey(ctx, "bogus"); err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestResolver_Preview_RedactsOverride(t *testing.T) {
	r := NewResolver(newFakeStore())
	ctx := context.Background()
	if err := r.SetOverride(ctx, TMDB, "abcdef1234567890xyz"); err != nil {
		t.Fatal(err)
	}
	preview, _, err := r.Preview(ctx, TMDB)
	if err != nil {
		t.Fatal(err)
	}
	want := "••••••••••••••••xyz"
	if preview != want {
		t.Errorf("preview = %q; want %q", preview, want)
	}
}
