package aiagent

import (
	"fmt"
	"regexp"
	"strings"
)

// initializeTemplates 初始化命令模板
func (s *NLUServiceImpl) initializeTemplates() {
	s.templates = []CommandTemplate{
		// 部署相关模板
		{
			Intent:      IntentDeploy,
			Pattern:     `部署\s*(\w+)`,
			Parameters:  []string{"service"},
			Examples:    []string{"部署nginx", "部署mysql", "部署redis"},
			Description: "部署指定的服务",
		},
		{
			Intent:      IntentInstall,
			Pattern:     `安装\s*(\w+)`,
			Parameters:  []string{"service"},
			Examples:    []string{"安装docker", "安装nginx", "安装mysql"},
			Description: "安装指定的软件或服务",
		},
		{
			Intent:      IntentCreate,
			Pattern:     `创建\s*(\w+)`,
			Parameters:  []string{"resource"},
			Examples:    []string{"创建容器", "创建数据库", "创建网站"},
			Description: "创建指定的资源",
		},
		
		// 查询相关模板
		{
			Intent:      IntentQuery,
			Pattern:     `查看\s*(\w+)`,
			Parameters:  []string{"target"},
			Examples:    []string{"查看状态", "查看日志", "查看配置"},
			Description: "查看系统或服务信息",
		},
		{
			Intent:      IntentList,
			Pattern:     `列出\s*(\w+)`,
			Parameters:  []string{"resource"},
			Examples:    []string{"列出容器", "列出服务", "列出进程"},
			Description: "列出指定类型的资源",
		},
		{
			Intent:      IntentShow,
			Pattern:     `显示\s*(\w+)`,
			Parameters:  []string{"target"},
			Examples:    []string{"显示详情", "显示配置", "显示状态"},
			Description: "显示详细信息",
		},
		
		// 管理相关模板
		{
			Intent:      IntentStart,
			Pattern:     `启动\s*(\w+)`,
			Parameters:  []string{"service"},
			Examples:    []string{"启动nginx", "启动mysql", "启动容器"},
			Description: "启动指定的服务",
		},
		{
			Intent:      IntentStop,
			Pattern:     `停止\s*(\w+)`,
			Parameters:  []string{"service"},
			Examples:    []string{"停止nginx", "停止mysql", "停止容器"},
			Description: "停止指定的服务",
		},
		{
			Intent:      IntentRestart,
			Pattern:     `重启\s*(\w+)`,
			Parameters:  []string{"service"},
			Examples:    []string{"重启nginx", "重启mysql", "重启系统"},
			Description: "重启指定的服务",
		},
		{
			Intent:      IntentUpdate,
			Pattern:     `更新\s*(\w+)`,
			Parameters:  []string{"target"},
			Examples:    []string{"更新配置", "更新系统", "更新软件"},
			Description: "更新指定的目标",
		},
		{
			Intent:      IntentDelete,
			Pattern:     `删除\s*(\w+)`,
			Parameters:  []string{"target"},
			Examples:    []string{"删除容器", "删除文件", "删除服务"},
			Description: "删除指定的资源",
		},
		
		// 诊断相关模板
		{
			Intent:      IntentDiagnose,
			Pattern:     `诊断\s*(\w+)`,
			Parameters:  []string{"target"},
			Examples:    []string{"诊断问题", "诊断性能", "诊断网络"},
			Description: "诊断系统或服务问题",
		},
		{
			Intent:      IntentAnalyze,
			Pattern:     `分析\s*(\w+)`,
			Parameters:  []string{"target"},
			Examples:    []string{"分析日志", "分析性能", "分析错误"},
			Description: "分析系统或服务状态",
		},
		{
			Intent:      IntentTroubleshoot,
			Pattern:     `排查\s*(\w+)`,
			Parameters:  []string{"problem"},
			Examples:    []string{"排查故障", "排查错误", "排查问题"},
			Description: "排查和解决问题",
		},
		
		// 配置相关模板
		{
			Intent:      IntentConfigure,
			Pattern:     `配置\s*(\w+)`,
			Parameters:  []string{"target"},
			Examples:    []string{"配置nginx", "配置数据库", "配置网络"},
			Description: "配置系统或服务",
		},
		{
			Intent:      IntentGenerate,
			Pattern:     `生成\s*(\w+)`,
			Parameters:  []string{"target"},
			Examples:    []string{"生成配置", "生成脚本", "生成证书"},
			Description: "生成配置文件或脚本",
		},
		
		// 英文模板
		{
			Intent:      IntentDeploy,
			Pattern:     `deploy\s+(\w+)`,
			Parameters:  []string{"service"},
			Examples:    []string{"deploy nginx", "deploy mysql", "deploy redis"},
			Description: "Deploy a service",
		},
		{
			Intent:      IntentQuery,
			Pattern:     `show\s+(\w+)`,
			Parameters:  []string{"target"},
			Examples:    []string{"show status", "show logs", "show config"},
			Description: "Show system or service information",
		},
		{
			Intent:      IntentStart,
			Pattern:     `start\s+(\w+)`,
			Parameters:  []string{"service"},
			Examples:    []string{"start nginx", "start mysql", "start container"},
			Description: "Start a service",
		},
		{
			Intent:      IntentStop,
			Pattern:     `stop\s+(\w+)`,
			Parameters:  []string{"service"},
			Examples:    []string{"stop nginx", "stop mysql", "stop container"},
			Description: "Stop a service",
		},
	}
}

// initializeServices 初始化服务定义
func (s *NLUServiceImpl) initializeServices() {
	s.services = []ServiceDefinition{
		// Web服务器
		{
			Name:        "nginx",
			Aliases:     []string{"web", "webserver", "反向代理"},
			Category:    "webserver",
			Description: "高性能Web服务器和反向代理",
			Ports:       []int{80, 443},
			Commands:    []string{"nginx", "systemctl nginx"},
		},
		{
			Name:        "apache",
			Aliases:     []string{"httpd", "apache2"},
			Category:    "webserver",
			Description: "Apache HTTP服务器",
			Ports:       []int{80, 443},
			Commands:    []string{"apache2", "httpd"},
		},
		{
			Name:        "caddy",
			Aliases:     []string{"caddyserver"},
			Category:    "webserver",
			Description: "现代化Web服务器，自动HTTPS",
			Ports:       []int{80, 443},
			Commands:    []string{"caddy"},
		},
		
		// 数据库
		{
			Name:        "mysql",
			Aliases:     []string{"mariadb", "数据库", "db"},
			Category:    "database",
			Description: "MySQL关系型数据库",
			Ports:       []int{3306},
			Commands:    []string{"mysql", "mysqld"},
		},
		{
			Name:        "postgresql",
			Aliases:     []string{"postgres", "pgsql"},
			Category:    "database",
			Description: "PostgreSQL关系型数据库",
			Ports:       []int{5432},
			Commands:    []string{"postgres", "psql"},
		},
		{
			Name:        "redis",
			Aliases:     []string{"缓存", "cache"},
			Category:    "cache",
			Description: "Redis内存数据库",
			Ports:       []int{6379},
			Commands:    []string{"redis-server", "redis-cli"},
		},
		{
			Name:        "mongodb",
			Aliases:     []string{"mongo", "文档数据库"},
			Category:    "database",
			Description: "MongoDB文档数据库",
			Ports:       []int{27017},
			Commands:    []string{"mongod", "mongo"},
		},
		
		// 消息队列
		{
			Name:        "rabbitmq",
			Aliases:     []string{"mq", "消息队列"},
			Category:    "messagequeue",
			Description: "RabbitMQ消息队列",
			Ports:       []int{5672, 15672},
			Commands:    []string{"rabbitmq-server"},
		},
		{
			Name:        "kafka",
			Aliases:     []string{"消息流"},
			Category:    "messagequeue",
			Description: "Apache Kafka流处理平台",
			Ports:       []int{9092},
			Commands:    []string{"kafka-server-start.sh"},
		},
		
		// 监控工具
		{
			Name:        "prometheus",
			Aliases:     []string{"监控", "metrics"},
			Category:    "monitoring",
			Description: "Prometheus监控系统",
			Ports:       []int{9090},
			Commands:    []string{"prometheus"},
		},
		{
			Name:        "grafana",
			Aliases:     []string{"仪表盘", "dashboard"},
			Category:    "monitoring",
			Description: "Grafana可视化平台",
			Ports:       []int{3000},
			Commands:    []string{"grafana-server"},
		},
		{
			Name:        "jaeger",
			Aliases:     []string{"链路追踪", "tracing"},
			Category:    "monitoring",
			Description: "Jaeger分布式追踪",
			Ports:       []int{14268, 16686},
			Commands:    []string{"jaeger-all-in-one"},
		},
		
		// 开发工具
		{
			Name:        "gitlab",
			Aliases:     []string{"git", "代码仓库"},
			Category:    "devtools",
			Description: "GitLab代码管理平台",
			Ports:       []int{80, 443, 22},
			Commands:    []string{"gitlab-ctl"},
		},
		{
			Name:        "jenkins",
			Aliases:     []string{"ci", "cd", "持续集成"},
			Category:    "devtools",
			Description: "Jenkins持续集成工具",
			Ports:       []int{8080},
			Commands:    []string{"jenkins"},
		},
		{
			Name:        "sonarqube",
			Aliases:     []string{"代码质量", "静态分析"},
			Category:    "devtools",
			Description: "SonarQube代码质量分析",
			Ports:       []int{9000},
			Commands:    []string{"sonar.sh"},
		},
		
		// 存储
		{
			Name:        "minio",
			Aliases:     []string{"对象存储", "s3"},
			Category:    "storage",
			Description: "MinIO对象存储",
			Ports:       []int{9000, 9001},
			Commands:    []string{"minio"},
		},
		{
			Name:        "nextcloud",
			Aliases:     []string{"网盘", "云存储"},
			Category:    "storage",
			Description: "NextCloud私有云存储",
			Ports:       []int{80, 443},
			Commands:    []string{"nextcloud"},
		},
		
		// 容器相关
		{
			Name:        "docker",
			Aliases:     []string{"容器", "container"},
			Category:    "container",
			Description: "Docker容器引擎",
			Ports:       []int{2375, 2376},
			Commands:    []string{"docker", "dockerd"},
		},
		{
			Name:        "kubernetes",
			Aliases:     []string{"k8s", "容器编排"},
			Category:    "orchestration",
			Description: "Kubernetes容器编排",
			Ports:       []int{6443, 8080},
			Commands:    []string{"kubectl", "kubelet"},
		},
	}
}

// compilePatterns 编译正则表达式模式
func (s *NLUServiceImpl) compilePatterns() {
	for _, template := range s.templates {
		if compiled, err := regexp.Compile(template.Pattern); err == nil {
			s.intentPatterns[template.Intent] = append(s.intentPatterns[template.Intent], *compiled)
		}
	}
}

// GetServiceByName 根据名称获取服务定义
func (s *NLUServiceImpl) GetServiceByName(name string) *ServiceDefinition {
	name = strings.ToLower(name)
	for _, service := range s.services {
		if strings.ToLower(service.Name) == name {
			return &service
		}
		for _, alias := range service.Aliases {
			if strings.ToLower(alias) == name {
				return &service
			}
		}
	}
	return nil
}

// GetServicesByCategory 根据分类获取服务列表
func (s *NLUServiceImpl) GetServicesByCategory(category string) []ServiceDefinition {
	var services []ServiceDefinition
	for _, service := range s.services {
		if service.Category == category {
			services = append(services, service)
		}
	}
	return services
}

// GetAllCategories 获取所有服务分类
func (s *NLUServiceImpl) GetAllCategories() []string {
	categories := make(map[string]bool)
	for _, service := range s.services {
		categories[service.Category] = true
	}
	
	var result []string
	for category := range categories {
		result = append(result, category)
	}
	return result
}

// MatchTemplate 匹配命令模板
func (s *NLUServiceImpl) MatchTemplate(text string) (*CommandTemplate, []string, error) {
	for _, template := range s.templates {
		if patterns, exists := s.intentPatterns[template.Intent]; exists {
			for _, pattern := range patterns {
				if matches := pattern.FindStringSubmatch(text); len(matches) > 1 {
					return &template, matches[1:], nil
				}
			}
		}
	}
	return nil, nil, fmt.Errorf("未找到匹配的模板")
}

// GetTemplatesByIntent 根据意图获取模板
func (s *NLUServiceImpl) GetTemplatesByIntent(intent Intent) []CommandTemplate {
	var templates []CommandTemplate
	for _, template := range s.templates {
		if template.Intent == intent {
			templates = append(templates, template)
		}
	}
	return templates
}

// GetExamplesByIntent 根据意图获取示例
func (s *NLUServiceImpl) GetExamplesByIntent(intent Intent) []string {
	var examples []string
	for _, template := range s.templates {
		if template.Intent == intent {
			examples = append(examples, template.Examples...)
		}
	}
	return examples
}