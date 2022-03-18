package reviewbot

import (
	"fmt"
	"net/http"

	"github.com/google/go-github/v42/github"
	"github.com/rs/zerolog/log"
)

const secretToken = "FooBar"
const appID = 169668

type Config struct {
	// Configuration for GitHub
	// TODO: future: option to get the below values from Secret Manager. See: `setKeySecret` in pkg/config/operator/operator.go
	GitHub struct {
		// The GitHub App's id.
		// See: https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps#authenticating-as-a-github-app
		AppId int

		// Path to private key
		PrivateKeyPath string

		// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/securing-your-webhooks
		SecretToken string
	}

	// The global minimum reviews reqiuired for approval
	MinReviewsRequired int

	// Port to listen on
	Port int
}

type WebookHandler struct {
	config Config
}

func HandleWebhooks(config *Config) error {
	w := WebookHandler{*config}

	http.HandleFunc("/", w.HandleRoot)

	address := fmt.Sprintf(":%d", config.Port)

	return http.ListenAndServe(address, nil)
}

func (h *WebookHandler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(secretToken))
	if err != nil {
		log.Error().Err(err).Msg("Got an invalid payload")
		w.WriteHeader(400)
		w.Write([]byte("Got an invalid payload"))
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse the webhook payload")
		w.WriteHeader(400)
		w.Write([]byte("Failed to parse the webhook payload"))
		return
	}

	var pr *github.PullRequestReviewEvent

	switch event := event.(type) {
	case *github.PullRequestReviewEvent:
		pr = event
	default:
		log.Warn().Interface("Event", event).Msg("Unknown event")
		w.WriteHeader(400)
		w.Write([]byte("Unknown GitHub Event"))
		return
	}

	log.Info().Interface("Pull Request Review", pr).Msg("Handling Pull Request Review Event")

	err = runPRCheck(h.config, pr)
	if err != nil {
		log.Error().Err(err).Msg("Error handling webhook")
		w.WriteHeader(500)
		w.Write([]byte("Error handling webhook"))
		return
	}
}
