package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type PushPlus struct {
	Token       string `json:"token"`
	Topic       string `json:"topic,omitempty"`
	Template    string `json:"template,omitempty"`
	Channel     string `json:"channel,omitempty"`
	Webhook     string `json:"webhook,omitempty"`
	CallbackUrl string `json:"callbackUrl,omitempty"`
	To          string `json:"to,omitempty"`
}

type pushPlusData struct {
	PushPlus
	Title   string `json:"title"`
	Content string `json:"content"`
}

type pushPlusResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data"`
}

func (p *PushPlus) Request(title, content string) (string, error) {
	url := "https://www.pushplus.plus/send"

	data := pushPlusData{
		PushPlus: *p,
		Title:    title,
		Content:  content,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		// Try old URL if first one fails or as fallback
		urlOld := "http://pushplus.hxtrip.com/send"
		resp, err = http.Post(urlOld, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return "", err
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res pushPlusResponse
	if err := json.Unmarshal(respBody, &res); err != nil {
		return string(respBody), err
	}

	if res.Code == 200 {
		return string(respBody), nil
	}

	return string(respBody), fmt.Errorf("PushPlus error: %s (code: %d)", res.Msg, res.Code)
}
