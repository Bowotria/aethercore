package security

import (
	"context"
	"regexp"
)

type RegexScanner struct {
	patterns map[string]*regexp.Regexp
}

func NewRegexScanner() *RegexScanner {
	return &RegexScanner{
		patterns: map[string]*regexp.Regexp{
			"SYSTEM_PROMPT_LEAK": regexp.MustCompile(`(?i)(reveal|show|print|output)\s+(your\s+)?(system\s+)?(prompt|instructions)`),
		},
	}
}

func (s *RegexScanner) Scan(ctx context.Context, text string, config GuardConfig) GuardResult {
	for category, pattern := range s.patterns {
		if loc := pattern.FindStringIndex(text); loc != nil {
			snippet := text[loc[0]:loc[1]]
			return GuardResult{
				IsSafe: false, Confidence: 0.9,
				Violations: []AdversarialMatch{{Category: category, Description: "Matched heuristic regex", Snippet: snippet, Severity: "HIGH"}},
			}
		}
	}
	return GuardResult{IsSafe: true}
}
