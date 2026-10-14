package minecraft

import (
	"Aether/pkg/models"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// DownloadTask represents a file to download with verification
type DownloadTask struct {
	URL  string
	Path string
	SHA1 string
	Size int64
}

// DownloadBatch downloads a list of files concurrently with progress updates
func DownloadBatch(
	tasks []DownloadTask,
	concurrency int,
	instanceID string,
	stage string,
	progressFn func(models.ProgressEvent),
) error {
	if len(tasks) == 0 {
		return nil
	}

	if concurrency <= 0 {
		concurrency = 16
	}

	total := len(tasks)
	var completed int64
	var downloadedBytes int64

	startTime := time.Now()
	taskChan := make(chan DownloadTask, total)
	for _, t := range tasks {
		taskChan <- t
	}
	close(taskChan)

	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	client := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 30,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				// Check if file already exists with matching SHA1
				if task.SHA1 != "" && verifyFileSHA1(task.Path, task.SHA1) {
					c := atomic.AddInt64(&completed, 1)
					if progressFn != nil && c%5 == 0 {
						sendProgress(progressFn, instanceID, stage, task.Path, c, total, downloadedBytes, startTime)
					}
					continue
				}

				n, err := downloadSingleFile(client, task)
				if err != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("lỗi tải file %s: %w", filepath.Base(task.Path), err)
					})
					return
				}

				atomic.AddInt64(&downloadedBytes, n)
				c := atomic.AddInt64(&completed, 1)

				if progressFn != nil {
					sendProgress(progressFn, instanceID, stage, filepath.Base(task.Path), c, total, downloadedBytes, startTime)
				}
			}
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return firstErr
	}

	// Final progress 100%
	if progressFn != nil {
		sendProgress(progressFn, instanceID, stage, "Hoàn tất", int64(total), total, downloadedBytes, startTime)
	}

	return nil
}

func sendProgress(
	fn func(models.ProgressEvent),
	instanceID, stage, currentItem string,
	completed int64,
	total int,
	downloadedBytes int64,
	startTime time.Time,
) {
	elapsed := time.Since(startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(downloadedBytes) / 1024.0 / elapsed
	}

	pct := (float64(completed) / float64(total)) * 100.0
	if pct > 100 {
		pct = 100
	}

	fn(models.ProgressEvent{
		InstanceID:     instanceID,
		Stage:          stage,
		Message:        fmt.Sprintf("Đang xử lý (%d/%d)", completed, total),
		Percentage:     pct,
		CurrentItem:    currentItem,
		CompletedItems: int(completed),
		TotalItems:     total,
		SpeedKbps:      speed,
	})
}

func downloadSingleFile(client *http.Client, task DownloadTask) (int64, error) {
	_ = os.MkdirAll(filepath.Dir(task.Path), 0755)

	resp, err := client.Get(task.URL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	tmpFile := task.Path + ".part"
	out, err := os.Create(tmpFile)
	if err != nil {
		return 0, err
	}

	hasher := sha1.New()
	writer := io.MultiWriter(out, hasher)

	n, err := io.Copy(writer, resp.Body)
	_ = out.Close()
	if err != nil {
		_ = os.Remove(tmpFile)
		return 0, err
	}

	// Check SHA1 if expected
	if task.SHA1 != "" {
		computedSHA1 := hex.EncodeToString(hasher.Sum(nil))
		if computedSHA1 != task.SHA1 {
			_ = os.Remove(tmpFile)
			return 0, fmt.Errorf("sai SHA-1 hash (nhận: %s, kỳ vọng: %s)", computedSHA1, task.SHA1)
		}
	}

	_ = os.Rename(tmpFile, task.Path)
	return n, nil
}

func verifyFileSHA1(filePath, expectedSHA1 string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	hasher := sha1.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return false
	}

	return hex.EncodeToString(hasher.Sum(nil)) == expectedSHA1
}
