package constant

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"math/rand"
	"sync"
)

//go:embed sentence1-10000.json
var sentenceData []byte

type Sentence struct {
	Name string `json:"name"`
	From string `json:"from"`
}

var (
	lineCount    int
	lineOffsets  []int64
	sentenceOnce sync.Once
)

// initSentences 初始化：统计行数和记录每行偏移
func initSentences() {
	sentenceOnce.Do(func() {
		data := sentenceData
		var offset int64 = 0
		for {
			// 记录当前行的起始位置
			lineOffsets = append(lineOffsets, offset)
			lineCount++

			// 寻找下一个换行符
			idx := bytes.IndexByte(data[offset:], '\n')
			if idx == -1 {
				// 最后一行没有换行符
				break
			}

			// 移动到下一行的起始位置
			offset += int64(idx) + 1
			if offset >= int64(len(data)) {
				break
			}
		}
	})
}

// GetRandomSentence 随机获取一条古诗词
func GetRandomSentence() string {
	initSentences()
	if lineCount <= 0 {
		return "欢迎使用白虎面板"
	}

	targetIndex := rand.Intn(lineCount)
	start := lineOffsets[targetIndex]

	// 确定当前行的结束位置
	var end int64
	if targetIndex < lineCount-1 {
		end = lineOffsets[targetIndex+1]
	} else {
		end = int64(len(sentenceData))
	}

	// 提取行并清理两端的空白字符（包括 \r, \n）
	line := bytes.TrimSpace(sentenceData[start:end])
	if len(line) == 0 {
		return "欢迎使用白虎面板"
	}

	var sData []string
	if err := json.Unmarshal(line, &sData); err == nil && len(sData) >= 1 {
		name := sData[0]
		from := ""
		if len(sData) >= 2 {
			from = sData[1]
		}

		if from != "" {
			return "\"" + name + "\"—— " + from
		}
		return name
	}

	return "欢迎使用白虎面板"
}
