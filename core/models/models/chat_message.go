package models

import (
	"github.com/goastian/astiango-hub/core/entity"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatMessage struct {
	any            `collection:"chat_messages"`
	BaseModel      `bson:",inline"`
	ConversationId primitive.ObjectID       `json:"conversation_id" bson:"conversation_id" description:"Conversation ID"`
	Role           string                   `json:"role" bson:"role" description:"Message role (user/assistant/system)"`
	Content        string                   `json:"content,omitempty" bson:"content,omitempty" description:"Message content for user/system only"`
	IsAgent        bool                     `json:"is_agent,omitempty" bson:"is_agent,omitempty" description:"Is agent"`
	ContentIds     []primitive.ObjectID     `json:"content_ids,omitempty" bson:"content_ids,omitempty" description:"Content IDs"`
	Contents       []ChatMessageContent     `json:"contents,omitempty" bson:"-" description:"Contents"`
	Model          string                   `json:"model" bson:"model" description:"AI model used"`
	Status         string                   `json:"status" bson:"status" description:"Message status (pending/completed/failed)"`
	Error          string                   `json:"error,omitempty" bson:"error,omitempty" description:"Error message if failed"`
	Usage          *entity.LLMResponseUsage `json:"usage,omitempty" bson:"-" description:"Usage"`
}

func (m *ChatMessage) GetContent() string {
	// If the message has a single content item, return it directly
	if m.Content != "" {
		return m.Content
	}

	// If the message has multiple content items, concatenate them
	var result string
	for _, content := range m.Contents {
		switch content.Type {
		case "text":
			result += content.Content
		case "action":
			// Format action content with status
			actionInfo := "[" + content.Action
			if content.ActionStatus != "" {
				actionInfo += " - " + string(content.ActionStatus)
			}
			actionInfo += "]"

			if content.Content != "" {
				result += actionInfo + ": " + content.Content
			} else {
				result += actionInfo
			}
		default:
			// For any unrecognized type, just add the content
			result += content.Content
		}

		// Add newline between content items
		result += "\n\n"
	}

	return result
}

func (m *ChatMessage) GetUsage(contents []ChatMessageContent) *entity.LLMResponseUsage {
	if contents == nil {
		contents = m.Contents
	}
	if len(contents) == 0 {
		return nil
	}
	var usage entity.LLMResponseUsage
	for _, content := range contents {
		if content.Usage != nil {
			// Accumulate usage
			usage.PromptTokens += content.Usage.PromptTokens
			usage.CompletionTokens += content.Usage.CompletionTokens
			usage.TotalTokens += content.Usage.TotalTokens
		}
	}
	return &usage
}
