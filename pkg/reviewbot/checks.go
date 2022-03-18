package reviewbot

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v42/github"
	"github.com/rs/zerolog/log"
)

func runPRCheck(config Config, pr *github.PullRequestReviewEvent) error {
	tr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appID, pr.GetInstallation().GetID(), config.GitHub.PrivateKeyPath)
	if err != nil {
		log.Error().Err(err).Msg("Could not read key")
		return err
	}

	client := github.NewClient(&http.Client{Transport: tr})
	ctx := context.Background()
	owner := pr.GetRepo().GetOwner().GetLogin()
	repo := pr.GetRepo().GetName()

	// TODO: get repo-level overwrites, if available
	minReviewsRequired := config.MinReviewsRequired

	optListReviews := &github.ListOptions{
		PerPage: 100,
	}

	// List of approvers
	// TODO: check pagination and ordering
	reviews, _, err := client.PullRequests.ListReviews(ctx, owner, repo, pr.GetPullRequest().GetNumber(), optListReviews)
	if err != nil {
		log.Error().Err(err).Msg("Could not list reviews")
		return err
	}

	var approvalCandidates = map[string]bool{
		// Add PR Creator as someone to check
		pr.GetPullRequest().GetUser().GetLogin(): true,
	}

	// Check reviews
	for _, review := range reviews {
		login := review.GetUser().GetLogin()
		association := review.GetAuthorAssociation()
		state := review.GetState()

		// "COLLABORATOR", "CONTRIBUTOR", "FIRST_TIMER", "FIRST_TIME_CONTRIBUTOR", "MEMBER", "OWNER", or "NONE".
		if association == "NONE" || state == "COMMENTED" {
			continue
		}

		log.Debug().Str("login", login).Str("association", association).Str("state", state).Msg("Found a review candidate")

		if state == "APPROVED" {
			approvalCandidates[login] = true
		} else {
			delete(approvalCandidates, login)
		}
	}

	// Points for approval
	points := 0

	for login := range approvalCandidates {
		permissionLevel, _, err := client.Repositories.GetPermissionLevel(ctx, owner, repo, login)
		if err != nil {
			return err
		}

		permission := permissionLevel.GetPermission()
		isAuthorized := permission == "admin" || permission == "write"

		log.Debug().Str("login", login).Str("permission", permission).Msg("Approver Authorization")

		if isAuthorized {
			points++

			if points == minReviewsRequired {
				// no need to waste resources - we have enough authorized approvers
				break
			}
		}
	}

	log.Info().Int("points", points).Msg("Check's State")

	statusComplete := "completed"
	titlePrefix := "⭐️ Allstar Pull Request Review Bot - "
	text := fmt.Sprintf("PR has %d authorized approvals, %d required", points, minReviewsRequired)
	timestamp := github.Timestamp{
		Time: time.Now(),
	}

	check := github.CreateCheckRunOptions{
		Name:        "Allstar Review Bot",
		Status:      &statusComplete,
		CompletedAt: &timestamp,
		Output: &github.CheckRunOutput{
			Text: &text,
		},
		HeadSHA: pr.PullRequest.GetHead().GetSHA(),
		// TODO: DetailsURL
	}

	if points >= minReviewsRequired {
		conclusion := "success"
		title := titlePrefix + conclusion
		summary := "PR has enough authorized approvals"

		check.Conclusion = &conclusion
		check.Output.Title = &title
		check.Output.Summary = &summary
	} else {
		conclusion := "failure"
		title := titlePrefix + conclusion
		summary := "PR does not have enough authorized approvals"

		check.Conclusion = &conclusion
		check.Output.Title = &title
		check.Output.Summary = &summary
	}

	checkRun, _, err := client.Checks.CreateCheckRun(ctx, owner, repo, check)
	if err != nil {
		return err
	}

	log.Info().Interface("Check Run", checkRun).Msg("Created Check Run")

	return nil
}
