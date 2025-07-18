package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// 全局变量，用于存储命令行参数
var (
	frontendAPIPrefix string
	backendURL        string
	port              string
)

func init() {
	// 定义命令行参数
	flag.StringVar(&frontendAPIPrefix, "prefix", "/api/", "前端API路径前缀 (默认: /api/)")
	flag.StringVar(&backendURL, "backend", "https://chat-stage.sensetime.com/api/test-cancel/v0.0.1/", "后端服务器地址")
	flag.StringVar(&port, "port", ":8080", "代理服务器监听端口 (默认: :8080)")

	// 解析命令行参数
	flag.Parse()

	// 验证参数
	if frontendAPIPrefix == "" {
		log.Fatal("前端API前缀不能为空")
	}
	if backendURL == "" {
		log.Fatal("后端URL不能为空")
	}
	if port == "" {
		log.Fatal("端口不能为空")
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
	fmt.Printf("API Proxy Configuration:\n")
	fmt.Printf("  Frontend API Prefix: %s\n", frontendAPIPrefix)
	fmt.Printf("  Backend URL: %s\n", backendURL)
	fmt.Printf("  Port: %s\n", port)
	fmt.Printf("  Path mapping: %s* -> %s*\n", frontendAPIPrefix, backendURL)
	fmt.Println()

	// 解析后端URL
	backend, err := url.Parse(backendURL)
	if err != nil {
		log.Fatal("Failed to parse backend URL:", err)
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
		log.Printf("Proxying request: %s %s -> %s", req.Method, before, after)
		log.Printf("Path mapping: %s -> %s", originalPath, req.URL.Path)
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
		log.Printf("Response received: %s", resp.Status)

		// 处理Set-Cookie头，确保cookie能正确传递到前端
		cookies := resp.Header.Values("Set-Cookie")
		if len(cookies) > 0 {
			log.Printf("Found %d Set-Cookie headers", len(cookies))
			for i, cookie := range cookies {
				log.Printf("Set-Cookie[%d]: %s", i, cookie)
			}
		}

		// 记录其他重要的响应头
		importantHeaders := []string{"Content-Type", "Content-Length", "Cache-Control", "Access-Control-Allow-Origin"}
		for _, header := range importantHeaders {
			if values := resp.Header.Values(header); len(values) > 0 {
				for _, value := range values {
					log.Printf("Response Header %s: %s", header, value)
				}
			}
		}

		return nil
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error for %s %s: %v", r.Method, r.URL.Path, err)

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
			log.Printf("Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

			// 记录请求头信息（用于调试）
			log.Printf("Request Headers:")
			for name, values := range r.Header {
				for _, value := range values {
					log.Printf("  %s: %s", name, value)
				}
			}

			// 记录Cookie信息
			if cookies := r.Cookies(); len(cookies) > 0 {
				log.Printf("Request Cookies:")
				for _, cookie := range cookies {
					log.Printf("  %s: %s", cookie.Name, cookie.Value)
				}
			}

			// 转发请求
			proxy.ServeHTTP(w, r)
		}),
	}

	fmt.Printf("API Proxy server starting on port %s\n", port)
	fmt.Printf("Frontend API prefix: %s\n", frontendAPIPrefix)
	fmt.Printf("Backend URL: %s\n", backendURL)
	fmt.Printf("Path mapping: %s* -> %s*\n", frontendAPIPrefix, backendURL)
	fmt.Println("Press Ctrl+C to stop the server")

	// 启动服务器
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
