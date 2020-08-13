package functions

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNotifier(t *testing.T) {
	yaml := `
teams:
  team1: https://webhook/team1
  team2: https://webhook/team2
  team3: https://webhook/team3
severity:
  - color: good
    regex: ^INFO
  - color: warning
    regex: ^WARN
  - color: danger
    regex: ^ERROR
rules:
  - name: sample1
    regex: sample1-+[0-9]
    targetTeams:
      - team1
  - name: sample2
    regex: sample2-+[0-9]
    targetTeams:
      - team2
  - name: sample23
    regex: sample23-+[0-9]
    targetTeams:
      - team2
      - team3
`
	n, err := NewConfig([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}

	expect := Config{
		Teams: map[string]string{
			"team1": "https://webhook/team1",
			"team2": "https://webhook/team2",
			"team3": "https://webhook/team3",
		},
		Severity: []Severity{
			{"good", "^INFO"},
			{"warning", "^WARN"},
			{"danger", "^ERROR"},
		},
		Rules: []Rule{
			{"sample1", "sample1-+[0-9]", []string{"team1"}},
			{"sample2", "sample2-+[0-9]", []string{"team2"}},
			{"sample23", "sample23-+[0-9]", []string{"team2", "team3"}},
		},
	}
	if !cmp.Equal(*n, expect) {
		t.Fatalf("diff: %#v", cmp.Diff(n, expect))
	}

	type Input struct {
		target  string
		message string
	}
	type Expect struct {
		targetURLs map[string]struct{}
		color      string
	}
	testCases := []struct {
		input  Input
		expect Expect
	}{
		{
			input: Input{"sample1-0", "INFO team1 message"},
			expect: Expect{
				map[string]struct{}{"https://webhook/team1": {}},
				"good",
			},
		},
		{
			input: Input{"sample23-0", "WARN team5 message"},
			expect: Expect{
				map[string]struct{}{
					"https://webhook/team2": {},
					"https://webhook/team3": {},
				},
				"warning",
			},
		},
		{
			input: Input{"sample4-0", "DEBUG team1 message"},
			expect: Expect{
				map[string]struct{}{},
				"good",
			},
		},
	}

	for _, tt := range testCases {
		teams, err := n.GetTeamSet(tt.input.target)
		if err != nil {
			t.Errorf("test case: %#v, err: %#v", tt, err)
		}
		targetURLs, err := n.ConvertTeamsToURLs(teams)
		if err != nil {
			t.Errorf("test case: %#v, err: %#v", tt, err)
		}
		color, err := n.GetColorFromMessage(tt.input.message)
		if err != nil {
			t.Errorf("test case: %#v, err: %#v", tt, err)
		}

		if !cmp.Equal(targetURLs, tt.expect.targetURLs) {
			t.Errorf("diff: %#v", cmp.Diff(targetURLs, tt.expect.targetURLs))
		}
		if color != tt.expect.color {
			t.Errorf("diff: %#v", cmp.Diff(color, tt.expect.color))
		}
	}
}
