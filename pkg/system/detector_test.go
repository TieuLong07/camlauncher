package system

import (
	"testing"
)

func TestDetectJavaRuntimes(t *testing.T) {
	rts := DetectJavaRuntimes()
	t.Logf("Found %d runtimes:", len(rts))
	for _, rt := range rts {
		t.Logf("  Java %d (%s) 64-bit=%v default=%v at %s", rt.MajorVersion, rt.Version, rt.Is64Bit, rt.IsDefault, rt.Path)
	}
}
