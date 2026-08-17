package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatConversation struct {
	any           `collection:"chat_conversations"`
	BaseModel     `bson:",inline"`
	Title         string             `json:"title" bson:"title" description:"Conversation title"`
	Description   string             `json:"description,omitempty" bson:"description,omitempty" description:"Conversation description"`
	UserId        primitive.ObjectID `json:"user_id" bson:"user_id" description:"User ID who owns this conversation"`
	Model         string             `json:"model" bson:"model" description:"Default AI model for this conversation"`
	Status        string             `json:"status" bson:"status" description:"Conversation status (active/archived/deleted)"`
	LastMessageAt primitive.DateTime `json:"last_message_at,omitempty" bson:"last_message_at,omitempty" description:"Last message timestamp"`
	Settings      map[string]any     `json:"settings,omitempty" bson:"settings,omitempty" description:"Conversation settings"`
	Tags          []string           `json:"tags,omitempty" bson:"tags,omitempty" description:"Conversation tags"`
	Messages      []*ChatMessage     `json:"messages,omitempty" bson:"-" description:"Messages in this conversation (populated field)"`
}
