package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type NotificationSetting struct {
	any         `collection:"notification_settings"`
	BaseModel   `bson:",inline"`
	Name        string `json:"name" bson:"name" description:"Name"`
	Description string `json:"description" bson:"description" description:"Description"`
	Enabled     bool   `json:"enabled" bson:"enabled" description:"Enabled"`

	Title                string `json:"title,omitempty" bson:"title,omitempty" description:"Title"`
	Template             string `json:"template" bson:"template" description:"Template"`
	TemplateMode         string `json:"template_mode" bson:"template_mode" description:"Template mode"`
	TemplateMarkdown     string `json:"template_markdown,omitempty" bson:"template_markdown,omitempty" description:"Template markdown"`
	TemplateRichText     string `json:"template_rich_text,omitempty" bson:"template_rich_text,omitempty" description:"Template rich text"`
	TemplateRichTextJson string `json:"template_rich_text_json,omitempty" bson:"template_rich_text_json,omitempty" description:"Template rich text JSON"`
	TemplateTheme        string `json:"template_theme,omitempty" bson:"template_theme,omitempty" description:"Template theme"`

	TaskTrigger string `json:"task_trigger" bson:"task_trigger" description:"Task trigger"`
	Trigger     string `json:"trigger" bson:"trigger" description:"Trigger"`

	SenderEmail          string   `json:"sender_email,omitempty" bson:"sender_email,omitempty" description:"Sender email"`
	UseCustomSenderEmail bool     `json:"use_custom_sender_email,omitempty" bson:"use_custom_sender_email,omitempty" description:"Use custom sender email"`
	SenderName           string   `json:"sender_name,omitempty" bson:"sender_name,omitempty" description:"Sender name"`
	MailTo               []string `json:"mail_to,omitempty" bson:"mail_to,omitempty" description:"Mail to"`
	MailCc               []string `json:"mail_cc,omitempty" bson:"mail_cc,omitempty" description:"Mail CC"`
	MailBcc              []string `json:"mail_bcc,omitempty" bson:"mail_bcc,omitempty" description:"Mail BCC"`

	ChannelIds []primitive.ObjectID `json:"channel_ids,omitempty" bson:"channel_ids,omitempty" description:"Channel IDs"`
}

type NotificationSettingDTO struct {
	NotificationSetting `json:",inline" bson:",inline"`

	Channels []NotificationChannel `json:"channels,omitempty" bson:"_channels,omitempty"`
}
