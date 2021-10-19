package slacknotifier

import (
	"fmt"
	"regexp"

	"sigs.k8s.io/yaml"
)

// "good" is recognized as red by slack API
const defaultColor = "good"

// SlackNotifierConfig is a Slack notifier
type SlackNotifierConfig struct {
	Teams    map[string]string `yaml:"teams"`
	Severity []Severity        `yaml:"severity"`
	Rules    []Rule            `yaml:"rules"`
}

// Severity represents relationships betwen color and regex
type Severity struct {
	Color string `yaml:"color"`
	Regex string `yaml:"regex"`
}

// Rule is a notification rule for Slack
type Rule struct {
	Name         string   `yaml:"name"`
	Regex        string   `yaml:"regex"`
	ExcludeRegex *string  `yaml:"excludeRegex"`
	TargetTeams  []string `yaml:"targetTeams"`
}

// NewSlackNotifierConfig creates new Notifier from config YAML
func NewSlackNotifierConfig(configYAML []byte) (*SlackNotifierConfig, error) {
	var n SlackNotifierConfig
	err := yaml.Unmarshal(configYAML, &n)
	if err != nil {
		return nil, err
	}

	return &n, nil
}

// FindTeamsByInstanceName returns matched URLs of target teams
func (c SlackNotifierConfig) FindTeamsByInstanceName(target string) (map[string]struct{}, error) {
	teams := make(map[string]struct{})
	for _, r := range c.Rules {
		matched, err := regexp.Match(r.Regex, []byte(target))
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}
		if r.ExcludeRegex != nil {
			exMatched, err := regexp.Match(*r.ExcludeRegex, []byte(target))
			if err != nil {
				return nil, err
			}
			if exMatched {
				continue
			}
		}
		for _, t := range r.TargetTeams {
			teams[t] = struct{}{}
		}
	}
	return teams, nil
}

// GetWebHookURLsFromTeams returns webhook URLs set from the given teams
func (c SlackNotifierConfig) GetWebHookURLsFromTeams(teams map[string]struct{}) (map[string]struct{}, error) {
	urls := make(map[string]struct{})
	for t := range teams {
		v, ok := c.Teams[t]
		if !ok {
			return nil, fmt.Errorf("cannot find %s in teams field", t)
		}
		urls[v] = struct{}{}
	}

	return urls, nil
}

// FindColorByMessage returns color by maching regex with message
func (c SlackNotifierConfig) FindColorByMessage(message string) (string, error) {
	for _, s := range c.Severity {
		matched, err := regexp.Match(s.Regex, []byte(message))
		if err != nil {
			return "", err
		}
		if !matched {
			continue
		}
		return s.Color, nil
	}

	return defaultColor, nil
}
