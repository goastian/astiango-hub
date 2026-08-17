package models

type NotificationChannel struct {
	any              `collection:"notification_channels"`
	BaseModel        `bson:",inline"`
	Type             string `json:"type" bson:"type" description:"Type" enum:"im,mail"`
	Name             string `json:"name" bson:"name" description:"Name"`
	Description      string `json:"description" bson:"description" description:"Description"`
	Provider         string `json:"provider" bson:"provider" description:"Provider"`
	SMTPServer       string `json:"smtp_server,omitempty" bson:"smtp_server,omitempty" description:"SMTP server"`
	SMTPPort         int    `json:"smtp_port,omitempty" bson:"smtp_port,omitempty" description:"SMTP port"`
	SMTPUsername     string `json:"smtp_username,omitempty" bson:"smtp_username,omitempty" description:"SMTP username"`
	SMTPPassword     string `json:"smtp_password,omitempty" bson:"smtp_password,omitempty" description:"SMTP password"`
	WebhookUrl       string `json:"webhook_url,omitempty" bson:"webhook_url,omitempty" description:"Webhook URL"`
	TelegramBotToken string `json:"telegram_bot_token,omitempty" bson:"telegram_bot_token,omitempty" description:"Telegram bot token"`
	TelegramChatId   string `json:"telegram_chat_id,omitempty" bson:"telegram_chat_id,omitempty" description:"Telegram chat ID"`
	GoogleOAuth2Json string `json:"google_oauth2_json,omitempty" bson:"google_oauth2_json,omitempty" description:"Google OAuth2 JSON"`
}
