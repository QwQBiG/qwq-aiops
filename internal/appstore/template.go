package appstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	// ErrInvalidTemplateType 无效的模板类型
	ErrInvalidTemplateType = errors.New("invalid template type")
	// ErrInvalidTemplateContent 无效的模板内容
	ErrInvalidTemplateContent = errors.New("invalid template content")
	// ErrMissingRequiredParameter 缺少必填参数
	ErrMissingRequiredParameter = errors.New("missing required parameter")
	// ErrInvalidParameterValue 无效的参数值
	ErrInvalidParameterValue = errors.New("invalid parameter value")
	// ErrParameterValidationFailed 参数验证失败
	ErrParameterValidationFailed = errors.New("parameter validation failed")
)

// TemplateService 模板服务
type TemplateService struct{}

// NewTemplateService 创建模板服务实例
func NewTemplateService() *TemplateService {
	return &TemplateService{}
}

// ParseTemplate 解析模板内容
// 根据模板类型解析 YAML 或 JSON 格式的模板内容
func (s *TemplateService) ParseTemplate(templateType TemplateType, content string) (map[string]interface{}, error) {
	if content == "" {
		return nil, ErrInvalidTemplateContent
	}

	var result map[string]interface{}

	switch templateType {
	case TemplateTypeDockerCompose:
		// 解析 Docker Compose YAML
		if err := yaml.Unmarshal([]byte(content), &result); err != nil {
			return nil, fmt.Errorf("failed to parse docker-compose template: %w", err)
		}
	case TemplateTypeHelmChart:
		// 解析 Helm Chart YAML
		if err := yaml.Unmarshal([]byte(content), &result); err != nil {
			return nil, fmt.Errorf("failed to parse helm chart template: %w", err)
		}
	default:
		return nil, ErrInvalidTemplateType
	}

	return result, nil
}

// ValidateTemplate 验证模板内容
// 检查模板格式是否正确，是否包含必要的字段
func (s *TemplateService) ValidateTemplate(template *AppTemplate) error {
	if template == nil {
		return errors.New("template is nil")
	}

	// 验证基本字段
	if template.Name == "" {
		return errors.New("template name is required")
	}
	if template.DisplayName == "" {
		return errors.New("template display name is required")
	}
	if template.Content == "" {
		return ErrInvalidTemplateContent
	}

	// 解析模板内容
	parsed, err := s.ParseTemplate(template.Type, template.Content)
	if err != nil {
		return err
	}

	// 根据模板类型进行特定验证
	switch template.Type {
	case TemplateTypeDockerCompose:
		return s.validateDockerComposeTemplate(parsed)
	case TemplateTypeHelmChart:
		return s.validateHelmChartTemplate(parsed)
	default:
		return ErrInvalidTemplateType
	}
}

// validateDockerComposeTemplate 验证 Docker Compose 模板
func (s *TemplateService) validateDockerComposeTemplate(parsed map[string]interface{}) error {
	// 检查是否包含 services 字段
	services, ok := parsed["services"]
	if !ok {
		return errors.New("docker-compose template must contain 'services' field")
	}

	// 检查 services 是否为 map
	servicesMap, ok := services.(map[string]interface{})
	if !ok {
		return errors.New("'services' field must be a map")
	}

	// 检查是否至少有一个服务
	if len(servicesMap) == 0 {
		return errors.New("docker-compose template must contain at least one service")
	}

	// 验证每个服务的基本结构
	for serviceName, serviceConfig := range servicesMap {
		serviceMap, ok := serviceConfig.(map[string]interface{})
		if !ok {
			return fmt.Errorf("service '%s' configuration must be a map", serviceName)
		}

		// 检查是否有 image 或 build 字段
		if _, hasImage := serviceMap["image"]; !hasImage {
			if _, hasBuild := serviceMap["build"]; !hasBuild {
				return fmt.Errorf("service '%s' must have either 'image' or 'build' field", serviceName)
			}
		}
	}

	return nil
}

// validateHelmChartTemplate 验证 Helm Chart 模板
func (s *TemplateService) validateHelmChartTemplate(parsed map[string]interface{}) error {
	// Helm Chart 的基本验证
	// 检查是否包含必要的字段（根据实际需求调整）
	if _, ok := parsed["apiVersion"]; !ok {
		return errors.New("helm chart template should contain 'apiVersion' field")
	}

	return nil
}

// RenderTemplate 渲染模板
// 使用提供的参数值替换模板中的占位符
func (s *TemplateService) RenderTemplate(template *AppTemplate, params map[string]interface{}) (string, error) {
	if template == nil {
		return "", errors.New("template is nil")
	}

	// 解析参数定义
	var paramDefs []TemplateParameter
	if template.Parameters != "" {
		if err := json.Unmarshal([]byte(template.Parameters), &paramDefs); err != nil {
			return "", fmt.Errorf("failed to parse parameter definitions: %w", err)
		}
	}

	// 验证参数
	if err := s.ValidateParameters(paramDefs, params); err != nil {
		return "", err
	}

	// 合并默认值
	mergedParams := s.mergeDefaultValues(paramDefs, params)

	// 渲染模板内容
	rendered := template.Content
	for key, value := range mergedParams {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		rendered = strings.ReplaceAll(rendered, placeholder, fmt.Sprintf("%v", value))
	}

	return rendered, nil
}

// ValidateParameters 验证参数
func (s *TemplateService) ValidateParameters(paramDefs []TemplateParameter, params map[string]interface{}) error {
	for _, paramDef := range paramDefs {
		value, exists := params[paramDef.Name]

		// 检查必填参数
		if paramDef.Required && !exists {
			return fmt.Errorf("%w: %s", ErrMissingRequiredParameter, paramDef.Name)
		}

		if !exists {
			continue
		}

		// 类型验证
		if err := s.validateParameterType(paramDef, value); err != nil {
			return err
		}

		// 正则验证
		if paramDef.Validation != "" {
			if err := s.validateParameterRegex(paramDef, value); err != nil {
				return err
			}
		}

		// 选项验证（用于 select 类型）
		if paramDef.Type == ParamTypeSelect && len(paramDef.Options) > 0 {
			if err := s.validateParameterOptions(paramDef, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateParameterType 验证参数类型
func (s *TemplateService) validateParameterType(paramDef TemplateParameter, value interface{}) error {
	switch paramDef.Type {
	case ParamTypeString, ParamTypePassword, ParamTypePath:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%w: parameter '%s' must be a string", ErrInvalidParameterValue, paramDef.Name)
		}
	case ParamTypeInt:
		switch value.(type) {
		case int, int32, int64, float64:
			// 允许数字类型
		default:
			return fmt.Errorf("%w: parameter '%s' must be an integer", ErrInvalidParameterValue, paramDef.Name)
		}
	case ParamTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%w: parameter '%s' must be a boolean", ErrInvalidParameterValue, paramDef.Name)
		}
	case ParamTypeSelect:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%w: parameter '%s' must be a string", ErrInvalidParameterValue, paramDef.Name)
		}
	}

	return nil
}

// validateParameterRegex 使用正则表达式验证参数
func (s *TemplateService) validateParameterRegex(paramDef TemplateParameter, value interface{}) error {
	strValue, ok := value.(string)
	if !ok {
		return nil // 非字符串类型跳过正则验证
	}

	matched, err := regexp.MatchString(paramDef.Validation, strValue)
	if err != nil {
		return fmt.Errorf("invalid validation regex for parameter '%s': %w", paramDef.Name, err)
	}

	if !matched {
		return fmt.Errorf("%w: parameter '%s' does not match validation pattern", ErrParameterValidationFailed, paramDef.Name)
	}

	return nil
}

// validateParameterOptions 验证参数选项
func (s *TemplateService) validateParameterOptions(paramDef TemplateParameter, value interface{}) error {
	strValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("%w: parameter '%s' must be a string for select type", ErrInvalidParameterValue, paramDef.Name)
	}

	for _, option := range paramDef.Options {
		if option == strValue {
			return nil
		}
	}

	return fmt.Errorf("%w: parameter '%s' value '%s' is not in allowed options", ErrInvalidParameterValue, paramDef.Name, strValue)
}

// mergeDefaultValues 合并默认值
func (s *TemplateService) mergeDefaultValues(paramDefs []TemplateParameter, params map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 先添加所有默认值
	for _, paramDef := range paramDefs {
		if paramDef.DefaultValue != nil {
			result[paramDef.Name] = paramDef.DefaultValue
		}
	}

	// 用户提供的值覆盖默认值
	for key, value := range params {
		result[key] = value
	}

	return result
}

// ExtractParameters 从模板内容中提取参数占位符
// 返回模板中使用的所有参数名称
func (s *TemplateService) ExtractParameters(content string) []string {
	// 匹配 {{.ParameterName}} 格式的占位符
	re := regexp.MustCompile(`\{\{\.(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)

	paramSet := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			paramSet[match[1]] = true
		}
	}

	// 转换为切片
	params := make([]string, 0, len(paramSet))
	for param := range paramSet {
		params = append(params, param)
	}

	return params
}

// ConvertToYAML 将 map 转换为 YAML 字符串
func (s *TemplateService) ConvertToYAML(data map[string]interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to convert to YAML: %w", err)
	}
	return string(yamlBytes), nil
}

// ConvertToJSON 将 map 转换为 JSON 字符串
func (s *TemplateService) ConvertToJSON(data map[string]interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to convert to JSON: %w", err)
	}
	return string(jsonBytes), nil
}
