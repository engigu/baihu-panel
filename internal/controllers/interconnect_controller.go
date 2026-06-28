package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/engigu/baihu-panel/internal/constant"
	"github.com/engigu/baihu-panel/internal/models"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/tunnel"
	"github.com/engigu/baihu-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type InterconnectController struct {
	interconnectService *services.InterconnectService
	httpClient          *http.Client
}

func NewInterconnectController(interconnectService *services.InterconnectService) *InterconnectController {
	return &InterconnectController{
		interconnectService: interconnectService,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetNodes 获取互联节点列表
func (ic *InterconnectController) GetNodes(c *gin.Context) {
	nodes, err := ic.interconnectService.GetNodes()
	if err != nil {
		utils.ServerError(c, "获取互联节点失败")
		return
	}
	utils.Success(c, nodes)
}

// CreateNode 创建互联节点
func (ic *InterconnectController) CreateNode(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		URL    string `json:"url"`
		Token  string `json:"token" binding:"required"`
		Remark string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	node, err := ic.interconnectService.CreateNode(req.Name, req.URL, req.Token, req.Remark)
	if err != nil {
		utils.ServerError(c, "创建互联节点失败")
		return
	}

	utils.Success(c, node)
}

// UpdateNode 更新互联节点
func (ic *InterconnectController) UpdateNode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的节点ID")
		return
	}

	var req struct {
		Name   string `json:"name" binding:"required"`
		URL    string `json:"url"`
		Token  string `json:"token" binding:"required"`
		Remark string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	node, err := ic.interconnectService.UpdateNode(id, req.Name, req.URL, req.Token, req.Remark)
	if err != nil {
		utils.ServerError(c, "更新互联节点失败")
		return
	}

	utils.Success(c, node)
}

// DeleteNode 删除互联节点
func (ic *InterconnectController) DeleteNode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.BadRequest(c, "无效的节点ID")
		return
	}

	err := ic.interconnectService.DeleteNode(id)
	if err != nil {
		utils.ServerError(c, "删除互联节点失败")
		return
	}

	utils.Success(c, nil)
}

// GetNodeStatus 获取单个子节点的状态
func (ic *InterconnectController) GetNodeStatus(c *gin.Context) {
	id := c.Param("id")
	node, err := ic.interconnectService.GetNodeByID(id)
	if err != nil {
		utils.NotFound(c, "节点不存在")
		return
	}

	// 针对反向隧道节点状态检测的特判
	if strings.HasPrefix(node.URL, "tunnel://") {
		sess := tunnel.GetSession(node.ID)
		if sess == nil {
			c.JSON(200, gin.H{"code": 500, "msg": "节点离线或反向隧道未建立", "data": nil})
			return
		}

		// 使用当前 Yamux Session 的虚拟底层连接进行拨号
		transport := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return sess.Session.Open()
			},
		}
		client := &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
		}

		req, err := http.NewRequest("GET", "http://tunnel.local/api/v1/monitor", nil)
		if err != nil {
			utils.ServerError(c, "构建检测请求失败")
			return
		}
		req.Header.Set("Authorization", "Bearer "+node.Token)

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(200, gin.H{"code": 500, "msg": "与子节点逆向连接通讯失败", "data": nil})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			c.JSON(200, gin.H{"code": 500, "msg": "子节点检测异常", "data": string(body)})
			return
		}

		var jsonResp map[string]interface{}
		if err := json.Unmarshal(body, &jsonResp); err != nil {
			utils.ServerError(c, "解析节点检测数据失败")
			return
		}

		if dataMap, ok := jsonResp["data"].(map[string]interface{}); ok {
			dataMap["tunnel_connected"] = true
			dataMap["tunnel_url"] = node.URL
			if hostMap, ok := dataMap["host"].(map[string]interface{}); ok {
				hostMap["tx_bytes"] = node.Metrics.TxBytes
				hostMap["rx_bytes"] = node.Metrics.RxBytes
			}
		}

		utils.Success(c, jsonResp["data"])
		return
	}

	apiURL := strings.TrimRight(node.URL, "/") + "/api/v1/monitor"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		utils.ServerError(c, "构建请求失败")
		return
	}
	req.Header.Set("Authorization", "Bearer "+node.Token)

	resp, err := ic.httpClient.Do(req)
	if err != nil {
		c.JSON(200, gin.H{"code": 500, "msg": "节点离线或网络不可达", "data": nil})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		c.JSON(200, gin.H{"code": 500, "msg": "节点返回异常", "data": string(body)})
		return
	}

	var jsonResp map[string]interface{}
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		utils.ServerError(c, "解析节点响应失败")
		return
	}

	utils.Success(c, jsonResp["data"])
}

// SyncScript 将脚本同步到指定的节点列表
func (ic *InterconnectController) SyncScript(c *gin.Context) {
	var req struct {
		NodeIDs  []string `json:"node_ids" binding:"required"`
		Filename string   `json:"filename" binding:"required"`
		Content  string   `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	results := make([]map[string]interface{}, 0)

	for _, nodeID := range req.NodeIDs {
		node, err := ic.interconnectService.GetNodeByID(nodeID)
		if err != nil {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": "节点不存在"})
			continue
		}

		client, apiURL, err := ic.getClientAndURL(node, "/api/v1/scripts/save")
		if err != nil {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": "反向隧道未连接"})
			continue
		}
		
		payload := map[string]interface{}{
			"filename": req.Filename,
			"content":  req.Content,
		}
		payloadBytes, _ := json.Marshal(payload)

		httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": "构建请求失败"})
			continue
		}
		httpReq.Header.Set("Authorization", "Bearer "+node.Token)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil || resp.StatusCode != 200 {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": "同步请求失败或超时"})
		} else {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": true, "msg": "同步成功"})
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	utils.Success(c, results)
}

// SyncEnv 将环境变量同步到指定的节点列表
func (ic *InterconnectController) SyncEnv(c *gin.Context) {
	var req struct {
		NodeIDs []string `json:"node_ids" binding:"required"`
		Envs    []struct{ ID string `json:"id"` } `json:"envs" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var envIDs []string
	for _, e := range req.Envs {
		envIDs = append(envIDs, e.ID)
	}

	dataService := services.NewDataService()
	exportData := dataService.ExportBusinessData(nil, envIDs)

	results := make([]map[string]interface{}, 0)

	for _, nodeID := range req.NodeIDs {
		node, err := ic.interconnectService.GetNodeByID(nodeID)
		if err != nil {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": "节点不存在"})
			continue
		}

		client, apiURL, err := ic.getClientAndURL(node, "/api/v1/system/import")
		if err != nil {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": "反向隧道未连接"})
			continue
		}

		payloadBytes, _ := json.Marshal(exportData)

		httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": "构建请求失败"})
			continue
		}
		httpReq.Header.Set("Authorization", "Bearer "+node.Token)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil || resp.StatusCode != 200 {
			msg := "同步失败"
			if err != nil {
				msg = err.Error()
			}
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": msg})
		} else {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": true, "msg": "同步成功"})
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	utils.Success(c, results)
}

// SyncTask 将任务同步到指定的节点列表
func (ic *InterconnectController) SyncTask(c *gin.Context) {
	var req struct {
		NodeIDs []string `json:"node_ids" binding:"required"`
		Tasks   []struct{ ID string `json:"id"` } `json:"tasks" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var taskIDs []string
	for _, t := range req.Tasks {
		taskIDs = append(taskIDs, t.ID)
	}

	dataService := services.NewDataService()
	exportData := dataService.ExportBusinessData(taskIDs, nil)

	results := make([]map[string]interface{}, 0)

	for _, nodeID := range req.NodeIDs {
		node, err := ic.interconnectService.GetNodeByID(nodeID)
		if err != nil {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": "节点不存在"})
			continue
		}

		client, apiURL, err := ic.getClientAndURL(node, "/api/v1/system/import")
		if err != nil {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": "反向隧道未连接"})
			continue
		}

		payloadBytes, _ := json.Marshal(exportData)

		httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": "构建请求失败"})
			continue
		}
		httpReq.Header.Set("Authorization", "Bearer "+node.Token)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil || resp.StatusCode != 200 {
			msg := "同步失败"
			if err != nil {
				msg = err.Error()
			}
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": false, "msg": msg})
		} else {
			results = append(results, map[string]interface{}{"node_id": nodeID, "success": true, "msg": "同步成功"})
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	utils.Success(c, results)
}

// HandleTunnel 接受子节点 WebSocket 连接请求
func (ic *InterconnectController) HandleTunnel(c *gin.Context) {
	tunnel.HandleTunnel(c)
}

// ProxyRequest 代理转发请求至目标节点
func (ic *InterconnectController) ProxyRequest(c *gin.Context) {
	nodeID := c.Param("node_id")
	path := c.Param("path")
	if nodeID == "" {
		utils.BadRequest(c, "Node ID required")
		return
	}

	node, err := ic.interconnectService.GetNodeByID(nodeID)
	if err != nil {
		utils.NotFound(c, "Node not found")
		return
	}

	if strings.HasPrefix(node.URL, "tunnel://") {
		// 走 WebSocket 逆向隧道 (基于 Yamux 流式多路复用)
		err := tunnel.ProxyHTTP(nodeID, c, path)
		if err != nil {
			utils.ServerError(c, "Tunnel request failed: "+err.Error())
		}
		return
	}

	// 走普通 HTTP 直连
	// Construct the target URL
	targetURL := strings.TrimRight(node.URL, "/") + path
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		utils.ServerError(c, "Failed to create proxy request")
		return
	}

	// Copy headers
	req.Header = c.Request.Header.Clone()
	
	// If the node token exists, append it as Bearer Auth
	if node.Token != "" {
		req.Header.Set("Authorization", "Bearer "+node.Token)
	}

	resp, err := ic.httpClient.Do(req)
	if err != nil {
		utils.ServerError(c, "Failed to connect to target node: "+err.Error())
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// ReportMonitorData 接收子节点上报的监控数据
func (ic *InterconnectController) ReportMonitorData(c *gin.Context) {
	var req models.NodeMetrics

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(401, gin.H{"error": "missing authorization"})
		return
	}
	tokenStr := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

	node, err := ic.interconnectService.GetNodeByToken(tokenStr)
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid token"})
		return
	}

	err = ic.interconnectService.UpdateNodeMonitorData(node.ID, req)
	if err != nil {
		utils.ServerError(c, "更新节点数据失败")
		return
	}
	utils.Success(c, gin.H{
		"tunnel_url": node.URL,
	})
}

// GetChildStatus 获取本机作为子节点的连接状态
func (ic *InterconnectController) GetChildStatus(c *gin.Context) {
	settingsSvc := services.NewSettingsService()
	parentURL := settingsSvc.Get(constant.SectionInterconnect, constant.KeyInterconnectParentURL)
	parentToken := settingsSvc.Get(constant.SectionInterconnect, constant.KeyInterconnectParentToken)

	connected := tunnel.IsTunnelConnected()
	tunnelURL := tunnel.GetLocalTunnelURL()

	utils.Success(c, gin.H{
		"parent_url":   parentURL,
		"parent_token": parentToken,
		"connected":    connected,
		"tunnel_url":   tunnelURL,
		"tx_bytes":     tunnel.GetTxBytes(),
		"rx_bytes":     tunnel.GetRxBytes(),
	})
}

// getClientAndURL 辅助方法：根据节点类型决定走直连还是隧道，并返回对应的 Client 和完整 URL
func (ic *InterconnectController) getClientAndURL(node *models.InterconnectNode, path string) (*http.Client, string, error) {
	if strings.HasPrefix(node.URL, "tunnel://") {
		sess := tunnel.GetSession(node.ID)
		if sess == nil {
			return nil, "", net.ErrClosed
		}
		transport := &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return sess.Session.Open()
			},
		}
		client := &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		}
		return client, "http://tunnel.local" + path, nil
	}

	targetURL := strings.TrimRight(node.URL, "/") + path
	return ic.httpClient, targetURL, nil
}
