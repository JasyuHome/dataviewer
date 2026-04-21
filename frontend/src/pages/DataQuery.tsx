import React, { useState, useEffect } from 'react';
import { Card, Table, Button, Space, Select, Input, InputNumber, Switch, message, Row, Col, Divider, Typography } from 'antd';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  DeleteOutlined,
  ExportOutlined,
} from '@ant-design/icons';
import { queryData, getTableStructure, exportToCSV, getFiles, listNotionTables } from '../services/api.ts';
import type { Condition, QueryResult, FileMetadata, NotionTableInfo } from '../types/index.ts';
import { useLocation } from 'react-router-dom';

const { Text } = Typography;

const DataQuery: React.FC = () => {
  const location = useLocation();
  const [tables, setTables] = useState<Array<{ id: number | string; table_name: string; source?: 'csv' | 'notion' }>>([]);
  const [selectedTable, setSelectedTable] = useState<string>('');
  const [columns, setColumns] = useState<{ name: string; type: string; displayName?: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const [queryResult, setQueryResult] = useState<QueryResult | null>(null);

  // Query conditions - start with empty array, add condition when field is selected
  const [conditions, setConditions] = useState<Condition[]>([]);
  const [logic, setLogic] = useState<'AND' | 'OR'>('AND');

  // Pagination
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [sortField, setSortField] = useState('');
  const [sortOrder, setSortOrder] = useState<'ASC' | 'DESC'>('ASC');

  useEffect(() => {
    loadTables();
    const state = location.state as { tableName?: string };
    if (state?.tableName) {
      setSelectedTable(state.tableName);
      loadColumns(state.tableName);
    }
  }, []);

  // Note: Pagination-triggered queries are handled by the useEffect below (lines 133-157)
  // which runs silently without showing success messages

  const loadTables = async () => {
    try {
      // Load CSV file tables
      const csvData = await getFiles();
      // Load Notion tables
      const notionData = await listNotionTables();
      // Merge both lists
      const allTables = [
        ...csvData.map((t) => ({ id: t.id, table_name: t.table_name, source: 'csv' as const })),
        ...notionData.tables.map((t) => ({ id: t.id, table_name: t.table_name, source: 'notion' as const })),
      ];
      setTables(allTables);
    } catch (error) {
      message.error('Failed to load tables');
    }
  };

  const loadColumns = async (tableName: string) => {
    try {
      const result = await getTableStructure(tableName);
      const columnsWithOriginal = result.columns.map((col: { name: string; type: string; original?: string }) => ({
        ...col,
        displayName: col.original || col.name,
      }));
      setColumns(columnsWithOriginal);
    } catch (error) {
      message.error('Failed to load table structure');
    }
  };

  const handleTableChange = (tableName: string) => {
    setSelectedTable(tableName);
    setConditions([]); // Clear conditions when switching table
    setQueryResult(null);
    loadColumns(tableName);
  };

  const addCondition = () => {
    setConditions([...conditions, { field: '', operator: 'eq', value: '' }]);
  };

  const removeCondition = (index: number) => {
    if (conditions.length === 1) {
      // Clear all conditions when removing the last one
      setConditions([]);
      return;
    }
    setConditions(conditions.filter((_, i) => i !== index));
  };

  const updateCondition = (index: number, key: keyof Condition, value: any) => {
    const newConditions = [...conditions];
    newConditions[index] = { ...newConditions[index], [key]: value };
    setConditions(newConditions);
  };

  const executeQuery = async () => {
    if (!selectedTable) {
      message.warning('Please select a table');
      return;
    }

    setLoading(true);
    try {
      const result = await queryData({
        table_name: selectedTable,
        page,
        page_size: pageSize,
        sort_field: sortField,
        sort_order: sortOrder,
        conditions: conditions.filter((c) => c.field && c.value !== ''),
        logic,
      });
      setQueryResult(result);
      message.success(`Query completed: ${result.data.length} records`);
    } catch (error: any) {
      message.error(error.response?.data?.error || 'Query failed');
    } finally {
      setLoading(false);
    }
  };

  // Re-execute query when pagination or sorting changes (silent, no success message)
  useEffect(() => {
    if (!selectedTable || !queryResult) return;

    const executeSilent = async () => {
      setLoading(true);
      try {
        const result = await queryData({
          table_name: selectedTable,
          page,
          page_size: pageSize,
          sort_field: sortField,
          sort_order: sortOrder,
          conditions: conditions.filter((c) => c.field && c.value !== ''),
          logic,
        });
        setQueryResult(result);
      } catch (error: any) {
        message.error(error.response?.data?.error || 'Query failed');
      } finally {
        setLoading(false);
      }
    };

    executeSilent();
  }, [page, pageSize, sortField, sortOrder]);

  const handleExport = () => {
    if (selectedTable) {
      exportToCSV(selectedTable);
    }
  };

  // Build table columns with original names
  const tableColumns = queryResult?.data[0]
    ? Object.keys(queryResult.data[0]).map((key) => {
        // Find matching column by name to get displayName
        const colDef = columns.find((c) => c.name === key);
        return {
          title: colDef?.displayName || key,
          dataIndex: key,
          key,
          sortable: true,
        };
      })
    : [];

  return (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <Card title="Query Builder" size="small">
        <Row gutter={[16, 16]} align="middle">
          {/* Select Table */}
          <Col span={6}>
            <Space direction="vertical" size="small" style={{ width: '100%' }}>
              <Text strong>Select Table:</Text>
              <Select
                value={selectedTable}
                onChange={handleTableChange}
                style={{ width: '100%' }}
                placeholder="Choose a table"
                options={tables.map((t) => ({ label: t.table_name, value: t.table_name }))}
              />
            </Space>
          </Col>

          {/* Conditions */}
          <Col span={12}>
            <Space direction="vertical" size="small" style={{ width: '100%' }}>
              <Text strong>Conditions:</Text>
              {conditions.length === 0 ? (
                <Text type="secondary">No conditions (querying all data)</Text>
              ) : (
                <Space wrap>
                  {conditions.map((condition, index) => (
                    <Space key={index} align="start">
                      {index === 0 ? <Text>WHERE</Text> : <Text>{logic}</Text>}
                      <Select
                        value={condition.field}
                        onChange={(value) => updateCondition(index, 'field', value)}
                        style={{ width: 120 }}
                        placeholder="Field"
                        options={columns.map((c) => ({ label: c.displayName || c.name, value: c.name }))}
                      />
                      <Select
                        value={condition.operator}
                        onChange={(value) => updateCondition(index, 'operator', value)}
                        style={{ width: 90 }}
                        options={[
                          { label: '=', value: 'eq' },
                          { label: '≠', value: 'ne' },
                          { label: '>', value: 'gt' },
                          { label: '<', value: 'lt' },
                          { label: '≥', value: 'gte' },
                          { label: '≤', value: 'lte' },
                          { label: 'Contains', value: 'like' },
                          { label: 'In', value: 'in' },
                          { label: 'Between', value: 'between' },
                        ]}
                      />
                      <Input
                        value={condition.value}
                        onChange={(e) => updateCondition(index, 'value', e.target.value)}
                        placeholder="Value"
                        style={{ width: 100 }}
                      />
                      <Button
                        type="text"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => removeCondition(index)}
                      />
                    </Space>
                  ))}
                </Space>
              )}
              <Space>
                <Button type="dashed" size="small" icon={<PlusOutlined />} onClick={addCondition}>
                  Add Condition
                </Button>
                {conditions.length > 0 && (
                  <Switch
                    size="small"
                    checked={logic === 'AND'}
                    onChange={(checked) => setLogic(checked ? 'AND' : 'OR')}
                    checkedChildren="AND"
                    unCheckedChildren="OR"
                  />
                )}
              </Space>
            </Space>
          </Col>

          {/* Sorting & Pagination */}
          <Col span={6}>
            <Space style={{ width: '100%' }} direction="vertical" size="small">
              <Text strong>Sorting & Pagination:</Text>
              <Space wrap>
                <Select
                  value={sortField}
                  onChange={setSortField}
                  style={{ width: 120 }}
                  placeholder="Sort Field"
                  allowClear
                  options={columns.map((c) => ({ label: c.displayName || c.name, value: c.name }))}
                />
                <Select
                  value={sortOrder}
                  onChange={(value) => setSortOrder(value)}
                  style={{ width: 100 }}
                  options={[
                    { label: '↑ ASC', value: 'ASC' },
                    { label: '↓ DESC', value: 'DESC' },
                  ]}
                />
                <InputNumber
                  value={pageSize}
                  onChange={(value) => setPageSize(value || 50)}
                  min={10}
                  max={500}
                  step={10}
                  style={{ width: 100 }}
                  addonBefore="Page:"
                />
              </Space>
            </Space>
          </Col>
        </Row>

        <Divider style={{ margin: '12px 0' }} />

        <Space>
          <Button
            type="primary"
            icon={<SearchOutlined />}
            onClick={executeQuery}
            loading={loading}
          >
            Execute Query
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => setQueryResult(null)}>
            Reset
          </Button>
          <Button
            icon={<ExportOutlined />}
            onClick={handleExport}
            disabled={!selectedTable}
          >
            Export to CSV
          </Button>
        </Space>
      </Card>

      {queryResult && (
        <Card
          title={`Results: ${queryResult.total} total records (Page ${queryResult.page}/${queryResult.total_pages})`}
          size="small"
        >
          <Table
            columns={tableColumns}
            dataSource={queryResult.data.map((row, i) => ({ key: i, ...row }))}
            pagination={{
              current: page,
              pageSize,
              total: queryResult.total,
              onChange: (p, ps) => {
                setPage(p);
                setPageSize(ps);
              },
              position: ['bottomRight'],
            }}
            loading={loading}
            scroll={{ x: true, y: 'calc(100vh - 400px)' }}
            size="small"
          />
        </Card>
      )}
    </Space>
  );
};

export default DataQuery;
