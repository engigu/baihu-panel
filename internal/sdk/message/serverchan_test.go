package message

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerChan_Request(t *testing.T) {
	// 模拟 ServerChan Turbo 成功响应
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm error: %v", err)
		}
		if r.FormValue("title") != "测试标题" {
			t.Errorf("Expected title=测试标题, got %s", r.FormValue("title"))
		}
		if r.FormValue("desp") != "测试内容" {
			t.Errorf("Expected desp=测试内容, got %s", r.FormValue("desp"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"","data":{"pushid":"123","error":"SUCCESS"}}`))
	}))
	defer ts.Close()

	sc := &ServerChan{
		SendKey: "SCT123456",
		APIURL:  ts.URL + "/{sendkey}.send",
	}

	res, err := sc.Request("测试标题", "测试内容")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if res == "" {
		t.Fatalf("Expected non-empty response")
	}
}

func TestServerChan_Request_Error(t *testing.T) {
	// 模拟 ServerChan 错误响应
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":40001,"message":"bad sendkey","data":{"error":"INVALID_KEY"}}`))
	}))
	defer ts.Close()

	sc := &ServerChan{
		SendKey: "INVALID_KEY",
		APIURL:  ts.URL + "/{sendkey}.send",
	}

	_, err := sc.Request("测试标题", "测试内容")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}
