package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"runtime"

	"github.com/sirupsen/logrus"
)

// 全局变量，用于存储命令行参数
var (
	frontendAPIPrefix string
	backendURL        string
	port              string
	logger            *logrus.Logger
)

func init() {
	// 初始化logrus
	logger = logrus.New()

	// 创建日志目录
	logDir := "/tmp/go_proxy"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.Fatal("Failed to create log directory:", err)
	}

	// 设置日志文件路径（按日期）
	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(logDir, fmt.Sprintf("go_proxy_%s.log", today))

	// 打开日志文件
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger.Fatal("Failed to open log file:", err)
	}

	// 设置日志输出到文件
	logger.SetOutput(file)

	// 设置日志格式，包含时间、文件行数等信息
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := filepath.Base(f.File)
			return "", fmt.Sprintf("%s:%d", filename, f.Line)
		},
	})

	// 启用调用者信息
	logger.SetReportCaller(true)

	// 设置日志级别
	logger.SetLevel(logrus.InfoLevel)

	// 定义命令行参数
	flag.StringVar(&frontendAPIPrefix, "prefix", "/api/", "前端API路径前缀 (默认: /api/)")
	flag.StringVar(&backendURL, "backend", "https://chat-stage.sensetime.com/api/test-cancel/v0.0.1/", "后端服务器地址")
	flag.StringVar(&port, "port", ":8080", "代理服务器监听端口 (默认: :8080)")

	// 解析命令行参数
	flag.Parse()

	// 验证参数
	if frontendAPIPrefix == "" {
		logger.Fatal("前端API前缀不能为空")
	}
	if backendURL == "" {
		logger.Fatal("后端URL不能为空")
	}
	if port == "" {
		logger.Fatal("端口不能为空")
	}

	// 确保前端API前缀以斜杠开头和结尾
	if !strings.HasPrefix(frontendAPIPrefix, "/") {
		frontendAPIPrefix = "/" + frontendAPIPrefix
	}
	if !strings.HasSuffix(frontendAPIPrefix, "/") {
		frontendAPIPrefix = frontendAPIPrefix + "/"
	}

	// 确保后端URL以斜杠结尾
	if !strings.HasSuffix(backendURL, "/") {
		backendURL = backendURL + "/"
	}
}

func main() {
	// 打印启动信息
	logger.Info("API Proxy Configuration:")
	logger.Infof("  Frontend API Prefix: %s", frontendAPIPrefix)
	logger.Infof("  Backend URL: %s", backendURL)
	logger.Infof("  Port: %s", port)
	logger.Infof("  Path mapping: %s* -> %s*", frontendAPIPrefix, backendURL)
	logger.Info("")

	// 解析后端URL
	backend, err := url.Parse(backendURL)
	if err != nil {
		logger.Fatal("Failed to parse backend URL:", err)
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(backend)

	// 自定义Director函数，处理路径映射和请求头
	proxy.Director = func(req *http.Request) {
		before := req.URL.String()

		// 设置目标服务器信息
		req.URL.Scheme = backend.Scheme
		req.URL.Host = backend.Host

		// 处理路径映射：移除前端API前缀，保留剩余路径
		originalPath := req.URL.Path
		if strings.HasPrefix(originalPath, frontendAPIPrefix) {
			// 移除前端API前缀
			originalPath = strings.TrimPrefix(originalPath, frontendAPIPrefix)
			// 如果路径为空，设置为根路径
			if originalPath == "" {
				originalPath = "/"
			}
		}

		// 构建后端路径
		req.URL.Path = backend.Path + strings.TrimPrefix(originalPath, "/")

		// 设置正确的Host头
		req.Host = backend.Host

		// 保留所有原始请求头，但移除可能导致问题的代理头
		req.Header.Del("X-Forwarded-Host")
		req.Header.Del("X-Real-IP")
		req.Header.Del("X-Forwarded-For")
		req.Header.Del("X-Forwarded-Proto")

		// 设置连接头
		req.Header.Set("Connection", "close")

		after := req.URL.String()
		logger.Infof("Proxying request: %s %s -> %s", req.Method, before, after)
		logger.Infof("Path mapping: %s -> %s", originalPath, req.URL.Path)
	}

	// 自定义Transport，处理TLS配置
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // 跳过SSL证书验证（仅用于开发环境）
		},
		// 设置超时时间
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       120 * time.Second,
		// 连接池设置
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		// 设置拨号超时
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// 启用HTTP/2
		ForceAttemptHTTP2: true,
	}

	// 自定义ModifyResponse函数，处理响应头和cookie
	proxy.ModifyResponse = func(resp *http.Response) error {
		logger.Infof("Response received: %s", resp.Status)

		// 处理Set-Cookie头，确保cookie能正确传递到前端
		cookies := resp.Header.Values("Set-Cookie")
		if len(cookies) > 0 {
			logger.Infof("Found %d Set-Cookie headers", len(cookies))
			for i, cookie := range cookies {
				logger.Infof("Set-Cookie[%d]: %s", i, cookie)
			}
		}

		// 记录其他重要的响应头
		importantHeaders := []string{"Content-Type", "Content-Length", "Cache-Control", "Access-Control-Allow-Origin"}
		for _, header := range importantHeaders {
			if values := resp.Header.Values(header); len(values) > 0 {
				for _, value := range values {
					logger.Infof("Response Header %s: %s", header, value)
				}
			}
		}

		return nil
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Errorf("Proxy error for %s %s: %v", r.Method, r.URL.Path, err)

		// 根据错误类型返回不同的状态码
		if strings.Contains(err.Error(), "timeout") {
			http.Error(w, "Gateway Timeout", http.StatusGatewayTimeout)
		} else if strings.Contains(err.Error(), "connection refused") {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}
	}

	// 创建HTTP服务器
	server := &http.Server{
		Addr: port,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 记录请求信息
			logger.Infof("Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

			// 记录请求头信息（用于调试）
			logger.Info("Request Headers:")
			for name, values := range r.Header {
				for _, value := range values {
					logger.Infof("  %s: %s", name, value)
				}
			}

			// 记录Cookie信息
			if cookies := r.Cookies(); len(cookies) > 0 {
				logger.Info("Request Cookies:")
				for _, cookie := range cookies {
					logger.Infof("  %s: %s", cookie.Name, cookie.Value)
				}
			}

			// 转发请求
			proxy.ServeHTTP(w, r)
		}),
	}

	logger.Infof("API Proxy server starting on port %s", port)
	logger.Infof("Frontend API prefix: %s", frontendAPIPrefix)
	logger.Infof("Backend URL: %s", backendURL)
	logger.Infof("Path mapping: %s* -> %s*", frontendAPIPrefix, backendURL)
	logger.Info("Press Ctrl+C to stop the server")

	// 启动服务器
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Server failed to start:", err)
	}
}
