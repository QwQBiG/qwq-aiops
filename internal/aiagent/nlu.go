package aiagent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// NLUServiceImpl 自然语言理解服务实现
type NLUServiceImpl struct {
	templates      []CommandTemplate
	services       []ServiceDefinition
	contexts       map[string]*ConversationContext
	contextMutex   sync.RWMutex
	intentPatterns map[Intent][]regexp.Regexp
	persistPath    string
}

// NewNLUService 创建新的NLU服务
func NewNLUService() *NLUServiceImpl {
	service := &NLUServiceImpl{
		contexts:       make(map[string]*ConversationContext),
		intentPatterns: make(map[Intent][]regexp.Regexp),
	}
	
	// 初始化命令模板和服务定义
	service.initializeTemplates()
	service.initializeServices()
	service.compilePatterns()
	
	return service
}

// Understand 理解用户输入的主要方法
func (s *NLUServiceImpl) Understand(ctx context.Context, req *NLURequest) (*NLUResponse, error) {
	// 1. 语言检测
	language := req.Language
	if language == "" {
		detectedLang, err := s.DetectLanguage(ctx, req.Text)
		if err == nil {
			language = detectedLang
		} else {
			language = "zh" // 默认中文
		}
	}
	
	// 2. 获取或创建对话上下文
	context := req.Context
	if context == nil && req.SessionID != "" {
		var err error
		context, err = s.GetContext(ctx, req.SessionID)
		if err != nil {
			// 创建新的上下文
			context = &ConversationContext{
				SessionID: req.SessionID,
				UserID:    req.UserID,
				Variables: make(map[string]interface{}),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
		}
	}
	
	// 3. 意图识别
	intent, confidence, err := s.RecognizeIntent(ctx, req.Text, context)
	if err != nil {
		return &NLUResponse{
			Intent:     IntentUnknown,
			Confidence: 0.0,
			Error:      fmt.Sprintf("意图识别失败: %v", err),
		}, err
	}
	
	// 4. 实体提取
	entities, err := s.ExtractEntities(ctx, req.Text, intent)
	if err != nil {
		return &NLUResponse{
			Intent:     intent,
			Confidence: confidence,
			Error:      fmt.Sprintf("实体提取失败: %v", err),
		}, err
	}
	
	// 5. 参数验证
	parameters, err := s.ValidateParameters(ctx, intent, entities)
	if err != nil {
		return &NLUResponse{
			Intent:     intent,
			Entities:   entities,
			Confidence: confidence,
			Error:      fmt.Sprintf("参数验证失败: %v", err),
		}, err
	}
	
	// 6. 生成建议
	suggestions := s.generateSuggestions(intent, entities, parameters)
	
	// 7. 更新对话上下文
	if context != nil && req.SessionID != "" {
		turn := &ConversationTurn{
			UserInput: req.Text,
			Intent:    intent,
			Entities:  entities,
			Timestamp: time.Now(),
		}
		s.UpdateContext(ctx, req.SessionID, turn)
	}
	
	return &NLUResponse{
		Intent:      intent,
		Entities:    entities,
		Confidence:  confidence,
		Parameters:  parameters,
		Suggestions: suggestions,
	}, nil
}

// RecognizeIntent 识别用户意图
func (s *NLUServiceImpl) RecognizeIntent(ctx context.Context, text string, context *ConversationContext) (Intent, float64, error) {
	text = strings.ToLower(strings.TrimSpace(text))
	
	// 1. 基于规则的快速匹配
	if intent, confidence := s.matchIntentByRules(text); intent != IntentUnknown {
		return intent, confidence, nil
	}
	
	// 2. 基于上下文的意图推断
	if context != nil && context.LastIntent != IntentUnknown {
		if intent, confidence := s.inferIntentFromContext(text, context); intent != IntentUnknown {
			return intent, confidence, nil
		}
	}
	
	// 3. 使用AI进行意图识别
	return s.recognizeIntentWithAI(ctx, text, context)
}

// matchIntentByRules 基于规则匹配意图
func (s *NLUServiceImpl) matchIntentByRules(text string) (Intent, float64) {
	// 部署相关关键词
	deployKeywords := []string{"部署", "安装", "创建", "搭建", "启动", "deploy", "install", "create", "setup", "start"}
	for _, keyword := range deployKeywords {
		if strings.Contains(text, keyword) {
			return IntentDeploy, 0.8
		}
	}
	
	// 查询相关关键词
	queryKeywords := []string{"查看", "看看", "显示", "状态", "列表", "show", "list", "status", "check", "view"}
	for _, keyword := range queryKeywords {
		if strings.Contains(text, keyword) {
			return IntentQuery, 0.8
		}
	}
	
	// 管理相关关键词
	manageKeywords := []string{"重启", "停止", "更新", "删除", "restart", "stop", "update", "delete", "remove"}
	for _, keyword := range manageKeywords {
		if strings.Contains(text, keyword) {
			if strings.Contains(text, "重启") || strings.Contains(text, "restart") {
				return IntentRestart, 0.8
			}
			if strings.Contains(text, "停止") || strings.Contains(text, "stop") {
				return IntentStop, 0.8
			}
			if strings.Contains(text, "更新") || strings.Contains(text, "update") {
				return IntentUpdate, 0.8
			}
			if strings.Contains(text, "删除") || strings.Contains(text, "delete") || strings.Contains(text, "remove") {
				return IntentDelete, 0.8
			}
		}
	}
	
	// 诊断相关关键词
	diagnoseKeywords := []string{"诊断", "分析", "问题", "故障", "错误", "diagnose", "analyze", "problem", "issue", "error"}
	for _, keyword := range diagnoseKeywords {
		if strings.Contains(text, keyword) {
			return IntentDiagnose, 0.8
		}
	}
	
	// 配置相关关键词
	configKeywords := []string{"配置", "设置", "生成", "configure", "config", "generate", "setup"}
	for _, keyword := range configKeywords {
		if strings.Contains(text, keyword) {
			if strings.Contains(text, "生成") || strings.Contains(text, "generate") {
				return IntentGenerate, 0.8
			}
			return IntentConfigure, 0.8
		}
	}
	
	return IntentUnknown, 0.0
}

// inferIntentFromContext 从上下文推断意图
func (s *NLUServiceImpl) inferIntentFromContext(text string, context *ConversationContext) (Intent, float64) {
	// 如果用户只说了一个服务名，根据上下文推断意图
	if s.isServiceName(text) {
		if context.LastIntent == IntentDeploy {
			return IntentDeploy, 0.7
		}
		// 默认查询状态
		return IntentQuery, 0.7
	}
	
	// 简单的确认词
	confirmWords := []string{"是", "好", "确定", "yes", "ok", "sure"}
	for _, word := range confirmWords {
		if strings.TrimSpace(text) == word {
			return context.LastIntent, 0.6
		}
	}
	
	return IntentUnknown, 0.0
}

// recognizeIntentWithAI 使用AI识别意图（简化版本，不依赖外部AI服务）
func (s *NLUServiceImpl) recognizeIntentWithAI(ctx context.Context, text string, context *ConversationContext) (Intent, float64, error) {
	// 简化的意图识别逻辑，基于关键词匹配
	text = strings.ToLower(text)
	
	// 使用更复杂的规则进行意图识别
	if strings.Contains(text, "部署") || strings.Contains(text, "deploy") {
		return IntentDeploy, 0.7, nil
	}
	if strings.Contains(text, "安装") || strings.Contains(text, "install") {
		return IntentInstall, 0.7, nil
	}
	if strings.Contains(text, "创建") || strings.Contains(text, "create") {
		return IntentCreate, 0.7, nil
	}
	if strings.Contains(text, "启动") || strings.Contains(text, "start") {
		return IntentStart, 0.7, nil
	}
	if strings.Contains(text, "查看") || strings.Contains(text, "show") || strings.Contains(text, "状态") || strings.Contains(text, "status") {
		return IntentQuery, 0.7, nil
	}
	if strings.Contains(text, "重启") || strings.Contains(text, "restart") {
		return IntentRestart, 0.7, nil
	}
	if strings.Contains(text, "停止") || strings.Contains(text, "stop") {
		return IntentStop, 0.7, nil
	}
	if strings.Contains(text, "配置") || strings.Contains(text, "configure") {
		return IntentConfigure, 0.7, nil
	}
	if strings.Contains(text, "生成") || strings.Contains(text, "generate") {
		return IntentGenerate, 0.7, nil
	}
	
	return IntentUnknown, 0.3, nil
}

// ExtractEntities 提取实体
func (s *NLUServiceImpl) ExtractEntities(ctx context.Context, text string, intent Intent) ([]Entity, error) {
	var entities []Entity
	
	// 1. 基于规则的实体提取
	ruleEntities := s.extractEntitiesByRules(text)
	entities = append(entities, ruleEntities...)
	
	// 2. 基于服务定义的实体提取
	serviceEntities := s.extractServiceEntities(text)
	entities = append(entities, serviceEntities...)
	
	// 3. 使用AI进行实体提取（对于复杂情况）
	if len(entities) == 0 || intent == IntentUnknown {
		aiEntities, err := s.extractEntitiesWithAI(ctx, text, intent)
		if err == nil {
			entities = append(entities, aiEntities...)
		}
	}
	
	return entities, nil
}

// extractEntitiesByRules 基于规则提取实体
func (s *NLUServiceImpl) extractEntitiesByRules(text string) []Entity {
	var entities []Entity
	
	// 端口号提取
	portRegex := regexp.MustCompile(`\b(\d{1,5})\b`)
	portMatches := portRegex.FindAllStringSubmatch(text, -1)
	for _, match := range portMatches {
		if len(match) > 1 {
			port := match[1]
			// 验证端口范围
			if len(port) <= 5 {
				entities = append(entities, Entity{
					Type:       EntityPort,
					Value:      port,
					Confidence: 0.8,
				})
			}
		}
	}
	
	// 路径提取
	pathRegex := regexp.MustCompile(`(/[a-zA-Z0-9_\-./]+)`)
	pathMatches := pathRegex.FindAllStringSubmatch(text, -1)
	for _, match := range pathMatches {
		if len(match) > 1 {
			entities = append(entities, Entity{
				Type:       EntityPath,
				Value:      match[1],
				Confidence: 0.7,
			})
		}
	}
	
	// 域名提取
	domainRegex := regexp.MustCompile(`\b([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}\b`)
	domainMatches := domainRegex.FindAllStringSubmatch(text, -1)
	for _, match := range domainMatches {
		if len(match) > 0 {
			entities = append(entities, Entity{
				Type:       EntityDomain,
				Value:      match[0],
				Confidence: 0.9,
			})
		}
	}
	
	return entities
}

// extractServiceEntities 提取服务相关实体
func (s *NLUServiceImpl) extractServiceEntities(text string) []Entity {
	var entities []Entity
	text = strings.ToLower(text)
	
	for _, service := range s.services {
		// 检查服务名称
		if strings.Contains(text, strings.ToLower(service.Name)) {
			entities = append(entities, Entity{
				Type:       EntityService,
				Value:      service.Name,
				Confidence: 0.9,
			})
		}
		
		// 检查别名
		for _, alias := range service.Aliases {
			if strings.Contains(text, strings.ToLower(alias)) {
				entities = append(entities, Entity{
					Type:       EntityService,
					Value:      service.Name, // 使用标准名称
					Confidence: 0.8,
				})
				break
			}
		}
	}
	
	return entities
}

// ValidateParameters 验证参数
func (s *NLUServiceImpl) ValidateParameters(ctx context.Context, intent Intent, entities []Entity) (map[string]string, error) {
	parameters := make(map[string]string)
	
	// 根据意图和实体构建参数
	for _, entity := range entities {
		switch entity.Type {
		case EntityService:
			parameters["service"] = entity.Value
		case EntityPort:
			parameters["port"] = entity.Value
		case EntityPath:
			parameters["path"] = entity.Value
		case EntityDomain:
			parameters["domain"] = entity.Value
		case EntityDatabase:
			parameters["database"] = entity.Value
		case EntityUser:
			parameters["user"] = entity.Value
		}
	}
	
	// 根据意图验证必需参数
	switch intent {
	case IntentDeploy, IntentInstall:
		if _, exists := parameters["service"]; !exists {
			return parameters, fmt.Errorf("部署操作需要指定服务名称")
		}
	case IntentQuery, IntentStart, IntentStop, IntentRestart:
		// 这些操作可以没有特定参数，会查询所有或提示用户选择
	case IntentDelete:
		if _, exists := parameters["service"]; !exists {
			return parameters, fmt.Errorf("删除操作需要指定服务名称")
		}
	}
	
	return parameters, nil
}

// 其他辅助方法...

// isServiceName 检查是否为服务名称
func (s *NLUServiceImpl) isServiceName(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	for _, service := range s.services {
		if strings.ToLower(service.Name) == text {
			return true
		}
		for _, alias := range service.Aliases {
			if strings.ToLower(alias) == text {
				return true
			}
		}
	}
	return false
}

// generateSuggestions 生成操作建议
func (s *NLUServiceImpl) generateSuggestions(intent Intent, entities []Entity, parameters map[string]string) []string {
	var suggestions []string
	
	switch intent {
	case IntentDeploy:
		if service, exists := parameters["service"]; exists {
			suggestions = append(suggestions, fmt.Sprintf("部署 %s 服务", service))
			suggestions = append(suggestions, fmt.Sprintf("配置 %s 的端口和存储", service))
		}
	case IntentQuery:
		suggestions = append(suggestions, "查看系统状态")
		suggestions = append(suggestions, "查看容器列表")
		suggestions = append(suggestions, "查看服务日志")
	case IntentDiagnose:
		suggestions = append(suggestions, "检查系统资源使用情况")
		suggestions = append(suggestions, "分析服务性能")
		suggestions = append(suggestions, "查看错误日志")
	}
	
	return suggestions
}

// buildIntentRecognitionPrompt 构建意图识别提示词
func (s *NLUServiceImpl) buildIntentRecognitionPrompt(text string, context *ConversationContext) string {
	prompt := fmt.Sprintf(`请分析以下用户输入的意图：

用户输入: "%s"

可能的意图类型:
- deploy: 部署、安装、创建服务
- query: 查询、查看状态
- start/stop/restart: 启动、停止、重启服务
- update: 更新、修改配置
- delete: 删除、移除服务
- diagnose: 诊断、分析问题
- configure: 配置、设置
- generate: 生成配置文件
- unknown: 未知意图

请返回JSON格式: {"intent": "意图类型", "confidence": 置信度(0-1)}`, text)

	if context != nil && len(context.History) > 0 {
		prompt += fmt.Sprintf("\n\n上下文信息:\n上一次意图: %s", context.LastIntent)
	}

	return prompt
}

// parseIntentFromText 从文本中解析意图
func (s *NLUServiceImpl) parseIntentFromText(text string) Intent {
	text = strings.ToLower(text)
	
	intentMap := map[string]Intent{
		"deploy":    IntentDeploy,
		"query":     IntentQuery,
		"start":     IntentStart,
		"stop":      IntentStop,
		"restart":   IntentRestart,
		"update":    IntentUpdate,
		"delete":    IntentDelete,
		"diagnose":  IntentDiagnose,
		"configure": IntentConfigure,
		"generate":  IntentGenerate,
	}
	
	for keyword, intent := range intentMap {
		if strings.Contains(text, keyword) {
			return intent
		}
	}
	
	return IntentUnknown
}

// extractEntitiesWithAI 使用AI提取实体
func (s *NLUServiceImpl) extractEntitiesWithAI(ctx context.Context, text string, intent Intent) ([]Entity, error) {
	// 这里可以实现更复杂的AI实体提取逻辑
	// 暂时返回空列表
	return []Entity{}, nil
}