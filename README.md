# Go API 转发器

这是一个灵活的Go语言API转发器，支持将前端请求转发到后端服务器，并保留cookie凭证等。

## 功能特性

- 支持命令行参数配置
- 保留原始请求路径
- 保留所有请求头（Headers）
- 保留查询参数
- 支持所有HTTP方法（GET, POST, PUT, DELETE等）
- 完整的Cookie处理（请求和响应）
- 详细的日志记录
- 跳过SSL证书验证（仅用于开发环境）

## 命令行参数

```bash
./proxy [选项]
```

### 可用选项

- `-prefix string`: 前端API路径前缀 (默认: "/api/")
- `-backend string`: 后端服务器地址 (默认: "https://chat-stage.sensetime.com/api/test-cancel/v0.0.1/")
- `-port string`: 代理服务器监听端口 (默认: ":8080")

## 使用方法

### 1. 使用默认配置启动

```bash
cd go_proxy
go run main.go
```

### 2. 使用自定义参数启动

```bash
# 自定义前端API前缀和后端URL
go run main.go -prefix="/v1/" -backend="https://api.example.com/v2/"

# 自定义端口
go run main.go -port=":9090"

# 完整自定义配置
go run main.go -prefix="/api/v1/" -backend="https://chat-stage.sensetime.com/api/test-cancel/v0.0.1/" -port=":8080"
```

### 3. 构建并运行

```bash
# 构建可执行文件
go build -o proxy main.go

# 运行
./proxy -prefix="/api/" -backend="https://chat-stage.sensetime.com/api/test-cancel/v0.0.1/"
```

## 请求转发规则

### 默认配置示例

| 前端请求 | 后端转发 |
|---------|---------|
| `/api/users` | `https://chat-stage.sensetime.com/api/test-cancel/v0.0.1/users` |
| `/api/login` | `https://chat-stage.sensetime.com/api/test-cancel/v0.0.1/login` |
| `/api/data?page=1` | `https://chat-stage.sensetime.com/api/test-cancel/v0.0.1/data?page=1` |

### 自定义前缀示例

如果使用 `-prefix="/v1/"`：

| 前端请求 | 后端转发 |
|---------|---------|
| `/v1/users` | `https://chat-stage.sensetime.com/api/test-cancel/v0.0.1/users` |
| `/v1/login` | `https://chat-stage.sensetime.com/api/test-cancel/v0.0.1/login` |

## 测试示例

### 1. 测试GET请求

```bash
curl -v http://localhost:8080/api/users
```

### 2. 测试POST请求

```bash
curl -v -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username": "test", "password": "password"}'
```

### 3. 测试带Cookie的请求

```bash
curl -v -b "session=abc123" http://localhost:8080/api/profile
```

### 4. 测试带查询参数的请求

```bash
curl -v "http://localhost:8080/api/search?q=test&page=1"
```

## 前端配置示例

### JavaScript (Fetch API)

```javascript
// 基础请求
fetch('/api/users', {
  credentials: 'include', // 包含cookie
  headers: {
    'Content-Type': 'application/json'
  }
})

// POST请求
fetch('/api/login', {
  method: 'POST',
  credentials: 'include',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    username: 'user',
    password: 'password'
  })
})
```

### JavaScript (Axios)

```javascript
// 配置axios
axios.defaults.baseURL = 'http://localhost:8080';
axios.defaults.withCredentials = true;

// 发送请求
axios.get('/api/users');
axios.post('/api/login', { username: 'user', password: 'password' });
```

## 注意事项

1. 代理服务器配置为跳过SSL证书验证，仅适用于开发环境
2. 所有请求头都会被保留并转发
3. Cookie会自动在请求和响应中传递
4. 响应状态码和内容会原样返回给客户端
5. 详细的请求和响应日志会输出到控制台
6. 前端API前缀必须以斜杠开头和结尾
7. 后端URL会自动添加斜杠后缀（如果不存在）

## 故障排除

### 常见问题

1. **403错误**: 检查后端URL是否正确，以及服务器是否有访问限制
2. **超时错误**: 检查网络连接和后端服务器状态
3. **路径映射错误**: 确认前端API前缀配置是否正确

### 调试模式

启动时会显示详细的配置信息和请求日志，便于调试。 