package security

import "context"

type SemanticAnalyzer struct{}

func NewSemanticAnalyzer() *SemanticAnalyzer { return &SemanticAnalyzer{} }
func (s *SemanticAnalyzer) Scan(ctx context.Context, text string, config GuardConfig) GuardResult {
	if text == "" {
		return GuardResult{IsSafe: true}
	}

	if violation, ok := s.checkTokenDensity(text); ok {
		return GuardResult{
			IsSafe:     false,
			Confidence: 0.8,
			Violations: []AdversarialMatch{violation},
		}
	}

	if violation, ok := s.checkSpecialCharRatio(text); ok {
		return GuardResult{
			IsSafe:     false,
			Confidence: 0.85,
			Violations: []AdversarialMatch{violation},
		}
	}

	return GuardResult{IsSafe: true}
}

func (s *SemanticAnalyzer) checkTokenDensity(text string) (AdversarialMatch, bool) {
	maxWordLen := 0
	currentWordLen := 0
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' {
			if currentWordLen > maxWordLen {
				maxWordLen = currentWordLen
			}
			currentWordLen = 0
		} else {
			currentWordLen++
		}
	}
	if currentWordLen > maxWordLen {
		maxWordLen = currentWordLen
	}

	if maxWordLen > 50 {
		return AdversarialMatch{
			Category:    "TOKEN_DENSITY_ANOMALY",
			Description: "Unusually long unbroken token detected",
			Severity:    "MEDIUM",
		}, true
	}
	return AdversarialMatch{}, false
}

func (s *SemanticAnalyzer) checkSpecialCharRatio(text string) (AdversarialMatch, bool) {
	specialChars := 0
	for _, r := range text {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != ' ' && r != '\n' && r != '\t' {
			specialChars++
		}
	}

	ratio := float64(specialChars) / float64(len(text))
	if len(text) > 20 && ratio > 0.4 {
		return AdversarialMatch{
			Category:    "PADDING_ABUSE",
			Description: "High concentration of special characters",
			Severity:    "HIGH",
		}, true
	}
	return AdversarialMatch{}, false
}
