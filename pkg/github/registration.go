package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

type clientWrapper struct {
	client *github.Client
}

// newClientFromPAT creates GitHub Actions Client from a personal access token (PAT).
func newClientFromPAT(pat string) *clientWrapper {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: pat},
	)
	tc := oauth2.NewClient(ctx, ts)
	return &clientWrapper{
		client: github.NewClient(tc),
	}
}

// newClientFromAppKey creates GitHub Actions Client from a private key of a GitHub app.
func newClientFromAppKey(appID, appInstallationID int64, privateKeyPath string) (*clientWrapper, error) {
	rt, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appID, appInstallationID, privateKeyPath)
	if err != nil {
		return nil, err
	}
	return &clientWrapper{
		client: github.NewClient(&http.Client{Transport: rt}),
	}, nil
}

func (c *clientWrapper) createRegistrationToken(ctx context.Context, owner, repo string) (*github.RegistrationToken, error) {
	var token *github.RegistrationToken
	var res *github.Response
	var err error
	token, res, err = c.client.Actions.CreateRegistrationToken(
		ctx,
		owner,
		repo,
	)
	if e, ok := err.(*url.Error); ok {
		// When url.Error came back, it was because the raw Responce leaked out as a string.
		return nil, fmt.Errorf("failed to create registration token: %s %s", e.Op, e.URL)
	}
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("invalid status code %d", res.StatusCode)
	}

	return token, nil
}

func CreateRegistrationToken(owner, repository, pat, privateKeyPath string, appId int64, installationId int64) (string, time.Time, error) {
	if pat != "" {
		token, err := newClientFromPAT(pat).createRegistrationToken(context.Background(), owner, repository)
		if err != nil {
			return "", time.Time{}, err
		}
		return token.GetToken(), token.GetExpiresAt().Time, nil
	} else if privateKeyPath != "" && appId != 0 {
		client, err := newClientFromAppKey(appId, installationId, privateKeyPath)
		if err != nil {
			return "", time.Time{}, err
		}
		token, err := client.createRegistrationToken(context.Background(), owner, repository)
		if err != nil {
			return "", time.Time{}, err
		}
		return token.GetToken(), token.GetExpiresAt().Time, nil
	}
	return "", time.Time{}, fmt.Errorf("neither pat nor application info are available")
}
