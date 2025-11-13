package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ================== 配置结构 ==================

type Config struct {
	Server struct {
		Listen string `yaml:"listen"`
	} `yaml:"server"`

	Upstream struct {
		URL string `yaml:"url"`
	} `yaml:"upstream"`

	Path struct {
		ShortPrefix string `yaml:"short_prefix"`
	} `yaml:"path"`

	Limit struct {
		Max    int    `yaml:"max"`    // 每个窗口最大请求次数
		Window string `yaml:"window"` // 窗口长度字符串，如 "24h"
	} `yaml:"limit"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 一些默认值
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}
	if cfg.Path.ShortPrefix == "" {
		cfg.Path.ShortPrefix = "/s/"
	}
	if !strings.HasSuffix(cfg.Path.ShortPrefix, "/") {
		cfg.Path.ShortPrefix += "/"
	}
	if cfg.Limit.Max <= 0 {
		cfg.Limit.Max = 10
	}
	if cfg.Limit.Window == "" {
		cfg.Limit.Window = "24h"
	}

	return &cfg, nil
}

// ================== 内存版限次器 ==================

type tokenEntry struct {
	Count   int
	ResetAt time.Time
}

type TokenLimiter struct {
	mu     sync.Mutex
	data   map[string]*tokenEntry
	max    int
	window time.Duration
}

func NewTokenLimiter(max int, window time.Duration) *TokenLimiter {
	return &TokenLimiter{
		data:   make(map[string]*tokenEntry),
		max:    max,
		window: window,
	}
}

// Allow：检查是否允许并自增计数
func (l *TokenLimiter) Allow(token string) (allowed bool, current int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	e, ok := l.data[token]
	if !ok || now.After(e.ResetAt) {
		// 新 token 或窗口已过，重置计数
		e = &tokenEntry{
			Count:   0,
			ResetAt: now.Add(l.window),
		}
		l.data[token] = e
	}

	if e.Count >= l.max {
		return false, e.Count
	}

	e.Count++
	return true, e.Count
}

// ================== 工具函数：提取 token ==================

// 根据配置的 short_prefix，从 URL 路径中提取 token
// 例如 prefix = "/s/"，path = "/s/2dddddd7733847468cad114"
// 返回 "2dddddd7733847468cad114"
func extractTokenFromRequest(r *http.Request, cfg *Config) string {
	prefix := cfg.Path.ShortPrefix
	if prefix == "" {
		prefix = "/s/"
	}

	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	tokenPart := strings.TrimPrefix(path, prefix)
	tokenPart = strings.Trim(tokenPart, "/")

	if tokenPart == "" {
		return ""
	}

	// 如果后面还有多级路径，只取第一段
	if idx := strings.IndexRune(tokenPart, '/'); idx >= 0 {
		tokenPart = tokenPart[:idx]
	}

	return tokenPart
}

// ================== 反向代理到 Xboard ==================

func newXboardProxy(upstream string) *httputil.ReverseProxy {
	target, err := url.Parse(upstream)
	if err != nil {
		log.Fatalf("invalid upstream url: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// 确保 Host 头是上游的主机名
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}

	return proxy
}

// ================== 中间件：限次 + 代理 ==================

func limitAndProxy(cfg *Config, limiter *TokenLimiter, proxy *httputil.ReverseProxy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractTokenFromRequest(r, cfg)
		if token == "" {
			http.Error(w, "无效的订阅链接", http.StatusBadRequest)
			return
		}

		allowed, count := limiter.Allow(token)
		if !allowed {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = fmt.Fprintf(w, "订阅更新次数超出限制\n")
			log.Printf("订阅链接 %s 超出限制 (%d)\n", token, count)
			return
		}

		log.Printf("订阅链接 %s 允许, 当前计数=%d\n", token, count)
		proxy.ServeHTTP(w, r)
	})
}

// ================== main ==================

func main() {
	// 允许通过 -config 指定配置文件路径，默认为 ./config.yaml
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	window, err := time.ParseDuration(cfg.Limit.Window)
	if err != nil {
		log.Fatalf("无效的限流窗口: %v", err)
	}

	limiter := NewTokenLimiter(cfg.Limit.Max, window)
	proxy := newXboardProxy(cfg.Upstream.URL)

	// 健康检查
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	// 所有 /s/* 的请求走限次 + 代理
	http.Handle(cfg.Path.ShortPrefix, limitAndProxy(cfg, limiter, proxy))

	log.Printf("监听 %s, 代理到 %s, 短前缀=%s, 最大次数=%d, 窗口=%s",
		cfg.Server.Listen, cfg.Upstream.URL, cfg.Path.ShortPrefix, cfg.Limit.Max, cfg.Limit.Window)

	if err := http.ListenAndServe(cfg.Server.Listen, nil); err != nil {
		log.Fatal(err)
	}
}
