package tickets

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/middleware"
	"github.com/Im-oldxu/new-api/model"
	"github.com/Im-oldxu/new-api/service"
	"github.com/Im-oldxu/new-api/service/authz"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTicketOwnershipAndWorkflow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "tickets.db")), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	previous, logs, redis, master := model.DB, model.LOG_DB, common.RedisEnabled, common.IsMasterNode
	model.DB, model.LOG_DB, common.RedisEnabled, common.IsMasterNode = db, db, false, true
	t.Cleanup(func() {
		model.DB, model.LOG_DB, common.RedisEnabled, common.IsMasterNode = previous, logs, redis, master
	})
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.AuditLog{}, &model.CasbinRule{}, &model.AuthzRole{}, &model.SystemTask{}))
	require.NoError(t, Migrate(db))
	require.NoError(t, authz.Init(db))
	roles := []int{common.RoleCommonUser, common.RoleCommonUser, common.RoleAdminUser, common.RoleRootUser, common.RoleAdminUser}
	tokens := make([]string, len(roles))
	for i, r := range roles {
		user := model.User{Id: i + 1, Username: "actor" + strconv.Itoa(i), Role: r, Status: common.UserStatusEnabled, AuthVersion: 1}
		require.NoError(t, db.Create(&user).Error)
		b, err := service.CreateLoginSession(user.Id, "password", "127.0.0.1", "ticket-test")
		require.NoError(t, err)
		tokens[i] = b.AccessToken
	}
	app := gin.New()
	apiGroup := app.Group("/api", middleware.RequestId(), middleware.BodyStorageCleanup())
	Register(apiGroup)
	request := func(method, url, token, ct string, body io.Reader) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, url, body)
		req.Header.Set("Content-Type", ct)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		return res
	}
	body := new(bytes.Buffer)
	form := multipart.NewWriter(body)
	require.NoError(t, form.WriteField("title", "Keyboard"))
	require.NoError(t, form.WriteField("description", "Replace the spare keyboard"))
	file, err := form.CreateFormFile("attachment", "notes.txt")
	require.NoError(t, err)
	_, err = file.Write([]byte("attachment-body"))
	require.NoError(t, err)
	require.NoError(t, form.Close())
	assert.Equal(t, 401, request("GET", "/api/tickets", "", "", nil).Code)
	res := request("POST", "/api/tickets", tokens[0], form.FormDataContentType(), body)
	require.Equal(t, 200, res.Code, res.Body.String())
	var created struct{ Data Ticket }
	require.NoError(t, common.Unmarshal(res.Body.Bytes(), &created))
	require.Equal(t, 1, created.Data.OwnerID)
	id := strconv.Itoa(created.Data.ID)
	for _, suffix := range []string{"", "/attachment"} {
		assert.Equal(t, 404, request("GET", "/api/tickets/"+id+suffix, tokens[1], "", nil).Code)
	}
	assert.Equal(t, 404, request("PUT", "/api/tickets/"+id, tokens[1], "", nil).Code)
	assert.Equal(t, "attachment-body", request("GET", "/api/tickets/"+id+"/attachment", tokens[0], "", nil).Body.String())
	assert.Equal(t, 403, request("GET", "/api/manage/tickets", tokens[2], "", nil).Code)
	require.NoError(t, authz.SetUserPermissions(3, authz.PermissionsMap{"ticket": {"read": true}}))
	assert.Equal(t, 200, request("GET", "/api/manage/tickets", tokens[2], "", nil).Code)
	transition := func(token, state string, version int) *httptest.ResponseRecorder {
		data, err := common.Marshal(map[string]any{"status": state, "version": version})
		require.NoError(t, err)
		return request(http.MethodPatch, "/api/manage/tickets/"+id+"/status", token, "application/json", bytes.NewReader(data))
	}
	assert.Equal(t, 403, transition(tokens[2], processing, 1).Code)
	require.NoError(t, authz.SetUserPermissions(3, authz.PermissionsMap{"ticket": {"write": true}}))
	assert.Equal(t, 403, transition(tokens[2], processing, 1).Code)

	require.NoError(t, authz.SetUserPermissions(3, authz.PermissionsMap{"ticket": {"read": true, "write": true}}))
	assert.Equal(t, 409, transition(tokens[2], completed, 1).Code)
	assert.Equal(t, 200, transition(tokens[2], processing, 1).Code)
	assert.Equal(t, 409, transition(tokens[3], completed, 1).Code)
	reply := func(token, path string, version int, text string) *httptest.ResponseRecorder {
		b, err := common.Marshal(map[string]any{"body": text, "version": version})
		require.NoError(t, err)
		return request(http.MethodPost, path+"/"+id+"/replies", token, "application/json", bytes.NewReader(b))
	}
	require.NoError(t, authz.SetUserPermissions(5, authz.PermissionsMap{"ticket": {"read": true, "write": true}}))
	assert.Equal(t, 200, request("GET", "/api/manage/tickets/"+id, tokens[4], "", nil).Code)
	assert.Equal(t, 403, reply(tokens[4], "/api/manage/tickets", 2, "Not my ticket").Code)
	assert.Equal(t, 403, transition(tokens[4], completed, 2).Code)
	assert.Equal(t, 404, reply(tokens[1], "/api/tickets", 2, "private").Code)
	assert.Equal(t, 400, reply(tokens[0], "/api/tickets", 2, "   ").Code)
	assert.Equal(t, 200, reply(tokens[0], "/api/tickets", 2, "More information").Code)
	assert.Equal(t, 409, reply(tokens[2], "/api/manage/tickets", 2, "Stale").Code)
	assert.Equal(t, 200, reply(tokens[2], "/api/manage/tickets", 3, "Replacement delivered").Code)
	assert.Equal(t, 200, transition(tokens[2], completed, 4).Code)
	assert.Equal(t, 409, reply(tokens[0], "/api/tickets", 5, "Too late").Code)
	var detail struct {
		Data struct {
			Ticket
			Entries []Entry
		}
	}
	require.NoError(t, common.Unmarshal(request("GET", "/api/tickets/"+id, tokens[0], "", nil).Body.Bytes(), &detail))
	require.Len(t, detail.Data.Entries, 5)
	assert.Equal(t, "Replacement delivered", detail.Data.Entries[3].Body)
	assert.Equal(t, "actor0", detail.Data.OwnerName)
	assert.Equal(t, "actor2", detail.Data.AssigneeName)
	require.NotNil(t, detail.Data.AssigneeID)
	assert.Equal(t, 3, *detail.Data.AssigneeID)
	var stored Ticket
	require.NoError(t, db.First(&stored, created.Data.ID).Error)
	assert.Equal(t, completed, stored.Status)
	assert.Equal(t, 5, stored.Version)
	var events []model.AuditLog
	require.NoError(t, db.Where("category = ? AND action LIKE ?", model.AuditCategoryOperation, "ticket.%").Order("id").Find(&events).Error)
	require.Len(t, events, 5)
	assert.Equal(t, 3, events[1].UserId)
	encoded, err := common.Marshal(events)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "attachment-body")
	assert.NotContains(t, string(encoded), "Replacement delivered")
	assert.Equal(t, 403, request("GET", "/api/manage/tickets/overview", tokens[0], "", nil).Code)
	for _, tc := range []struct {
		url, credential string
		total           int
	}{
		{"/api/tickets/overview?days=7&owner_id=1", tokens[1], 0},
		{"/api/tickets/overview?days=7", tokens[0], 1},
		{"/api/manage/tickets/overview?days=7", tokens[2], 1},
	} {
		res := request("GET", tc.url, tc.credential, "", nil)
		require.Equal(t, 200, res.Code)
		var result struct {
			Data struct {
				Total  int
				Counts map[string]int
				Trend  []int
				Recent []Ticket
			}
		}
		require.NoError(t, common.Unmarshal(res.Body.Bytes(), &result))
		assert.Equal(t, tc.total, result.Data.Total)
		assert.Equal(t, tc.total, result.Data.Counts[completed])
		require.Len(t, result.Data.Trend, 7)
		assert.Equal(t, tc.total, result.Data.Trend[6])
		assert.Len(t, result.Data.Recent, tc.total)
	}
	setLimit := func(token string, limit int) int {
		data, err := common.Marshal(map[string]int{"attachment_limit_kib": limit})
		require.NoError(t, err)
		return request("PUT", "/api/tickets/settings", token, "application/json", bytes.NewReader(data)).Code
	}
	assert.Equal(t, 403, setLimit(tokens[2], 32))
	assert.Equal(t, 400, setLimit(tokens[3], 0))
	assert.Equal(t, 400, setLimit(tokens[3], 1025))
	assert.Equal(t, 200, setLimit(tokens[3], 16))
	assert.Contains(t, request("GET", "/api/tickets/settings", tokens[0], "", nil).Body.String(), `"attachment_limit_kib":16`)

}
