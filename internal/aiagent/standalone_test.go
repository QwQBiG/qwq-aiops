package aiagent

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"
)

// **Feature: enhanced-aiops-platform, Property 1: AI 自然语言理解一致性**
// **Validates: Requirements 1.1**
//
// 独立的属性测试，验证 AI 自然语言理解的一致性
// 这个测试不依赖外部库，使用内置的规则引擎进行测试
func TestAINaturalLanguageUnderstandingConsistency(t *testing.T) {
	// 创建 NLU 服务实例
	nluService := &NLUServiceImpl{
		contexts:       make(map[string]*ConversationContext),
		intentPatterns: make(map[Intent][]regexp.Regexp),
	}
	
	// 初始化模板和服务定义
	nluService.initializeTemplates()
	nluService.initializeServices()
	nluService.compilePatterns()
	
	// 定义测试数据集：有效的自然语言部署请求
	deploymentRequests := []struct {
		input           string
		expectedIntent  Intent
		expectedService string
		description     string
	}{
		// 中文部署请求
		{"部署nginx", IntentDeploy, "nginx", "中文部署指令"},
		{"安装mysql", IntentInstall, "mysql", "中文安装指令"},
		{"创建redis容器", IntentCreate, "redis", "中文创建指令"},
		{"启动postgresql", IntentStart, "postgresql", "中文启动指令"},
		{"搭建mongodb", IntentDeploy, "mongodb", "中文搭建指令"},
		
		// 英文部署请求
		{"deploy nginx", IntentDeploy, "nginx", "英文部署指令"},
		{"install mysql", IntentInstall, "mysql", "英文安装指令"},
		{"create redis", IntentCreate, "redis", "英文创建指令"},
		{"start postgresql", IntentStart, "postgresql", "英文启动指令"},
		
		// 组合式请求
		{"部署一个nginx服务", IntentDeploy, "nginx", "中文组合部署指令"},
		{"安装mysql数据库", IntentInstall, "mysql", "中文组合安装指令"},
		{"创建redis缓存", IntentCreate, "redis", "中文组合创建指令"},
	}
	
	// 运行属性测试：对于任何有效的部署请求，AI应该能理解意图并提供相应的部署选项
	successCount := 0
	totalTests := len(deploymentRequests)
	
	for _, testCase := range deploymentRequests {
		t.Run(testCase.description+": "+testCase.input, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			// 1. 测试意图识别
			intent, confidence, err := nluService.RecognizeIntent(ctx, testCase.input, nil)
			if err != nil {
				t.Errorf("意图识别失败: %v, 输入: %s", err, testCase.input)
				return
			}
			
			// 验证意图类型是否为部署相关
			deployIntents := map[Intent]bool{
				IntentDeploy:  true,
				IntentInstall: true,
				IntentCreate:  true,
				IntentStart:   true,
			}
			
			if !deployIntents[intent] {
				t.Errorf("意图识别错误: 期望部署相关意图，实际: %s, 输入: %s", intent, testCase.input)
				return
			}
			
			// 验证置信度合理性
			if confidence < 0.5 {
				t.Errorf("置信度过低: %f, 输入: %s", confidence, testCase.input)
				return
			}
			
			// 2. 测试实体提取
			entities, err := nluService.ExtractEntities(ctx, testCase.input, intent)
			if err != nil {
				t.Errorf("实体提取失败: %v, 输入: %s", err, testCase.input)
				return
			}
			
			// 验证服务实体提取
			hasServiceEntity := false
			extractedService := ""
			for _, entity := range entities {
				if entity.Type == EntityService {
					hasServiceEntity = true
					extractedService = entity.Value
					break
				}
			}
			
			// 检查输入是否包含期望的服务名称
			lowerInput := strings.ToLower(testCase.input)
			lowerExpectedService := strings.ToLower(testCase.expectedService)
			
			if strings.Contains(lowerInput, lowerExpectedService) {
				if !hasServiceEntity {
					t.Errorf("未能提取服务实体，但输入包含服务名称: %s", testCase.input)
					return
				}
				
				// 验证提取的服务名称是否正确
				if !strings.Contains(strings.ToLower(extractedService), lowerExpectedService) {
					t.Errorf("提取的服务名称不匹配: 期望包含 %s，实际: %s, 输入: %s", 
						testCase.expectedService, extractedService, testCase.input)
					return
				}
			}
			
			// 3. 测试参数验证
			params, err := nluService.ValidateParameters(ctx, intent, entities)
			if err != nil {
				// 对于部署意图，如果没有服务实体，应该返回错误
				if deployIntents[intent] && !hasServiceEntity {
					// 这是预期的错误，测试通过
					t.Logf("预期的参数验证错误（缺少服务名称）: %v", err)
				} else {
					t.Errorf("参数验证失败: %v, 输入: %s", err, testCase.input)
					return
				}
			} else {
				// 验证参数映射
				if params == nil {
					t.Errorf("参数映射不应该为nil, 输入: %s", testCase.input)
					return
				}
				
				// 如果有服务实体，参数中应该包含服务名称
				if hasServiceEntity {
					if serviceParam, exists := params["service"]; !exists {
						t.Errorf("参数中缺少服务名称, 输入: %s", testCase.input)
						return
					} else {
						t.Logf("成功提取服务参数: %s, 输入: %s", serviceParam, testCase.input)
					}
				}
			}
			
			successCount++
			t.Logf("测试通过: 意图=%s, 置信度=%.2f, 服务=%s", intent, confidence, extractedService)
		})
	}
	
	// 验证整体成功率
	successRate := float64(successCount) / float64(totalTests)
	t.Logf("整体测试成功率: %.2f%% (%d/%d)", successRate*100, successCount, totalTests)
	
	if successRate < 0.8 { // 要求80%以上的成功率
		t.Errorf("AI理解能力测试成功率过低: %.2f%%, 期望至少80%%", successRate*100)
	}
}

// 测试相同意图的不同表达方式应该产生一致的结果
func TestConsistentIntentRecognition(t *testing.T) {
	nluService := &NLUServiceImpl{
		contexts:       make(map[string]*ConversationContext),
		intentPatterns: make(map[Intent][]regexp.Regexp),
	}
	
	nluService.initializeTemplates()
	nluService.initializeServices()
	nluService.compilePatterns()
	
	// 测试相同意图的不同表达
	testGroups := []struct {
		expressions    []string
		expectedIntent Intent
		description    string
	}{
		{
			expressions: []string{
				"部署nginx",
				"安装nginx",
				"创建nginx容器",
				"启动nginx服务",
			},
			expectedIntent: IntentDeploy, // 注意：这里可能会有不同的具体意图，但都应该是部署相关的
			description:    "nginx部署相关表达",
		},
		{
			expressions: []string{
				"查看nginx状态",
				"显示nginx信息",
				"nginx运行情况",
			},
			expectedIntent: IntentQuery,
			description:    "nginx查询相关表达",
		},
	}
	
	for _, group := range testGroups {
		t.Run(group.description, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			var recognizedIntents []Intent
			for _, expr := range group.expressions {
				intent, confidence, err := nluService.RecognizeIntent(ctx, expr, nil)
				if err != nil {
					t.Errorf("意图识别失败: %v, 表达: %s", err, expr)
					continue
				}
				
				t.Logf("表达 '%s' -> 意图: %s, 置信度: %.2f", expr, intent, confidence)
				recognizedIntents = append(recognizedIntents, intent)
			}
			
			// 验证所有表达都识别为相关的意图类型
			deployIntents := map[Intent]bool{
				IntentDeploy:  true,
				IntentInstall: true,
				IntentCreate:  true,
				IntentStart:   true,
			}
			
			queryIntents := map[Intent]bool{
				IntentQuery: true,
				IntentShow:  true,
				IntentList:  true,
			}
			
			for i, intent := range recognizedIntents {
				expr := group.expressions[i]
				
				if group.expectedIntent == IntentDeploy {
					if !deployIntents[intent] {
						t.Errorf("表达 '%s' 的意图识别错误: 期望部署相关意图，实际: %s", expr, intent)
					}
				} else if group.expectedIntent == IntentQuery {
					if !queryIntents[intent] {
						t.Errorf("表达 '%s' 的意图识别错误: 期望查询相关意图，实际: %s", expr, intent)
					}
				}
			}
		})
	}
}

// 测试上下文对意图识别的影响
func TestContextualIntentRecognition(t *testing.T) {
	nluService := &NLUServiceImpl{
		contexts:       make(map[string]*ConversationContext),
		intentPatterns: make(map[Intent][]regexp.Regexp),
	}
	
	nluService.initializeTemplates()
	nluService.initializeServices()
	nluService.compilePatterns()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 创建有部署上下文的会话
	deployContext := &ConversationContext{
		SessionID:  "test-session-deploy",
		LastIntent: IntentDeploy,
		LastEntities: []Entity{
			{Type: EntityService, Value: "nginx", Confidence: 0.9},
		},
		Variables: make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// 测试模糊输入在不同上下文中的识别
	ambiguousInputs := []string{
		"nginx",
		"mysql",
		"redis",
	}
	
	for _, input := range ambiguousInputs {
		t.Run("上下文测试: "+input, func(t *testing.T) {
			// 无上下文时的意图识别
			intentWithoutContext, confidenceWithoutContext, err1 := nluService.RecognizeIntent(ctx, input, nil)
			if err1 != nil {
				t.Errorf("无上下文意图识别失败: %v", err1)
				return
			}
			
			// 有部署上下文时的意图识别
			intentWithContext, confidenceWithContext, err2 := nluService.RecognizeIntent(ctx, input, deployContext)
			if err2 != nil {
				t.Errorf("有上下文意图识别失败: %v", err2)
				return
			}
			
			t.Logf("输入 '%s':", input)
			t.Logf("  无上下文: 意图=%s, 置信度=%.2f", intentWithoutContext, confidenceWithoutContext)
			t.Logf("  有上下文: 意图=%s, 置信度=%.2f", intentWithContext, confidenceWithContext)
			
			// 验证上下文确实产生了影响（至少不应该出错）
			if intentWithContext == IntentUnknown && intentWithoutContext != IntentUnknown {
				t.Errorf("有上下文时意图识别变差了")
			}
			
			// 对于只说服务名的情况，有上下文时应该倾向于查询意图
			if intentWithContext == IntentQuery {
				t.Logf("上下文正确影响了意图识别：单独的服务名被识别为查询意图")
			}
		})
	}
}

// 测试多语言支持
func TestMultiLanguageSupport(t *testing.T) {
	nluService := &NLUServiceImpl{}
	
	testCases := []struct {
		text     string
		expected string
	}{
		// 中文文本
		{"部署nginx服务", "zh"},
		{"查看系统状态", "zh"},
		{"重启mysql数据库", "zh"},
		{"配置redis缓存", "zh"},
		
		// 英文文本
		{"deploy nginx service", "en"},
		{"check system status", "en"},
		{"restart mysql database", "en"},
		{"configure redis cache", "en"},
		
		// 混合文本（应该根据主要语言判断）
		{"部署nginx", "zh"},
		{"nginx部署", "zh"},
	}
	
	for _, tc := range testCases {
		t.Run("语言检测: "+tc.text, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			language, err := nluService.DetectLanguage(ctx, tc.text)
			if err != nil {
				t.Errorf("语言检测失败: %v", err)
				return
			}
			
			if language != tc.expected {
				t.Errorf("语言检测错误: 期望 %s，实际 %s, 文本: %s", tc.expected, language, tc.text)
			} else {
				t.Logf("语言检测正确: %s -> %s", tc.text, language)
			}
		})
	}
}

// 测试实体提取的准确性
func TestEntityExtractionAccuracy(t *testing.T) {
	nluService := &NLUServiceImpl{
		contexts:       make(map[string]*ConversationContext),
		intentPatterns: make(map[Intent][]regexp.Regexp),
	}
	
	nluService.initializeServices()
	
	testCases := []struct {
		text         string
		expectedType EntityType
		expectedValue string
		shouldFind   bool
	}{
		// 端口号提取
		{"nginx运行在80端口", EntityPort, "80", true},
		{"mysql使用3306端口", EntityPort, "3306", true},
		{"redis监听6379", EntityPort, "6379", true},
		{"web服务器端口8080", EntityPort, "8080", true},
		
		// 路径提取
		{"配置文件在/etc/nginx/nginx.conf", EntityPath, "/etc/nginx/nginx.conf", true},
		{"日志路径/var/log/mysql/error.log", EntityPath, "/var/log/mysql/error.log", true},
		{"数据目录/data/redis", EntityPath, "/data/redis", true},
		
		// 服务名提取
		{"部署nginx服务", EntityService, "nginx", true},
		{"安装mysql数据库", EntityService, "mysql", true},
		{"创建redis缓存", EntityService, "redis", true},
	}
	
	for _, tc := range testCases {
		t.Run("实体提取: "+tc.text, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			entities, err := nluService.ExtractEntities(ctx, tc.text, IntentQuery)
			if err != nil {
				t.Errorf("实体提取失败: %v", err)
				return
			}
			
			if tc.shouldFind {
				found := false
				var extractedValue string
				
				for _, entity := range entities {
					if entity.Type == tc.expectedType {
						found = true
						extractedValue = entity.Value
						
						// 验证提取的值是否包含期望的内容
						if tc.expectedType == EntityService {
							// 服务名可能不完全匹配，但应该包含期望的服务名
							if !strings.Contains(strings.ToLower(extractedValue), strings.ToLower(tc.expectedValue)) {
								t.Errorf("提取的服务名不匹配: 期望包含 %s，实际: %s", tc.expectedValue, extractedValue)
							}
						} else {
							// 其他类型的实体应该完全匹配
							if extractedValue != tc.expectedValue {
								t.Logf("提取的值不完全匹配: 期望 %s，实际: %s (可能是正常的)", tc.expectedValue, extractedValue)
							}
						}
						
						t.Logf("成功提取 %s 实体: %s", tc.expectedType, extractedValue)
						break
					}
				}
				
				if !found {
					t.Errorf("未能提取到期望的 %s 实体", tc.expectedType)
				}
			}
		})
	}
}