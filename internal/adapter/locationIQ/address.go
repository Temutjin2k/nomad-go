package locationIQ

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

var (
	ErrLocationNotFound = fmt.Errorf("location not found")
)

type LocationIQClient struct {
	apiKey string
}

func New(apiKey string) *LocationIQClient {
	return &LocationIQClient{
		apiKey: apiKey,
	}
}

var domain = "https://us1.locationiq.com"

type AddressPayload struct {
	Address string `json:"display_name"`
}

func (c *LocationIQClient) GetAddress(ctx context.Context, longitude, latitude float64) (string, error) {
	const op = "LocationIQClient.GetAddress"

	url := fmt.Sprintf("%s/v1/reverse?key=%s&lat=%f&lon=%f&format=json", domain, c.apiKey, latitude, longitude)

	resp, err := http.Get(url)
	if err != nil {
		ctx = wrap.WithAction(ctx, types.ActionExternalServiceFailed)
		return "", wrap.Error(ctx, fmt.Errorf("%s: failed to make request to LocationIQ: %w", op, err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ctx = wrap.WithAction(ctx, types.ActionExternalServiceFailed)
		return "", wrap.Error(ctx, fmt.Errorf("%s: unexpected response status %d", op, resp.StatusCode))
	}

	var payload AddressPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		ctx = wrap.WithAction(ctx, "decode_address_payload")
		return "", wrap.Error(ctx, fmt.Errorf("%s: failed to decode data from LocationIQ response: %w", op, err))
	}

	return payload.Address, nil
}

// GetLocation fetches the longitude and latitude for a given address using the LocationIQ API.
func (c *LocationIQClient) GetLocation(ctx context.Context, address string) (float64, float64, error) {
	ctx = wrap.WithAction(ctx, "locationiq_get_location")

	url := fmt.Sprintf("%s/v1/search?key=%s&q=%s&format=json", domain, c.apiKey, address)

	resp, err := http.Get(url)
	if err != nil {
		ctx = wrap.WithAction(ctx, types.ActionExternalServiceFailed)
		return 0, 0, wrap.Error(ctx, fmt.Errorf("failed to make request to LocationIQ: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ctx = wrap.WithAction(ctx, types.ActionExternalServiceFailed)
		return 0, 0, wrap.Error(ctx, fmt.Errorf("unexpected response status %d", resp.StatusCode))
	}

	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0, wrap.Error(ctx, fmt.Errorf(" failed to decode data from LocationIQ response: %w", err))
	}

	if len(results) == 0 {
		return 0, 0, wrap.Error(ctx, ErrLocationNotFound)
	}

	lat, err := parseStringToFloat(results[0].Lat)
	if err != nil {
		return 0, 0, wrap.Error(ctx, fmt.Errorf("failed to parse latitude: %w", err))
	}
	lon, err := parseStringToFloat(results[0].Lon)
	if err != nil {
		return 0, 0, wrap.Error(ctx, fmt.Errorf("failed to parse longitude: %w", err))
	}

	return lon, lat, nil
}

func parseStringToFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
