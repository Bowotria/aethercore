// Package telegram implements a native AetherCore gateway adapter for the
// Telegram Bot API.
//
// The adapter uses Telegram's HTTP long-polling API exclusively — no external
// dependencies, pure stdlib net/http.  Agents plug into Telegram chats by
// registering a [Bot] instance and wiring it to an sdk.ModuleRegistry.
//
// Architecture:
//
//	Telegram API
//	    │  HTTPS long-poll (getUpdates)
//	    ▼
//	[Poller]  ──updates──▶  [Router]  ──ModuleTask──▶  [Adapter]
//	                                                         │
//	                                                    sdk.ModuleRegistry
//	                                                         │
//	                                                    ModuleResult
//	                                                         │
//	                                              sendMessage ──▶ Telegram API
package telegram

import "fmt"

// -----------------------------------------------------------------------------
// Core Telegram Bot API types
// https://core.telegram.org/bots/api
// -----------------------------------------------------------------------------

// User represents a Telegram user or bot.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat represents a Telegram chat (private, group, supergroup, channel).
type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"` // "private" | "group" | "supergroup" | "channel"
	Title string `json:"title,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      Chat   `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
}

// Update represents a Telegram webhook/getUpdates payload.
// Only the message field is handled; callback queries and other event types
// are ignored in this implementation.
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

// ChatID is a type-safe Telegram chat identifier.
type ChatID = int64

// -----------------------------------------------------------------------------
// API response wrappers
// -----------------------------------------------------------------------------

// apiResponse is the generic envelope returned by every Telegram Bot API call.
type apiResponse[T any] struct {
	OK          bool   `json:"ok"`
	Result      T      `json:"result"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
}

// APIError is returned when the Telegram Bot API reports ok=false.
type APIError struct {
	Code        int
	Description string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram: API error %d: %s", e.Code, e.Description)
}

// SendMessageRequest is the payload for the sendMessage Bot API method.
type SendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"` // "Markdown" | "HTML" | ""
}

// getUpdatesRequest is the payload for the getUpdates Bot API method.
type getUpdatesRequest struct {
	Offset  int64 `json:"offset,omitempty"`
	Limit   int   `json:"limit,omitempty"`
	Timeout int   `json:"timeout,omitempty"`
}
