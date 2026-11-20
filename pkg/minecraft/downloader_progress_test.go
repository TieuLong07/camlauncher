package minecraft

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"Aether/pkg/models"
)

func TestDownloadBatchReportsAllBytes(t *testing.T) {
	content := bytes.Repeat([]byte("12345678"), 128*1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(content) }))
	defer server.Close()
	var last models.ProgressEvent
	dest := filepath.Join(t.TempDir(), "test.bin")
	err := DownloadBatch([]DownloadTask{{URL: server.URL, Path: dest, Size: int64(len(content))}}, 1, "test", "download", func(event models.ProgressEvent) {
		last = event
	})
	if err != nil {
		t.Fatal(err)
	}
	if last.Percentage != 100 || last.CompletedItems != 1 {
		t.Fatalf("unexpected final progress: %+v", last)
	}
}

func TestDownloadBatchRejectsWrongSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("incomplete")) }))
	defer server.Close()
	dest := filepath.Join(t.TempDir(), "test.bin")
	err := DownloadBatch([]DownloadTask{{URL: server.URL, Path: dest, Size: 100}}, 1, "test", "download", nil)
	if err == nil {
		t.Fatal("expected size mismatch")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("incomplete destination exists: %v", err)
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatalf("partial file exists: %v", err)
	}
}
