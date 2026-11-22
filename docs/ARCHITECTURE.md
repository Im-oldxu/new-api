# 模板架构

Go 主程序组装 Gin 路由、GORM 主库和审计库。通用账号、会话、权限、审计、设置与运维逻辑位于原有的 middleware、service、controller、model 等包中。React 客户端通过注册路由、共享布局、组件和 API 客户端访问后端。

示例通过主程序的业务模块注册点接入。模块负责自己的迁移与 API，沿用 API 中间件和数据库连接。主节点在启动时执行已安装模块的迁移。身份取自已验证的请求上下文，个人数据范围体现在数据库查询条件中。跨账号处理同时检查角色和 Casbin 资源权限。

示例迁移管理 example_tickets 和 example_ticket_entries。可空处理人字段兼容旧工单；旧处理中工单可认领，已有状态保持原值。创建、编辑、认领、回复、完成操作在同一主库事务内校验版本、更新工单并写入参与者可见的记录。详情从已授权工单读取处理记录，附件沿用同一数据范围。跨工单的操作审计使用现有审计库和管理员审计入口。读取列表时只取摘要，附件由鉴权接口下载。版本号防止并发覆盖，状态推进与原状态、版本同时匹配。通用审计仅保存工单编号、动作及状态等非敏感信息。

新装只迁移已安装模块的表。卸载模块时移除注册与示例代码，既有数据库的业务数据保留。复制模板从源码、依赖锁文件和配置示例开始，数据库、会话、密钥、日志及上传内容由各安装环境独立管理。

审计全局和个人读取均经过 AdminAuth。全局个人令牌仍需 audit:read，个人审计入口限管理会话。全站读取由系统角色决定。model.GetAuditLogs 在返回数据前投影摘要和详情，过滤条件涉及受保护标识时不返回对应的特权事件；历史表保持原样。

示例通过 business-overview.ts 提供概览组件。/api/tickets/overview 按提交人聚合，/api/manage/tickets/overview 另需 ticket:read。状态分组使用整个可见范围；创建时间按 UTC 日聚合，最近记录有固定数量上限。React Query 按账号、范围和时间窗口区分，业务变更触发失效。example_ticket_settings 由示例独立迁移；读取供上传表单使用，Root 写入生效，服务端在读请求体前重新获取限制。纯底座的首页及通用内容面板独立于示例。

系统设置仅加载和写入已支持的独立键及已注册配置字段。历史未知键留在 options 表。对于包含新旧字段的顶部与侧栏导航 JSON，读取只投影当前字段；显式写入在数据库事务内合并，保留历史子字段。异常原文不能通过默认值隐式修复。管理员区审计可见性可读取旧的 console.audit 设置，实际授权仍以服务端角色规则为准。

发布检查按构建时指定的 GitHub 仓库生成 API 和发布页地址，本地缓存按仓库区分。浏览器延迟加载工单概览；移除工单后保留通用概览和启用的站点内容。

## 工具链和兼容约束

宿主 Go 模块路径为 `github.com/Im-oldxu/new-api`。宿主包导入、外部扩展和构建时的 `common.Version` 链接参数使用该路径。RelayKit 和 TokenKit 的模块标识分别由各自的 `go.mod` 定义。

- 根模块和 TokenKit 使用各自 go.mod 声明的 Go 1.26 工具链。RelayKit 使用 Go 1.25.1 或其兼容工具链。三者在各自根目录用 GOWORK=off 独立构建、测试。两个 kit 不得依赖宿主包，TokenKit 也不得依赖 RelayKit。独立协议 DTO 的可选标量使用指针以保留显式零值。完整仓库的 relaykit/README.md 和 tokenkit/README.md 分别说明其接口。应用导出包仅包含宿主。
- Go 服务沿用 Gin、GORM v2、Casbin、内存缓存和可选 Redis。Web 使用 React 19、TypeScript、Rsbuild、TanStack 和 Bun 冻结锁文件构建。优先复用现有业务组件、状态及 API 客户端。前端详细约定见 [Web 开发规则](../web/AGENTS.md)。
- SQLite、MySQL 5.7.8+、PostgreSQL 9.6+ 支持主库，独立审计库另支持 ClickHouse。修改数据库行为须在三个真实主库分别验收；迁移变更覆盖空库与有代表性的发布版非空旧库，各启动两次，检查数据、索引和约束。涉及最低版本特性的改动须在该版本验证。GORM 与方言驱动作为兼容组合维护。标准行锁使用 model 包的 lockForUpdate，SQLite 跳过不支持的 FOR UPDATE。保留旧字段、历史记录、索引及无法解析的原始设置。
- 公共 JSON 编解码使用 common/json.go；RelayKit 使用 kitutil。constant 仅放共享常量和启动参数，避免引入业务包。新增 Go 代码遵循当前 go.mod、gofmt 和现有包边界。回归优先扩展同职责的测试文件，Go 测试使用 testify 的 require/assert。
- 身份安全按 OWASP ASVS 与 Authentication、Session Management 及对应机制的 Cheat Sheets 检查。服务端校验角色、令牌范围、数据归属、二次验证及会话失效。日志、截图和审计均须排除可用凭据、恢复码及验证码。多语种邮件和通知由后端 i18n 翻译；Web 控制台翻译英文消息键，七种语言同步。保留 AGPL、NOTICE、第三方许可、上游作者与界面署名。
- GitHub 议题与 PR 沿用 .agents/github/ 检查和仓库模板。非原项目核心开发者提交 AI 辅助变更时，在 PR 说明来源。构建、启停及清理命令在独立项目和运行目录验证。

应用源码包只含运行模板及其开发资料；两个独立 kit、Electron、GitHub 工作流由完整 Git 仓库交付。
