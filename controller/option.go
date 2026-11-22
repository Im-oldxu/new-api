package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/model"
	"github.com/Im-oldxu/new-api/pkg/outboundhttp"
	"github.com/Im-oldxu/new-api/setting/console_setting"
	"github.com/Im-oldxu/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func GetOptions(c *gin.Context) {
	var options []*model.Option
	common.OptionMapRWMutex.RLock()
	for key, value := range common.OptionMap {
		if key == "theme.frontend" || key == "ResendKeyConfigured" {
			continue
		}
		valueString := common.Interface2String(value)
		isSensitiveKey := strings.HasSuffix(key, "Token") ||
			strings.HasSuffix(key, "Secret") ||
			strings.HasSuffix(key, "Key") ||
			strings.HasSuffix(key, "secret") ||
			strings.HasSuffix(key, "api_key") ||
			strings.HasSuffix(key, "private_key")
		if isSensitiveKey && key != "TurnstileSiteKey" {
			continue
		}
		options = append(options, &model.Option{Key: key, Value: valueString})
	}
	options = append(options, &model.Option{
		Key:   "ResendKeyConfigured",
		Value: strconv.FormatBool(strings.TrimSpace(common.OptionMap["ResendApiKey"]) != ""),
	})
	common.OptionMapRWMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    options,
	})
}

type OptionUpdateRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func UpdatePasskeyDomains(c *gin.Context) {
	var request struct {
		RPID                *string `json:"rp_id"`
		LegacyRPIDs         *string `json:"legacy_rp_ids"`
		Origins             *string `json:"origins"`
		Preview             bool    `json:"preview"`
		RemovalConfirmation string  `json:"removal_confirmation"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.RPID == nil || request.LegacyRPIDs == nil || request.Origins == nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	change, err := model.UpdatePasskeyDomainOptions(map[string]string{
		"passkey.rp_id": *request.RPID, "passkey.legacy_rp_ids": *request.LegacyRPIDs, "passkey.origins": *request.Origins,
	}, request.Preview, request.RemovalConfirmation)
	if err != nil {
		writePasskeyDomainSettingsError(c, err)
		if !request.Preview {
			recordPasskeyDomainAudit(c, change, request.RemovalConfirmation != "", err)
		}
		return
	}
	if !request.Preview {
		recordPasskeyDomainAudit(c, change, request.RemovalConfirmation != "", nil)
	}
	common.ApiSuccess(c, change)
}

func writePasskeyDomainSettingsError(c *gin.Context, err error) {
	var removal *model.PasskeyDomainRemovalError
	if errors.As(err, &removal) {
		common.ApiErrorStatus(c, http.StatusConflict, common.NewMessage("Review the affected Passkeys and confirm the domain removal. If the settings or impact have changed, confirmation is required again."), gin.H{
			"code": "PASSKEY_RP_ID_REMOVAL_CONFIRMATION_REQUIRED",
			"data": removal.Change,
		})
		return
	}
	if errors.Is(err, system_setting.ErrPasskeyRPIDInvalid) {
		writeSecurityOperationError(c, err)
		return
	}
	common.ApiError(c, err)
}

// A zero maximum leaves the upper bound open, matching the corresponding form.
var numericOptionBounds = map[string]struct{ min, max int }{
	"SMTPPort": {1, 65535},
	"performance_setting.disk_cache_threshold_mb":  {1, 0},
	"performance_setting.disk_cache_max_size_mb":   {100, 0},
	"performance_setting.monitor_cpu_threshold":    {0, 0},
	"performance_setting.monitor_memory_threshold": {0, 100},
	"performance_setting.monitor_disk_threshold":   {0, 100},
}

func UpdateOption(c *gin.Context) {
	var option OptionUpdateRequest
	err := common.DecodeJson(c.Request.Body, &option)
	if err != nil {
		common.ApiErrorStatus(c, http.StatusBadRequest, common.NewMessage("Invalid parameters"))
		return
	}
	switch v := option.Value.(type) {
	case string, bool, float64:
		option.Value = common.Interface2String(v)
	default:
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	value := option.Value.(string)
	switch option.Key {
	case "PasswordLoginEnabled", "PasswordRegisterEnabled", "RegisterEnabled", "EmailVerificationEnabled", "EmailAliasRestrictionEnabled",
		"DefaultCollapseSidebar", "DemoSiteEnabled", "SelfUseModeEnabled", "SMTPForceAuthLogin", "SMTPInsecureSkipVerify", "SMTPStartTLSEnabled", "SMTPSSLEnabled",
		"passkey.enabled", "passkey.allow_insecure_origin", "console_setting.announcements_enabled", "console_setting.faq_enabled", "console_setting.uptime_kuma_enabled",
		"performance_setting.disk_cache_enabled", "performance_setting.monitor_enabled", "fetch_setting.enable_ssrf_protection", "fetch_setting.allow_private_ip",
		"fetch_setting.apply_ip_filter_for_domain", "fetch_setting.domain_filter_mode", "fetch_setting.ip_filter_mode",
		"EmailDomainRestrictionEnabled", "GitHubOAuthEnabled", "discord.enabled", "oidc.enabled", "LinuxDOOAuthEnabled", "TelegramOAuthEnabled", "WeChatAuthEnabled", "TurnstileCheckEnabled":
		if value != "true" && value != "false" {
			common.ApiErrorT(c, "Invalid parameters")
			return
		}
	}
	if bounds, ok := numericOptionBounds[option.Key]; ok {
		number, err := strconv.Atoi(value)
		if err != nil || number < bounds.min || (bounds.max > 0 && number > bounds.max) {
			common.ApiErrorT(c, "Invalid parameters")
			return
		}
	}
	switch option.Key {
	case "passkey.user_verification":
		if value != "required" && value != "preferred" && value != "discouraged" {
			common.ApiErrorT(c, "Invalid parameters")
			return
		}
	case "passkey.attachment_preference":
		if value != "" && value != "platform" && value != "cross-platform" {
			common.ApiErrorT(c, "Invalid parameters")
			return
		}
	case "fetch_setting.allowed_ports":
		var ports []string
		if err := common.UnmarshalJsonStr(option.Value.(string), &ports); err != nil || ports == nil {
			common.ApiErrorT(c, "Invalid parameters")
			return
		}
		if _, err := outboundhttp.NewSSRFProtectionFromFetchSetting(false, false, false, nil, nil, ports, true); err != nil {
			common.ApiErrorT(c, "Invalid parameters")
			return
		}
	case "GitHubOAuthEnabled":
		if option.Value == "true" && (strings.TrimSpace(common.GitHubClientId) == "" || strings.TrimSpace(common.GitHubClientSecret) == "") {
			common.ApiErrorT(c, "Cannot enable {{provider}}. Enter the {{provider}} Client ID and Client Secret first.", map[string]any{"provider": "GitHub OAuth"})
			return
		}
	case "discord.enabled":
		if option.Value == "true" && (strings.TrimSpace(system_setting.GetDiscordSettings().ClientId) == "" || strings.TrimSpace(system_setting.GetDiscordSettings().ClientSecret) == "") {
			common.ApiErrorT(c, "Cannot enable {{provider}}. Enter the {{provider}} Client ID and Client Secret first.", map[string]any{"provider": "Discord OAuth"})
			return
		}
	case "oidc.enabled":
		if option.Value == "true" && (strings.TrimSpace(system_setting.GetOIDCSettings().ClientId) == "" || strings.TrimSpace(system_setting.GetOIDCSettings().ClientSecret) == "") {
			common.ApiErrorT(c, "Cannot enable {{provider}}. Enter the {{provider}} Client ID and Client Secret first.", map[string]any{"provider": "OIDC"})
			return
		}
		if option.Value == "true" {
			cfg := system_setting.GetOIDCSettings()
			if cfg.AuthorizationEndpoint == "" || cfg.TokenEndpoint == "" || cfg.UserInfoEndpoint == "" {
				common.ApiErrorT(c, "Invalid parameters")
				return
			}
		}
	case "LinuxDOOAuthEnabled":
		if option.Value == "true" && (strings.TrimSpace(common.LinuxDOClientId) == "" || strings.TrimSpace(common.LinuxDOClientSecret) == "") {
			common.ApiErrorT(c, "Cannot enable {{provider}}. Enter the {{provider}} Client ID and Client Secret first.", map[string]any{"provider": "LinuxDO OAuth"})
			return
		}
	case "EmailDomainRestrictionEnabled":
		if option.Value == "true" && len(common.EmailDomainWhitelist) == 0 {
			common.ApiErrorT(c, "Cannot enable the email domain restriction. Enter the allowed email domains first.")
			return
		}
	case "WeChatAuthEnabled":
		if option.Value == "true" && common.WeChatServerAddress == "" {
			common.ApiErrorT(c, "Cannot enable WeChat login. Enter the WeChat login settings first.")
			return
		}
	case "TurnstileCheckEnabled":
		if option.Value == "true" && (strings.TrimSpace(common.TurnstileSiteKey) == "" || strings.TrimSpace(common.TurnstileSecretKey) == "") {
			common.ApiErrorT(c, "Cannot enable Turnstile verification. Enter the Turnstile settings first.")
			return
		}
	case "TelegramOAuthEnabled":
		if option.Value == "true" && !system_setting.GetTelegramSettings().IsConfigured() {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"code":    "TELEGRAM_OAUTH_NOT_CONFIGURED",
				"message": "Telegram OAuth is not configured or enabled. Please contact your administrator.",
			})
			return
		}
	case "console_setting.announcements":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "Announcements")
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "console_setting.faq":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "FAQ")
		if err != nil {
			common.ApiError(c, err)
			return
		}
	case "console_setting.uptime_kuma_groups":
		err = console_setting.ValidateConsoleSettings(option.Value.(string), "UptimeKumaGroups")
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if model.IsPasskeyDomainOption(option.Key) {
		change, updateErr := model.UpdatePasskeyDomainOptions(map[string]string{option.Key: option.Value.(string)}, false, "")
		if updateErr != nil {
			writePasskeyDomainSettingsError(c, updateErr)
			recordPasskeyDomainAudit(c, change, false, updateErr)
			return
		}
		recordPasskeyDomainAudit(c, change, false, nil)
		common.ApiSuccess(c, change)
		return
	}
	err = model.UpdateOption(option.Key, option.Value.(string))
	if errors.Is(err, model.ErrRetiredGatewayOption) {
		common.ApiErrorStatus(c, http.StatusBadRequest, model.ErrRetiredGatewayOption)
		return
	}
	if err != nil {
		if errors.Is(err, system_setting.ErrPasskeyRPIDInvalid) {
			writeSecurityOperationError(c, err)
		} else {
			common.ApiError(c, err)
		}
		return
	}
	// 出于安全考虑只记录被修改的配置项名称，不记录配置值（可能含密钥等敏感信息）。
	recordManageAudit(c, "option.update", map[string]any{
		"key": option.Key,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
