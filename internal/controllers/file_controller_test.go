package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFileController_GetFileTree_Lazy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建临时测试目录
	tempDir, err := os.MkdirTemp("", "baihu-file-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 构建目录结构:
	// root/
	//   file1.txt
	//   dirA/
	//     subfileA.txt
	//     subDirAA/
	//       deep.txt
	//   dirB/ (empty)
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("hello"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "dirA", "subDirAA"), 0755)
	os.WriteFile(filepath.Join(tempDir, "dirA", "subfileA.txt"), []byte("sub a"), 0644)
	os.WriteFile(filepath.Join(tempDir, "dirA", "subDirAA", "deep.txt"), []byte("deep"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "dirB"), 0755)

	fc := NewFileController(tempDir)

	r := gin.New()
	r.GET("/files/tree", fc.GetFileTree)

	// 1. 测试根目录懒加载：只返回直接子项，不递归
	req, _ := http.NewRequest(http.MethodGet, "/files/tree", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp struct {
		Code int         `json:"code"`
		Data []*FileNode `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON decode error: %v", err)
	}

	// 根目录下应该有 3 项: file1.txt, dirA, dirB
	if len(resp.Data) != 3 {
		t.Fatalf("Expected 3 items in root, got %d", len(resp.Data))
	}

	var dirANode, dirBNode *FileNode
	for _, n := range resp.Data {
		if n.Name == "dirA" {
			dirANode = n
		} else if n.Name == "dirB" {
			dirBNode = n
		}
	}

	if dirANode == nil || !dirANode.IsDir || !dirANode.HasChildren {
		t.Fatalf("dirA should be a dir with HasChildren=true, got %+v", dirANode)
	}
	// 懒加载模式下，children 必须为空数组，不递归！
	if len(dirANode.Children) != 0 {
		t.Fatalf("dirA children should be empty in lazy mode, got %d", len(dirANode.Children))
	}

	if dirBNode == nil || !dirBNode.IsDir || dirBNode.HasChildren {
		t.Fatalf("dirB should be empty dir with HasChildren=false, got %+v", dirBNode)
	}

	// 2. 测试子目录懒加载：path=dirA
	req2, _ := http.NewRequest(http.MethodGet, "/files/tree?path=dirA", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w2.Code)
	}

	var resp2 struct {
		Code int         `json:"code"`
		Data []*FileNode `json:"data"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("JSON decode error: %v", err)
	}

	// dirA 目录下应该有 2 项: subfileA.txt, subDirAA
	if len(resp2.Data) != 2 {
		t.Fatalf("Expected 2 items in dirA, got %d", len(resp2.Data))
	}

	for _, n := range resp2.Data {
		if n.Name == "subDirAA" {
			if !n.IsDir || !n.HasChildren {
				t.Fatalf("subDirAA should have children, got %+v", n)
			}
			if len(n.Children) != 0 {
				t.Fatalf("subDirAA children should be empty, got %d", len(n.Children))
			}
		}
	}
}

func TestFileController_SearchFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "baihu-file-search-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建多层文件
	os.MkdirAll(filepath.Join(tempDir, "pkg", "auth"), 0755)
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tempDir, "pkg", "main_test.go"), []byte("package pkg"), 0644)
	os.WriteFile(filepath.Join(tempDir, "pkg", "auth", "main_auth.go"), []byte("package auth"), 0644)
	os.WriteFile(filepath.Join(tempDir, "readme.md"), []byte("# readme"), 0644)

	fc := NewFileController(tempDir)
	r := gin.New()
	r.GET("/files/search", fc.SearchFiles)

	// 搜索 "main"
	req, _ := http.NewRequest(http.MethodGet, "/files/search?keyword=main&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp struct {
		Code int         `json:"code"`
		Data []*FileNode `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON decode error: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Fatalf("Expected 3 files matching 'main', got %d", len(resp.Data))
	}

	// 验证 limit 生效
	req2, _ := http.NewRequest(http.MethodGet, "/files/search?keyword=main&limit=1", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp2 struct {
		Code int         `json:"code"`
		Data []*FileNode `json:"data"`
	}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if len(resp2.Data) != 1 {
		t.Fatalf("Expected 1 file with limit=1, got %d", len(resp2.Data))
	}
}

