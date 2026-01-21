package constant

import (
	"bufio"
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
		scanner := bufio.NewScanner(bytes.NewReader(sentenceData))
		var offset int64 = 0
		for scanner.Scan() {
			lineOffsets = append(lineOffsets, offset)
			offset += int64(len(scanner.Bytes())) + 1 // +1 for newline
			lineCount++
		}
	})
}

// GetRandomSentence 随机获取一条古诗词
func GetRandomSentence() string {
	initSentences()
	if lineCount == 0 {
		return "欢迎使用白虎面板"
	}

	targetLine := rand.Intn(lineCount)

	scanner := bufio.NewScanner(bytes.NewReader(sentenceData))
	currentLine := 0
	for scanner.Scan() {
		if currentLine == targetLine {
			var s Sentence
			if err := json.Unmarshal(scanner.Bytes(), &s); err == nil && s.Name != "" {
				if s.From != "" {
					return "\"" + s.Name + "\"—— " + s.From
				}
				return s.Name
			}
			break
		}
		currentLine++
	}

	return "欢迎使用白虎面板"
}
