package middleware

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestFilesCleanup(t *testing.T) {
	previousConfig, previousLimit := common.GetDiskCacheConfig(), constant.MaxFileDownloadMB
	t.Cleanup(func() {
		common.SetDiskCacheConfig(previousConfig)
		constant.MaxFileDownloadMB = previousLimit
	})
	constant.MaxFileDownloadMB = 1
	for _, scenario := range []struct {
		name   string
		status int
	}{
		{name: "return", status: http.StatusNoContent},
		{name: "abort", status: http.StatusBadRequest},
		{name: "panic", status: http.StatusInternalServerError},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			common.SetDiskCacheConfig(common.DiskCacheConfig{Enabled: true, MaxSizeMB: 16, Path: t.TempDir()})
			baseline := common.GetDiskCacheStats()
			var payload bytes.Buffer
			writer := multipart.NewWriter(&payload)
			part, err := writer.CreateFormFile("file", "fixture.txt")
			require.NoError(t, err)
			_, err = part.Write(bytes.Repeat([]byte("f"), 1<<20+1))
			require.NoError(t, err)
			require.NoError(t, writer.Close())

			var replay io.ReadCloser
			var multipartPath string
			engine := gin.New()
			engine.Use(gin.RecoveryWithWriter(io.Discard), BodyStorageCleanup())
			engine.POST("/upload", func(c *gin.Context) {
				storage, err := common.GetBodyStorage(c)
				require.NoError(t, err)
				replay, err = storage.NewReader()
				require.NoError(t, err)
				t.Cleanup(func() { _ = replay.Close() })

				form, err := common.ParseMultipartFormReusable(c)
				require.NoError(t, err)
				file, err := form.File["file"][0].Open()
				require.NoError(t, err)
				diskFile, ok := file.(*os.File)
				require.True(t, ok)
				multipartPath = diskFile.Name()
				require.NoError(t, file.Close())

				cachePath, err := common.WriteDiskCacheFile(common.DiskCacheTypeFile, []byte("fixture"))
				require.NoError(t, err)
				common.RegisterRequestFileCleanup(c, func() error { return common.RemoveDiskCacheFile(cachePath) })

				dir := common.GetDiskCacheDir()
				entries, err := os.ReadDir(dir)
				require.NoError(t, err)
				require.Len(t, entries, 2)
				old := time.Now().Add(-time.Hour)
				for _, entry := range entries {
					require.NoError(t, os.Chtimes(filepath.Join(dir, entry.Name()), old, old))
				}
				orphan := filepath.Join(dir, "file-0123abcd-1.tmp")
				unmanaged := filepath.Join(dir, "unmanaged.tmp")
				for _, path := range []string{orphan, unmanaged} {
					require.NoError(t, os.WriteFile(path, []byte("fixture"), 0600))
					require.NoError(t, os.Chtimes(path, old, old))
				}
				require.NoError(t, common.CleanupOldDiskCacheFiles(time.Minute))
				assert.NoFileExists(t, orphan)
				assert.FileExists(t, unmanaged)
				for _, entry := range entries {
					assert.FileExists(t, filepath.Join(dir, entry.Name()))
				}
				assert.Equal(t, baseline.ActiveDiskFiles+2, common.GetDiskCacheStats().ActiveDiskFiles)

				switch scenario.status {
				case http.StatusBadRequest:
					c.AbortWithStatus(scenario.status)
				case http.StatusInternalServerError:
					panic("fixture handler failure")
				default:
					c.Status(scenario.status)
				}
			})
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(payload.Bytes()))
			request.Header.Set("Content-Type", writer.FormDataContentType())
			engine.ServeHTTP(response, request)
			assert.Equal(t, scenario.status, response.Code)
			assert.NoFileExists(t, multipartPath)
			assert.Equal(t, baseline.ActiveDiskFiles+1, common.GetDiskCacheStats().ActiveDiskFiles)
			require.NotNil(t, replay)
			body, err := io.ReadAll(replay)
			require.NoError(t, err)
			assert.Equal(t, payload.Bytes(), body)
			require.NoError(t, replay.Close())
			require.NoError(t, replay.Close())
			assert.Equal(t, baseline.ActiveDiskFiles, common.GetDiskCacheStats().ActiveDiskFiles)
			assert.Equal(t, baseline.CurrentDiskUsageBytes, common.GetDiskCacheStats().CurrentDiskUsageBytes)
			entries, err := os.ReadDir(common.GetDiskCacheDir())
			require.NoError(t, err)
			require.Len(t, entries, 1)
			assert.Equal(t, "unmanaged.tmp", entries[0].Name())
		})
	}
}
