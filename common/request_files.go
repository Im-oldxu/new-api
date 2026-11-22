package common

import (
	"sync"

	"github.com/gin-gonic/gin"
)

const requestFileCleanupKey = "request_file_cleanup"

type requestFileCleanup struct {
	mu       sync.Mutex
	cleanups []func() error
}

func BeginRequestFileCleanup(c *gin.Context) func() {
	files := &requestFileCleanup{}
	c.Set(requestFileCleanupKey, files)
	return func() {
		files.mu.Lock()
		cleanups := files.cleanups
		files.cleanups = nil
		files.mu.Unlock()
		for index := len(cleanups) - 1; index >= 0; index-- {
			if err := cleanups[index](); err != nil {
				SysError(LogText("Failed to clean up request file: %v", err))
			}
		}
	}
}

func RegisterRequestFileCleanup(c *gin.Context, cleanup func() error) {
	if c == nil {
		return
	}
	value, exists := c.Get(requestFileCleanupKey)
	if !exists {
		return
	}
	files := value.(*requestFileCleanup)
	files.mu.Lock()
	defer files.mu.Unlock()
	files.cleanups = append(files.cleanups, cleanup)
}
