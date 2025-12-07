package container

import (
	"strings"
	"testing"
)

func TestComposeParser_Parse(t *testing.T) {
	parser := NewComposeParser()

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "有效的基本 Compose 文件",
			content: `version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`,
			wantErr: false,
		},
		{
			name:    "空内容",
			content: "",
			wantErr: true,
		},
		{
			name: "缺少版本",
			content: `services:
  web:
    image: nginx:latest
`,
			wantErr: true,
		},
		{
			name: "缺少服务",
			content: `version: '3.8'
networks:
  frontend:
`,
			wantErr: true,
		},
		{
			name: "不支持的版本",
			content: `version: '2.0'
services:
  web:
    image: nginx:latest
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parser.Parse(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("Parse() returned nil config for valid input")
			}
		})
	}
}

func TestComposeParser_Validate(t *testing.T) {
	parser := NewComposeParser()

	tests := []struct {
		name       string
		config     *ComposeConfig
		wantValid  bool
		wantErrors int
	}{
		{
			name: "有效配置",
			config: &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image: "nginx:latest",
						Ports: []string{"80:80"},
					},
				},
			},
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name: "缺少版本",
			config: &ComposeConfig{
				Services: map[string]*Service{
					"web": {
						Image: "nginx:latest",
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "缺少服务",
			config: &ComposeConfig{
				Version:  "3.8",
				Services: map[string]*Service{},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "服务缺少 image 和 build",
			config: &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Ports: []string{"80:80"},
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "无效的端口格式",
			config: &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image: "nginx:latest",
						Ports: []string{"invalid-port"},
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "无效的重启策略",
			config: &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image:   "nginx:latest",
						Restart: "invalid-policy",
					},
				},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "未定义的网络引用",
			config: &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image:    "nginx:latest",
						Networks: []interface{}{"undefined-network"},
					},
				},
				Networks: map[string]*Network{},
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "未定义的卷引用",
			config: &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image:   "nginx:latest",
						Volumes: []string{"undefined-volume:/data"},
					},
				},
				Volumes: map[string]*Volume{},
			},
			wantValid:  false,
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Validate(tt.config)
			if result.Valid != tt.wantValid {
				t.Errorf("Validate() valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if len(result.Errors) != tt.wantErrors {
				t.Errorf("Validate() errors count = %d, want %d", len(result.Errors), tt.wantErrors)
				for _, err := range result.Errors {
					t.Logf("  Error: %s - %s", err.Field, err.Message)
				}
			}
		})
	}
}

func TestComposeParser_Render(t *testing.T) {
	parser := NewComposeParser()

	config := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image: "nginx:latest",
				Ports: []string{"80:80"},
			},
		},
	}

	rendered, err := parser.Render(config)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if rendered == "" {
		t.Error("Render() returned empty string")
	}

	// 验证渲染的内容可以被解析回来
	parsedConfig, err := parser.Parse(rendered)
	if err != nil {
		t.Errorf("Rendered content cannot be parsed: %v", err)
	}

	if parsedConfig.Version != config.Version {
		t.Errorf("Rendered version = %v, want %v", parsedConfig.Version, config.Version)
	}
}

func TestComposeParser_isValidPortMapping(t *testing.T) {
	parser := NewComposeParser()

	tests := []struct {
		port string
		want bool
	}{
		{"80", true},
		{"8080:80", true},
		{"127.0.0.1:8080:80", true},
		{"8080-8090:80-90", true},
		{"80/tcp", true},
		{"80/udp", true},
		{"8080:80/tcp", true},
		{"invalid-port", false},
		{"80:80:80:80", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.port, func(t *testing.T) {
			if got := parser.isValidPortMapping(tt.port); got != tt.want {
				t.Errorf("isValidPortMapping(%q) = %v, want %v", tt.port, got, tt.want)
			}
		})
	}
}

func TestComposeParser_GetCompletions(t *testing.T) {
	parser := NewComposeParser()

	tests := []struct {
		name     string
		context  string
		minItems int
	}{
		{
			name:     "服务级别补全",
			context:  "services:\n  web:\n    ",
			minItems: 5,
		},
		{
			name:     "镜像补全",
			context:  "services:\n  web:\n    image: ",
			minItems: 3,
		},
		{
			name:     "重启策略补全",
			context:  "services:\n  web:\n    restart: ",
			minItems: 4,
		},
		{
			name:     "网络补全",
			context:  "networks:\n  frontend:\n    ",
			minItems: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions := parser.GetCompletions(tt.context, 0)
			if len(completions) < tt.minItems {
				t.Errorf("GetCompletions() returned %d items, want at least %d", len(completions), tt.minItems)
			}
		})
	}
}

func TestComposeParser_ComplexConfiguration(t *testing.T) {
	parser := NewComposeParser()

	complexContent := `version: '3.8'

services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html
      - web_logs:/var/log/nginx
    networks:
      - frontend
    restart: always
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
    deploy:
      replicas: 2
      resources:
        limits:
          cpus: '0.5'
          memory: 512M

  api:
    build:
      context: ./api
      dockerfile: Dockerfile
    environment:
      NODE_ENV: production
      DB_HOST: db
    depends_on:
      - db
    networks:
      - frontend
      - backend

  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - db_data:/var/lib/postgresql/data
    networks:
      - backend

networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge

volumes:
  web_logs:
  db_data:
`

	// 解析
	config, err := parser.Parse(complexContent)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// 验证服务数量
	if len(config.Services) != 3 {
		t.Errorf("Services count = %d, want 3", len(config.Services))
	}

	// 验证网络数量
	if len(config.Networks) != 2 {
		t.Errorf("Networks count = %d, want 2", len(config.Networks))
	}

	// 验证卷数量
	if len(config.Volumes) != 2 {
		t.Errorf("Volumes count = %d, want 2", len(config.Volumes))
	}

	// 验证配置
	result := parser.Validate(config)
	if !result.Valid {
		t.Errorf("Validate() failed with %d errors:", len(result.Errors))
		for _, err := range result.Errors {
			t.Logf("  - %s: %s", err.Field, err.Message)
		}
	}

	// 验证特定服务的配置
	webService := config.Services["web"]
	if webService == nil {
		t.Fatal("web service not found")
	}

	if webService.Image != "nginx:latest" {
		t.Errorf("web.image = %v, want nginx:latest", webService.Image)
	}

	if len(webService.Ports) != 1 {
		t.Errorf("web.ports count = %d, want 1", len(webService.Ports))
	}

	if webService.HealthCheck == nil {
		t.Error("web.healthcheck is nil")
	}

	if webService.Deploy == nil {
		t.Error("web.deploy is nil")
	} else if webService.Deploy.Replicas != 2 {
		t.Errorf("web.deploy.replicas = %d, want 2", webService.Deploy.Replicas)
	}
}

func TestComposeParser_RoundTrip(t *testing.T) {
	parser := NewComposeParser()

	originalContent := `version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    restart: always
networks:
  frontend:
    driver: bridge
`

	// 解析
	config, err := parser.Parse(originalContent)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// 渲染
	rendered, err := parser.Render(config)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// 再次解析
	config2, err := parser.Parse(rendered)
	if err != nil {
		t.Fatalf("Second Parse() error = %v", err)
	}

	// 验证版本一致
	if config2.Version != config.Version {
		t.Errorf("Version mismatch: %v != %v", config2.Version, config.Version)
	}

	// 验证服务数量一致
	if len(config2.Services) != len(config.Services) {
		t.Errorf("Services count mismatch: %d != %d", len(config2.Services), len(config.Services))
	}

	// 验证网络数量一致
	if len(config2.Networks) != len(config.Networks) {
		t.Errorf("Networks count mismatch: %d != %d", len(config2.Networks), len(config.Networks))
	}
}

func TestComposeParser_HealthCheckValidation(t *testing.T) {
	parser := NewComposeParser()

	tests := []struct {
		name      string
		config    *ComposeConfig
		wantValid bool
	}{
		{
			name: "有效的健康检查",
			config: &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image: "nginx:latest",
						HealthCheck: &HealthCheck{
							Test:     []interface{}{"CMD", "curl", "-f", "http://localhost"},
							Interval: "30s",
							Timeout:  "10s",
							Retries:  3,
						},
					},
				},
			},
			wantValid: true,
		},
		{
			name: "缺少测试命令的健康检查",
			config: &ComposeConfig{
				Version: "3.8",
				Services: map[string]*Service{
					"web": {
						Image: "nginx:latest",
						HealthCheck: &HealthCheck{
							Interval: "30s",
							Timeout:  "10s",
						},
					},
				},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Validate(tt.config)
			if result.Valid != tt.wantValid {
				t.Errorf("Validate() valid = %v, want %v", result.Valid, tt.wantValid)
				if !result.Valid {
					for _, err := range result.Errors {
						t.Logf("  Error: %s - %s", err.Field, err.Message)
					}
				}
			}
		})
	}
}

func TestComposeParser_NetworkReferences(t *testing.T) {
	parser := NewComposeParser()

	// 测试数组形式的网络引用
	configWithArrayNetworks := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image:    "nginx:latest",
				Networks: []interface{}{"frontend", "backend"},
			},
		},
		Networks: map[string]*Network{
			"frontend": {Driver: "bridge"},
			"backend":  {Driver: "bridge"},
		},
	}

	result := parser.Validate(configWithArrayNetworks)
	if !result.Valid {
		t.Errorf("Array networks validation failed: %d errors", len(result.Errors))
	}

	// 测试映射形式的网络引用
	configWithMapNetworks := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image: "nginx:latest",
				Networks: map[string]interface{}{
					"frontend": map[string]interface{}{"aliases": []string{"web"}},
				},
			},
		},
		Networks: map[string]*Network{
			"frontend": {Driver: "bridge"},
		},
	}

	result = parser.Validate(configWithMapNetworks)
	if !result.Valid {
		t.Errorf("Map networks validation failed: %d errors", len(result.Errors))
	}

	// 测试未定义的网络
	configWithUndefinedNetwork := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image:    "nginx:latest",
				Networks: []interface{}{"undefined"},
			},
		},
		Networks: map[string]*Network{},
	}

	result = parser.Validate(configWithUndefinedNetwork)
	if result.Valid {
		t.Error("Undefined network should fail validation")
	}
}

func TestComposeParser_VolumeReferences(t *testing.T) {
	parser := NewComposeParser()

	// 测试命名卷
	configWithNamedVolume := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"db": {
				Image:   "postgres:15",
				Volumes: []string{"db_data:/var/lib/postgresql/data"},
			},
		},
		Volumes: map[string]*Volume{
			"db_data": {Driver: "local"},
		},
	}

	result := parser.Validate(configWithNamedVolume)
	if !result.Valid {
		t.Errorf("Named volume validation failed: %d errors", len(result.Errors))
	}

	// 测试绑定挂载（应该通过验证）
	configWithBindMount := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image:   "nginx:latest",
				Volumes: []string{"./html:/usr/share/nginx/html"},
			},
		},
	}

	result = parser.Validate(configWithBindMount)
	if !result.Valid {
		t.Errorf("Bind mount validation failed: %d errors", len(result.Errors))
	}

	// 测试未定义的命名卷
	configWithUndefinedVolume := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"db": {
				Image:   "postgres:15",
				Volumes: []string{"undefined_volume:/data"},
			},
		},
		Volumes: map[string]*Volume{},
	}

	result = parser.Validate(configWithUndefinedVolume)
	if result.Valid {
		t.Error("Undefined volume should fail validation")
	}
}

func TestComposeParser_EmptyAndNilHandling(t *testing.T) {
	parser := NewComposeParser()

	// 测试 nil 配置
	t.Run("Nil config", func(t *testing.T) {
		_, err := parser.Render(nil)
		if err == nil {
			t.Error("Render(nil) should return error")
		}
	})

	// 测试空字符串
	t.Run("Empty string", func(t *testing.T) {
		_, err := parser.Parse("")
		if err == nil {
			t.Error("Parse(\"\") should return error")
		}
	})

	// 测试只有空白字符
	t.Run("Whitespace only", func(t *testing.T) {
		_, err := parser.Parse("   \n\t  ")
		if err == nil {
			t.Error("Parse(whitespace) should return error")
		}
	})
}

func TestComposeParser_ServiceCompletions(t *testing.T) {
	parser := NewComposeParser()

	completions := parser.getServiceCompletions()
	
	if len(completions) == 0 {
		t.Error("getServiceCompletions() returned empty list")
	}

	// 验证包含关键属性
	expectedLabels := []string{"image", "build", "ports", "volumes", "environment", "restart"}
	for _, expected := range expectedLabels {
		found := false
		for _, completion := range completions {
			if completion.Label == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected completion label %q not found", expected)
		}
	}
}

func TestComposeParser_RestartPolicyCompletions(t *testing.T) {
	parser := NewComposeParser()

	completions := parser.getRestartPolicyCompletions()
	
	if len(completions) != 4 {
		t.Errorf("getRestartPolicyCompletions() returned %d items, want 4", len(completions))
	}

	expectedPolicies := []string{"no", "always", "on-failure", "unless-stopped"}
	for _, expected := range expectedPolicies {
		found := false
		for _, completion := range completions {
			if completion.Label == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected restart policy %q not found", expected)
		}
	}
}

func TestComposeParser_YAMLFormatting(t *testing.T) {
	parser := NewComposeParser()

	config := &ComposeConfig{
		Version: "3.8",
		Services: map[string]*Service{
			"web": {
				Image: "nginx:latest",
				Ports: []string{"80:80"},
				Environment: map[string]string{
					"KEY1": "value1",
					"KEY2": "value2",
				},
			},
		},
	}

	rendered, err := parser.Render(config)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// 验证 YAML 格式
	if !strings.Contains(rendered, "version:") {
		t.Error("Rendered YAML missing 'version' field")
	}

	if !strings.Contains(rendered, "services:") {
		t.Error("Rendered YAML missing 'services' field")
	}

	if !strings.Contains(rendered, "web:") {
		t.Error("Rendered YAML missing 'web' service")
	}
}
