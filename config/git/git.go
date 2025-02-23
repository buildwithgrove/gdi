package git

import (
	"errors"
	"regexp"
)

var tokenPattern = regexp.MustCompile(`^(ghp_|gho_|ghu_|ghs_|ghr_|github_pat_)[a-zA-Z0-9_]{36,}$`)

var (
	errGitConfigNotFound          = errors.New("config error: Git config not found")
	errPersonalAccessTokenMissing = errors.New("config error: personal access token is missing")
	errInvalidPersonalAccessToken = errors.New("config error: personal access token is invalid")
)

type Config struct {
	PersonalAccessToken string `yaml:"personal_access_token"`
}

func (c *Config) Validate() error {
	if c == nil {
		return errGitConfigNotFound
	}
	if c.PersonalAccessToken == "" {
		return errPersonalAccessTokenMissing
	}
	if !tokenPattern.MatchString(c.PersonalAccessToken) {
		return errInvalidPersonalAccessToken
	}
	return nil
}
