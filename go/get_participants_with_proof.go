package gonkaopenai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gonka-ai/gonka-utils/go/contracts"
	"github.com/gonka-ai/gonka-utils/go/utils"
	"io"
	"net/http"
	"os"
)

var ErrInvalidEpoch = errors.New("invalid epoch")

func getParticipantsWithProof(baseURL string) utils.GetParticipantsFn {
	return func(ctx context.Context, epoch string) (*contracts.ActiveParticipantWithProof, error) {
		if epoch == "" {
			return nil, ErrInvalidEpoch
		}

		// Ensure baseURL doesn't end with a slash
		if baseURL != "" && baseURL[len(baseURL)-1] == '/' {
			baseURL = baseURL[:len(baseURL)-1]
		}

		url := fmt.Sprintf("%s/v1/epochs/%v/participants", baseURL, epoch)
		fmt.Println("url", url)

		// Create a new HTTP request
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch participants with proof: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch participants with proof: status code %d", resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var participantResp contracts.ActiveParticipantWithProof
		if err := json.Unmarshal(bodyBytes, &participantResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &participantResp, nil
	}
}

// GetParticipantsWithProof fetches participants from the specified base URL and epoch,
// verifies the proof, and returns a list of Endpoints.
// This function is independent of the GonkaOpenAI client.
// Specify "current" as the epoch to fetch the current participants.
func GetParticipantsWithProof(ctx context.Context, baseURL string, epoch string) ([]Endpoint, error) {
	if epoch == "" {
		return nil, ErrInvalidEpoch
	}

	fn := getParticipantsWithProof(baseURL)
	participantResp, err := fn(ctx, epoch)
	if err != nil {
		return nil, err
	}

	verify := os.Getenv(verifyEnabledEnv) == "1"
	if verify {
		expectedAppHashHex := os.Getenv("GONKA_APP_HASH_HEX")
		err := utils.VerifyParticipants(ctx, expectedAppHashHex, fn, epoch)
		if err != nil {
			return nil, err
		}
	}

	endpoints := make([]Endpoint, 0, len(participantResp.ActiveParticipants.Participants))
	for _, participant := range participantResp.ActiveParticipants.Participants {
		inferenceUrl := participant.InferenceUrl
		endpoints = append(endpoints, Endpoint{
			URL:     inferenceUrl + "/v1",
			Address: participant.Index,
		})
	}
	return endpoints, nil
}
