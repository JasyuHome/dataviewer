import React, { useEffect, useState } from 'react';
import { Card, Table, Button, Space, message, Input, Typography, Tag, Popconfirm } from 'antd';
import {
  CloudOutlined,
  SyncOutlined,
  DatabaseOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import {
  saveNotionData,
  syncNotionData,
  listNotionTables,
  deleteNotionTable,
  type NotionTableInfo,
} from '../services/api';
import { useNavigate } from 'react-router-dom';

const { Text } = Typography;

const Notion: React.FC = () => {
  const navigate = useNavigate();
  const [cachedTables, setCachedTables] = useState<NotionTableInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [databaseId, setDatabaseId] = useState('');
  const [tableName, setTableName] = useState('');

  // Load cached tables on mount
  useEffect(() => {
    loadCachedTables();
  }, []);

  const loadCachedTables = async () => {
    setLoading(true);
    try {
      const result = await listNotionTables();
      setCachedTables(result.tables);
    } catch (error: any) {
      message.error(`Failed to load cached tables: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveData = async () => {
    if (!databaseId.trim()) {
      message.error('Please enter a Database ID');
      return;
    }
    if (!tableName.trim()) {
      message.error('Please enter a table name');
      return;
    }

    setLoading(true);
    try {
      await saveNotionData(databaseId.trim(), tableName.trim());
      message.success('Data saved successfully');
      setDatabaseId('');
      setTableName('');
      loadCachedTables();
    } catch (error: any) {
      message.error(`Failed to save data: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleSyncTable = async (table: NotionTableInfo) => {
    setLoading(true);
    try {
      await syncNotionData(table.table_name);
      message.success(`Table "${table.table_name}" synced successfully`);
      loadCachedTables();
    } catch (error: any) {
      message.error(`Failed to sync table: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteTable = async (tableName: string) => {
    setLoading(true);
    try {
      await deleteNotionTable(tableName);
      message.success('Table deleted successfully');
      loadCachedTables();
    } catch (error: any) {
      message.error(`Failed to delete table: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    {
      title: 'Table Name',
      dataIndex: 'table_name',
      key: 'table_name',
      render: (text: string, record: NotionTableInfo) => (
        <Space>
          <DatabaseOutlined />
          <Text strong>{text}</Text>
        </Space>
      ),
    },
    {
      title: 'Database ID',
      dataIndex: 'notion_database_id',
      key: 'notion_database_id',
      render: (text: string) => (
        <Text code style={{ fontSize: 12 }}>
          {text}
        </Text>
      ),
    },
    {
      title: 'Row Count',
      dataIndex: 'row_count',
      key: 'row_count',
      render: (count: number) => <Tag color="blue">{count} rows</Tag>,
    },
    {
      title: 'Last Sync',
      dataIndex: 'last_sync_time',
      key: 'last_sync_time',
      render: (_: string, record: NotionTableInfo) => (
        <Space>
          <CheckCircleOutlined />
          <Text type="secondary">{record.last_sync_time.replace('T', ' ').substring(0, 19)}</Text>
        </Space>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: NotionTableInfo) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<SyncOutlined />}
            onClick={() => handleSyncTable(record)}
          >
            Sync
          </Button>
          <Popconfirm
            title={`Delete table "${record.table_name}"?`}
            description="This will delete the local cache only"
            onConfirm={() => handleDeleteTable(record.table_name)}
            okText="Delete"
            cancelText="Cancel"
          >
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
            >
              Delete
            </Button>
          </Popconfirm>
          <Button
            type="link"
            size="small"
            onClick={() => navigate('/query', { state: { tableName: record.table_name } })}
          >
            Query
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px 0' }}>
      <Card
        title={
          <Space>
            <CloudOutlined />
            <span>Notion Integration</span>
          </Space>
        }
        extra={
          <Button icon={<SyncOutlined spin={loading} />} onClick={loadCachedTables}>
            Refresh
          </Button>
        }
      >
        {/* Save new data section */}
        <Card
          type="inner"
          title="Save Notion Data to Local Database"
          style={{ marginBottom: 24 }}
        >
          <Space direction="vertical" size="small" style={{ width: '100%' }}>
            <Space>
              <Text strong>Database ID:</Text>
              <Input
                value={databaseId}
                onChange={(e) => setDatabaseId(e.target.value)}
                placeholder="Enter Notion Database ID"
                style={{ width: 400 }}
                onPressEnter={handleSaveData}
              />
            </Space>
            <Space>
              <Text strong>Table Name:</Text>
              <Input
                value={tableName}
                onChange={(e) => setTableName(e.target.value)}
                placeholder="Enter local table name"
                style={{ width: 300 }}
                onPressEnter={handleSaveData}
              />
              <Button
                type="primary"
                onClick={handleSaveData}
                disabled={!databaseId.trim() || !tableName.trim()}
                loading={loading}
              >
                Save
              </Button>
            </Space>
          </Space>
        </Card>

        {/* Cached tables */}
        <Card type="inner" title="Cached Tables">
          <Table
            columns={columns}
            dataSource={cachedTables.map((table) => ({
              key: table.id,
              ...table,
            }))}
            loading={loading}
            pagination={false}
          />
        </Card>
      </Card>
    </div>
  );
};

export default Notion;
