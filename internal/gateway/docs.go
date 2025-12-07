package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OpenAPISpec OpenAPI 3.0 规范结构
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       APIInfo                `json:"info"`
	Servers    []APIServer            `json:"servers"`
	Paths      map[string]interface{} `json:"paths"`
	Components APIComponents          `json:"components"`
}

// APIInfo API信息
type APIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Contact     APIContact `json:"contact"`
}

// APIContact 联系信息
type APIContact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

// APIServer 服务器信息
type APIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// APIComponents API组件
type APIComponents struct {
	SecuritySchemes map[string]interface{} `json:"securitySchemes"`
	Schemas         map[string]interface{} `json:"schemas"`
}

// DocsGenerator API文档生成器
type DocsGenerator struct {
	gateway *Gateway
	spec    *OpenAPISpec
}

// NewDocsGenerator 创建新的文档生成器
func NewDocsGenerator(gateway *Gateway) *DocsGenerator {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: APIInfo{
			Title:       "qwq Enhanced AIOps Platform API",
			Description: "智能运维管理平台 API 文档",
			Version:     "1.0.0",
			Contact: APIContact{
				Name:  "qwq Team",
				Email: "support@qwq.ai",
				URL:   "https://qwq.ai",
			},
		},
		Servers: []APIServer{
			{
				URL:         "http://localhost:8080",
				Description: "开发环境",
			},
			{
				URL:         "https://api.qwq.ai",
				Description: "生产环境",
			},
		},
		Paths: make(map[string]interface{}),
		Components: APIComponents{
			SecuritySchemes: map[string]interface{}{
				"basicAuth": map[string]interface{}{
					"type":   "http",
					"scheme": "basic",
				},
			},
			Schemas: map[string]interface{}{
				"APIResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "请求是否成功",
						},
						"data": map[string]interface{}{
							"description": "响应数据",
						},
						"error": map[string]interface{}{
							"type":        "string",
							"description": "错误信息",
						},
						"code": map[string]interface{}{
							"type":        "integer",
							"description": "HTTP状态码",
						},
						"version": map[string]interface{}{
							"type":        "string",
							"description": "API版本",
						},
					},
				},
			},
		},
	}

	return &DocsGenerator{
		gateway: gateway,
		spec:    spec,
	}
}

// GenerateSpec 生成OpenAPI规范
func (dg *DocsGenerator) GenerateSpec() *OpenAPISpec {
	dg.generatePaths()
	return dg.spec
}

// generatePaths 生成路径文档
func (dg *DocsGenerator) generatePaths() {
	// AI Agent 服务路径
	dg.addPath("/api/v1/ai/chat", map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"AI Agent"},
			"summary":     "AI智能对话",
			"description": "与AI助手进行自然语言对话",
			"requestBody": map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"message": map[string]interface{}{
									"type":        "string",
									"description": "用户消息",
								},
								"session_id": map[string]interface{}{
									"type":        "string",
									"description": "会话ID",
								},
							},
							"required": []string{"message"},
						},
					},
				},
			},
			"responses": dg.getStandardResponses(),
		},
	})

	// 应用商店服务路径
	dg.addPath("/api/v1/apps", map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"Application Store"},
			"summary":     "获取应用列表",
			"description": "获取应用商店中的所有应用",
			"parameters": []map[string]interface{}{
				{
					"name":        "category",
					"in":          "query",
					"description": "应用分类",
					"schema":      map[string]interface{}{"type": "string"},
				},
			},
			"responses": dg.getStandardResponses(),
		},
		"post": map[string]interface{}{
			"tags":        []string{"Application Store"},
			"summary":     "安装应用",
			"description": "安装指定的应用",
			"requestBody": map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"app_id": map[string]interface{}{
									"type":        "string",
									"description": "应用ID",
								},
								"config": map[string]interface{}{
									"type":        "object",
									"description": "应用配置",
								},
							},
							"required": []string{"app_id"},
						},
					},
				},
			},
			"responses": dg.getStandardResponses(),
		},
	})

	// 容器服务路径
	dg.addPath("/api/v1/containers", map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"Container Service"},
			"summary":     "获取容器列表",
			"description": "获取所有容器的状态信息",
			"responses":   dg.getStandardResponses(),
		},
	})

	// 网站管理路径
	dg.addPath("/api/v1/websites", map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"Website Manager"},
			"summary":     "获取网站列表",
			"description": "获取所有管理的网站",
			"responses":   dg.getStandardResponses(),
		},
		"post": map[string]interface{}{
			"tags":        []string{"Website Manager"},
			"summary":     "创建网站",
			"description": "创建新的网站配置",
			"requestBody": map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"domain": map[string]interface{}{
									"type":        "string",
									"description": "域名",
								},
								"backend": map[string]interface{}{
									"type":        "string",
									"description": "后端服务地址",
								},
							},
							"required": []string{"domain", "backend"},
						},
					},
				},
			},
			"responses": dg.getStandardResponses(),
		},
	})

	// 数据库管理路径
	dg.addPath("/api/v1/databases", map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"Database Manager"},
			"summary":     "获取数据库连接列表",
			"description": "获取所有数据库连接",
			"responses":   dg.getStandardResponses(),
		},
	})

	// 备份服务路径
	dg.addPath("/api/v1/backups", map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"Backup Service"},
			"summary":     "获取备份列表",
			"description": "获取所有备份记录",
			"responses":   dg.getStandardResponses(),
		},
	})

	// 监控服务路径
	dg.addPath("/api/v1/monitoring/metrics", map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"Monitoring"},
			"summary":     "获取监控指标",
			"description": "获取系统监控指标数据",
			"responses":   dg.getStandardResponses(),
		},
	})

	// 网关管理路径
	dg.addPath("/api/v1/gateway/services", map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"Gateway"},
			"summary":     "获取服务列表",
			"description": "获取网关注册的所有服务",
			"responses":   dg.getStandardResponses(),
		},
	})
}

// addPath 添加路径到规范
func (dg *DocsGenerator) addPath(path string, pathItem interface{}) {
	dg.spec.Paths[path] = pathItem
}

// getStandardResponses 获取标准响应定义
func (dg *DocsGenerator) getStandardResponses() map[string]interface{} {
	return map[string]interface{}{
		"200": map[string]interface{}{
			"description": "成功",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/APIResponse",
					},
				},
			},
		},
		"400": map[string]interface{}{
			"description": "请求参数错误",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/APIResponse",
					},
				},
			},
		},
		"401": map[string]interface{}{
			"description": "未授权",
		},
		"404": map[string]interface{}{
			"description": "资源不存在",
		},
		"500": map[string]interface{}{
			"description": "服务器内部错误",
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"$ref": "#/components/schemas/APIResponse",
					},
				},
			},
		},
	}
}

// ServeSwaggerUI 提供Swagger UI界面
func (dg *DocsGenerator) ServeSwaggerUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/docs/openapi.json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dg.GenerateSpec())
		return
	}

	// 简单的Swagger UI HTML
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>qwq API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/docs/openapi.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// AddDocsRoutes 添加文档路由到网关
func (g *Gateway) AddDocsRoutes() {
	docsGen := NewDocsGenerator(g)
	
	// 添加文档路由处理
	g.mu.Lock()
	defer g.mu.Unlock()
	
	// 添加到路由表
	g.routes["/docs"] = &Route{
		Path:        "/docs",
		ServiceName: "gateway-docs",
		Methods:     []string{"GET"},
	}
	
	g.routes["/api/v1/gateway/"] = &Route{
		Path:        "/api/v1/gateway/",
		ServiceName: "gateway-api",
		Methods:     []string{"GET"},
	}
	
	// 注册虚拟服务处理文档
	g.registry.mu.Lock()
	g.registry.services["gateway-docs"] = &ServiceInfo{
		Name:     "gateway-docs",
		URL:      "internal://docs",
		Health:   "",
		Status:   "healthy",
		LastSeen: time.Now(),
		Version:  "1.0",
	}
	
	g.registry.services["gateway-api"] = &ServiceInfo{
		Name:     "gateway-api",
		URL:      "internal://api",
		Health:   "",
		Status:   "healthy",
		LastSeen: time.Now(),
		Version:  "1.0",
	}
	g.registry.mu.Unlock()
	
	// Use docsGen for documentation generation
	_ = docsGen
}