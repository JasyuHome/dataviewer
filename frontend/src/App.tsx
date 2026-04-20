import React from 'react';
import { Layout, Menu, theme } from 'antd';
import {
  HomeOutlined,
  TableOutlined,
  SearchOutlined,
  BarChartOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { BrowserRouter, Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import Home from './pages/Home.tsx';
import DataManage from './pages/DataManage.tsx';
import DataQuery from './pages/DataQuery.tsx';
import DataViz from './pages/DataViz.tsx';
import Settings from './pages/Settings.tsx';

const { Header, Content, Footer } = Layout;

const AppContent: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const {
    token: { colorBgContainer },
  } = theme.useToken();

  const menuItems = [
    { key: '/', icon: <HomeOutlined />, label: 'Home' },
    { key: '/manage', icon: <TableOutlined />, label: 'Data Management' },
    { key: '/query', icon: <SearchOutlined />, label: 'Data Query' },
    { key: '/viz', icon: <BarChartOutlined />, label: 'Data Visualization' },
    { key: '/settings', icon: <SettingOutlined />, label: 'Settings' },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', padding: '0 24px' }}>
        <div style={{ color: 'white', fontSize: 20, fontWeight: 'bold', marginRight: 24 }}>
          Data Viewer
        </div>
        <Menu
          theme="dark"
          mode="horizontal"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{ flex: 1, minWidth: 0 }}
        />
      </Header>
      <Content style={{ padding: '24px 48px', background: colorBgContainer }}>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/manage" element={<DataManage />} />
          <Route path="/query" element={<DataQuery />} />
          <Route path="/viz" element={<DataViz />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </Content>
      <Footer style={{ textAlign: 'center' }}>
        Data Viewer ©{new Date().getFullYear()} - CSV Data Management System
      </Footer>
    </Layout>
  );
};

const App: React.FC = () => {
  return (
    <BrowserRouter>
      <AppContent />
    </BrowserRouter>
  );
};

export default App;
