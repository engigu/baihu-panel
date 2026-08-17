package constant

const (
	// IDSchemaSignature 数据库表结构指纹的固定 ID
	IDSchemaSignature = "sys_schema_sig"

	// KeySchemaSignature 数据库表结构指纹，用于判断是否需要全量自动建表
	KeySchemaSignature = "schema_signature"
	
	// KeyTaskEnvsMigrated 任务环境变量迁移标记 (v2版本强制重跑过)
	KeyTaskEnvsMigrated = "task_envs_migrated_v2"
	
	// KeyTaskTagsMigrated 任务标签迁移标记 (v2版本强制重跑过)
	KeyTaskTagsMigrated = "task_tags_migrated_v2"

	// 以下是旧表或字段名，用于前置结构迁移时重命名或删除
	TableMigrateQlTokens        = "ql_tokens"
	ColumnMigrateQlTokenCode    = "code"
	ColumnMigrateQlTokenToken   = "token"
	ColumnMigrateDependencyType = "type"
)
