import axios from 'axios';
import type {
  FileMetadata,
  UploadResponse,
  QueryParams,
  QueryResult,
  ChartParams,
  ChartResponse,
  NotionDatabase,
  NotionPage,
  NotionQueryRequest,
  NotionCreatePageRequest,
  NotionTableInfo,
} from '../types';

const API_BASE_URL = '/api';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// File upload APIs
export const uploadFile = async (file: File): Promise<UploadResponse> => {
  const formData = new FormData();
  formData.append('file', file);

  const response = await api.post<UploadResponse>('/upload', formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });

  return response.data;
};

export const getFiles = async (): Promise<FileMetadata[]> => {
  const response = await api.get<{ files: FileMetadata[] }>('/files');
  return response.data.files;
};

export const deleteFile = async (id: number): Promise<void> => {
  await api.delete(`/files/${id}`);
};

export const renameFile = async (id: number, newName: string): Promise<void> => {
  await api.put(`/files/${id}/rename`, { new_name: newName });
};

// Query APIs
export const queryData = async (params: QueryParams): Promise<QueryResult> => {
  const response = await api.post<QueryResult>('/query', params);
  return response.data;
};

export const getTableData = async (
  tableName: string,
  page: number = 1,
  pageSize: number = 50,
  sortField?: string,
  sortOrder?: string
): Promise<QueryResult> => {
  const response = await api.get<QueryResult>(`/tables/${tableName}/data`, {
    params: { page, page_size: pageSize, sort_field: sortField, sort_order: sortOrder },
  });
  return response.data;
};

export const getTableStructure = async (tableName: string): Promise<{ columns: { name: string; type: string }[] }> => {
  const response = await api.get(`/tables/${tableName}/structure`);
  return response.data;
};

export const exportToCSV = async (tableName: string): Promise<void> => {
  window.open(`${API_BASE_URL}/tables/${tableName}/export`);
};

// Chart APIs
export const generateChart = async (params: ChartParams): Promise<ChartResponse> => {
  const response = await api.post<ChartResponse>('/charts/generate', params);
  return response.data;
};

export const getChartData = async (
  tableName: string,
  chartType: string,
  xField: string,
  yField: string,
  limit: number = 100
): Promise<ChartResponse> => {
  const response = await api.get<ChartResponse>(`/charts/${tableName}/data`, {
    params: { type: chartType, x_field: xField, y_field: yField, limit },
  });
  return response.data;
};

// Health check
export const healthCheck = async (): Promise<{ status: string; version: string }> => {
  const response = await api.get('/health');
  return response.data as { status: string; version: string };
};

// Notion APIs
export const listNotionDatabases = async (): Promise<NotionDatabase[]> => {
  const response = await api.get<{ databases: NotionDatabase[] }>('/notion/databases');
  return response.data.databases;
};

export const getNotionDatabase = async (databaseID: string): Promise<NotionDatabase> => {
  const response = await api.get<NotionDatabase>(`/notion/databases/${databaseID}`);
  return response.data;
};

export const queryNotionDatabase = async (
  databaseID: string,
  filter?: Record<string, any>,
  sorts?: Array<{ property: string; direction: string }>,
  pageSize?: number
): Promise<{ results: NotionPage[]; has_more: boolean; next_cursor: string }> => {
  const response = await api.post(`/notion/databases/${databaseID}/query`, {
    filter,
    sorts,
    page_size: pageSize,
  });
  return response.data;
};

export const createNotionPage = async (
  databaseID: string,
  properties: Record<string, any>
): Promise<{ id: string; url: string; properties: Record<string, any> }> => {
  const response = await api.post('/notion/pages', {
    database_id: databaseID,
    properties,
  });
  return response.data;
};

export const updateNotionPage = async (
  pageID: string,
  properties: Record<string, any>
): Promise<{ id: string; url: string; properties: Record<string, any> }> => {
  const response = await api.put(`/notion/pages/${pageID}`, {
    properties,
  });
  return response.data;
};

export const deleteNotionPage = async (pageID: string): Promise<void> => {
  await api.delete(`/notion/pages/${pageID}`);
};

export const searchNotion = async (
  query: string,
  filter?: string
): Promise<{ results: NotionPage[] }> => {
  const response = await api.get('/notion/search', {
    params: { q: query, filter },
  });
  return response.data;
};

// Notion table cache APIs
export type { NotionTableInfo };

export const saveNotionData = async (
  databaseID: string,
  tableName: string
): Promise<{ message: string; table_name: string; row_count: number }> => {
  const response = await api.post('/notion/save', {
    database_id: databaseID,
    table_name: tableName,
  });
  return response.data;
};

export const syncNotionData = async (
  tableName: string
): Promise<{ message: string; table_name: string; row_count: number }> => {
  const response = await api.post('/notion/sync', {
    table_name: tableName,
  });
  return response.data;
};

export const listNotionTables = async (): Promise<{ tables: NotionTableInfo[] }> => {
  const response = await api.get('/notion/tables');
  return response.data;
};

export const deleteNotionTable = async (tableName: string): Promise<{ message: string }> => {
  const response = await api.delete(`/notion/tables/${tableName}`);
  return response.data;
};
