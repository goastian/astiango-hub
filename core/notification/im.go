package notification

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/apex/log"
	"github.com/goastian/astiango-hub/core/utils"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/goastian/astiango-hub/core/models/models"
)

type ResBody struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// RequestParam represents parameters for HTTP requests
type RequestParam map[string]interface{}

// performRequest performs an HTTP request with JSON body
func performRequest(method, url string, data interface{}) (*http.Response, []byte, error) {
	var reqBody io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := utils.NewHttpClient(15 * time.Second).Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to perform request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return resp, body, nil
}

func SendIMNotification(ch *models.NotificationChannel, title, content string) error {
	switch ch.Provider {
	case ChannelIMProviderLark:
		return sendIMLark(ch, title, content)
	case ChannelIMProviderDingtalk:
		return sendIMDingTalk(ch, title, content)
	case ChannelIMProviderWechatWork:
		return sendIMWechatWork(ch, title, content)
	case ChannelIMProviderSlack:
		return sendIMSlack(ch, title, content)
	case ChannelIMProviderTelegram:
		return sendIMTelegram(ch, title, content)
	case ChannelIMProviderDiscord:
		return sendIMDiscord(ch, title, content)
	case ChannelIMProviderMSTeams:
		return sendIMMSTeams(ch, title, content)
	}

	// request data
	data := RequestParam{
		"msgtype": "markdown",
		"markdown": RequestParam{
			"title":   title,
			"text":    content,
			"content": content,
		},
		"at": RequestParam{
			"atMobiles": []string{},
			"isAtAll":   false,
		},
		"text": content,
	}
	if strings.Contains(strings.ToLower(ch.WebhookUrl), "feishu") {
		data = RequestParam{
			"msg_type": "text",
			"content": RequestParam{
				"text": content,
			},
		}
	}

	// perform request
	resp, body, err := performRequest("POST", ch.WebhookUrl, data)
	if err != nil {
		log.Errorf("IM request error: %v", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	// parse response
	var resBody ResBody
	if err := json.Unmarshal(body, &resBody); err != nil {
		log.Errorf("Parsing IM response error: %v", err)
		return err
	}

	// validate response code
	if resBody.ErrCode != 0 {
		log.Errorf("IM response error: %s", resBody.ErrMsg)
		return errors.New(resBody.ErrMsg)
	}

	return nil
}

func performIMRequest(webhookUrl string, data RequestParam) ([]byte, error) {
	resp, body, err := performRequest("POST", webhookUrl, data)
	if err != nil {
		logger.Errorf("IM request error: %v", err)
		return nil, err
	}

	if resp.StatusCode >= 400 {
		logger.Errorf("IM response status code: %d", resp.StatusCode)
		return nil, fmt.Errorf("IM error response %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func performIMRequestWithJson[T any](webhookUrl string, data RequestParam) (resBody T, err error) {
	body, err := performIMRequest(webhookUrl, data)
	if err != nil {
		return resBody, err
	}

	// parse response
	if err := json.Unmarshal(body, &resBody); err != nil {
		logger.Warnf("Parsing IM response error: %v", err)
		logger.Infof("IM response: %s", string(body))
		return resBody, nil
	}

	return resBody, nil
}

func convertMarkdownToSlack(markdown string) string {
	// Convert bold text
	reBold := regexp.MustCompile(`\*\*(.*?)\*\*`)
	slack := reBold.ReplaceAllString(markdown, `*$1*`)

	// Convert italic text
	reItalic := regexp.MustCompile(`\*(.*?)\*`)
	slack = reItalic.ReplaceAllString(slack, `_$1_`)

	// Convert links
	reLink := regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
	slack = reLink.ReplaceAllString(slack, `<$2|$1>`)

	// Convert inline code
	reInlineCode := regexp.MustCompile("`(.*?)`")
	slack = reInlineCode.ReplaceAllString(slack, "`$1`")

	// Convert unordered list
	slack = strings.ReplaceAll(slack, "- ", "• ")

	// Convert ordered list
	reOrderedList := regexp.MustCompile(`^\d+\. `)
	slack = reOrderedList.ReplaceAllStringFunc(slack, func(s string) string {
		return strings.Replace(s, ". ", ". ", 1)
	})

	// Convert blockquote
	reBlockquote := regexp.MustCompile(`^> (.*)`)
	slack = reBlockquote.ReplaceAllString(slack, `> $1`)

	return slack
}

func convertMarkdownToTelegram(markdownText string) string {
	// Combined regex to handle bold and italic
	re := regexp.MustCompile(`(?m)(\*\*)(.*)(\*\*)|(__)(.*)(__)|(\*)(.*)(\*)|(_)(.*)(_)`)
	markdownText = re.ReplaceAllStringFunc(markdownText, func(match string) string {
		groups := re.FindStringSubmatch(match)
		if groups[1] != "" || groups[4] != "" {
			// Handle bold
			return "*" + match[2:len(match)-2] + "*"
		} else if groups[6] != "" || groups[9] != "" {
			// Handle italic
			return "_" + match[1:len(match)-1] + "_"
		} else {
			// No match
			return match
		}
	})

	// Convert unordered list
	re = regexp.MustCompile(`(?m)^- (.*)`)
	markdownText = re.ReplaceAllString(markdownText, "• $1")

	// Escape characters
	escapeChars := []string{"#", "-", "."}
	for _, c := range escapeChars {
		markdownText = strings.ReplaceAll(markdownText, c, "\\"+c)
	}

	return markdownText
}

func sendIMLark(ch *models.NotificationChannel, title, content string) error {
	data := RequestParam{
		"msg_type": "interactive",
		"card": RequestParam{
			"header": RequestParam{
				"title": RequestParam{
					"tag":     "plain_text",
					"content": title,
				},
			},
			"elements": []RequestParam{
				{
					"tag":     "markdown",
					"content": content,
				},
			},
		},
	}
	resBody, err := performIMRequestWithJson[ResBody](ch.WebhookUrl, data)
	if err != nil {
		return err
	}
	if resBody.ErrCode != 0 {
		return errors.New(resBody.ErrMsg)
	}
	return nil
}

func sendIMDingTalk(ch *models.NotificationChannel, title string, content string) error {
	data := RequestParam{
		"msgtype": "markdown",
		"markdown": RequestParam{
			"title": title,
			"text":  fmt.Sprintf("# %s\n\n%s", title, content),
		},
	}
	resBody, err := performIMRequestWithJson[ResBody](ch.WebhookUrl, data)
	if err != nil {
		return err
	}
	if resBody.ErrCode != 0 {
		return errors.New(resBody.ErrMsg)
	}
	return nil
}

func sendIMWechatWork(ch *models.NotificationChannel, title string, content string) error {
	data := RequestParam{
		"msgtype": "markdown",
		"markdown": RequestParam{
			"content": fmt.Sprintf("# %s\n\n%s", title, content),
		},
	}
	resBody, err := performIMRequestWithJson[ResBody](ch.WebhookUrl, data)
	if err != nil {
		return err
	}
	if resBody.ErrCode != 0 {
		return errors.New(resBody.ErrMsg)
	}
	return nil
}

func sendIMSlack(ch *models.NotificationChannel, title, content string) error {
	data := RequestParam{
		"blocks": []RequestParam{
			{"type": "header", "text": RequestParam{"type": "plain_text", "text": title}},
			{"type": "section", "text": RequestParam{"type": "mrkdwn", "text": convertMarkdownToSlack(content)}},
		},
	}
	_, err := performIMRequest(ch.WebhookUrl, data)
	if err != nil {
		return err
	}
	return nil
}

func sendIMTelegram(ch *models.NotificationChannel, title string, content string) error {
	type ResBody struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description"`
	}

	// chat id
	chatId := ch.TelegramChatId
	if !strings.HasPrefix("@", ch.TelegramChatId) {
		chatId = fmt.Sprintf("@%s", ch.TelegramChatId)
	}

	// webhook url
	webhookUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", ch.TelegramBotToken)

	// original Markdown text
	text := fmt.Sprintf("**%s**\n\n%s", title, content)

	// convert to Telegram MarkdownV2
	text = convertMarkdownToTelegram(text)

	// request data
	data := RequestParam{
		"chat_id":    chatId,
		"text":       text,
		"parse_mode": "MarkdownV2",
	}

	// perform request
	_, err := performIMRequest(webhookUrl, data)
	if err != nil {
		return err
	}
	return nil
}

func sendIMDiscord(ch *models.NotificationChannel, title string, content string) error {
	data := RequestParam{
		"embeds": []RequestParam{
			{
				"title":       title,
				"description": content,
			},
		},
	}
	_, err := performIMRequest(ch.WebhookUrl, data)
	if err != nil {
		return err
	}
	return nil
}

func sendIMMSTeams(ch *models.NotificationChannel, title string, content string) error {
	data := RequestParam{
		"type": "message",
		"attachments": []RequestParam{{
			"contentType": "application/vnd.microsoft.card.adaptive",
			"contentUrl":  nil,
			"content": RequestParam{
				"$schema": "https://adaptivecards.io/schemas/adaptive-card.json",
				"type":    "AdaptiveCard",
				"version": "1.2",
				"body": []RequestParam{
					{
						"type": "TextBlock",
						"text": fmt.Sprintf("**%s**", title),
						"size": "Large",
					},
					{
						"type": "TextBlock",
						"text": content,
					},
				},
			},
		}},
	}
	_, err := performIMRequest(ch.WebhookUrl, data)
	if err != nil {
		return err
	}
	return nil
}
