# 通用管理后台与工单示例

交付物包含可独立运行的管理底座、可移除的工单示例和使用新数据库的复制安装流程。新安装的用户、权限、审计和运维功能均应能在没有原网关业务表的环境中使用。旧库中的原始记录和配置供历史留存。

## 能力、入口与验收

| 通用能力 | 使用入口 | 工单调用或独立验收结果 | 复制后配置及复验 |
| --- | --- | --- | --- |
| 注册、登录、退出、会话 | 登录注册页；各角色的安全与访问页 | 三库测试通过公开注册接口建号。Edge 实测三角色登录、重新登录。后端覆盖退出、会话撤销和禁用账号后的失效。 | 为每个实例生成独立 SESSION_SECRET、CRYPTO_SECRET，初始化 Root，建立 Admin、User，分别登录后查看自己的资料与工单。 |
| OAuth、Passkey、2FA | 安全与访问；Root 配置身份提供方和可信域 | OAuth 模拟提供方、Passkey 验证与域保护、TOTP 启用及会话轮换测试通过。在纯底座的独立账号上，Edge 完成密码二次验证、TOTP 注册及新会话的密码加验证码登录。外部 OAuth 登录、实体 Passkey 和手机验证器配对待现场联调。 | 在系统设置的认证页配置实际提供方、客户端密钥、回调地址与 Passkey RP ID、可信 Origin。分别验证授权和取消、绑定、解绑、设备注册与重新登录。每个账号在安全与访问中配对自己的 TOTP。 |
| 个人访问令牌 | 安全与访问，各角色管理自己的令牌 | 三库中通过二次验证签发受限令牌，读取个人资料成功，越权读取设置返回 403，删除测试账号后原令牌失效。 | 在安全与访问中二次验证后签发最小范围令牌，验证允许的读取，撤销后重复请求确认失效。工单浏览器流程使用登录身份。 |
| 个人资料、语言和侧栏 | 个人资料，各角色 | Edge 访问个人资料。偏好保存、已有 JSON 字段保留及缓存读取通过回归测试。 | 设置本实例默认语言，按账号保存偏好。切换语言并刷新，核对工单状态、时间及侧栏。 |
| 用户管理、授权 | 用户管理，Admin 与 Root；Root 配置管理权限 | 浏览器及 API 检查 User 受限、Admin 可查看用户、Root 可管理。工单测试验证管理员获只读授权后仍无法推进状态。 | Root 给办理员授予 ticket:read 与 ticket:write。用普通账号与只读办理员分别核对访问范围和写入边界。 |
| 审计及日志脱敏 | 管理员 → 审计日志，Admin 和 Root | Admin 可读全站摘要及普通详情。Root 和角色未知事件仅显示时间、类别、结果等摘要。Root 可查看原有专属详情。User 的全局和个人审计读取均返回 403，账号安全和令牌管理仍可用。 | 默认共用主库。使用独立审计库时配置 LOG_SQL_DSN。完成工单后以 Admin、Root 检索认领、回复与完成事件，核对权限投影。 |
| 系统设置、状态及运维 | Root 的系统设置、系统信息；公开状态 API | Root 在三库读写通用设置，旧业务键拒写。状态、日志、任务列表与筛选通过接口或自动化测试。 | Root 配置站点名称、导航及部署所需选项，保存后刷新并重启核验。发布仓库的 Web 构建配置见 README。 |
| 邮件与密码恢复 | Root 配置 SMTP 或 Resend；登录页找回密码 | 本地 SMTP/TLS、邮件绑定和验证测试通过。演示库缺少发送配置，真实收信和点开重置链接待验证。 | 在系统设置 → 运维 → 邮件中配置发件人、SMTP 主机、端口、认证与 TLS，或 Resend 的 API 密钥和已验证发件人。以测试邮箱走找回密码流程并核对实际收信。 |
| HTTP、SSRF、Webhook 与文件 | 请求策略、通知服务代码、文件下载及工单附件 | DNS/IP、重定向、代理及临时文件通过定向测试。Webhook 载荷和签名经过本地验证。Edge 上传和下载附件、逐字节核对及跨账号拒绝通过。外部通知送达待配置。 | 见下方组件表。为接收端配置受保护 HTTP 可达的 HTTPS 地址。工单办理使用附件及审计，Webhook 可由接入的新业务显式调用。 |
| 工单 | 所有角色的个人工单；获权 Admin 与 Root 的全部工单 | 已验证分页、标题及状态筛选、详情、编辑、状态流转、版本冲突、附件和重启持久化。三角色浏览器操作覆盖创建到完成。 | 显式导入演示数据。Root 在工单设置中保存附件上限并重启，验证边界字节数和历史附件下载。 |

工单状态依次为待处理、处理中、已完成。“待认领”筛选尚未完成且没有处理人的工单，包括旧库中无人认领的处理中工单。“由我处理”筛选当前账号认领的处理中工单。处理人为空时人员栏显示“无”。用户可编辑自己的待处理工单，并可向自己尚未完成的工单补充说明。管理员在全部工单中认领待处理工单，处理人可回复、完成，Root 可处理已认领工单。旧版无处理人的处理中工单允许认领。完成后详情、附件和处理记录均可继续查阅。列表展示提交人、处理人和更新时间。管理员分组下的审计日志入口沿用原审计路由、数据范围及筛选，日志配置在系统设置中管理。每张工单支持一个可替换附件，演示数据由显式操作导入。安装步骤与验收状态分别维护在 README 和交接记录中。

工单概览的数量卡片、状态分布和列表采用相同的角色范围。新增趋势按 UTC 日统计近 7／30 天，最近记录按更新时间、编号倒序排列。工单模块的附件上限由 Root 保存，默认 1024 KiB，可取 1–1024 KiB。系统维护从部署指定的发布仓库检查更新，未配置时禁用。站点品牌、公告、FAQ、状态监控及邮件认证配置由通用系统设置管理。


## 系统设置的复用与启停

以下路径相对于 `/system-settings/`。页面注册和默认表单在 `web/src/features/system-settings/{site,auth,security,content,operations}/`。Root 用 `GET /api/option/` 读取设置，用 `PUT /api/option/` 保存单项。页面按改动项提交，服务端可能用 HTTP 200、`success:false` 返回业务错误。连续保存多个普通键时，先前成功的键可能已生效，可刷新核对后重试。专用 Passkey 域变更使用单独的预览及确认接口。多节点还须等待选项同步。账号密码、客户端密钥、会话及测试数据均按实例准备。

| 分节与用途 | 配置键或接口及消费位置 | 长期演示状态、复用和验收 |
| --- | --- | --- |
| 站点 → 系统信息，基础 | `SystemName, Logo, Footer, ServerAddress, About, HomePageContent, general_setting.docs_link, legal.user_agreement, legal.privacy_policy`。公开状态、首页、关于页、法律声明和登录／注册页读取。 | 默认 Admin Workspace。复制后设置自己的名称、站点基址、Logo 和正文。首页、关于和文档地址默认空。Edge 保存站名并刷新，验证自定义首页 Markdown、关于文字和文档链接的目标。Logo 可使用站点根相对路径。 |
| 站点 → 系统通知，基础 | `Notice` 经 `/api/notice` 返回。 | 演示无覆盖。保存文字、刷新及读回已验。清空可撤下当前通知。 |
| 站点 → 顶部导航／侧栏模块，基础 | `HeaderNavModules, SidebarModulesAdmin`。顶部识别 home、console、docs、about，侧栏按工作台、个人和管理员区域读取。 | 按新站点调整条目。文档导航需要填写 docs_link 并启用 docs。菜单可见性只控制入口，直接访问仍按服务端权限。设置投影、历史字段保留和角色隔离复用定版验证。 |
| 认证 → 基本身份验证，基础 | `RegisterEnabled, PasswordRegisterEnabled, PasswordLoginEnabled, EmailVerificationEnabled, EmailDomainRestrictionEnabled, EmailDomainWhitelist, EmailAliasRestrictionEnabled`。登录、注册及验证码接口直接检查相应规则。 | 演示开放密码注册、登录，未要求邮件验证。关闭注册或仅关闭密码注册时，注册表单和后端同步限制。切换前应先用另一种方式成功登录管理员账号。隔离实例已实测拒绝注册、错误密码、域限制和邮件验证码注册。 |
| 认证 → OAuth，选配 | GitHub、LinuxDO 使用各自的 OAuthEnabled、ClientId、ClientSecret；LinuxDO 另有 LinuxDOMinimumTrustLevel。Discord、OIDC、Telegram 使用对应的 `discord.*`、`oidc.*`、`telegram.*` 字段，Telegram 开关为 TelegramOAuthEnabled。WeChat 使用 WeChatAuthEnabled、WeChatServerAddress、WeChatServerToken 和二维码地址。 | 演示全部关闭。先填 ServerAddress，在提供方登记页面显示的回调，再填凭据并启用。OIDC 可通过发现文档填充授权、令牌和用户信息端点。微信需要配套验证码服务。停用后登录入口和对应新认证请求应关闭。配置、状态签发与停用已验；真实授权、拒绝授权及回调需自己的客户端与回调域。 |
| 认证 → 通行密钥，选配 | `passkey.enabled, rp_display_name, rp_id, legacy_rp_ids, origins, allow_insecure_origin, user_verification, attachment_preference`（均属 passkey 命名空间）。域设置经 `PUT /api/option/passkey/domains`。 | 演示关闭。部署 HTTPS 后填 RP ID、精确 Origin、显示名和设备偏好。在「安全与访问」注册，换会话登录并移除，验证关闭后 begin 请求拒绝。Edge 虚拟 CTAP2 已完成完整链路。实体密钥及平台认证器仍需实机。迁移 RP ID 前用预览检查已有凭据，并保留必要旧域。 |
| 认证 → 机器人保护，选配 | `TurnstileSiteKey, TurnstileSecretKey, TurnstileCheckEnabled`。公开站点键渲染 widget，服务端校验挑战令牌。 | 演示关闭。填写本站域名的两把键后启用。隔离验证公开键可回显、私钥隐藏、缺少私钥拒绝启用、开启后缺令牌拒绝登录、停用后登录恢复。真实挑战需要获准域名及 Cloudflare 键。 |
| 认证 → 自定义 OAuth，选配 | Root 的 `/api/custom-oauth-provider/` CRUD 与 discovery。配置 slug、客户端、授权／令牌／用户信息端点、scope、字段映射、认证方式和访问条件。启用项由公开状态发布。 | 演示启用数为零。本机提供方替身完成获取凭据、带二次验证的绑定、同一账号登录、解绑及停用；空密钥编辑保留原值。对真实提供方按页面的回调 URL 走同一流程。删除前先处理已有绑定，可优先停用。 |
| 安全 → SSRF，基础安全策略 | `fetch_setting.enable_ssrf_protection, allow_private_ip, domain_filter_mode, domain_list, ip_filter_mode, ip_list, allowed_ports, apply_ip_filter_for_domain`。公共客户端用于下载、Webhook、Kuma 查询和自定义发现。 | 演示使用默认保护。按部署明确允许访问的主机、IP 和端口，端口可用逗号分隔的单值或范围。Edge 已保存并回读端口范围。DNS、IP、重定向与代理策略有定向测试。隔离 Kuma 仅向明确许可的本机接收端发请求。 |
| 内容 → 公告／FAQ，选配 | `console_setting.announcements, announcements_enabled, faq, faq_enabled`。由 `/api/status` 条件发布，概览按内容渲染。 | 演示开关开启，显示还需要条目。填写内容并启用后核对概览，停用后公开状态不返回条目。有效内容、错误格式拒绝、启停及持久化已验。 |
| 内容 → Uptime Kuma，选配 | `console_setting.uptime_kuma_groups, uptime_kuma_enabled`。每组填分类名、URL 和状态页 slug，`GET /api/uptime/status` 读取组和心跳。 | 演示无配置分组。隔离状态页响应验证可见监控；关闭开关时接口返回空数组且接收端请求增量为零。生产需可达的 Kuma 公开状态页和相应出站许可。 |
| 运维 → 系统行为，基础 | `DefaultCollapseSidebar, DemoSiteEnabled, SelfUseModeEnabled`。控制侧栏初始偏好、演示标识和登录页注册链接。 | 演示／自用模式均关。自用模式控制展示，禁止注册须关闭 RegisterEnabled。三项保存和重启读回已验。 |
| 运维 → 邮件，选配 | `EmailProvider` 选择 smtp／resend。SMTPServer、SMTPPort、SMTPFrom、SMTPAccount、SMTPToken 及 TLS、STARTTLS、认证选项，或 ResendApiKey、ResendFrom。 | 演示使用 SMTP 默认方式，未配发信服务。SMTP 本地接收器实际收信并完成注册，TLS 有定向测试；Resend 仅替身验证。正式部署需可用 SMTP 账号或 Resend 已验证发件域和 API 密钥。通过测试邮箱验证发信、邮件绑定及找回链路。切换发送方式后再停用旧凭据。 |
| 运维 → 日志，基础 | `system_task_setting.retention_days` 控制已完成系统任务历史的保留期。运行日志文件管理为 `/api/performance/logs`，操作审计独立在管理员菜单。 | 正整数天数保存及重启读回，Root 日志与任务列表读取通过。定时到期和破坏性清理未在长期预览执行。复制后确定保留周期、日志目录及磁盘权限。 |
| 运维 → 性能，基础／选配 | `performance_setting.disk_cache_enabled, disk_cache_threshold_mb, disk_cache_max_size_mb, disk_cache_path, monitor_enabled, monitor_cpu_threshold, monitor_memory_threshold, monitor_disk_threshold`。运行配置同步至请求体缓存和系统采样，`/api/performance/stats` 读取。 | 演示默认采样开、磁盘缓存关。复制后选可写路径并按主机容量设阈值。保存、重启和统计接口已验，采样关闭后后台跳过采集。实际负载和阈值告警需目标主机验证。 |
| 运维 → 系统维护，选配 | 当前版本来自 `/api/status`。发布源为 Web 构建变量 `VITE_RELEASE_REPOSITORY`，配置更新需要重新构建。 | 演示未配置。沿用已验证的空配置禁用与隔离构建请求目标测试。真实版本发布需本项目公开仓库，操作见 README。 |
| 工单设置，业务专属 | 工单页的附件上限及 `example_ticket_settings` 归示例模块。 | 沿用定版的实际上传边界、重启及模块移除验收。 |
| 原网关 API 信息面板，已退役 | 旧选项 `ApiInfo, console_setting.api_info, console_setting.api_info_enabled`。 | 管理写入拒绝、运行时无注册面板。用于读取旧安装样本的历史迁移与校验保留。 |

**停用和永久移除。** 可选能力优先通过所属开关停用，同时验证匿名入口和后端发起／完成路径。永久移除提供方时，要解除 OAuth 注册、登录和账号安全页面、绑定／解绑／验证操作、状态投影、配置写入、测试及迁移注册，保留已有身份和审计数据。移除内容集成时同时解除概览、公开 API、出站请求及配置注册。邮件仍有找回密码和安全通知调用方，永久移除前须先处理这些依赖。外部 Webhook 是通知服务提供的代码接口，详见下表。

## 公共组件的复用与验证

| 能力 | 代码及调用场景 | 验收与新实例配置 |
| --- | --- | --- |
| 请求与下载 | `pkg/outboundhttp/` 负责 DNS、IP 和重定向校验。通知服务走受保护传输。 | 安全边界有 Go 测试。按部署环境配置请求策略，使用许可的测试目标验证下载和重定向。 |
| 文件生命周期 | `common/request_files.go`、请求体清理中间件回收临时资源。工单自己的附件在授权的工单记录中保存，由 `examples/tickets/` 提供下载。 | 请求清理测试、工单上传边界、下载字节及跨账号隔离已验证。在工单设置中指定上传上限。 |
| 日志与通知 | `common/log_url.go`、`logger/` 提供脱敏与运行日志。`service/user_notify.go`、`service/webhook.go` 提供通用通知发送函数。 | 日志及签名有本地测试。当前个人资料页没有 Webhook 配置表单。业务接入时传入接收 URL 及可选共享密钥，在实际接收端核对请求体、X-Webhook-Signature 和成功应答。外部送达待联调。 |
| 设置与统计 | `model/option.go` 管理通用选项。`examples/tickets/overview.go` 管理示例设置与工单统计，`web/src/business-overview.ts` 注册模块面板。 | 三角色统计、状态分布、时间趋势及列表范围已核对。授权办理员可切换个人／全部范围。移除工单时移除示例注册、设置及统计接口。 |
| 界面与接口复用 | `web/src/components/` 的表格、表单、弹窗及 `web/src/lib/api.ts` 的共享客户端支撑工单列表、回复、加载与失败重试。 | 浏览器与组件测试已覆盖。新增业务复用这些组件，通过路由和业务菜单注册页面。 |

运行模板交付根 Go 模块、React 前端和可移除的工单示例。维护仓库中的 RelayKit、TokenKit、Electron 和 CI 独立维护，工单及通用后台启动均无需加载两个 kit。
