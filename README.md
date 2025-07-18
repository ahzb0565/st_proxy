# Go API 转发器

这是一个灵活的Go语言API转发器，支持将前端请求转发到后端服务器，并保留cookie凭证等。


### 可用选项

- `-prefix string`: 前端API路径前缀 (默认: "/api/")
- `-backend string`: 后端服务器地址 (默认: "https://xxx.com/api/test/v0.0.1/")
- `-port string`: 代理服务器监听端口 (默认: ":8080")

## 使用方法

```bash
# 自定义前端API前缀和后端URL
go run main.go -prefix="/v1/" -backend="https://api.example.com/v2/"

# 自定义端口
go run main.go -port=":9090"

# 完整自定义配置
go run main.go -prefix="/api/v1/" -backend="https://xxx.com/api/test/v0.0.1/" -port=":8080"
```
