import React, { useState, useEffect } from 'react';
import { Table, Card, Space, Button, Input, Popconfirm, message, Tag, Typography } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  DownloadOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { getFiles, deleteFile, renameFile, exportToCSV } from '../services/api.ts';
import type { FileMetadata } from '../types/index.ts';
import { useNavigate } from 'react-router-dom';

const { Text } = Typography;

const DataManage: React.FC = () => {
  const [files, setFiles] = useState<FileMetadata[]>([]);
  const [loading, setLoading] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [newName, setNewName] = useState('');
  const navigate = useNavigate();

  const loadFiles = async () => {
    setLoading(true);
    try {
      const data = await getFiles();
      setFiles(data);
    } catch (error: any) {
      message.error('Failed to load files');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadFiles();
  }, []);

  const handleDelete = async (id: number) => {
    try {
      await deleteFile(id);
      message.success('File deleted successfully');
      loadFiles();
    } catch (error: any) {
      message.error('Failed to delete file');
    }
  };

  const handleRename = async (id: number) => {
    if (!newName.trim()) {
      message.warning('Please enter a new name');
      return;
    }
    try {
      await renameFile(id, newName);
      message.success('File renamed successfully');
      setEditingId(null);
      setNewName('');
      loadFiles();
    } catch (error: any) {
      message.error(error.response?.data?.error || 'Failed to rename file');
    }
  };

  const handleViewData = (tableName: string) => {
    navigate('/query', { state: { tableName } });
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: 'Filename',
      dataIndex: 'filename',
      key: 'filename',
      render: (text: string) => <Text ellipsis>{text}</Text>,
    },
    {
      title: 'Table Name',
      dataIndex: 'table_name',
      key: 'table_name',
      render: (text: string) => <Text code>{text}</Text>,
    },
    {
      title: 'Rows',
      dataIndex: 'row_count',
      key: 'row_count',
      width: 80,
      render: (count: number) => count.toLocaleString(),
    },
    {
      title: 'Columns',
      dataIndex: 'column_count',
      key: 'column_count',
      width: 80,
    },
    {
      title: 'Upload Time',
      dataIndex: 'upload_time',
      key: 'upload_time',
      width: 180,
      render: (time: string) => new Date(time).toLocaleString(),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'gray'}>
          {status.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 200,
      render: (_: unknown, record: FileMetadata) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleViewData(record.table_name)}
          >
            View
          </Button>
          <Button
            type="link"
            size="small"
            icon={<DownloadOutlined />}
            onClick={() => exportToCSV(record.table_name)}
          >
            Export
          </Button>
          {editingId === record.id ? (
            <Input
              size="small"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onBlur={() => handleRename(record.id)}
              onPressEnter={() => handleRename(record.id)}
              autoFocus
              style={{ width: 120 }}
            />
          ) : (
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => {
                setEditingId(record.id);
                setNewName(record.table_name);
              }}
            />
          )}
          <Popconfirm
            title="Delete this file?"
            onConfirm={() => handleDelete(record.id)}
            okText="Yes"
            cancelText="No"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card
      title="Data Management"
      extra={
        <Button icon={<ReloadOutlined />} onClick={loadFiles} loading={loading}>
          Refresh
        </Button>
      }
    >
      <Table
        columns={columns}
        dataSource={files}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 20, showSizeChanger: true }}
        scroll={{ x: 1000 }}
      />
    </Card>
  );
};

export default DataManage;
