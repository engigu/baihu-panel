package message

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

type CustomWebhook struct {
	Webhook string
	Body    string
}

var Client = &http.Client{
	Timeout: 5 * time.Second,
}

func (cw *CustomWebhook) Request(url string, msg string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(msg)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}
