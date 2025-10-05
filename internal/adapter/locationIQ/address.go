package locationIQ

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
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

	fmt.Println(url)

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
