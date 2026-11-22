package model

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Im-oldxu/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAuthCacheIntegrityDatabaseMatrix(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(":memory:")
			case "mysql":
				dsn := os.Getenv("TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN is not configured")
				}
				driver = mysql.Open(dsn)
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN is not configured")
				}
				driver = postgres.Open(dsn)
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			previousDB, previousType := DB, common.MainDatabaseType()
			DB = db
			common.SetMainDatabaseType(common.DatabaseType(dialect))
			initCol()
			t.Cleanup(func() {
				DB = previousDB
				common.SetMainDatabaseType(previousType)
				initCol()
				assert.NoError(t, sqlDB.Close())
			})
			require.NoError(t, db.AutoMigrate(&User{}, &UserSession{}))
			var version string
			query := "SELECT version()"
			if dialect == "sqlite" {
				query = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(query).Scan(&version).Error)
			t.Logf("database version: %s", version)
			useUserCacheMiniRedis(t)

			user := User{Username: "cache-integrity-user", Role: common.RoleCommonUser,
				Status: common.UserStatusEnabled, AuthVersion: 1, Group: "default"}
			require.NoError(t, DB.Create(&user).Error)
			t.Cleanup(func() { assert.NoError(t, db.Unscoped().Delete(&user).Error) })
			require.NoError(t, populateUserCache(user))

			t.Run("user_metadata_and_partial_refresh", func(t *testing.T) {
				for _, fields := range []map[string]any{
					{"Role": common.RoleRootUser},
					{"AuthVersion": 99, "Status": common.UserStatusDisabled},
					{"Group": "admin", "Setting": `{"language":"evil"}`},
				} {
					require.NoError(t, common.RDB.HSet(t.Context(), getUserCacheKey(user.Id), fields).Err())
					_, err := cacheGetUserBase(user.Id)
					require.Error(t, err)
					// The version fence intentionally rejects snapshots below a cached version.
					if _, ok := fields["AuthVersion"]; ok {
						require.NoError(t, common.RDB.HSet(t.Context(), getUserCacheKey(user.Id), "AuthVersion", user.AuthVersion).Err())
					}
					loaded, err := GetUserCache(user.Id)
					require.NoError(t, err)
					assert.Equal(t, user.Role, loaded.Role)
					assert.Equal(t, user.Group, loaded.Group)
					assert.Equal(t, user.Status, loaded.Status)
				}
				require.NoError(t, common.RDB.HSet(t.Context(), getUserCacheKey(user.Id), "Role", common.RoleRootUser).Err())
				require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("group", "paid").Error)
				require.NoError(t, RefreshUserGroupCache(user.Id))
				cached, err := cacheGetUserBase(user.Id)
				require.NoError(t, err)
				assert.Equal(t, common.RoleCommonUser, cached.Role)
				assert.Equal(t, "paid", cached.Group)
			})

			session := newTestUserSession("cache-integrity-session", user.Id, time.Now().Unix())
			require.NoError(t, CreateUserSession(session))
			t.Cleanup(func() { assert.NoError(t, db.Delete(session).Error) })
			t.Run("session_owner_and_versions", func(t *testing.T) {
				require.NoError(t, common.RDB.HSet(t.Context(), userSessionCacheKey(session.SID),
					"UserID", user.Id+1, "Version", 2, "UserAuthVersion", 2, "ExpiresAt", session.ExpiresAt+3600).Err())
				_, err := getUserSessionCache(session.SID)
				require.Error(t, err)
				// A higher cached version is a deny fence for delayed active snapshots.
				require.NoError(t, common.RDB.HSet(t.Context(), userSessionCacheKey(session.SID), "Version", 1).Err())
				loaded, err := GetUserSessionCached(session.SID)
				require.NoError(t, err)
				assert.Equal(t, session.UserID, loaded.UserID)
				assert.Equal(t, session.UserAuthVersion, loaded.UserAuthVersion)
				assert.Equal(t, session.ExpiresAt, loaded.ExpiresAt)
			})

			t.Run("unsigned_and_expired_cache", func(t *testing.T) {
				for _, key := range []string{getUserCacheKey(user.Id), userSessionCacheKey(session.SID)} {
					require.NoError(t, common.RDB.HDel(t.Context(), key, "CacheMAC", "CacheExpiresAt").Err())
				}
				_, err := GetUserCache(user.Id)
				require.NoError(t, err)

				_, err = GetUserSessionCached(session.SID)
				require.NoError(t, err)
				cachedUser, err := cacheGetUserBase(user.Id)
				require.NoError(t, err)

				cachedSession, err := getUserSessionCache(session.SID)
				require.NoError(t, err)
				expired := time.Now().Add(-time.Second).UnixMilli()
				cachedUser.CacheExpiresAt = expired
				cachedSession.CacheExpiresAt = expired
				userMAC, err := cachedUser.cacheMAC()
				require.NoError(t, err)

				sessionMAC, err := authCacheMAC(userSessionCacheKey(session.SID), cachedSession)
				require.NoError(t, err)
				for key, mac := range map[string]string{getUserCacheKey(user.Id): userMAC, userSessionCacheKey(session.SID): sessionMAC} {
					require.NoError(t, common.RDB.HSet(t.Context(), key, "CacheExpiresAt", expired, "CacheMAC", mac).Err())
					require.NoError(t, common.RDB.Persist(t.Context(), key).Err())
				}
				_, err = cacheGetUserBase(user.Id)
				assert.Error(t, err)

				_, err = getUserSessionCache(session.SID)
				assert.Error(t, err)
				_, err = GetUserCache(user.Id)
				require.NoError(t, err)

				_, err = GetUserSessionCached(session.SID)
				require.NoError(t, err)
			})

			t.Run("cross_identity_copy_and_database_failure", func(t *testing.T) {
				for source, destination := range map[string]string{
					getUserCacheKey(user.Id):         getUserCacheKey(user.Id + 1000),
					userSessionCacheKey(session.SID): userSessionCacheKey("unknown-session"),
				} {
					fields, err := common.RDB.HGetAll(t.Context(), source).Result()
					require.NoError(t, err)
					require.NoError(t, common.RDB.HSet(t.Context(), destination, fields).Err())
				}
				require.NoError(t, common.RDB.HSet(t.Context(), getUserCacheKey(user.Id+1000), "Id", user.Id+1000).Err())
				require.NoError(t, common.RDB.HSet(t.Context(), userSessionCacheKey("unknown-session"), "SID", "unknown-session").Err())
				_, err := GetUserCache(user.Id + 1000)
				assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

				_, err = GetUserSessionCached("unknown-session")
				assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
				require.NoError(t, DB.Callback().Query().Before("gorm:query").Register("auth_cache_db_failure", func(tx *gorm.DB) {
					tx.AddError(errors.New("database unavailable"))
				}))
				defer func() { require.NoError(t, DB.Callback().Query().Remove("auth_cache_db_failure")) }()

				_, err = GetUserSessionCached(session.SID)
				require.NoError(t, err, "valid session cache must not query the database")
				require.NoError(t, common.RDB.HSet(t.Context(), userSessionCacheKey(session.SID), "CacheMAC", "invalid").Err())

				_, err = GetUserSessionCached(session.SID)
				assert.Error(t, err)
			})

			t.Run("session_revocation_cannot_be_cleared", func(t *testing.T) {
				revoked, err := RevokeUserSession(user.Id, session.SID, "test")
				require.NoError(t, err)
				require.True(t, revoked)
				require.NoError(t, common.RDB.HSet(t.Context(), userSessionCacheKey(session.SID),
					"Status", UserSessionStatusActive, "RevokedAt", 0, "RevokedReason", "").Err())
				_, err = GetUserSessionCached(session.SID)
				assert.ErrorIs(t, err, ErrUserSessionInactive)
			})

		})
	}
}
