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
		getMongoDBTemplate(),
		getGitLabTemplate(),
		getJenkinsTemplate(),
		getSonarQubeTemplate(),
		getGrafanaTemplate(),
		getJaegerTemplate(),
		getRabbitMQTemplate(),
		getKafkaTemplate(),
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

// getMongoDBTemplate 获取 MongoDB 模板
func getMongoDBTemplate() *AppTemplate {
	params := []TemplateParameter{
		{Name: "port", DisplayName: "端口", Type: ParamTypeInt, DefaultValue: 27017, Required: true},
		{Name: "root_username", DisplayName: "Root 用户名", Type: ParamTypeString, DefaultValue: "root", Required: true},
		{Name: "root_password", DisplayName: "Root 密码", Type: ParamTypePassword, Required: true},
		{Name: "data_path", DisplayName: "数据目录", Type: ParamTypePath, DefaultValue: "./mongodb_data", Required: true},
	}
	paramsJSON, _ := json.Marshal(params)
	
	return &AppTemplate{
		Name: "mongodb", DisplayName: "MongoDB", Version: "7.0", Category: CategoryDatabase,
		Description: "MongoDB 是一个基于分布式文件存储的数据库", Icon: "mongodb.png",
		Tags: "nosql,database,mongodb", Parameters: string(paramsJSON), Status: TemplateStatusPublished,
		Content: `version: '3.8'
services:
  mongodb:
    image: mongo:7.0
    container_name: {{.name}}_mongodb
    ports: ["{{.port}}:27017"]
    environment:
      MONGO_INITDB_ROOT_USERNAME: {{.root_username}}
      MONGO_INITDB_ROOT_PASSWORD: {{.root_password}}
    volumes: ["{{.data_path}}:/data/db"]
    restart: unless-stopped`,
	}
}

// getGitLabTemplate 获取 GitLab 模板
func getGitLabTemplate() *AppTemplate {
	params := []TemplateParameter{
		{Name: "http_port", DisplayName: "HTTP 端口", Type: ParamTypeInt, DefaultValue: 8080, Required: true},
		{Name: "ssh_port", DisplayName: "SSH 端口", Type: ParamTypeInt, DefaultValue: 2222, Required: true},
		{Name: "data_path", DisplayName: "数据目录", Type: ParamTypePath, DefaultValue: "./gitlab_data", Required: true},
	}
	paramsJSON, _ := json.Marshal(params)
	
	return &AppTemplate{
		Name: "gitlab", DisplayName: "GitLab CE", Version: "latest", Category: CategoryDevTools,
		Description: "GitLab 是一个开源的 DevOps 平台", Icon: "gitlab.png",
		Tags: "git,devops,ci/cd", Parameters: string(paramsJSON), Status: TemplateStatusPublished,
		Content: `version: '3.8'
services:
  gitlab:
    image: gitlab/gitlab-ce:latest
    container_name: {{.name}}_gitlab
    ports: ["{{.http_port}}:80", "{{.ssh_port}}:22"]
    volumes: ["{{.data_path}}/config:/etc/gitlab", "{{.data_path}}/logs:/var/log/gitlab", "{{.data_path}}/data:/var/opt/gitlab"]
    restart: unless-stopped`,
	}
}

// getJenkinsTemplate 获取 Jenkins 模板
func getJenkinsTemplate() *AppTemplate {
	params := []TemplateParameter{
		{Name: "port", DisplayName: "Web 端口", Type: ParamTypeInt, DefaultValue: 8080, Required: true},
		{Name: "data_path", DisplayName: "数据目录", Type: ParamTypePath, DefaultValue: "./jenkins_data", Required: true},
	}
	paramsJSON, _ := json.Marshal(params)
	
	return &AppTemplate{
		Name: "jenkins", DisplayName: "Jenkins", Version: "lts", Category: CategoryDevTools,
		Description: "Jenkins 是一个开源的持续集成工具", Icon: "jenkins.png",
		Tags: "ci/cd,automation,jenkins", Parameters: string(paramsJSON), Status: TemplateStatusPublished,
		Content: `version: '3.8'
services:
  jenkins:
    image: jenkins/jenkins:lts
    container_name: {{.name}}_jenkins
    ports: ["{{.port}}:8080", "50000:50000"]
    volumes: ["{{.data_path}}:/var/jenkins_home"]
    restart: unless-stopped`,
	}
}

// getSonarQubeTemplate 获取 SonarQube 模板
func getSonarQubeTemplate() *AppTemplate {
	params := []TemplateParameter{
		{Name: "port", DisplayName: "Web 端口", Type: ParamTypeInt, DefaultValue: 9000, Required: true},
	}
	paramsJSON, _ := json.Marshal(params)
	
	return &AppTemplate{
		Name: "sonarqube", DisplayName: "SonarQube", Version: "community", Category: CategoryDevTools,
		Description: "SonarQube 是一个代码质量管理平台", Icon: "sonarqube.png",
		Tags: "code-quality,static-analysis", Parameters: string(paramsJSON), Status: TemplateStatusPublished,
		Content: `version: '3.8'
services:
  sonarqube:
    image: sonarqube:community
    container_name: {{.name}}_sonarqube
    ports: ["{{.port}}:9000"]
    restart: unless-stopped`,
	}
}

// getGrafanaTemplate 获取 Grafana 模板
func getGrafanaTemplate() *AppTemplate {
	params := []TemplateParameter{
		{Name: "port", DisplayName: "Web 端口", Type: ParamTypeInt, DefaultValue: 3000, Required: true},
		{Name: "data_path", DisplayName: "数据目录", Type: ParamTypePath, DefaultValue: "./grafana_data", Required: true},
	}
	paramsJSON, _ := json.Marshal(params)
	
	return &AppTemplate{
		Name: "grafana", DisplayName: "Grafana", Version: "latest", Category: CategoryMonitoring,
		Description: "Grafana 是一个开源的监控和可视化平台", Icon: "grafana.png",
		Tags: "monitoring,visualization,metrics", Parameters: string(paramsJSON), Status: TemplateStatusPublished,
		Content: `version: '3.8'
services:
  grafana:
    image: grafana/grafana:latest
    container_name: {{.name}}_grafana
    ports: ["{{.port}}:3000"]
    volumes: ["{{.data_path}}:/var/lib/grafana"]
    restart: unless-stopped`,
	}
}

// getJaegerTemplate 获取 Jaeger 模板
func getJaegerTemplate() *AppTemplate {
	params := []TemplateParameter{
		{Name: "ui_port", DisplayName: "UI 端口", Type: ParamTypeInt, DefaultValue: 16686, Required: true},
	}
	paramsJSON, _ := json.Marshal(params)
	
	return &AppTemplate{
		Name: "jaeger", DisplayName: "Jaeger", Version: "latest", Category: CategoryMonitoring,
		Description: "Jaeger 是一个分布式追踪系统", Icon: "jaeger.png",
		Tags: "tracing,monitoring,observability", Parameters: string(paramsJSON), Status: TemplateStatusPublished,
		Content: `version: '3.8'
services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    container_name: {{.name}}_jaeger
    ports: ["{{.ui_port}}:16686", "6831:6831/udp"]
    restart: unless-stopped`,
	}
}

// getRabbitMQTemplate 获取 RabbitMQ 模板
func getRabbitMQTemplate() *AppTemplate {
	params := []TemplateParameter{
		{Name: "port", DisplayName: "AMQP 端口", Type: ParamTypeInt, DefaultValue: 5672, Required: true},
		{Name: "management_port", DisplayName: "管理界面端口", Type: ParamTypeInt, DefaultValue: 15672, Required: true},
		{Name: "username", DisplayName: "用户名", Type: ParamTypeString, DefaultValue: "admin", Required: true},
		{Name: "password", DisplayName: "密码", Type: ParamTypePassword, Required: true},
	}
	paramsJSON, _ := json.Marshal(params)
	
	return &AppTemplate{
		Name: "rabbitmq", DisplayName: "RabbitMQ", Version: "management", Category: CategoryMessageQueue,
		Description: "RabbitMQ 是一个开源的消息代理软件", Icon: "rabbitmq.png",
		Tags: "message-queue,amqp,rabbitmq", Parameters: string(paramsJSON), Status: TemplateStatusPublished,
		Content: `version: '3.8'
services:
  rabbitmq:
    image: rabbitmq:management
    container_name: {{.name}}_rabbitmq
    ports: ["{{.port}}:5672", "{{.management_port}}:15672"]
    environment:
      RABBITMQ_DEFAULT_USER: {{.username}}
      RABBITMQ_DEFAULT_PASS: {{.password}}
    restart: unless-stopped`,
	}
}

// getKafkaTemplate 获取 Kafka 模板
func getKafkaTemplate() *AppTemplate {
	params := []TemplateParameter{
		{Name: "port", DisplayName: "Kafka 端口", Type: ParamTypeInt, DefaultValue: 9092, Required: true},
	}
	paramsJSON, _ := json.Marshal(params)
	
	return &AppTemplate{
		Name: "kafka", DisplayName: "Apache Kafka", Version: "latest", Category: CategoryMessageQueue,
		Description: "Kafka 是一个分布式流处理平台", Icon: "kafka.png",
		Tags: "message-queue,streaming,kafka", Parameters: string(paramsJSON), Status: TemplateStatusPublished,
		Content: `version: '3.8'
services:
  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    container_name: {{.name}}_zookeeper
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
  kafka:
    image: confluentinc/cp-kafka:latest
    container_name: {{.name}}_kafka
    depends_on: [zookeeper]
    ports: ["{{.port}}:9092"]
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:{{.port}}
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    restart: unless-stopped`,
	}
}
