/**
 * 数据库概览信息
 */
export interface DBOverview {
  type: string;
  version: string;
  name: string;
  size: string;
  table_count: number;
  connections: number;
}

/**
 * 动态数据表分页数据响应
 */
export interface TableDataResponse {
  columns: string[];
  total: number;
  results: Record<string, unknown>[];
}

export interface DatabaseMaintenancePreview {
  database_type: string;
  expired_task_executions: number;
  cache_keys: number;
  cache_bytes: number;
  operations: string[];
}

export interface DatabaseMaintenanceSelection {
  cleanup_task_executions: boolean;
  clear_disk_cache: boolean;
  optimize_database: boolean;
}
