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
	var totalBytes int64
	for _, t := range tasks {
		totalBytes += t.Size
	}

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
					if task.Size > 0 {
						atomic.AddInt64(&downloadedBytes, task.Size)
					}
					if progressFn != nil {
						sendProgress(progressFn, instanceID, stage, filepath.Base(task.Path), c, total, atomic.LoadInt64(&downloadedBytes), totalBytes, startTime)
					}
					continue
				}

				onChunk := func(n int64) {
					curr := atomic.AddInt64(&downloadedBytes, n)
					if progressFn != nil {
						c := atomic.LoadInt64(&completed)
						sendProgress(progressFn, instanceID, stage, filepath.Base(task.Path), c, total, curr, totalBytes, startTime)
					}
				}

				n, err := downloadSingleFileWithProgress(client, task, onChunk)
				if err != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("lỗi tải file %s: %w", filepath.Base(task.Path), err)
					})
					return
				}

				c := atomic.AddInt64(&completed, 1)
				if progressFn != nil {
					sendProgress(progressFn, instanceID, stage, filepath.Base(task.Path), c, total, atomic.LoadInt64(&downloadedBytes), totalBytes, startTime)
				}
				_ = n
			}
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return firstErr
	}

	// Final progress 100%
	if progressFn != nil {
		sendProgress(progressFn, instanceID, stage, "Hoàn tất", int64(total), total, downloadedBytes, totalBytes, startTime)
	}

	return nil
}

func sendProgress(
	fn func(models.ProgressEvent),
	instanceID, stage, currentItem string,
	completed int64,
	total int,
	downloadedBytes int64,
	totalBytes int64,
	startTime time.Time,
) {
	elapsed := time.Since(startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(downloadedBytes) / 1024.0 / elapsed
	}

	var pct float64
	if totalBytes > 0 {
		pct = (float64(downloadedBytes) / float64(totalBytes)) * 100.0
	} else if total > 0 {
		pct = (float64(completed) / float64(total)) * 100.0
	}
	if pct > 100 {
		pct = 100
	}

	msg := fmt.Sprintf("Đang xử lý (%d/%d)", completed, total)
	if totalBytes > 0 && downloadedBytes > 0 {
		msg = fmt.Sprintf("%.1f / %.1f MB", float64(downloadedBytes)/(1024*1024), float64(totalBytes)/(1024*1024))
	}

	fn(models.ProgressEvent{
		InstanceID:     instanceID,
		Stage:          stage,
		Message:        msg,
		Percentage:     pct,
		CurrentItem:    currentItem,
		CompletedItems: int(completed),
		TotalItems:     total,
		SpeedKbps:      speed,
	})
}

func downloadSingleFile(client *http.Client, task DownloadTask) (int64, error) {
	return downloadSingleFileWithProgress(client, task, nil)
}

func downloadSingleFileWithProgress(client *http.Client, task DownloadTask, onChunk func(int64)) (int64, error) {
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

	buf := make([]byte, 64*1024)
	var totalRead int64
	var lastReport time.Time

	for {
		nr, rerr := resp.Body.Read(buf)
		if nr > 0 {
			nw, werr := writer.Write(buf[0:nr])
			if werr != nil {
				_ = out.Close()
				_ = os.Remove(tmpFile)
				return totalRead, werr
			}
			totalRead += int64(nw)
			if onChunk != nil && time.Since(lastReport) > 80*time.Millisecond {
				onChunk(int64(nw))
				lastReport = time.Now()
			}
		}
		if rerr != nil {
			if rerr != io.EOF {
				_ = out.Close()
				_ = os.Remove(tmpFile)
				return totalRead, rerr
			}
			break
		}
	}

	_ = out.Close()

	// Check SHA1 if expected
	if task.SHA1 != "" {
		computedSHA1 := hex.EncodeToString(hasher.Sum(nil))
		if computedSHA1 != task.SHA1 {
			_ = os.Remove(tmpFile)
			return 0, fmt.Errorf("sai SHA-1 hash (nhận: %s, kỳ vọng: %s)", computedSHA1, task.SHA1)
		}
	}

	_ = os.Rename(tmpFile, task.Path)
	return totalRead, nil
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
