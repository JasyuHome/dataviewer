// Column definition
export interface ColumnDef {
  name: string;
  type: string;
  original?: string;  // Original column name from CSV
}

// File metadata
export interface FileMetadata {
  id: number;
  filename: string;
  table_name: string;
  upload_time: string;
  row_count: number;
  column_count: number;
  status: string;
  column_defs?: ColumnDef[];
}

// Upload response
export interface UploadResponse {
  id: number;
  filename: string;
  table_name: string;
  row_count: number;
  preview: string[][];    // First 10 rows for preview
  rows?: string[][];      // All rows (for local pagination in preview)
  columns: ColumnDef[];
}

// Query condition
export interface Condition {
  field: string;
  operator: string;
  value: any;
}

// Query parameters
export interface QueryParams {
  table_name: string;
  page: number;
  page_size: number;
  sort_field: string;
  sort_order: string;
  conditions: Condition[];
  logic: string;
}

// Query result
export interface QueryResult {
  data: Record<string, any>[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// Chart parameters
export interface ChartParams {
  table_name: string;
  chart_type: string;
  x_field: string;
  y_field: string;
  series?: string;
  limit: number;
}

// Chart response
export interface ChartResponse {
  chart_type: string;
  data: Record<string, any>[];
  chart_config: any;
  x_field: string;
  y_field: string;
}

// API response types
export interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}
