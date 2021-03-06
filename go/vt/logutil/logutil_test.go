package logutil

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

func TestParsing(t *testing.T) {
	ts, err := parseTimestamp("/tmp/something.foo/zkocc.goedel.szopa.log.INFO.20130806-151006.10530")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if want := time.Date(2013, 8, 6, 15, 10, 06, 0, time.UTC); ts != want {
		t.Errorf("timestamp: want %v, got %v", want, ts)
	}
}

func TestPurge(t *testing.T) {
	logDir := path.Join(os.TempDir(), fmt.Sprintf("%v-%v", os.Args[0], os.Getpid()))
	if err := os.MkdirAll(logDir, 0777); err != nil {
		t.Fatalf("os.MkdirAll: %v", err)
	}
	defer os.RemoveAll(logDir)

	now := time.Date(2013, 8, 6, 15, 10, 06, 0, time.UTC)
	files := []string{
		"zkocc.goedel.szopa.log.INFO.20130806-121006.10530",
		"zkocc.goedel.szopa.log.INFO.20130806-131006.10530",
		"zkocc.goedel.szopa.log.INFO.20130806-141006.10530",
		"zkocc.goedel.szopa.log.INFO.20130806-151006.10530",
	}

	for _, file := range files {
		if _, err := os.Create(path.Join(logDir, file)); err != nil {
			t.Fatalf("os.Create: %v", err)
		}
	}
	if err := os.Symlink(files[1], path.Join(logDir, "zkocc.INFO")); err != nil {
		t.Fatalf("os.Symlink: %v", err)
	}

	purgeLogsOnce(now, logDir, "zkocc", 30*time.Minute)

	left, err := filepath.Glob(path.Join(logDir, "zkocc.*"))
	if err != nil {
		t.Fatalf("filepath.Glob: %v", err)
	}

	if len(left) != 2 {
		// 151006 is still good, 131006 is the "current" log
		// (symlinked to zkocc.INFO), the rest should be
		// removed.
		t.Errorf("wrong number of files remain: want %v, got %v", 2, len(left))
	}

}
