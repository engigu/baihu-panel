package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/engigu/baihu-panel/internal/controllers"
	"github.com/engigu/baihu-panel/internal/services/app"
	"github.com/gin-gonic/gin"
)

func TestOpenAPIAppRoutes_RegistrationAndAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	appController := controllers.NewAppController(app.DefaultAppService)
	ctrls := &Controllers{
		App: appController,
	}

	root := engine.Group("")
	initOpenAPIV1Routes(root, ctrls)

	// 1. 测试未授权访问返回 401
	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/open2api/v1/apps"},
		{http.MethodGet, "/open2api/v1/apps/store"},
		{http.MethodGet, "/open2api/v1/apps/updates"},
		{http.MethodGet, "/open2api/v1/apps/test-app-id"},
		{http.MethodPost, "/open2api/v1/apps/apply"},
		{http.MethodPost, "/open2api/v1/apps/test-app-id/update"},
		{http.MethodPost, "/open2api/v1/apps/test-app-id/rollback"},
		{http.MethodPost, "/open2api/v1/apps/test-app-id/auto-update"},
		{http.MethodPost, "/open2api/v1/apps/test-app-id/switch"},
		{http.MethodPost, "/open2api/v1/apps/test-app-id/rebuild"},
		{http.MethodDelete, "/open2api/v1/apps/test-app-id"},
	}

	for _, ep := range endpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(ep.method, ep.path, nil)
		engine.ServeHTTP(w, req)

		// 未携带 token 应被 OpenapiRequired 中间件拦截并返回 401
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Endpoint %s %s expected 401 Unauthorized, got %d", ep.method, ep.path, w.Code)
		}
	}
}

func TestOpenAPIAppRoutes_RegisteredRoutesList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	appController := controllers.NewAppController(app.DefaultAppService)
	ctrls := &Controllers{
		App: appController,
	}

	root := engine.Group("")
	initOpenAPIV1Routes(root, ctrls)

	expectedRoutes := map[string]string{
		"GET:/open2api/v1/apps":                  "GetApps",
		"GET:/open2api/v1/apps/store":            "GetMarketplace",
		"GET:/open2api/v1/apps/updates":          "GetAppUpdates",
		"GET:/open2api/v1/apps/:id":              "GetApp",
		"POST:/open2api/v1/apps/apply":           "ApplyApp",
		"POST:/open2api/v1/apps/:id/update":      "UpdateApp",
		"POST:/open2api/v1/apps/:id/rollback":    "RollbackApp",
		"POST:/open2api/v1/apps/:id/auto-update": "SetAppAutoUpdate",
		"POST:/open2api/v1/apps/:id/switch":      "SwitchScenario",
		"POST:/open2api/v1/apps/:id/rebuild":     "RebuildApp",
		"DELETE:/open2api/v1/apps/:id":           "RemoveApp",
	}

	routes := engine.Routes()
	registered := make(map[string]bool)
	for _, r := range routes {
		registered[r.Method+":"+r.Path] = true
	}

	for expKey := range expectedRoutes {
		if !registered[expKey] {
			t.Errorf("Route %s must be registered in OpenAPI v1", expKey)
		}
	}
}
