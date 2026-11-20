//go:build windows

package sync

import (
	"golang.org/x/sys/windows"
	"os"
)

func lockFile(file *os.File) (func(), error) {
	var overlapped windows.Overlapped
	if err := windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &overlapped); err != nil {
		return nil, err
	}
	return func() { _ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &overlapped) }, nil
}
