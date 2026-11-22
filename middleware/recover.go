package middleware

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/Im-oldxu/new-api/common"
	"github.com/gin-gonic/gin"
)

func PanicRecover() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		var details strings.Builder
		fmt.Fprintf(&details, "[Recovery] %s | request_id=%q | %d | %q %q\n",
			time.Now().UTC().Format(time.RFC3339), c.GetString(common.RequestIdKey),
			http.StatusInternalServerError, c.Request.Method, c.FullPath())
		var programCounters [64]uintptr
		stack := runtime.CallersFrames(programCounters[:runtime.Callers(2, programCounters[:])])
		for {
			frame, more := stack.Next()
			fmt.Fprintf(&details, "\t%s:%d %s\n", frame.File, frame.Line, frame.Function)
			if !more {
				break
			}
		}
		_, _ = common.ErrorLogWriter.Write([]byte(details.String()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Panic detected. Please submit an issue here: https://github.com/Calcium-Ion/new-api",
				"type":    "new_api_panic",
			},
		})
	})
}
