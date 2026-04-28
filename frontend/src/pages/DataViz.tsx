import React, { useState, useEffect, useRef } from 'react';
import { Card, Select, Button, Space, message, Typography } from 'antd';
import { BarChartOutlined, LineChartOutlined, PieChartOutlined, DownloadOutlined } from '@ant-design/icons';
import ReactECharts from 'echarts-for-react';
import { generateChart, getFiles, listNotionTables, getTableStructure } from '../services/api.ts';
import type { FileMetadata, ChartParams, NotionTableInfo } from '../types/index.ts';

const { Text } = Typography;

const DataViz: React.FC = () => {
  const [tables, setTables] = useState<Array<{ id: number | string; table_name: string; source?: 'csv' | 'notion' }>>([]);
  const [selectedTable, setSelectedTable] = useState('');
  const [columns, setColumns] = useState<{ name: string; type: string; displayName?: string }[]>([]);
  const [chartType, setChartType] = useState<'bar' | 'line' | 'pie'>('bar');
  const [xField, setXField] = useState('');
  const [yField, setYField] = useState('');
  const [chartOption, setChartOption] = useState<any>(null);
  const [chartDataCount, setChartDataCount] = useState<number>(0);
  const [dataLimit, setDataLimit] = useState<number>(100);
  const [loading, setLoading] = useState(false);
  const chartRef = useRef<ReactECharts>(null);
  const [chartKey, setChartKey] = useState(0);

  useEffect(() => {
    loadTables();
  }, []);

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

  const handleTableChange = async (tableName: string) => {
    setSelectedTable(tableName);
    try {
      const result = await getTableStructure(tableName);
      const columnsWithOriginal = result.columns.map((col: { name: string; type: string; original?: string }) => ({
        ...col,
        displayName: col.original || col.name,
      }));
      setColumns(columnsWithOriginal);
      if (columnsWithOriginal.length > 0) {
        setXField(columnsWithOriginal[0].name);
        if (columnsWithOriginal.length > 1) {
          setYField(columnsWithOriginal[1].name);
        }
      }
    } catch (error) {
      message.error('Failed to load table structure');
    }
    setChartOption(null);
  };

  const generate = async () => {
    if (!selectedTable || !xField || !yField) {
      message.warning('Please select table and fields');
      return;
    }

    setLoading(true);
    try {
      const params: ChartParams = {
        table_name: selectedTable,
        chart_type: chartType,
        x_field: xField,
        y_field: yField,
        limit: dataLimit,
      };

      const result = await generateChart(params);
      setChartOption(result.chart_config);
      setChartDataCount(result.data.length);
      setChartKey(prev => prev + 1);
      message.success(`Chart generated successfully with ${result.data.length} records`);
    } catch (error: any) {
      message.error(error.response?.data?.error || 'Failed to generate chart');
    } finally {
      setLoading(false);
    }
  };

  const handleExportChart = () => {
    if (chartRef.current) {
      const chart = chartRef.current.getEchartsInstance();
      const url = chart.getDataURL({ type: 'png', pixelRatio: 2, backgroundColor: '#fff' });
      const link = document.createElement('a');
      link.href = url;
      link.download = `chart-${selectedTable}-${chartType}.png`;
      link.click();
      message.success('Chart exported successfully');
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      <Card title="Chart Configuration" size="small">
        <Space style={{ width: '100%' }} size="middle" wrap>
          {/* Select Table */}
          <div style={{ minWidth: 200 }}>
            <div style={{ marginBottom: 8 }}>
              <Text strong>Select Table:</Text>
            </div>
            <Select
              value={selectedTable}
              onChange={handleTableChange}
              style={{ width: '100%' }}
              placeholder="Choose a table"
              options={tables.map((t) => ({ label: t.table_name, value: t.table_name }))}
            />
          </div>

          {/* Chart Type */}
          <div>
            <div style={{ marginBottom: 8 }}>
              <Text strong>Chart Type:</Text>
            </div>
            <Space>
              <Button
                type={chartType === 'bar' ? 'primary' : 'default'}
                icon={<BarChartOutlined />}
                onClick={() => setChartType('bar')}
              >
                Bar
              </Button>
              <Button
                type={chartType === 'line' ? 'primary' : 'default'}
                icon={<LineChartOutlined />}
                onClick={() => setChartType('line')}
              >
                Line
              </Button>
              <Button
                type={chartType === 'pie' ? 'primary' : 'default'}
                icon={<PieChartOutlined />}
                onClick={() => setChartType('pie')}
              >
                Pie
              </Button>
            </Space>
          </div>

          {/* X Axis */}
          <div style={{ minWidth: 180 }}>
            <div style={{ marginBottom: 8 }}>
              <Text strong>X Axis (Category):</Text>
            </div>
            <Select
              value={xField}
              onChange={setXField}
              style={{ width: '100%' }}
              placeholder="Select X field"
              options={columns.map((c) => ({ label: c.displayName || c.name, value: c.name }))}
            />
          </div>

          {/* Y Axis */}
          <div style={{ minWidth: 180 }}>
            <div style={{ marginBottom: 8 }}>
              <Text strong>Y Axis (Value):</Text>
            </div>
            <Select
              value={yField}
              onChange={setYField}
              style={{ width: '100%' }}
              placeholder="Select Y field"
              options={columns.map((c) => ({ label: c.displayName || c.name, value: c.name }))}
            />
          </div>

          {/* Data Limit */}
          <div style={{ minWidth: 150 }}>
            <div style={{ marginBottom: 8 }}>
              <Text strong>Data Limit:</Text>
            </div>
            <Select
              value={dataLimit}
              onChange={setDataLimit}
              style={{ width: '100%' }}
              options={[
                { label: '50 records', value: 50 },
                { label: '100 records', value: 100 },
                { label: '200 records', value: 200 },
                { label: '500 records', value: 500 },
                { label: '1000 records', value: 1000 },
              ]}
            />
          </div>

          {/* Action Buttons */}
          <div style={{ marginLeft: 'auto' }}>
            <Space>
              <Button
                type="primary"
                icon={<BarChartOutlined />}
                onClick={generate}
                loading={loading}
              >
                Generate Chart
              </Button>
              <Button
                icon={<DownloadOutlined />}
                onClick={handleExportChart}
                disabled={!chartOption}
              >
                Export PNG
              </Button>
            </Space>
          </div>
        </Space>
      </Card>

      <Card
        title={
          chartOption
            ? `Chart Preview (${chartDataCount} records)`
            : 'Chart Preview'
        }
        size="small"
        styles={{ body: { padding: 0, height: 'calc(100vh - 280px)', minHeight: 500 } }}
      >
        {chartOption ? (
          <ReactECharts
            key={chartKey}
            ref={chartRef}
            option={chartOption}
            style={{ height: '100%', width: '100%' }}
            opts={{ renderer: 'canvas' }}
            notMerge={true}
          />
        ) : (
          <div
            style={{
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#999',
            }}
          >
            <div style={{ textAlign: 'center' }}>
              <BarChartOutlined style={{ fontSize: 48, marginBottom: 16 }} />
              <div>Select data and click "Generate Chart" to visualize</div>
            </div>
          </div>
        )}
      </Card>
    </div>
  );
};

export default DataViz;
