package appstore

import (
	"testing"
)

// TestParseDockerComposeTemplate 测试 Docker Compose 模板解析
func TestParseDockerComposeTemplate(t *testing.T) {
	service := NewTemplateService()

	validContent := `version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`

	parsed, err := service.ParseTemplate(TemplateTypeDockerCompose, validContent)
	if err != nil {
		t.Fatalf("解析有效的 Docker Compose 模板失败: %v", err)
	}

	if parsed == nil {
		t.Fatal("解析结果为 nil")
	}

	// 检查是否包含 services 字段
	if _, ok := parsed["services"]; !ok {
		t.Error("解析结果缺少 services 字段")
	}
}

// TestParseInvalidTemplate 测试解析无效模板
func TestParseInvalidTemplate(t *testing.T) {
	service := NewTemplateService()

	invalidContent := `invalid yaml content: [[[`

	_, err := service.ParseTemplate(TemplateTypeDockerCompose, invalidContent)
	if err == nil {
		t.Error("应该返回错误，但没有")
	}
}

// TestValidateDockerComposeTemplate 测试 Docker Compose 模板验证
func TestValidateDockerComposeTemplate(t *testing.T) {
	service := NewTemplateService()

	tests := []struct {
		name        string
		template    *AppTemplate
		expectError bool
	}{
		{
			name: "有效的模板",
			template: &AppTemplate{
				Name:        "test",
				DisplayName: "Test",
				Type:        TemplateTypeDockerCompose,
				Content: `version: '3.8'
services:
  web:
    image: nginx:latest
`,
			},
			expectError: false,
		},
		{
			name: "缺少 services 字段",
			template: &AppTemplate{
				Name:        "test",
				DisplayName: "Test",
				Type:        TemplateTypeDockerCompose,
				Content: `version: '3.8'
networks:
  default:
`,
			},
			expectError: true,
		},
		{
			name: "服务缺少 image 和 build",
			template: &AppTemplate{
				Name:        "test",
				DisplayName: "Test",
				Type:        TemplateTypeDockerCompose,
				Content: `version: '3.8'
services:
  web:
    ports:
      - "80:80"
`,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateTemplate(tt.template)
			if tt.expectError && err == nil {
				t.Error("期望返回错误，但没有")
			}
			if !tt.expectError && err != nil {
				t.Errorf("不期望返回错误，但得到: %v", err)
			}
		})
	}
}

// TestRenderTemplate 测试模板渲染
func TestRenderTemplate(t *testing.T) {
	service := NewTemplateService()

	template := &AppTemplate{
		Name:        "test",
		DisplayName: "Test",
		Type:        TemplateTypeDockerCompose,
		Content: `version: '3.8'
services:
  web:
    image: nginx:{{.version}}
    ports:
      - "{{.port}}:80"
`,
		Parameters: `[
			{
				"name": "version",
				"type": "string",
				"default_value": "latest",
				"required": true
			},
			{
				"name": "port",
				"type": "int",
				"default_value": 80,
				"required": true
			}
		]`,
	}

	params := map[string]interface{}{
		"version": "1.21",
		"port":    8080,
	}

	rendered, err := service.RenderTemplate(template, params)
	if err != nil {
		t.Fatalf("渲染模板失败: %v", err)
	}

	// 检查渲染结果
	if rendered == "" {
		t.Error("渲染结果为空")
	}

	// 检查参数是否被正确替换
	if !contains(rendered, "nginx:1.21") {
		t.Error("版本参数未被正确替换")
	}

	if !contains(rendered, "8080:80") {
		t.Error("端口参数未被正确替换")
	}
}

// TestValidateParameters 测试参数验证
func TestValidateParameters(t *testing.T) {
	service := NewTemplateService()

	paramDefs := []TemplateParameter{
		{
			Name:     "required_field",
			Type:     ParamTypeString,
			Required: true,
		},
		{
			Name:         "optional_field",
			Type:         ParamTypeString,
			Required:     false,
			DefaultValue: "default",
		},
		{
			Name:       "email",
			Type:       ParamTypeString,
			Required:   true,
			Validation: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		},
		{
			Name:     "env",
			Type:     ParamTypeSelect,
			Required: true,
			Options:  []string{"dev", "staging", "prod"},
		},
	}

	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
	}{
		{
			name: "所有参数有效",
			params: map[string]interface{}{
				"required_field": "value",
				"email":          "test@example.com",
				"env":            "prod",
			},
			expectError: false,
		},
		{
			name: "缺少必填参数",
			params: map[string]interface{}{
				"email": "test@example.com",
				"env":   "prod",
			},
			expectError: true,
		},
		{
			name: "邮箱格式无效",
			params: map[string]interface{}{
				"required_field": "value",
				"email":          "invalid-email",
				"env":            "prod",
			},
			expectError: true,
		},
		{
			name: "选项值无效",
			params: map[string]interface{}{
				"required_field": "value",
				"email":          "test@example.com",
				"env":            "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateParameters(paramDefs, tt.params)
			if tt.expectError && err == nil {
				t.Error("期望返回错误，但没有")
			}
			if !tt.expectError && err != nil {
				t.Errorf("不期望返回错误，但得到: %v", err)
			}
		})
	}
}

// TestExtractParameters 测试参数提取
func TestExtractParameters(t *testing.T) {
	service := NewTemplateService()

	content := `version: '3.8'
services:
  web:
    image: {{.image}}
    ports:
      - "{{.port}}:80"
    environment:
      APP_ENV: {{.env}}
      DB_HOST: {{.db_host}}
`

	params := service.ExtractParameters(content)

	expectedParams := map[string]bool{
		"image":   true,
		"port":    true,
		"env":     true,
		"db_host": true,
	}

	if len(params) != len(expectedParams) {
		t.Errorf("期望提取 %d 个参数，实际提取 %d 个", len(expectedParams), len(params))
	}

	for _, param := range params {
		if !expectedParams[param] {
			t.Errorf("提取了意外的参数: %s", param)
		}
	}
}

// TestMergeDefaultValues 测试默认值合并
func TestMergeDefaultValues(t *testing.T) {
	service := NewTemplateService()

	paramDefs := []TemplateParameter{
		{
			Name:         "field1",
			DefaultValue: "default1",
		},
		{
			Name:         "field2",
			DefaultValue: "default2",
		},
		{
			Name:         "field3",
			DefaultValue: "default3",
		},
	}

	userParams := map[string]interface{}{
		"field2": "user2",
		"field4": "user4",
	}

	merged := service.mergeDefaultValues(paramDefs, userParams)

	// 检查默认值
	if merged["field1"] != "default1" {
		t.Error("field1 应该使用默认值")
	}

	// 检查用户值覆盖默认值
	if merged["field2"] != "user2" {
		t.Error("field2 应该使用用户提供的值")
	}

	// 检查未定义默认值的字段
	if merged["field3"] != "default3" {
		t.Error("field3 应该使用默认值")
	}

	// 检查用户额外提供的值
	if merged["field4"] != "user4" {
		t.Error("field4 应该保留用户提供的值")
	}
}

// TestParameterTypeValidation 测试参数类型验证
func TestParameterTypeValidation(t *testing.T) {
	service := NewTemplateService()

	tests := []struct {
		name        string
		paramDef    TemplateParameter
		value       interface{}
		expectError bool
	}{
		{
			name:        "字符串类型 - 有效",
			paramDef:    TemplateParameter{Name: "test", Type: ParamTypeString},
			value:       "hello",
			expectError: false,
		},
		{
			name:        "字符串类型 - 无效",
			paramDef:    TemplateParameter{Name: "test", Type: ParamTypeString},
			value:       123,
			expectError: true,
		},
		{
			name:        "整数类型 - 有效",
			paramDef:    TemplateParameter{Name: "test", Type: ParamTypeInt},
			value:       123,
			expectError: false,
		},
		{
			name:        "整数类型 - 无效",
			paramDef:    TemplateParameter{Name: "test", Type: ParamTypeInt},
			value:       "not a number",
			expectError: true,
		},
		{
			name:        "布尔类型 - 有效",
			paramDef:    TemplateParameter{Name: "test", Type: ParamTypeBool},
			value:       true,
			expectError: false,
		},
		{
			name:        "布尔类型 - 无效",
			paramDef:    TemplateParameter{Name: "test", Type: ParamTypeBool},
			value:       "true",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateParameterType(tt.paramDef, tt.value)
			if tt.expectError && err == nil {
				t.Error("期望返回错误，但没有")
			}
			if !tt.expectError && err != nil {
				t.Errorf("不期望返回错误，但得到: %v", err)
			}
		})
	}
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
