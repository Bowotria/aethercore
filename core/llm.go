package core

import "context"

// LLMAdapter defines the contract for any LLM provider used by the kernel.
// By abstracting this, AetherCore can seamlessly switch between OpenAI, Anthropic, or local endpoints like Ollama.
type LLMAdapter interface {
	// Generate takes a system prompt and user input, returning the raw response.
	Generate(ctx context.Context, systemPrompt, userInput string) (string, error)

	// GenerateWithTools handles tool calling capabilities natively for the provider, operating over a conversation history.
	GenerateWithTools(ctx context.Context, messages []Message, tools []ToolManifest) (LLMResponse, error)

	// Name returns the identifier for the provider.
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
	Arguments string `json:"arguments"` // JSON string representation of the parsed arguments
}

// TokenUsage tracks the resource utilization per request.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
