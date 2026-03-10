package llm

import "context"

// LLMAdapter defines the contract for any LLM provider used by the kernel.
type LLMAdapter interface {
	Generate(ctx context.Context, systemPrompt, userInput string) (string, error)
	GenerateWithTools(ctx context.Context, messages []Message, tools []ToolManifest) (LLMResponse, error)
	Name() string
}

// LLMResponse encapsulates the response from the LLM, including tool invocations if any.
type LLMResponse struct {
	Content    string
	ToolCalls  []ToolCall
	TokenUsage TokenUsage
}

// Message represents a single turn in a conversational ReAct loop history.
type Message struct {
	Role        string // "system", "user", "assistant", "tool"
	Content     string
	ToolCalls   []ToolCall
	ToolResults []ToolResultMessage
}

// ToolResultMessage holds the feedback from an executed local or sandboxed tool.
type ToolResultMessage struct {
	ToolCallID string
	Content    string
	IsError    bool
}

// ToolCall represents a deterministic request from the LLM to execute a tool.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// TokenUsage tracks the resource utilization per request.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ToolManifest is a placeholder for tool definitions (will be refined later)
type ToolManifest struct {
	Name        string
	Description string
	Parameters  interface{}
}
