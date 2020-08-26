package functions

import (
	"fmt"
	"regexp"

	"sigs.k8s.io/yaml"
)

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
	Name        string   `yaml:"name"`
	Regex       string   `yaml:"regex"`
	TargetTeams []string `yaml:"targetTeams"`
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

// GetTeamSet returns matched URLs of target teams
func (c SlackNotifierConfig) GetTeamSet(target string) (map[string]struct{}, error) {
	teams := make(map[string]struct{})
	for _, r := range c.Rules {
		matched, err := regexp.Match(r.Regex, []byte(target))
		if err != nil {
			return nil, err
		}
		if !matched {
			continue
		}
		for _, t := range r.TargetTeams {
			teams[t] = struct{}{}
		}
	}
	return teams, nil
}

// ConvertTeamsToURLs converts teams set to URLs set
func (c SlackNotifierConfig) ConvertTeamsToURLs(teams map[string]struct{}) (map[string]struct{}, error) {
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

// GetColorFromMessage returns color by maching regex with message
func (c SlackNotifierConfig) GetColorFromMessage(message string) (string, error) {
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
