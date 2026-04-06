package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/beacon-media/prism/internal/safedialer"
)

const (
	plexPinURL   = "https://plex.tv/api/v2/pins"
	plexClientID = "prism"
	plexProduct  = "Prism"
)

// RegisterPlexAuthRoutes registers the Plex pin-based auth endpoints.
func RegisterPlexAuthRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-plex-pin",
		Method:      http.MethodPost,
		Path:        "/api/v1/plex/pin",
		Summary:     "Create a Plex auth pin",
		Tags:        []string{"Plex"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body PlexPinResponse }, error) {
		pin, err := createPlexPin(ctx)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("failed to create Plex pin", err)
		}
		return &struct{ Body PlexPinResponse }{Body: pin}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "check-plex-pin",
		Method:      http.MethodGet,
		Path:        "/api/v1/plex/pin/{id}",
		Summary:     "Check if a Plex pin has been claimed",
		Tags:        []string{"Plex"},
	}, func(ctx context.Context, input *struct {
		ID int64 `path:"id"`
	}) (*struct{ Body PlexPinStatus }, error) {
		status, err := checkPlexPin(ctx, input.ID)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("failed to check Plex pin", err)
		}
		return &struct{ Body PlexPinStatus }{Body: status}, nil
	})
}

// PlexPinResponse is returned when creating a new Plex auth pin.
type PlexPinResponse struct {
	ID      int64  `json:"id"`
	Code    string `json:"code"`
	AuthURL string `json:"auth_url"`
}

// PlexPinStatus is returned when checking a Plex pin.
type PlexPinStatus struct {
	Claimed bool   `json:"claimed"`
	Token   string `json:"token,omitempty"`
}

func plexHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: safedialer.Transport(),
	}
}

func createPlexPin(ctx context.Context) (PlexPinResponse, error) {
	body := strings.NewReader("strong=true")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, plexPinURL, body)
	if err != nil {
		return PlexPinResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", plexClientID)
	req.Header.Set("X-Plex-Product", plexProduct)

	resp, err := plexHTTPClient().Do(req)
	if err != nil {
		return PlexPinResponse{}, fmt.Errorf("requesting pin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return PlexPinResponse{}, fmt.Errorf("plex returned %d: %s", resp.StatusCode, string(respBody))
	}

	var pin struct {
		ID   int64  `json:"id"`
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pin); err != nil {
		return PlexPinResponse{}, fmt.Errorf("decoding pin: %w", err)
	}

	authURL := fmt.Sprintf(
		"https://app.plex.tv/auth#?clientID=%s&code=%s&context%%5Bdevice%%5D%%5Bproduct%%5D=%s",
		plexClientID, pin.Code, plexProduct,
	)

	return PlexPinResponse{
		ID:      pin.ID,
		Code:    pin.Code,
		AuthURL: authURL,
	}, nil
}

func checkPlexPin(ctx context.Context, pinID int64) (PlexPinStatus, error) {
	url := fmt.Sprintf("%s/%d", plexPinURL, pinID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PlexPinStatus{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Client-Identifier", plexClientID)
	req.Header.Set("X-Plex-Product", plexProduct)

	resp, err := plexHTTPClient().Do(req)
	if err != nil {
		return PlexPinStatus{}, fmt.Errorf("checking pin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return PlexPinStatus{}, fmt.Errorf("plex returned %d", resp.StatusCode)
	}

	var pin struct {
		AuthToken string `json:"authToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pin); err != nil {
		return PlexPinStatus{}, fmt.Errorf("decoding pin status: %w", err)
	}

	return PlexPinStatus{
		Claimed: pin.AuthToken != "",
		Token:   pin.AuthToken,
	}, nil
}
