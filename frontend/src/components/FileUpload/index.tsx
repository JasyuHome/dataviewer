import React, { useState } from 'react';
import { Table, message, Spin, Card } from 'antd';
import { InboxOutlined, UploadOutlined } from '@ant-design/icons';
import type { UploadFile } from 'antd/es/upload/interface';
import { uploadFile } from '../../services/api.ts';
import type { UploadResponse, ColumnDef } from '../../types/index.ts';

interface FileUploadProps {
  onUploadSuccess?: (data: UploadResponse) => void;
}

const FileUpload: React.FC<FileUploadProps> = ({ onUploadSuccess }) => {
  const [uploading, setUploading] = useState(false);
  const [preview, setPreview] = useState<UploadResponse | null>(null);
  const [pageSize, setPageSize] = useState(10);
  const [currentPage, setCurrentPage] = useState(1);

  const handleUpload = async (file: File): Promise<void> => {
    if (!file) return;

    setUploading(true);
    try {
      const result = await uploadFile(file);
      setPreview(result);
      setCurrentPage(1);
      setPageSize(10);
      message.success(`File uploaded successfully! ${result.row_count} rows imported.`);
      onUploadSuccess?.(result);
    } catch (error: any) {
      message.error(error.response?.data?.error || 'Upload failed');
    } finally {
      setUploading(false);
    }
  };

  const columns: { title: string; dataIndex: string; key: string }[] = [
    { title: 'ID', dataIndex: '0', key: '0' },
  ];

  if (preview?.columns) {
    preview.columns.forEach((col: ColumnDef, index: number) => {
      // Use original name if available, otherwise use sanitized name
      const displayName = col.original || col.name;
      columns.push({
        title: `${displayName} (${col.type})`,
        dataIndex: String(index + 1),
        key: String(index + 1),
      });
    });
  }

  const previewData = preview?.rows?.map((row, index) => ({
    key: String(index),
    ...row.reduce((acc, val, idx) => ({ ...acc, [String(idx + 1)]: val }), {}),
  })) || [];

  const totalRows = preview?.row_count || 0;

  // Slice data for current page
  const startIndex = (currentPage - 1) * pageSize;
  const endIndex = startIndex + pageSize;
  const currentPageData = previewData.slice(startIndex, endIndex);

  return (
    <div style={{ marginTop: 20 }}>
      <Card title="Upload CSV File" style={{ marginBottom: 20 }}>
        <div
          className="ant-upload-drag"
          style={{
            padding: '32px 16px',
            textAlign: 'center',
            border: '1px dashed #d9d9d9',
            borderRadius: '6px',
            cursor: 'pointer',
          }}
          onClick={() => {
            const input = document.querySelector('input[type="file"]') as HTMLInputElement;
            input?.click();
          }}
        >
          <input
            type="file"
            accept=".csv"
            style={{ display: 'none' }}
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) {
                handleUpload(file);
              }
            }}
          />
          <p className="ant-upload-drag-icon">
            {uploading ? <Spin /> : <InboxOutlined />}
          </p>
          <p className="ant-upload-text">
            {uploading ? 'Uploading...' : 'Click to select CSV file to upload'}
          </p>
          <p className="ant-upload-hint">
            Support for CSV files with automatic type detection
          </p>
        </div>
      </Card>

      {preview && (
        <Card
          title={
            <span>
              <UploadOutlined /> Preview: {preview.filename} ({preview.row_count} rows, {preview.columns.length} columns)
            </span>
          }
          size="small"
          style={{ marginBottom: 20 }}
        >
          <Table
            columns={columns}
            dataSource={currentPageData}
            pagination={{
              current: currentPage,
              pageSize: pageSize,
              total: totalRows,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total) => `Total ${total} rows`,
              pageSizeOptions: ['10', '20', '50', '100'],
              onChange: (page, size) => {
                setCurrentPage(page);
                if (size) setPageSize(size);
              },
              onShowSizeChange: (_current, size) => {
                setPageSize(size);
                setCurrentPage(1);
              },
              showPrevNextJumpers: true,
            }}
            scroll={{ x: true, y: 400 }}
            size="small"
          />
        </Card>
      )}
    </div>
  );
};

export default FileUpload;
