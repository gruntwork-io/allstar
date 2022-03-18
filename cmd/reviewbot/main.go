// Copyright 2022 Allstar Authors

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"os"
	"strconv"

	"github.com/ossf/allstar/pkg/reviewbot"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const defaultPort = 8080
const defaultGitHubSecretToken = "FooBar"
const defaultGitHubAppID = 169668

func main() {
	setupLog()

	config := reviewbot.Config{}

	if err := determineConfig(&config); err != nil {
		log.Fatal().Err(err).Msg("Error determining configuration")
	}

	reviewbot.HandleWebhooks(&config)
}

func determineConfigFromEnv(config *reviewbot.Config) error {
	if envPort, ok := os.LookupEnv("PORT"); ok {
		port, err := strconv.ParseUint(envPort, 10, 16)

		if err != nil {
			return err
		}

		config.Port = int(port)
	}

	if envAppId, ok := os.LookupEnv("APP_ID"); ok {
		appId, err := strconv.ParseUint(envAppId, 10, 64)

		if err != nil {
			return err
		}

		config.GitHub.AppId = int(appId)
	}

	if envPrivateKeyPath, ok := os.LookupEnv("PRIVATE_KEY_PATH"); ok {
		config.GitHub.PrivateKeyPath = envPrivateKeyPath
	}

	if envSecretToken, ok := os.LookupEnv("SECRET_TOKEN"); ok {
		config.GitHub.SecretToken = envSecretToken
	}

	return nil
}

func determineConfigFromFlags(config *reviewbot.Config) error {
	flagAppID := flag.Int("app-id", defaultGitHubAppID, "A GitHub App Id")
	flagPrivateKeyPath := flag.String("private-key-path", "", "A path to a GitHub Private Key")
	flagSecretToken := flag.String("secret-token", defaultGitHubSecretToken, "GitHub Private Key")
	flagPort := flag.Int("port", defaultPort, "A port to listen on")

	flag.Parse()

	if *flagAppID != defaultGitHubAppID {
		config.GitHub.AppId = *flagAppID
	}

	if *flagPrivateKeyPath != "" {
		config.GitHub.PrivateKeyPath = *flagPrivateKeyPath
	}

	if *flagSecretToken != "" {
		config.GitHub.PrivateKeyPath = *flagSecretToken
	}

	if *flagPort != defaultPort {
		config.Port = *flagPort
	}

	return nil
}

func determineConfig(config *reviewbot.Config) error {
	// Set defaults
	config.GitHub.AppId = defaultGitHubAppID
	config.GitHub.SecretToken = defaultGitHubSecretToken
	config.Port = defaultPort

	// Determine from environment variables
	determineConfigFromEnv(config)

	// Determine from flags
	determineConfigFromFlags(config)

	return nil
}

func setupLog() {
	// Match expected values in GCP
	zerolog.LevelFieldName = "severity"
	zerolog.LevelTraceValue = "DEFAULT"
	zerolog.LevelDebugValue = "DEBUG"
	zerolog.LevelInfoValue = "INFO"
	zerolog.LevelWarnValue = "WARNING"
	zerolog.LevelErrorValue = "ERROR"
	zerolog.LevelFatalValue = "CRITICAL"
	zerolog.LevelPanicValue = "CRITICAL"
}
