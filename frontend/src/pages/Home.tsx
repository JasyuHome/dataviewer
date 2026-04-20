import React from 'react';
import { Card, Row, Col, Statistic, Typography, Space } from 'antd';
import {
  UploadOutlined,
  TableOutlined,
  BarChartOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import FileUpload from '../components/FileUpload';

const { Title, Paragraph } = Typography;

const Home: React.FC = () => {
  const navigate = useNavigate();

  const features = [
    {
      title: 'Upload CSV',
      icon: <UploadOutlined style={{ fontSize: 48, color: '#1890ff' }} />,
      description: 'Upload and parse CSV files with automatic type detection',
      action: () => document.querySelector('.ant-upload-drag-container')?.scrollIntoView({ behavior: 'smooth' }),
    },
    {
      title: 'Data Management',
      icon: <TableOutlined style={{ fontSize: 48, color: '#52c41a' }} />,
      description: 'View, manage, and export your data tables',
      action: () => navigate('/manage'),
    },
    {
      title: 'Data Query',
      icon: <SearchOutlined style={{ fontSize: 48, color: '#faad14' }} />,
      description: 'Build complex queries with visual query builder',
      action: () => navigate('/query'),
    },
    {
      title: 'Data Visualization',
      icon: <BarChartOutlined style={{ fontSize: 48, color: '#722ed1' }} />,
      description: 'Create beautiful charts and visualizations',
      action: () => navigate('/viz'),
    },
  ];

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="large">
      <Card>
        <Row align="middle" justify="space-between">
          <Col>
            <Title level={2}>Data Viewer</Title>
            <Paragraph style={{ fontSize: 16, color: '#666' }}>
              A powerful web-based data management system for CSV files.
              Upload, query, and visualize your data with ease.
            </Paragraph>
          </Col>
          <Col>
            <Statistic title="System Status" value="Online" valueStyle={{ color: '#52c41a' }} />
          </Col>
        </Row>
      </Card>

      <FileUpload />

      <Card title="Features" size="small">
        <Row gutter={16}>
          {features.map((feature, index) => (
            <Col span={6} key={index}>
              <Card
                hoverable
                onClick={feature.action}
                style={{ textAlign: 'center', cursor: 'pointer' }}
              >
                <Space direction="vertical" size="small">
                  {feature.icon}
                  <Title level={5}>{feature.title}</Title>
                  <Paragraph style={{ fontSize: 12, color: '#999', margin: 0 }}>
                    {feature.description}
                  </Paragraph>
                </Space>
              </Card>
            </Col>
          ))}
        </Row>
      </Card>
    </Space>
  );
};

export default Home;
