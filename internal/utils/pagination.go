package utils

import (
	"reflect"
	"strconv"

	"github.com/engigu/baihu-panel/internal/cache"
	"github.com/engigu/baihu-panel/internal/constant"

	"github.com/gin-gonic/gin"
)

// ToInt 解析字符串为整数，如果解析失败则返回默认值
func ToInt(s string, defaultVal int) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// ParseInt 解析字符串为整数
func ParseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// Pagination 分页参数
type Pagination struct {
	Page     int
	PageSize int
}

// getDefaultPageSize 从缓存获取默认分页大小
func getDefaultPageSize() int {
	pageSizeStr := cache.GetSiteCache(constant.KeyPageSize)
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		return 10
	}
	return pageSize
}

// ParsePagination 从请求中解析分页参数
func ParsePagination(c *gin.Context) Pagination {
	defaultPageSize := getDefaultPageSize()
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(defaultPageSize)))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 10000 {
		pageSize = defaultPageSize
	}

	return Pagination{Page: page, PageSize: pageSize}
}

// Offset 计算偏移量
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// PaginationData 分页数据
type PaginationData struct {
	Data     interface{} `json:"data"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// ensureNonNilSlice 将 nil 切片统一归一化为空切片，避免 json.Marshal 序列化为 null
func ensureNonNilSlice(data interface{}) interface{} {
	if data == nil {
		return []interface{}{}
	}
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Slice && val.IsNil() {
		return []interface{}{}
	}
	return data
}

// PaginatedResponse 分页响应
func PaginatedResponse(c *gin.Context, data interface{}, total int64, p Pagination) {
	Success(c, PaginationData{
		Data:     ensureNonNilSlice(data),
		Total:    total,
		Page:     p.Page,
		PageSize: p.PageSize,
	})
}
