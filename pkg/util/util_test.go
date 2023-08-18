package util

import (
	"io"
	"os"
	"testing"
)

func TestAppendStringToFile(t *testing.T) {
	tempDir := t.TempDir()
	newFilePath := tempDir + "/new.txt"
	existingFilePath := tempDir + "/existing.txt"
	existingFile, err := os.Create(existingFilePath)
	if err != nil {
		t.Errorf("error creating & opening file for test file: %s", err.Error())
	}
	io.WriteString(existingFile, "existing content")

	var tests = []struct {
		name    string
		path    string
		content string
		want    string
	}{
		{"NewFile", newFilePath, "some content", "some content"},
		{"FileCreatedByFunction", newFilePath, "some more content", "some contentsome more content"},
		{"ExistingFile", existingFilePath, "appended content", "existing contentappended content"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AppendStringToFile(tt.path, tt.content)
			if err != nil {
				t.Errorf("got %s, want %s", err.Error(), tt.want)
			}
			file, err := os.OpenFile(tt.path, os.O_RDONLY, 0777)
			if err != nil {
				t.Errorf("error opening targetted file: %s", err.Error())
			}
			content, err := io.ReadAll(file)
			if err != nil {
				t.Errorf("error reading targetted file: %s", err.Error())
			}
			strContent := string(content)
			if strContent != tt.want {
				t.Errorf("got %s, want %s", strContent, tt.want)
			}
		})
	}
}

func TestRandString(t *testing.T) {
	t.Run("TestRandStringLength", func(t *testing.T) {
		lengths := []uint{0, 1, 10, 100, 1000}
		for _, length := range lengths {
			result := RandString(length)
			if len(result) != int(length) {
				t.Errorf("Expected string of length %d, but got length %d", length, len(result))
			}
		}
	})

	t.Run("TestRandStringCharacters", func(t *testing.T) {
		for i := 1; i <= 1000; i *= 10 {
			result := RandString(uint(i))
			for _, char := range result {
				if !isValidCharacter(char) {
					t.Errorf("Invalid character found in the random string: %c", char)
				}
			}
		}
	})
}

func isValidCharacter(char rune) bool {
	for _, validChar := range letterBytes {
		if rune(validChar) == char {
			return true
		}
	}
	return false
}

func BenchmarkRandString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandString(10)
	}
}

func FuzzRandString(f *testing.F) {
	f.Fuzz(func(t *testing.T, a uint) {
		t.Logf("%d: %s\n", a, RandString(a))
	})
}
