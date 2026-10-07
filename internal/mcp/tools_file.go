package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerFileTools 注册所有文件与脚本管理相关工具
func registerFileTools(s *server.MCPServer, deps *Deps) {
	registerGetFileTreeTool(s, deps)
	registerReadScriptTool(s, deps)
	registerSaveScriptTool(s, deps)
	registerSearchScriptsTool(s, deps)
	registerDeleteScriptTool(s, deps)
}

// 1. get_file_tree
func registerGetFileTreeTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("get_file_tree",
		mcp.WithDescription("查询脚本目录下的文件与文件夹列表（单层直接子项）"),
		mcp.WithString("path", mcp.Description("相对脚本目录的路径，留空代表脚本根目录")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetFileTree(ctx, deps, req)
	})
}

func handleGetFileTree(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	subPath := req.GetString("path", "")
	nodes, err := deps.FileService.GetFileTree(subPath)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	data, _ := json.MarshalIndent(nodes, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// 2. read_script
func registerReadScriptTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("read_script",
		mcp.WithDescription("读取指定脚本文件的文本内容"),
		mcp.WithString("path", mcp.Required(), mcp.Description("相对于脚本目录的文件路径（如 'test.js' 或 'myrepo/task.py'）")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleReadScript(ctx, deps, req)
	})
}

func handleReadScript(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 path"), nil
	}

	vo, err := deps.FileService.GetFileContent(filePath)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if vo.IsBinary {
		return mcp.NewToolResultError("目标文件为二进制文件，不支持文本读取"), nil
	}

	return mcp.NewToolResultText(vo.Content), nil
}

// 3. save_script
func registerSaveScriptTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("save_script",
		mcp.WithDescription("新建或保存脚本文件内容"),
		mcp.WithString("path", mcp.Required(), mcp.Description("相对于脚本目录的文件路径")),
		mcp.WithString("content", mcp.Required(), mcp.Description("待写入的脚本文件文本内容")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSaveScript(ctx, deps, req)
	})
}

func handleSaveScript(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 path"), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 content"), nil
	}

	if err := deps.FileService.SaveFileContent(filePath, content); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("成功保存文件: %s", filePath)), nil
}

// 4. search_scripts
func registerSearchScriptsTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("search_scripts",
		mcp.WithDescription("在脚本目录下根据关键字模糊搜索文件"),
		mcp.WithString("keyword", mcp.Required(), mcp.Description("文件名关键字")),
		mcp.WithInteger("limit", mcp.Description("最多返回结果数量，默认 50，最大 200")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSearchScripts(ctx, deps, req)
	})
}

func handleSearchScripts(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword, err := req.RequireString("keyword")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 keyword"), nil
	}
	limit := req.GetInt("limit", 50)
	if limit < 1 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}

	nodes, err := deps.FileService.SearchFiles(keyword, limit, true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	data, _ := json.MarshalIndent(nodes, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// 5. delete_script
func registerDeleteScriptTool(s *server.MCPServer, deps *Deps) {
	s.AddTool(mcp.NewTool("delete_script",
		mcp.WithDescription("删除指定脚本文件或文件夹"),
		mcp.WithString("path", mcp.Required(), mcp.Description("相对于脚本目录的文件或文件夹路径")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDeleteScript(ctx, deps, req)
	})
}

func handleDeleteScript(_ context.Context, deps *Deps, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("缺少必填参数 path"), nil
	}

	if err := deps.FileService.DeleteFile(filePath); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("成功删除文件: %s", filePath)), nil
}
