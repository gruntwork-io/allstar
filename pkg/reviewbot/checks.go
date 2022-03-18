package reviewbot

import (
	"context"
	"net/http"

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

	// client
	// - determine if user is a member of org OR
	// - has write perms to repo OR
	// - has write perms in org OR

	opt := github.ListCollaboratorsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
		Affiliation: "all",
	}

	ctx := context.Background()

	owner := pr.GetRepo().GetOwner().GetLogin()
	repo := pr.GetRepo().GetName()

	users, _, err := client.Repositories.ListCollaborators(ctx, owner, repo, &opt)
	if err != nil {
		log.Error().Err(err).Msg("Could not list collaborators")
		return err
	}

	// TODO: this list could be too long - perhaps query by approvers
	var pushers = map[string]bool{}
	for _, u := range users {
		// NOTE: other permissions: https://docs.github.com/en/rest/reference/collaborators#list-repository-collaborators
		if u.GetPermissions()["push"] {
			log.Debug().Str("pusher", u.GetLogin()).Msg("Found a pusher")
			pushers[u.GetLogin()] = true
		}
	}

	optListReviews := &github.ListOptions{
		PerPage: 100,
	}

	// List of approvers
	reviews, _, err := client.PullRequests.ListReviews(ctx, owner, repo, pr.GetPullRequest().GetNumber(), optListReviews)
	if err != nil {
		log.Error().Err(err).Msg("Could not list reviews")
		return err
	}

	// Points for approval
	points := 0

	if pushers[pr.GetPullRequest().GetUser().GetLogin()] {
		log.Debug().Str("sender", pr.GetPullRequest().GetUser().GetLogin()).Msg("Sender is authorized")
		points++
	}

	// TODO: determine the precendence and which, should or maybe, trusted
	// "COLLABORATOR", "CONTRIBUTOR", "FIRST_TIMER", "FIRST_TIME_CONTRIBUTOR", "MEMBER", "OWNER", or "NONE".
	var authorizedAssociations = map[string]bool{}

	// Check reviews
	for _, review := range reviews {
		isApprover := pushers[review.GetUser().GetLogin()]

		review.GetAuthorAssociation()

		a, b, c := client.Repositories.GetPermissionLevel(ctx, owner, repo, &opt)

		a.Permission

		log.Debug().Str("Review", review.GetUser().GetLogin()).Str("State", review.GetState()).Msg("Found Review")

		isAuthorized := authorizedAssociations[review.AuthorAssociation]

		// review.AuthorAssociation

		if review.GetState() == "APPROVED" && isAuthorized {
			log.Debug().Str("approver", review.GetUser().GetLogin()).Msg("Found an authorized approver")

			points++
		}
	}

	totalPointsRequired := 2

	log.Info().Int("points", points).Msg("Total points")
	if points >= totalPointsRequired {
		// client.Checks.
		log.Info().Msg("Should pass")
	} else {
		log.Info().Msg("Should fail")
	}

	return nil
}
