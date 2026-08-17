package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type NotificationRequest struct {
	any         `collection:"notification_requests"`
	BaseModel   `bson:",inline"`
	Status      string               `json:"status" bson:"status" description:"Status"`
	Error       string               `json:"error,omitempty" bson:"error,omitempty" description:"Error"`
	Title       string               `json:"title" bson:"title" description:"Title"`
	Content     string               `json:"content" bson:"content" description:"Content"`
	SenderEmail string               `json:"sender_email,omitempty" bson:"sender_email,omitempty" description:"Sender email"`
	SenderName  string               `json:"sender_name,omitempty" bson:"sender_name,omitempty" description:"Sender name"`
	MailTo      []string             `json:"mail_to,omitempty" bson:"mail_to,omitempty" description:"Mail to"`
	MailCc      []string             `json:"mail_cc,omitempty" bson:"mail_cc,omitempty" description:"Mail CC"`
	MailBcc     []string             `json:"mail_bcc,omitempty" bson:"mail_bcc,omitempty" description:"Mail BCC"`
	SettingId   primitive.ObjectID   `json:"setting_id" bson:"setting_id" description:"Setting ID"`
	ChannelId   primitive.ObjectID   `json:"channel_id" bson:"channel_id" description:"Channel ID"`
	Setting     *NotificationSetting `json:"setting,omitempty" bson:"-" binding:"-"`
	Channel     *NotificationChannel `json:"channel,omitempty" bson:"-" binding:"-"`
	Test        bool                 `json:"test,omitempty" bson:"test,omitempty" description:"Test"`
}

type NotificationRequestDTO struct {
	NotificationRequest `json:",inline" bson:",inline"`

	Setting *NotificationSetting `json:"setting,omitempty" bson:"_setting,omitempty"`
	Channel *NotificationChannel `json:"channel,omitempty" bson:"_channel,omitempty"`
}
