package message

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type VoceChat struct {
	Server     string
	APIKey     string
	TargetType string // "user" or "group"
	TargetID   string
}

func (v *VoceChat) Request(title, content string) ([]byte, error) {
	if v.Server == "" || v.APIKey == "" || v.TargetID == "" {
		return nil, fmt.Errorf("vocechat config missing: server, api_key and target_id are required")
	}

	server := strings.TrimSuffix(v.Server, "/")
	endpoint := "send_to_user"
	if v.TargetType == "group" {
		endpoint = "send_to_group"
	}

	url := fmt.Sprintf("%s/api/bot/%s/%s", server, endpoint, v.TargetID)

	// Use text/plain for now as requested
	body := content
	if title != "" {
		body = fmt.Sprintf("%s\n\n%s", title, content)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("x-api-key", v.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return respBody, fmt.Errorf("vocechat response error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
