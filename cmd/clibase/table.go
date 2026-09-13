package clibase

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
)

// Alignment 定义列的文本对齐方式
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// Column 定义表格的单列配置
type Column struct {
	Title    string    // 列标题
	Width    int       // 强制固定宽度。若 > 0 则严格使用该宽度；若 <= 0 则全自适应
	MaxWidth int       // 最大允许宽度限制。若 > 0 且自适应宽度超过该值，则截断为该值
	Align    Alignment // 文本对齐方式，默认 AlignLeft
}

// Table 基于 go-runewidth 深度驱动的现代终端表格排版组件
type Table struct {
	columns   []Column
	rows      [][]string
	separator string // 列与列之间的分隔符，默认 " | "
}

// NewTable 创建一个全新的自适应表格，只需传入表头标题字符串
// 每列宽度将根据内容及表头的实际视觉列宽全自动精准匹配计算
func NewTable(headers ...string) *Table {
	columns := make([]Column, len(headers))
	for i, h := range headers {
		columns[i] = Column{
			Title: h,
			Align: AlignLeft,
		}
	}
	return &Table{
		columns:   columns,
		rows:      make([][]string, 0),
		separator: " | ",
	}
}

// NewTableWithColumns 基于自定义 Column 结构创建表格，适用于需要精细化预设属性的场景
func NewTableWithColumns(columns ...Column) *Table {
	cols := make([]Column, len(columns))
	copy(cols, columns)
	return &Table{
		columns:   cols,
		rows:      make([][]string, 0),
		separator: " | ",
	}
}

// SetSeparator 自定义列分隔符
func (t *Table) SetSeparator(sep string) *Table {
	t.separator = sep
	return t
}

// SetAlign 设置指定列索引的对齐方式（如 AlignRight 适合数值/耗时等）
func (t *Table) SetAlign(colIndex int, align Alignment) *Table {
	if colIndex >= 0 && colIndex < len(t.columns) {
		t.columns[colIndex].Align = align
	}
	return t
}

// SetWidth 设置指定列的强制固定宽度
func (t *Table) SetWidth(colIndex int, width int) *Table {
	if colIndex >= 0 && colIndex < len(t.columns) {
		t.columns[colIndex].Width = width
	}
	return t
}

// SetMaxWidth 设置指定列的最大宽度限制（防止超长文本撑破终端屏幕）
func (t *Table) SetMaxWidth(colIndex int, maxWidth int) *Table {
	if colIndex >= 0 && colIndex < len(t.columns) {
		t.columns[colIndex].MaxWidth = maxWidth
	}
	return t
}

// AddRow 追加一行数据，支持任意类型输入（自动转换为字符串），无需显式转换
func (t *Table) AddRow(cells ...any) *Table {
	row := make([]string, len(cells))
	for i, c := range cells {
		if s, ok := c.(string); ok {
			row[i] = s
		} else {
			row[i] = fmt.Sprint(c)
		}
	}
	t.rows = append(t.rows, row)
	return t
}

// RowCount 返回当前数据行总数
func (t *Table) RowCount() int {
	return len(t.rows)
}

// ResetRows 清空已添加的数据行
func (t *Table) ResetRows() *Table {
	t.rows = make([][]string, 0)
	return t
}

// FormatCell 根据指定的视觉宽度和对齐方式，精准格式化单元格文本内容
func FormatCell(text string, targetWidth int, align Alignment) string {
	if targetWidth <= 0 {
		return text
	}

	w := runewidth.StringWidth(text)
	if w > targetWidth {
		text = runewidth.Truncate(text, targetWidth, "..")
		w = runewidth.StringWidth(text)
	}

	pad := targetWidth - w
	if pad <= 0 {
		return text
	}

	switch align {
	case AlignRight:
		return strings.Repeat(" ", pad) + text
	case AlignCenter:
		leftPad := pad / 2
		rightPad := pad - leftPad
		return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
	case AlignLeft:
		fallthrough
	default:
		return text + strings.Repeat(" ", pad)
	}
}

// computeEffectiveWidths 计算各列的实际生效视觉宽度
func (t *Table) computeEffectiveWidths() []int {
	widths := make([]int, len(t.columns))
	for i, col := range t.columns {
		if col.Width > 0 {
			widths[i] = col.Width
			continue
		}

		// 全自动自适应：取标题与各行该列的最大宽度
		maxW := runewidth.StringWidth(col.Title)
		for _, row := range t.rows {
			if i < len(row) {
				w := runewidth.StringWidth(row[i])
				if w > maxW {
					maxW = w
				}
			}
		}

		// 如果设置了 MaxWidth 上限约束
		if col.MaxWidth > 0 && maxW > col.MaxWidth {
			maxW = col.MaxWidth
		}

		if maxW <= 0 {
			maxW = 1
		}
		widths[i] = maxW
	}
	return widths
}

// computeTotalVisualWidth 计算包含分隔符在内的单行实际总视觉宽度
func (t *Table) computeTotalVisualWidth(colWidths []int) int {
	if len(colWidths) == 0 {
		return 0
	}
	total := 0
	for _, w := range colWidths {
		total += w
	}
	if len(colWidths) > 1 {
		sepWidth := runewidth.StringWidth(t.separator)
		total += (len(colWidths) - 1) * sepWidth
	}
	return total
}

// String 将整个表格完整渲染为终端格式化字符串
func (t *Table) String() string {
	if len(t.columns) == 0 {
		return ""
	}

	colWidths := t.computeEffectiveWidths()
	totalWidth := t.computeTotalVisualWidth(colWidths)

	var sb strings.Builder
	borderOuter := strings.Repeat("=", totalWidth)
	borderInner := strings.Repeat("-", totalWidth)

	// 1. 顶部分隔线
	sb.WriteString(borderOuter)
	sb.WriteString("\n")

	// 2. 表头
	headers := make([]string, len(t.columns))
	for i, col := range t.columns {
		headers[i] = FormatCell(col.Title, colWidths[i], col.Align)
	}
	sb.WriteString(strings.Join(headers, t.separator))
	sb.WriteString("\n")

	// 3. 表头与数据分隔线
	sb.WriteString(borderInner)
	sb.WriteString("\n")

	// 4. 数据行
	for _, row := range t.rows {
		formattedRow := make([]string, len(t.columns))
		for i, col := range t.columns {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			formattedRow[i] = FormatCell(cell, colWidths[i], col.Align)
		}
		sb.WriteString(strings.Join(formattedRow, t.separator))
		sb.WriteString("\n")
	}

	// 5. 底部分隔线
	sb.WriteString(borderOuter)

	return sb.String()
}

// Render 将格式化后的表格直接输出至标准输出 os.Stdout
func (t *Table) Render() {
	_ = t.RenderTo(os.Stdout)
}

// RenderTo 将格式化后的表格输出至指定的 io.Writer
func (t *Table) RenderTo(w io.Writer) error {
	str := t.String()
	if str == "" {
		return nil
	}
	_, err := fmt.Fprintln(w, str)
	return err
}
