package security

import "context"

type MockPromptGuard struct {
	ForceFail  bool
	MockReason string
}

func (m *MockPromptGuard) Scan(ctx context.Context, text string, config GuardConfig) GuardResult {
	if m.ForceFail {
		return GuardResult{
			IsSafe: false, Confidence: 0.99,
			Violations: []AdversarialMatch{{Category: "MOCK_INJECTION", Description: m.MockReason, Severity: "CRITICAL"}},
		}
	}
	return GuardResult{IsSafe: true, Confidence: 1.0}
}
