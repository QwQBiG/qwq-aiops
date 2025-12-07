package appstore

import "encoding/json"

// GetBuiltinTemplates 获取内置的应用模板
func GetBuiltinTemplates() []*AppTemplate {
	return []*AppTemplate{
		getNginxTemplate(),
		getMySQLTemplate(),
		getRedisTemplate(),
		getPostgreSQLTemplate(),
		getPrometheusTemplate(),
	}
}

// getNginxTemplate 获取 Nginx 模板
func getNginxTemplate() *AppTemplate {
	params := []TemplateParameter{
		{
			Name:         "port",
			DisplayName:  "HTTP 端口",
			Description:  "Nginx 监听的 HTTP 端口",
			Type:         ParamTypeInt,
			DefaultValue: 80,
			Required:     true,
		},
		{
			Name:         "https_port",
			DisplayName:  "HTTPS 端口",
			Description:  "Nginx 监听的 HTTPS 端口",
			Type:         ParamTypeInt,
			DefaultValue: 443,
			Required:     false,
		},
		{
			Name:         "html_path",
			DisplayName:  "HTML 目录",
			Description:  "静态文件存放路径",
			Type:         ParamTypePath,
			DefaultValue: "./html",
			Required:     true,
		},
	}

	paramsJSON, _ := json.Marshal(params)

	content := `version: '3.8'
services:
  nginx:
    image: nginx:latest
    container_name: nginx-{{.port}}
    ports:
      - "{{.port}}:80"
      - "{{.https_port}}:443"
    volumes:
      - {{.html_path}}:/usr/share/nginx/html:ro
    restart: unless-stopped
    networks:
      - web

networks:
  web:
    driver: bridge
`

	return &AppTemplate{
		Name:        "nginx",
		DisplayName: "Nginx Web Server",
		Description: "高性能的 HTTP 和反向代理服务器",
		Category:    CategoryWebServer,
		Type:        TemplateTypeDockerCompose,
		Version:     "1.0.0",
		Icon:        "https://nginx.org/nginx.png",
		Author:      "qwq",
		Status:      TemplateStatusPublished,
		Tags:        "web,proxy,http",
		Content:     content,
		Parameters:  string(paramsJSON),
	}
}

// getMySQLTemplate 获取 MySQL 模板
func getMySQLTemplate() *AppTemplate {
	params := []TemplateParameter{
		{
			Name:        "root_password",
			DisplayName: "Root 密码",
			Description: "MySQL root 用户密码",
			Type:        ParamTypePassword,
			Required:    true,
			Validation:  "^.{8,}$", // 至少8位
		},
		{
			Name:         "database",
			DisplayName:  "数据库名",
			Description:  "初始化创建的数据库名称",
			Type:         ParamTypeString,
			DefaultValue: "myapp",
			Required:     true,
		},
		{
			Name:         "port",
			DisplayName:  "端口",
			Description:  "MySQL 监听端口",
			Type:         ParamTypeInt,
			DefaultValue: 3306,
			Required:     true,
		},
		{
			Name:         "data_path",
			DisplayName:  "数据目录",
			Description:  "MySQL 数据存储路径",
			Type:         ParamTypePath,
			DefaultValue: "./mysql-data",
			Required:     true,
		},
	}

	paramsJSON, _ := json.Marshal(params)

	content := `version: '3.8'
services:
  mysql:
    image: mysql:8.0
    container_name: mysql-{{.port}}
    environment:
      MYSQL_ROOT_PASSWORD: {{.root_password}}
      MYSQL_DATABASE: {{.database}}
    ports:
      - "{{.port}}:3306"
    volumes:
      - {{.data_path}}:/var/lib/mysql
    restart: unless-stopped
    networks:
      - database

networks:
  database:
    driver: bridge
`

	return &AppTemplate{
		Name:        "mysql",
		DisplayName: "MySQL Database",
		Description: "流行的开源关系型数据库",
		Category:    CategoryDatabase,
		Type:        TemplateTypeDockerCompose,
		Version:     "8.0",
		Icon:        "https://www.mysql.com/common/logos/logo-mysql-170x115.png",
		Author:      "qwq",
		Status:      TemplateStatusPublished,
		Tags:        "database,sql,mysql",
		Content:     content,
		Parameters:  string(paramsJSON),
	}
}

// getRedisTemplate 获取 Redis 模板
func getRedisTemplate() *AppTemplate {
	params := []TemplateParameter{
		{
			Name:         "port",
			DisplayName:  "端口",
			Description:  "Redis 监听端口",
			Type:         ParamTypeInt,
			DefaultValue: 6379,
			Required:     true,
		},
		{
			Name:        "password",
			DisplayName: "密码",
			Description: "Redis 访问密码（可选）",
			Type:        ParamTypePassword,
			Required:    false,
		},
		{
			Name:         "data_path",
			DisplayName:  "数据目录",
			Description:  "Redis 数据持久化路径",
			Type:         ParamTypePath,
			DefaultValue: "./redis-data",
			Required:     true,
		},
		{
			Name:        "max_memory",
			DisplayName: "最大内存",
			Description: "Redis 最大使用内存（如 256mb）",
			Type:        ParamTypeString,
			DefaultValue: "256mb",
			Required:    false,
		},
	}

	paramsJSON, _ := json.Marshal(params)

	content := `version: '3.8'
services:
  redis:
    image: redis:7-alpine
    container_name: redis-{{.port}}
    command: redis-server --requirepass {{.password}} --maxmemory {{.max_memory}} --maxmemory-policy allkeys-lru
    ports:
      - "{{.port}}:6379"
    volumes:
      - {{.data_path}}:/data
    restart: unless-stopped
    networks:
      - cache

networks:
  cache:
    driver: bridge
`

	return &AppTemplate{
		Name:        "redis",
		DisplayName: "Redis Cache",
		Description: "高性能的内存数据库和缓存",
		Category:    CategoryDatabase,
		Type:        TemplateTypeDockerCompose,
		Version:     "7.0",
		Icon:        "https://redis.io/images/redis-white.png",
		Author:      "qwq",
		Status:      TemplateStatusPublished,
		Tags:        "cache,nosql,redis",
		Content:     content,
		Parameters:  string(paramsJSON),
	}
}

// getPostgreSQLTemplate 获取 PostgreSQL 模板
func getPostgreSQLTemplate() *AppTemplate {
	params := []TemplateParameter{
		{
			Name:        "postgres_password",
			DisplayName: "Postgres 密码",
			Description: "PostgreSQL 超级用户密码",
			Type:        ParamTypePassword,
			Required:    true,
			Validation:  "^.{8,}$",
		},
		{
			Name:         "database",
			DisplayName:  "数据库名",
			Description:  "初始化创建的数据库名称",
			Type:         ParamTypeString,
			DefaultValue: "myapp",
			Required:     true,
		},
		{
			Name:         "port",
			DisplayName:  "端口",
			Description:  "PostgreSQL 监听端口",
			Type:         ParamTypeInt,
			DefaultValue: 5432,
			Required:     true,
		},
		{
			Name:         "data_path",
			DisplayName:  "数据目录",
			Description:  "PostgreSQL 数据存储路径",
			Type:         ParamTypePath,
			DefaultValue: "./postgres-data",
			Required:     true,
		},
	}

	paramsJSON, _ := json.Marshal(params)

	content := `version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    container_name: postgres-{{.port}}
    environment:
      POSTGRES_PASSWORD: {{.postgres_password}}
      POSTGRES_DB: {{.database}}
    ports:
      - "{{.port}}:5432"
    volumes:
      - {{.data_path}}:/var/lib/postgresql/data
    restart: unless-stopped
    networks:
      - database

networks:
  database:
    driver: bridge
`

	return &AppTemplate{
		Name:        "postgresql",
		DisplayName: "PostgreSQL Database",
		Description: "强大的开源对象关系型数据库",
		Category:    CategoryDatabase,
		Type:        TemplateTypeDockerCompose,
		Version:     "15.0",
		Icon:        "https://www.postgresql.org/media/img/about/press/elephant.png",
		Author:      "qwq",
		Status:      TemplateStatusPublished,
		Tags:        "database,sql,postgresql",
		Content:     content,
		Parameters:  string(paramsJSON),
	}
}

// getPrometheusTemplate 获取 Prometheus 模板
func getPrometheusTemplate() *AppTemplate {
	params := []TemplateParameter{
		{
			Name:         "port",
			DisplayName:  "Web 端口",
			Description:  "Prometheus Web UI 端口",
			Type:         ParamTypeInt,
			DefaultValue: 9090,
			Required:     true,
		},
		{
			Name:         "config_path",
			DisplayName:  "配置文件路径",
			Description:  "Prometheus 配置文件路径",
			Type:         ParamTypePath,
			DefaultValue: "./prometheus.yml",
			Required:     true,
		},
		{
			Name:         "data_path",
			DisplayName:  "数据目录",
			Description:  "Prometheus 数据存储路径",
			Type:         ParamTypePath,
			DefaultValue: "./prometheus-data",
			Required:     true,
		},
		{
			Name:         "retention",
			DisplayName:  "数据保留时间",
			Description:  "监控数据保留时间（如 15d）",
			Type:         ParamTypeString,
			DefaultValue: "15d",
			Required:     false,
		},
	}

	paramsJSON, _ := json.Marshal(params)

	content := `version: '3.8'
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus-{{.port}}
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time={{.retention}}'
    ports:
      - "{{.port}}:9090"
    volumes:
      - {{.config_path}}:/etc/prometheus/prometheus.yml:ro
      - {{.data_path}}:/prometheus
    restart: unless-stopped
    networks:
      - monitoring

networks:
  monitoring:
    driver: bridge
`

	return &AppTemplate{
		Name:        "prometheus",
		DisplayName: "Prometheus Monitoring",
		Description: "开源的监控和告警工具",
		Category:    CategoryMonitoring,
		Type:        TemplateTypeDockerCompose,
		Version:     "2.0",
		Icon:        "https://prometheus.io/assets/prometheus_logo_grey.svg",
		Author:      "qwq",
		Status:      TemplateStatusPublished,
		Tags:        "monitoring,metrics,prometheus",
		Content:     content,
		Parameters:  string(paramsJSON),
	}
}
