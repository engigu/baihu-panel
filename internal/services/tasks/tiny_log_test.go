package tasks

import (
	"bytes"
	"testing"
)

func TestTinyLog_UTF8Splitting(t *testing.T) {
	tl, err := NewTinyLog("test-utf8", nil)
	if err != nil {
		t.Fatalf("Failed to create TinyLog: %v", err)
	}
	defer tl.Close()

	// "你好" in UTF-8: E4 BD A0, E5 a5 bd
	part1 := []byte{0xE4, 0xBD}       // Partial "你"
	part2 := []byte{0xA0, 0xE5, 0xA5} // Rest of "你", partial "好"
	part3 := []byte{0xBD, '\n'}       // Rest of "好", newline

	_, _ = tl.Write(part1)
	if len(tl.remainder) != 2 {
		t.Errorf("Expected remainder len 2, got %d", len(tl.remainder))
	}

	_, _ = tl.Write(part2)
	// Currently it should collect both parts but still no newline,
	// so remainder should be 5 bytes.
	if len(tl.remainder) != 5 {
		t.Errorf("Expected remainder len 5, got %d", len(tl.remainder))
	}

	_, _ = tl.Write(part3)
	if len(tl.remainder) != 0 {
		t.Errorf("Expected remainder len 0 after newline, got %d", len(tl.remainder))
	}

	// Read and verify
	data, err := tl.ReadLastLines(1)
	if err != nil {
		t.Fatalf("ReadLastLines failed: %v", err)
	}
	if !bytes.Contains(data, []byte("你好")) {
		t.Errorf("Expected output to contain '你好', got %q", data)
	}
}

func TestTinyLog_CarriageReturn(t *testing.T) {
	tl, err := NewTinyLog("test-cr", nil)
	if err != nil {
		t.Fatalf("Failed to create TinyLog: %v", err)
	}
	defer tl.Close()

	input := []byte("progress: 50%\rprogress: 100%\r\n")
	_, _ = tl.Write(input)

	data, err := tl.ReadLastLines(10)
	if err != nil {
		t.Fatalf("ReadLastLines failed: %v", err)
	}

	// Should contain both progress lines (or at least be split correctly)
	if !bytes.Contains(data, []byte("progress: 50%")) {
		t.Errorf("Expected output to contain 'progress: 50%%', got %q", data)
	}
	if !bytes.Contains(data, []byte("progress: 100%")) {
		t.Errorf("Expected output to contain 'progress: 100%%', got %q", data)
	}
}

func TestTinyLog_LongLineCut(t *testing.T) {
	tl, err := NewTinyLog("test-long", nil)
	if err != nil {
		t.Fatalf("Failed to create TinyLog: %v", err)
	}
	defer tl.Close()

	// Create a buffer of maxLogBufferLen-1 bytes with a multi-byte character at the maxLogBufferLen boundary
	// We want to ensure it doesn't cut in the middle of a 3-byte char.
	longData := make([]byte, maxLogBufferLen-1)
	for i := range longData {
		longData[i] = 'A'
	}
	// "你" is E4 BD A0
	longData = append(longData, 0xE4, 0xBD, 0xA0) // This starts at index maxLogBufferLen-1.
	// Index maxLogBufferLen-1: E4
	// Index maxLogBufferLen: BD
	// Index maxLogBufferLen+1: A0
	// If we cut at maxLogBufferLen, we split E4 and BD.

	_, _ = tl.Write(longData)

	// Since it's > maxLogBufferLen and no newline, it should trigger the cut.
	// Our logic finds the last safe boundary before maxLogBufferLen.
	// RuneStart(E4) at maxLogBufferLen-1 is true. FullRune(E4 at maxLogBufferLen-1 in payload[:maxLogBufferLen]) is false.
	// So lastSafe should be maxLogBufferLen-1.

	// The first maxLogBufferLen-1 bytes (all 'A') should be processed.
	// The "你" should be in remainder.

	if len(tl.remainder) != 3 {
		t.Errorf("Expected remainder len 3 (the char '你'), got %d", len(tl.remainder))
	}
}
