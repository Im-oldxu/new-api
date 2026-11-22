package model

import (
	"maps"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Im-oldxu/new-api/common"
	"github.com/Im-oldxu/new-api/setting/config"
	"github.com/Im-oldxu/new-api/setting/operation_setting"
	"github.com/Im-oldxu/new-api/setting/performance_setting"
	"github.com/Im-oldxu/new-api/setting/system_setting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Option struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}

func AllOption() ([]*Option, error) {
	var options []*Option
	var err error
	err = DB.Find(&options).Error
	return options, err
}

func InitOptionMap() {
	common.OptionMapRWMutex.Lock()
	common.OptionMap = make(map[string]string)

	// 添加原有的系统配置
	common.OptionMap["PasswordLoginEnabled"] = strconv.FormatBool(common.PasswordLoginEnabled)
	common.OptionMap["PasswordRegisterEnabled"] = strconv.FormatBool(common.PasswordRegisterEnabled)
	common.OptionMap["EmailVerificationEnabled"] = strconv.FormatBool(common.EmailVerificationEnabled)
	common.OptionMap["GitHubOAuthEnabled"] = strconv.FormatBool(common.GitHubOAuthEnabled)
	common.OptionMap["LinuxDOOAuthEnabled"] = strconv.FormatBool(common.LinuxDOOAuthEnabled)
	common.OptionMap["TelegramOAuthEnabled"] = strconv.FormatBool(common.TelegramOAuthEnabled)
	common.OptionMap["WeChatAuthEnabled"] = strconv.FormatBool(common.WeChatAuthEnabled)
	common.OptionMap["TurnstileCheckEnabled"] = strconv.FormatBool(common.TurnstileCheckEnabled)
	common.OptionMap["RegisterEnabled"] = strconv.FormatBool(common.RegisterEnabled)
	common.OptionMap["EmailDomainRestrictionEnabled"] = strconv.FormatBool(common.EmailDomainRestrictionEnabled)
	common.OptionMap["EmailAliasRestrictionEnabled"] = strconv.FormatBool(common.EmailAliasRestrictionEnabled)
	common.OptionMap["EmailDomainWhitelist"] = strings.Join(common.EmailDomainWhitelist, ",")
	common.OptionMap["SMTPServer"] = ""
	common.OptionMap["EmailProvider"] = "smtp"
	common.OptionMap["ResendApiKey"] = ""
	common.OptionMap["ResendFrom"] = ""
	common.OptionMap["SMTPFrom"] = ""
	common.OptionMap["SMTPPort"] = strconv.Itoa(common.SMTPPort)
	common.OptionMap["SMTPAccount"] = ""
	common.OptionMap["SMTPToken"] = ""
	common.OptionMap["SMTPSSLEnabled"] = strconv.FormatBool(common.SMTPSSLEnabled)
	common.OptionMap["SMTPStartTLSEnabled"] = strconv.FormatBool(common.SMTPStartTLSEnabled)
	common.OptionMap["SMTPInsecureSkipVerify"] = strconv.FormatBool(common.SMTPInsecureSkipVerify)
	common.OptionMap["SMTPForceAuthLogin"] = strconv.FormatBool(common.SMTPForceAuthLogin)
	common.OptionMap["Notice"] = ""
	common.OptionMap["About"] = ""
	common.OptionMap["HomePageContent"] = ""
	common.OptionMap["Footer"] = common.Footer
	common.OptionMap["SystemName"] = common.SystemName
	common.OptionMap["Logo"] = common.Logo
	common.OptionMap["ServerAddress"] = ""
	common.OptionMap["WorkerUrl"] = system_setting.WorkerUrl
	common.OptionMap["WorkerValidKey"] = system_setting.WorkerValidKey
	common.OptionMap["WorkerAllowHttpImageRequestEnabled"] = strconv.FormatBool(system_setting.WorkerAllowHttpImageRequestEnabled)
	common.OptionMap["GitHubClientId"] = ""
	common.OptionMap["GitHubClientSecret"] = ""
	common.OptionMap["TelegramBotToken"] = ""
	common.OptionMap["TelegramBotName"] = ""
	common.OptionMap["WeChatServerAddress"] = ""
	common.OptionMap["WeChatServerToken"] = ""
	common.OptionMap["WeChatAccountQRCodeImageURL"] = ""
	common.OptionMap["TurnstileSiteKey"] = ""
	common.OptionMap["TurnstileSecretKey"] = ""

	common.OptionMap["DefaultCollapseSidebar"] = strconv.FormatBool(common.DefaultCollapseSidebar)
	common.OptionMap["DemoSiteEnabled"] = strconv.FormatBool(operation_setting.DemoSiteEnabled)
	common.OptionMap["SelfUseModeEnabled"] = strconv.FormatBool(operation_setting.SelfUseModeEnabled)

	// 自动添加所有注册的模型配置
	modelConfigs := config.GlobalConfig.ExportAllConfigs()
	maps.DeleteFunc(modelConfigs, func(key, _ string) bool { return isRetiredGatewayOption(key) })
	maps.Copy(common.OptionMap, modelConfigs)

	common.OptionMapRWMutex.Unlock()
	if err := LoadOptionsFromDatabase(); err != nil {
		common.SysError(common.LogText("failed to load options from database: %s", err.Error()))
	}
}

// LoadOptionsFromDatabase applies every stored option. A failed read keeps
// the options in memory as they are.
func LoadOptionsFromDatabase() error {
	passkeyOptionMutex.Lock()
	defer passkeyOptionMutex.Unlock()
	options, err := AllOption()
	if err != nil {
		return err
	}
	passkeyOptions := make(map[string]string)
	for _, option := range options {
		if !isSupportedOption(option.Key) {
			continue
		}
		if IsPasskeyDomainOption(option.Key) {
			passkeyOptions[option.Key] = option.Value
			continue
		}
		err := updateOptionMap(option.Key, option.Value)
		if err != nil {
			common.SysLog(common.LogText("failed to update option map: %s", err.Error()))
		}
	}
	applyPasskeyDomainOptions(passkeyOptions)
	return nil
}

func SyncOptions(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		common.SysLog(common.LogText("syncing options from database"))
		if err := LoadOptionsFromDatabase(); err != nil {
			common.SysError(common.LogText("failed to load options from database: %s", err.Error()))
		}
	}
}

func isRetiredGatewayOption(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	namespace, _, nested := strings.Cut(key, ".")
	if nested {
		switch namespace {
		case "channel_affinity_setting", "monitor_setting", "billing_setting", "tool_price_setting", "group_ratio_setting",
			"quota_setting", "payment_setting", "checkin_setting", "token_setting", "log_setting", "global", "claude", "gemini", "grok", "request_plugin_setting":
			return true
		}
	}
	if strings.HasPrefix(key, "stripe") || strings.HasPrefix(key, "creem") || strings.HasPrefix(key, "waffo") {
		return true
	}
	switch key {
	case "general_setting.ping_interval_enabled", "general_setting.ping_interval_seconds", "console_setting.api_info", "console_setting.api_info_enabled", retiredThemeOptionKey, "apiinfo", "announcements", "faq", "uptimekumaurl", "uptimekumaslug", "chats", "drawingenabled", "taskenabled", "taskpublicaddress", "autogroups", "customcallbackaddress", "defaultuseautogroup", "displayincurrencyenabled",
		"displaytokenstatenabled", "epayid", "epaykey", "mintopup",
		"payaddress", "paymethods", "preconsumedquota", "price",
		"quotaforinvitee", "quotaforinviter", "quotafornewuser", "quotaperunit",
		"quotaremindenabled", "quotaremindthreshold", "topuplink", "topupgroupratio",
		"usdexchangerate", "userusablegroups", "general_setting.quota_display_type", "general_setting.custom_currency_symbol",
		"general_setting.custom_currency_exchange_rate", "modelbillingmode", "modelbillingexpr":
		return true
	case "logconsumeenabled", "dataexportenabled", "dataexportinterval", "dataexportdefaulttime", "maxtokenautogroups",
		"modelprice", "modelratio", "audiocompletionratio", "audioratio", "cacheratio", "completionratio",
		"createcacheratio", "imageratio", "groupratio", "groupgroupratio", "exposeratioenabled",
		"retrytimes", "automaticretrystatuscodes", "automaticdisablestatuscodes", "automaticdisablekeywords",
		"automaticdisablechannelenabled", "automaticenablechannelenabled", "channeldisablethreshold",
		"modelrequestratelimitcount", "modelrequestratelimitdurationminutes", "modelrequestratelimitsuccesscount",
		"modelrequestratelimitgroup", "modelrequestratelimitenabled", "checksensitiveenabled",
		"checksensitiveonpromptenabled", "sensitivewords", "chatlink", "chatlink2":
		return true
	}
	return false
}

var ErrRetiredGatewayOption = common.NewMessage("Unsupported operation")

// Unknown legacy rows stay in storage. Only active settings may enter the runtime map.
var registeredOptionKeys = sync.OnceValue(func() map[string]bool {
	keys := map[string]bool{}
	for key := range config.GlobalConfig.ExportAllConfigs() {
		keys[key] = true
	}
	return keys
})

func isSupportedOption(key string) bool {
	switch key {
	case "About", "DefaultCollapseSidebar", "DemoSiteEnabled", "EmailAliasRestrictionEnabled",
		"EmailDomainRestrictionEnabled", "EmailDomainWhitelist", "EmailProvider", "EmailVerificationEnabled",
		"Footer", "GitHubClientId", "GitHubClientSecret", "GitHubOAuthEnabled",
		"HeaderNavModules", "HomePageContent", "LinuxDOClientId", "LinuxDOClientSecret",
		"LinuxDOMinimumTrustLevel", "LinuxDOOAuthEnabled", "Logo", "Notice",
		"PasswordLoginEnabled", "PasswordRegisterEnabled", "RegisterEnabled", "ResendApiKey",
		"ResendFrom", "SMTPAccount", "SMTPForceAuthLogin", "SMTPFrom",
		"SMTPInsecureSkipVerify", "SMTPPort", "SMTPSSLEnabled", "SMTPServer",
		"SMTPStartTLSEnabled", "SMTPToken", "SelfUseModeEnabled", "ServerAddress",
		"SidebarModulesAdmin", "SystemName", "TelegramBotName", "TelegramBotToken",
		"TelegramOAuthEnabled", "TurnstileCheckEnabled", "TurnstileSecretKey", "TurnstileSiteKey",
		"WeChatAccountQRCodeImageURL", "WeChatAuthEnabled", "WeChatServerAddress", "WeChatServerToken",
		"WorkerAllowHttpImageRequestEnabled", "WorkerUrl", "WorkerValidKey":
		return true
	}
	return registeredOptionKeys()[key] && !isRetiredGatewayOption(key)
}

func validateOptionValue(key string, value string) error {
	if isRetiredGatewayOption(key) {
		return ErrRetiredGatewayOption
	}
	if key == legacyAccessTokenRetireAtKey {
		return errLegacyRetireAtReadOnly
	}
	if key == "EmailProvider" || key == "ResendApiKey" || key == "ResendFrom" {
		if key == "EmailProvider" && value != "smtp" && value != "resend" {
			return common.NewMessage("Email sending method must be SMTP or Resend")
		}
		common.OptionMapRWMutex.RLock()
		settings := map[string]string{
			"EmailProvider": common.OptionMap["EmailProvider"],
			"ResendApiKey":  common.OptionMap["ResendApiKey"],
			"ResendFrom":    common.OptionMap["ResendFrom"],
		}
		common.OptionMapRWMutex.RUnlock()
		settings[key] = value
		// While Resend is the active method, a change must keep settings it can
		// send with. Inactive drafts are checked when Resend is turned on.
		if settings["EmailProvider"] == "resend" {
			return common.ValidateResendConfiguration(settings["ResendApiKey"], settings["ResendFrom"])
		}
	}
	if key == "ResendKeyConfigured" {
		return common.NewMessage("Resend key configuration status is read-only")
	}
	if !isSupportedOption(key) {
		return ErrRetiredGatewayOption
	}
	if key == operation_setting.SystemTaskRetentionDaysOptionKey {
		return operation_setting.ValidateSystemTaskRetentionDays(value)
	}
	return nil
}

func saveOptionValue(tx *gorm.DB, key, value string) (string, error) {
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&Option{Key: key, Value: value})
	if result.Error != nil {
		return "", result.Error
	}
	var stored Option
	if err := lockForUpdate(tx).Select("key", "value").Where(clause.Eq{Column: clause.Column{Name: "key"}, Value: key}).Take(&stored).Error; err != nil {
		return "", err
	}
	if !isSupportedOption(stored.Key) || stored.Key != key {
		return "", ErrRetiredGatewayOption
	}
	if key == "HeaderNavModules" || key == "SidebarModulesAdmin" {
		merged, err := navigationOption(key, stored.Value, &value)
		if err != nil {
			return "", err
		}
		value = merged
	}
	return value, tx.Model(&stored).Update("value", value).Error
}

func UpdateOption(key string, value string) error {
	if IsPasskeyDomainOption(key) {
		_, err := UpdatePasskeyDomainOptions(map[string]string{key: value}, false, "")
		return err
	}
	if err := validateOptionValue(key, value); err != nil {
		return err
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		value, err = saveOptionValue(tx, key, value)
		return err
	}); err != nil {
		return err
	}
	BumpSyncTopic(SyncTopicOptions)
	// Update OptionMap
	return updateOptionMap(key, value)
}

// UpdateOptionsBulk persists multiple key/value pairs in a single database
// transaction, then dispatches them through updateOptionMap in one pass. If
// any DB write fails the whole transaction rolls back and no in-memory state
// is touched — safe for callers that must commit a set of related options
// atomically (e.g. payment gateway binding).
func UpdateOptionsBulk(values map[string]string) error {
	if len(values) == 0 {
		return nil
	}
	for key := range values {
		if IsPasskeyDomainOption(key) {
			_, err := UpdatePasskeyDomainOptions(values, false, "")
			return err
		}
	}
	for key, value := range values {
		if err := validateOptionValue(key, value); err != nil {
			return err
		}
	}
	values = maps.Clone(values)
	err := DB.Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			stored, err := saveOptionValue(tx, key, value)
			if err != nil {
				return err
			}
			values[key] = stored
		}
		return nil
	})
	if err != nil {
		return err
	}
	BumpSyncTopic(SyncTopicOptions)
	for k, v := range values {
		if err := updateOptionMap(k, v); err != nil {
			return err
		}
	}
	return nil
}

func updateOptionMap(key string, value string) (err error) {
	if !isSupportedOption(key) {
		common.OptionMapRWMutex.Lock()
		delete(common.OptionMap, key)
		common.OptionMapRWMutex.Unlock()
		return nil
	}
	common.OptionMapRWMutex.Lock()
	defer common.OptionMapRWMutex.Unlock()
	if (key == "HeaderNavModules" || key == "SidebarModulesAdmin") && value != "" {
		value, err = navigationOption(key, value, nil)
		if err != nil {
			return err
		}
	}
	common.OptionMap[key] = value

	// 检查是否是模型配置 - 使用更规范的方式处理
	if handleConfigUpdate(key, value) {
		return nil // 已由配置系统处理
	}

	// 处理传统配置项...
	if strings.HasSuffix(key, "Enabled") || key == "DefaultCollapseSidebar" || key == "SMTPForceAuthLogin" || key == "SMTPInsecureSkipVerify" {
		boolValue := value == "true"
		switch key {
		case "PasswordRegisterEnabled":
			common.PasswordRegisterEnabled = boolValue
		case "PasswordLoginEnabled":
			common.PasswordLoginEnabled = boolValue
		case "EmailVerificationEnabled":
			common.EmailVerificationEnabled = boolValue
		case "GitHubOAuthEnabled":
			common.GitHubOAuthEnabled = boolValue
		case "LinuxDOOAuthEnabled":
			common.LinuxDOOAuthEnabled = boolValue
		case "WeChatAuthEnabled":
			common.WeChatAuthEnabled = boolValue
		case "TelegramOAuthEnabled":
			common.TelegramOAuthEnabled = boolValue
		case "TurnstileCheckEnabled":
			common.TurnstileCheckEnabled = boolValue
		case "RegisterEnabled":
			common.RegisterEnabled = boolValue
		case "EmailDomainRestrictionEnabled":
			common.EmailDomainRestrictionEnabled = boolValue
		case "EmailAliasRestrictionEnabled":
			common.EmailAliasRestrictionEnabled = boolValue

		case "DefaultCollapseSidebar":
			common.DefaultCollapseSidebar = boolValue
		case "DemoSiteEnabled":
			operation_setting.DemoSiteEnabled = boolValue
		case "SelfUseModeEnabled":
			operation_setting.SelfUseModeEnabled = boolValue
		case "SMTPSSLEnabled":
			common.SMTPSSLEnabled = boolValue
		case "SMTPStartTLSEnabled":
			common.SMTPStartTLSEnabled = boolValue
		case "SMTPInsecureSkipVerify":
			common.SMTPInsecureSkipVerify = boolValue
		case "SMTPForceAuthLogin":
			common.SMTPForceAuthLogin = boolValue
		case "WorkerAllowHttpImageRequestEnabled":
			system_setting.WorkerAllowHttpImageRequestEnabled = boolValue
		}
	}
	switch key {
	case "EmailDomainWhitelist":
		common.EmailDomainWhitelist = strings.Split(value, ",")
	case "SMTPServer":
		common.SMTPServer = value
	case "SMTPPort":
		intValue, _ := strconv.Atoi(value)
		common.SMTPPort = intValue
	case "SMTPAccount":
		common.SMTPAccount = value
	case "SMTPFrom":
		common.SMTPFrom = value
	case "SMTPToken":
		common.SMTPToken = value
	case "ServerAddress":
		system_setting.ServerAddress = value

	case "WorkerUrl":
		system_setting.WorkerUrl = value
	case "WorkerValidKey":
		system_setting.WorkerValidKey = value

	case "GitHubClientId":
		common.GitHubClientId = value
	case "GitHubClientSecret":
		common.GitHubClientSecret = value
	case "LinuxDOClientId":
		common.LinuxDOClientId = value
	case "LinuxDOClientSecret":
		common.LinuxDOClientSecret = value
	case "LinuxDOMinimumTrustLevel":
		common.LinuxDOMinimumTrustLevel, _ = strconv.Atoi(value)
	case "Footer":
		common.Footer = value
	case "SystemName":
		common.SystemName = value
	case "Logo":
		common.Logo = value
	case "WeChatServerAddress":
		common.WeChatServerAddress = value
	case "WeChatServerToken":
		common.WeChatServerToken = value
	case "WeChatAccountQRCodeImageURL":
		common.WeChatAccountQRCodeImageURL = value
	case "TelegramBotToken":
		common.TelegramBotToken = value
	case "TelegramBotName":
		common.TelegramBotName = value
	case "TurnstileSiteKey":
		common.TurnstileSiteKey = value
	case "TurnstileSecretKey":
		common.TurnstileSecretKey = value

	}
	return err
}

// handleConfigUpdate 处理分层配置更新，返回是否已处理
func handleConfigUpdate(key, value string) bool {

	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return false // 不是分层配置
	}

	configName := parts[0]
	configKey := parts[1]

	// 获取配置对象
	cfg := config.GlobalConfig.Get(configName)
	if cfg == nil {
		return false // 未注册的配置
	}

	// 更新配置
	configMap := map[string]string{
		configKey: value,
	}
	config.UpdateConfigFromMap(cfg, configMap)

	// 特定配置的后处理
	if configName == "performance_setting" {
		performance_setting.UpdateAndSync()
	}

	return true // 已处理
}
