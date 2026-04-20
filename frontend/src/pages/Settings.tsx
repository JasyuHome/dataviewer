import React, { useState, useEffect } from 'react';
import { Card, Descriptions, Space, Tag, Statistic, Row, Col, message } from 'antd';
import { InfoCircleOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { healthCheck, getFiles } from '../services/api.ts';
import type { FileMetadata } from '../types/index.ts';

interface HealthStatus {
  status: string;
  version: string;
}

const Settings: React.FC = () => {
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [tableCount, setTableCount] = useState(0);
  const [totalRows, setTotalRows] = useState(0);

  useEffect(() => {
    checkHealth();
    loadStats();
  }, []);

  const checkHealth = async () => {
    try {
      const result = await healthCheck();
      setHealth(result);
    } catch (error: any) {
      console.error('Health check failed:', error);
      setHealth({ status: 'error', version: '-' });
    }
  };

  const loadStats = async () => {
    try {
      const files = await getFiles();
      setTableCount(files.length);
      setTotalRows(files.reduce((sum: number, f: FileMetadata) => sum + f.row_count, 0));
    } catch (error) {
      console.error('Failed to load stats:', error);
    }
  };

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="large">
      <Card
        title="System Settings"
        extra={<InfoCircleOutlined />}
      >
        <Descriptions bordered column={2}>
          <Descriptions.Item label="Application Name">Data Viewer</Descriptions.Item>
          <Descriptions.Item label="Version">1.0.0</Descriptions.Item>
          <Descriptions.Item label="Backend Status">
            <Space>
              {health?.status === 'ok' ? (
                <>
                  <CheckCircleOutlined style={{ color: '#52c41a' }} />
                  <Tag color="green">Online</Tag>
                </>
              ) : (
                <Tag color="red">Offline</Tag>
              )}
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="Backend Version">{health?.version || '-'}</Descriptions.Item>
          <Descriptions.Item label="Database">SQLite</Descriptions.Item>
          <Descriptions.Item label="Storage Path">./storage/uploads</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="Statistics">
        <Row gutter={16}>
          <Col span={8}>
            <Statistic
              title="Total Tables"
              value={tableCount}
              prefix={<InfoCircleOutlined />}
            />
          </Col>
          <Col span={8}>
            <Statistic
              title="Total Records"
              value={totalRows}
              precision={0}
            />
          </Col>
          <Col span={8}>
            <Statistic
              title="System Status"
              value={health?.status === 'ok' ? 'Healthy' : 'Unknown'}
              valueStyle={{ color: health?.status === 'ok' ? '#52c41a' : '#faad14' }}
            />
          </Col>
        </Row>
      </Card>

      <Card title="About">
        <Space direction="vertical" style={{ width: '100%' }}>
          <div>
            <strong>Tech Stack:</strong>
            <ul>
              <li>Frontend: React 18 + TypeScript + Ant Design + ECharts</li>
              <li>Backend: Golang 1.21 + Gin Framework</li>
              <li>Database: SQLite</li>
              <li>CSV Processing: PapaParse (frontend) + encoding/csv (backend)</li>
            </ul>
          </div>
          <div>
            <strong>Features:</strong>
            <ul>
              <li>CSV file upload with drag-and-drop support</li>
              <li>Automatic data type detection</li>
              <li>Dynamic table creation in SQLite</li>
              <li>Advanced query builder with multiple conditions</li>
              <li>Data visualization with bar, line, and pie charts</li>
              <li>Export data to CSV format</li>
            </ul>
          </div>
        </Space>
      </Card>
    </Space>
  );
};

export default Settings;
