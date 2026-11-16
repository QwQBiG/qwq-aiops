package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type TelegramNotificationService struct {
	token, chatID, endpoint string
	httpClient              *http.Client
}

func NewTelegramNotificationService(token, chatID string) *TelegramNotificationService {
	return &TelegramNotificationService{
		token: token, chatID: chatID,
		endpoint:   "https://api.telegram.org/bot" + token + "/sendMessage",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TelegramNotificationService) ValidateConfig() error {
	if t.token == "" || t.chatID == "" {
		return fmt.Errorf("Telegram token and chat ID must both be configured")
	}
	return nil
}

func (t *TelegramNotificationService) SendAlertContext(parent context.Context, title, content string) error {
	if err := t.ValidateConfig(); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]string{
		"chat_id": t.chatID,
		"text":    title + "\n\n" + content,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(payload))
	if err != nil {
		return notificationRequestError(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return notificationRequestError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		OK          bool   `json:"ok"`
		Code        int    `json:"error_code"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&result); err != nil {
		return fmt.Errorf("invalid Telegram response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("Telegram rejected notification (%d): %s", result.Code, result.Description)
	}
	return nil
}

// Transport errors often include the complete request URL and its credentials.
func notificationRequestError(err error) error {
	var requestErr *url.Error
	if errors.As(err, &requestErr) {
		err = requestErr.Err
	}
	return fmt.Errorf("notification request failed: %w", err)
}
