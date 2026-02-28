package constant

const (

	// ConfigPath 配置文件路径
	ConfigPath = "configs/config.ini"

	// DataDir 数据目录
	DataDir = "./data"

	// DefaultDBPath 默认数据库路径
	DefaultDBPath = "./data/baihu.db"

	// WebDistDir 前端构建目录
	WebDistDir = "./web/dist"

	// DefaultRole 默认用户角色
	DefaultRole = "user"

	// AdminRole 管理员角色
	AdminRole = "admin"

	// ScriptsWorkDir 脚本工作目录
	ScriptsWorkDir = "./data/scripts"

	// CookieName Cookie 名称
	CookieName = "BHToken"

	// DefaultTaskTimeout 默认任务超时时间（分钟）
	DefaultTaskTimeout = 30

	// Settings Section 常量
	SectionSite      = "site"
	SectionSystem    = "system"
	SectionScheduler = "scheduler"
	SectionSecurity  = "security"

	// Site Settings Key 常量
	KeyTitle      = "title"
	KeySubtitle   = "subtitle"
	KeyIcon       = "icon"
	KeyPageSize       = "page_size"
	KeyCookieDays     = "cookie_days"
	KeyApiToken       = "api_token"

	// Security Settings Key 常量
	KeySecret = "secret"

	// System Settings Key 常量
	KeyInitialized = "initialized"

	// Scheduler Settings Key 常量
	KeyWorkerCount  = "worker_count"
	KeyQueueSize    = "queue_size"
	KeyRateInterval = "rate_interval"

	// WebSocket 消息类型
	WSTypeHeartbeat     = "heartbeat"
	WSTypeHeartbeatAck  = "heartbeat_ack"
	WSTypeTasks         = "tasks"
	WSTypeTaskResult    = "task_result"
	WSTypeTaskLog       = "task_log"
	WSTypeExecute       = "execute"
	WSTypeUpdate        = "update"
	WSTypeDisconnect    = "disconnect"
	WSTypeConnected     = "connected"
	WSTypeDisabled      = "disabled"
	WSTypeEnabled       = "enabled"
	WSTypeFetchTasks    = "fetch_tasks"
	WSTypeTaskHeartbeat = "task_heartbeat"
	WSTypeStop          = "stop"

	// 任务状态
	TaskStatusSuccess   = "success"
	TaskStatusFailed    = "failed"
	TaskStatusRunning   = "running"
	TaskStatusPending   = "pending"
	TaskStatusTimeout   = "timeout"
	TaskStatusCancelled = "cancelled"
	TaskStatusQueued    = "queued"

	// 工作流触发条件
	WorkflowConditionSuccess   = "success"
	WorkflowConditionOnSuccess = "on_success"
	WorkflowConditionFailed    = "failed"
	WorkflowConditionOnError   = "on_error"
	WorkflowConditionError     = "error"
	WorkflowConditionAlways    = "always"

	TaskTypeNormal  = "task"
	TaskTypeRepo    = "repo"
	TaskTypeControl = "control" // 虚拟控制节点，不执行真实任务只处理流程逻辑

	TriggerTypeCron         = "cron"
	TriggerTypeBaihuStartup = "baihu_startup"
	TriggerTypeManual       = "manual"
	TriggerTypeWorkflow     = "workflow" // 代表此任务是由另一个工作流节点触发的

	// Agent 状态
	AgentStatusOnline  = "online"
	AgentStatusOffline = "offline"
)

// TablePrefix 表前缀，从配置文件读取
var TablePrefix string

// Secret JWT和密码salt密钥，运行中自动从数据库加载
var Secret string

// DemoMode 演示模式，从环境变量读取
var DemoMode bool

// DefaultIcon 默认站点图标
var DefaultIcon = `<svg t="1766107903919" class="icon" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg" p-id="1942" width="200" height="200"><path d="M884.992 273.05984c4.10624 0 5.0688-2.36544 2.10944-5.25312 0 0-64.28672-65.55648-111.7696-75.55072-47.48288-9.984-45.47584-59.37152-80.56832-75.02848-72.0896-32.16384-158.6176-34.3552-158.6176-34.3552s-91.37152-4.92544-138.752-9.89184c-19.46624-2.03776-54.46656-9.58464-54.46656-9.58464-4.0448-0.84992-10.63936-1.19808-14.66368-0.44032 0 0-30.63808-0.07168-44.30848 43.35616-8.8576 28.11904 1.792 104.79616 1.792 104.79616 1.46432 12.27776-3.42016 30.21824-10.72128 40.20224 0 0-36.77184 46.03904-58.9312 100.34176-22.15936 54.30272 118.15936 145.05984 208.98816 205.27104C507.82208 613.89824 502.03648 743.424 502.03648 743.424s-74.19904-96.75776-194.00704-156.9792C188.2112 526.22336 150.30272 442.23488 150.30272 442.23488c-2.89792-5.45792-5.94944-4.94592-6.71744 1.21856 0 0-15.1552 91.61728 16.25088 147.8144 70.92224 126.88384 141.74208 112.88576 197.03808 183.27552s54.272 164.9152 54.272 164.9152-97.62816-141.29152-235.66336-205.55776c-91.53536-61.27616-74.7008-125.91104-85.49376-101.85728-10.79296 24.05376 26.73664 192.41984 65.40288 222.2592 80.57856 62.18752 94.16704 101.98016 94.16704 101.98016h175.53408S544.512 814.85824 572.928 725.73952c44.41088-139.30496 40.20224-191.26272 40.20224-191.26272 0.08192-8.22272 6.81984-14.19264 14.98112-13.29152 0 0 46.45888 4.64896 66.64192 9.68704 23.53152 5.86752 55.35744 26.20416 55.35744 26.20416 3.49184 2.14016 7.39328 0.68608 8.69376-3.1744l34.4576-102.77888c1.30048-3.8912-0.63488-5.51936-4.352-3.75808 0 0-45.37344 25.09824-88.17664 12.1856-20.15232-6.08256-59.60704-14.82752-74.69056-32.6656-16.95744-20.03968-15.59552-71.3728 26.66496-79.21664 48.0256-8.9088 33.13664 15.14496 65.91488 24.64768 27.42272 7.95648 22.29248-1.69984 26.69568 5.21216 1.67936 2.63168 0.38912 32.65536 0.38912 32.65536-0.21504 6.144 3.39968 7.84384 8.0384 3.79904l41.89184-36.46464c17.37728-11.24352 30.86336-0.57344 48.24064-11.81696 14.73536-9.53344 29.58336-43.66336 29.58336-43.66336 4.48512-9.24672 0.60416-20.41856-8.63232-24.92416l-42.58816-20.80768c-3.69664-1.80224-3.38944-3.26656 0.74752-3.26656h62.0032zM422.54336 123.87328s-49.88928 50.7392-74.5472 50.66752c-24.65792-0.07168-33.24928-56.12544-3.92192-56.12544 18.00192 0 76.32896 0.21504 76.32896 0.21504 4.13696 0.02048 5.0688 2.36544 2.14016 5.24288z m123.09504 249.64096s-3.31776-25.53856-33.16736-40.05888c-29.8496-14.52032-52.5312-41.13408-59.648-68.95616-12.1856-54.8864 48.29184-104.192 48.29184-104.192s-30.1056 73.6768 3.5328 106.60864c54.272 53.12512 40.99072 106.5984 40.99072 106.5984z m155.56608-164.46464c-7.94624 10.32192-9.60512 37.92896-59.2384 20.736-49.63328-17.2032-71.00416-54.5792-71.00416-54.5792-2.27328-3.40992-0.79872-6.49216 3.328-6.8096 0 0 43.55072-5.03808 80.06656 6.44096 24.91392 7.82336 54.79424 23.88992 46.848 34.21184z" fill="#272636" p-id="1943"></path><path d="M366.30528 259.26656c1.05472-1.76128 1.51552-1.57696 1.19808 0.43008 0 0-7.55712 19.0464 22.8864 87.63392 27.0848 61.02016 87.49056 68.7104 118.66112 98.23232 55.808 52.82816 53.52448 123.45344 53.52448 123.45344s-25.82528-49.85856-85.98528-78.83776-92.6208-50.52416-127.7952-85.77024c-45.02528-45.12768 17.5104-145.14176 17.5104-145.14176zM500.0704 961.44384h134.49216s95.8464-138.07616 46.68416-291.25632c-16.19968-50.46272-43.45856-80.00512-43.45856-80.00512-5.18144-6.38976-8.89856-4.88448-8.448 3.34848 0 0 9.15456 99.80928-19.44576 194.00704-28.60032 94.208-109.824 173.90592-109.824 173.90592zM681.61536 956.8768h105.24672s22.38464-73.5232 16.61952-130.21184c-9.30816-91.57632-48.31232-132.72064-48.31232-132.72064-3.80928-4.77184-6.0928-3.67616-5.2224 2.38592 0 0 16.75264 89.1904-14.4896 150.1184-30.9248 60.30336-53.84192 110.42816-53.84192 110.42816zM869.30432 811.35616c-2.88768-2.9184-4.80256-1.95584-4.38272 2.10944 0 0 6.79936 44.99456-7.76192 81.22368-10.12736 25.1904-28.91776 58.60352-28.91776 58.60352h107.17184s5.34528-50.31936-19.89632-83.97824c-11.56096-23.99232-46.21312-57.9584-46.21312-57.9584z" fill="#272636" p-id="1944"></path></svg>`

// DefaultSettings 默认系统设置
var DefaultSettings = map[string]map[string]string{
	SectionSite: {
		KeyTitle:      "白虎面板",
		KeySubtitle:   "极致轻量、高性能的自动化任务调度平台",
		KeyIcon:       DefaultIcon,
		KeyPageSize:   "10",
		KeyCookieDays: "7",
	},
	SectionScheduler: {
		KeyWorkerCount:  "4",
		KeyQueueSize:    "100",
		KeyRateInterval: "200",
	},
}
