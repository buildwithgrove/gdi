package git

import (
	"errors"
	"regexp"
)

var tokenPattern = regexp.MustCompile(`^(ghp_|gho_|ghu_|ghs_|ghr_|github_pat_)[a-zA-Z0-9_]{36,}$`)

var (
	errPersonalAccessTokenMissing = errors.New("config error: personal access token is missing")
	errInvalidPersonalAccessToken = errors.New("config error: personal access token is invalid")
)

type Config struct {
	// Valid Personal Access Token for GitHub. Required if performing actions on a private repository.
	// Should have at least `write` scope for `repo`.
	PersonalAccessToken string `yaml:"personal_access_token"`
}

func (c *Config) Validate() error {
	if c != nil {
		if c.PersonalAccessToken == "" {
			return errPersonalAccessTokenMissing
		}
		if !tokenPattern.MatchString(c.PersonalAccessToken) {
			return errInvalidPersonalAccessToken
		}
	}
	return nil
}
