package core

// VerifiedTool wraps a standard Tool to explicitly indicate it passed cryptographic checks.
type VerifiedTool struct {
	Tool
}
