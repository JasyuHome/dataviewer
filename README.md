# Data Viewer - Web 数据管理系统

一个基于 React + Golang 的完整 Web 数据管理系统，支持 CSV 文件上传、数据存储、查询和可视化。

## 技术栈

### 前端
- React 18 + TypeScript
- Ant Design UI 组件库
- ECharts 图表库
- React Router 路由
- Axios HTTP 客户端

### 后端
- Golang 1.21+
- Gin Web 框架
- SQLite 数据库
- encoding/csv CSV 解析
- Resty HTTP 客户端

## 功能特性

### 1. CSV 文件上传
- 拖拽上传和文件选择器
- 自动检测分隔符（逗号、分号、制表符）
- 智能数据类型推断（INTEGER, REAL, TEXT, DATETIME）
- 上传后预览前 10 行数据

### 2. 数据管理
- 数据表列表展示
- 表重命名功能
- 数据表删除（带确认）
- 数据导出为 CSV

### 3. 数据查询
- 可视化查询构建器
- 多条件组合查询（AND/OR 逻辑）
- 支持的操作符：=、≠、>、<、≥、≤、包含、范围
- 分页和排序功能

### 4. 数据可视化
- 三种图表类型：柱状图、折线图、饼图
- 动态字段映射配置
- 图表导出为 PNG 图片
- 交互式图表（缩放、数据提示）

### 5. Notion 集成
- 连接 Notion 工作区
- 浏览和查询 Notion 数据库
- 创建、编辑和删除页面
- 搜索 Notion 页面

## 项目结构

```
dataviewer/
├── backend/                 # Go 后端
│   ├── main.go              # 应用入口
│   ├── config/              # 配置
│   ├── handlers/            # HTTP 处理器
│   ├── models/              # 数据模型
│   ├── services/            # 业务逻辑
│   └── storage/             # 文件存储
│
├── frontend/                # React 前端
│   ├── src/
│   │   ├── components/      # 组件
│   │   ├── pages/           # 页面
│   │   ├── services/        # API 服务
│   │   └── types/           # 类型定义
│   └── package.json
│
└── README.md
```

## 快速开始

### 启动后端

```bash
cd backend
go mod tidy
go run main.go
```

后端服务将在 http://localhost:8080 启动

### 启动前端

```bash
cd frontend
pnpm install
pnpm dev
```

前端开发服务器将在 http://localhost:3000 启动

## API 接口

### 文件上传
- `POST /api/upload` - 上传 CSV 文件
- `GET /api/files` - 获取文件列表
- `DELETE /api/files/:id` - 删除文件
- `PUT /api/files/:id/rename` - 重命名文件

### 数据查询
- `POST /api/query` - 执行数据查询
- `GET /api/tables/:tableName/data` - 获取表数据
- `GET /api/tables/:tableName/structure` - 获取表结构
- `GET /api/tables/:tableName/export` - 导出 CSV

### 图表
- `POST /api/charts/generate` - 生成图表
- `GET /api/charts/:tableName/data` - 获取图表数据

### Notion 集成
- `GET /api/notion/databases` - 获取数据库列表
- `GET /api/notion/databases/:id` - 获取数据库详情
- `POST /api/notion/databases/:id/query` - 查询数据库
- `POST /api/notion/pages` - 创建页面
- `PUT /api/notion/pages/:id` - 更新页面
- `DELETE /api/notion/pages/:id` - 删除页面
- `GET /api/notion/search` - 搜索页面

## 环境变量

| 变量名 | 默认值 | 描述 |
|--------|--------|------|
| SERVER_PORT | 8080 | 服务器端口 |
| DATABASE_PATH | ./storage/dataviewer.db | 数据库路径 |
| STORAGE_PATH | ./storage/uploads | 文件存储路径 |
| NOTION_API_KEY | - | Notion API 密钥（从 https://www.notion.so/my-integrations 获取） |
| NOTION_VERSION | 2022-06-28 | Notion API 版本 |

## 使用说明

1. 在首页上传 CSV 文件
2. 在"数据管理"页面查看和管理已上传的文件
3. 在"数据查询"页面构建查询条件并查看结果
4. 在"数据可视化"页面创建图表

## 开发说明

- 前端使用 Vite 作为构建工具
- 后端使用 Gin 框架提供 REST API
- 数据库使用 SQLite 存储元信息和业务数据
- CSV 文件自动解析并创建对应的数据表

## License

MIT
