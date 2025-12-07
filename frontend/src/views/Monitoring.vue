<template>
  <div class="monitoring-container">
    <el-tabs v-model="activeTab">
      <!-- 监控指标标签页 -->
      <el-tab-pane label="监控指标" name="metrics">
        <!-- 系统指标卡片展示 -->
        <el-row :gutter="20">
          <el-col :span="6" v-for="metric in systemMetrics" :key="metric.name">
            <el-card class="metric-card">
              <div class="metric-icon">
                <el-icon :size="32" :color="metric.color">
                  <component :is="metric.icon" />
                </el-icon>
              </div>
              <div class="metric-info">
                <div class="metric-name">{{ metric.label }}</div>
                <div class="metric-value">{{ metric.value }}</div>
              </div>
            </el-card>
          </el-col>
        </el-row>

        <!-- 监控图表展示区域 -->
        <el-row :gutter="20" class="charts-row">
          <el-col :span="12">
            <el-card>
              <template #header>CPU使用率</template>
              <div ref="cpuChart" class="chart-container"></div>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card>
              <template #header>内存使用率</template>
              <div ref="memoryChart" class="chart-container"></div>
            </el-card>
          </el-col>
        </el-row>

        <el-row :gutter="20" class="charts-row">
          <el-col :span="12">
            <el-card>
              <template #header>网络流量</template>
              <div ref="networkChart" class="chart-container"></div>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card>
              <template #header>磁盘I/O</template>
              <div ref="diskChart" class="chart-container"></div>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <!-- 告警规则标签页 -->
      <el-tab-pane label="告警规则" name="rules">
        <el-button type="primary" @click="showCreateRule" style="margin-bottom: 20px">
          <el-icon><Plus /></el-icon>
          创建规则
        </el-button>

        <!-- 告警规则列表 -->
        <el-table :data="alertRules" style="width: 100%">
          <el-table-column prop="name" label="规则名称" width="200" />
          <el-table-column prop="metric" label="监控指标" width="150" />
          <el-table-column label="条件" width="200">
            <template #default="{ row }">
              {{ row.operator }} {{ row.threshold }}
            </template>
          </el-table-column>
          <el-table-column prop="duration" label="持续时间" width="120">
            <template #default="{ row }">
              {{ row.duration }}秒
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag v-if="row.enabled" type="success">启用</el-tag>
              <el-tag v-else type="info">禁用</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" fixed="right" width="200">
            <template #default="{ row }">
              <el-button size="small" @click="toggleRule(row)">
                {{ row.enabled ? '禁用' : '启用' }}
              </el-button>
              <el-button size="small" type="danger" @click="deleteRule(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 告警历史标签页 -->
      <el-tab-pane label="告警历史" name="alerts">
        <el-table :data="alerts" style="width: 100%">
          <el-table-column label="级别" width="100">
            <template #default="{ row }">
              <el-tag :type="getSeverityType(row.severity)">
                {{ row.severity }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="message" label="告警信息" />
          <el-table-column prop="fired_at" label="触发时间" width="180">
            <template #default="{ row }">
              {{ formatDate(row.fired_at) }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" fixed="right" width="200">
            <template #default="{ row }">
              <el-button v-if="row.status === 'firing'" size="small" @click="acknowledgeAlert(row)">
                确认
              </el-button>
              <el-button v-if="row.status === 'acknowledged'" size="small" type="success" @click="resolveAlert(row)">
                解决
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 创建告警规则对话框 -->
    <el-dialog v-model="createRuleVisible" title="创建告警规则" width="600px">
      <el-form :model="ruleForm" label-width="100px">
        <el-form-item label="规则名称">
          <el-input v-model="ruleForm.name" />
        </el-form-item>
        <el-form-item label="监控指标">
          <el-select v-model="ruleForm.metric_name" style="width: 100%">
            <el-option label="CPU使用率" value="cpu_usage" />
            <el-option label="内存使用率" value="memory_usage" />
            <el-option label="磁盘使用率" value="disk_usage" />
            <el-option label="网络流量" value="network_traffic" />
          </el-select>
        </el-form-item>
        <el-form-item label="条件">
          <el-row :gutter="10">
            <el-col :span="8">
              <el-select v-model="ruleForm.operator">
                <el-option label="大于" value=">" />
                <el-option label="大于等于" value=">=" />
                <el-option label="小于" value="<" />
                <el-option label="小于等于" value="<=" />
              </el-select>
            </el-col>
            <el-col :span="16">
              <el-input-number v-model="ruleForm.threshold" :min="0" :max="100" style="width: 100%" />
            </el-col>
          </el-row>
        </el-form-item>
        <el-form-item label="持续时间">
          <el-input-number v-model="ruleForm.duration" :min="60" :step="60" />
          <span style="margin-left: 10px">秒</span>
        </el-form-item>
        <el-form-item label="严重程度">
          <el-select v-model="ruleForm.severity">
            <el-option label="严重" value="critical" />
            <el-option label="警告" value="warning" />
            <el-option label="信息" value="info" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createRuleVisible = false">取消</el-button>
        <el-button type="primary" @click="createRule" :loading="creating">
          创建
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
// 监控告警视图组件 - 提供系统监控、告警规则管理和告警历史查看功能
import { ref, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'

// 响应式数据
const activeTab = ref('metrics')              // 当前激活的标签页
const systemMetrics = ref([])                 // 系统指标数据
const alertRules = ref([])                    // 告警规则列表
const alerts = ref([])                        // 告警历史列表
const createRuleVisible = ref(false)          // 创建规则对话框显示状态
const creating = ref(false)                   // 创建规则加载状态

// 图表实例引用
const cpuChart = ref(null)
const memoryChart = ref(null)
const networkChart = ref(null)
const diskChart = ref(null)
let cpuChartInstance = null
let memoryChartInstance = null
let networkChartInstance = null
let diskChartInstance = null

// 告警规则表单数据
const ruleForm = ref({
  name: '',
  metric_name: 'cpu_usage',
  operator: '>',
  threshold: 80,
  duration: 300,
  severity: 'warning'
})

// 加载系统指标数据
const loadMetrics = async () => {
  try {
    const response = await axios.get('/api/v1/monitoring/metrics')
    systemMetrics.value = [
      { name: 'cpu', label: 'CPU使用率', value: '45%', color: '#409EFF', icon: 'Cpu' },
      { name: 'memory', label: '内存使用率', value: '62%', color: '#67C23A', icon: 'MemoryCard' },
      { name: 'disk', label: '磁盘使用率', value: '38%', color: '#E6A23C', icon: 'Coin' },
      { name: 'network', label: '网络流量', value: '125MB/s', color: '#F56C6C', icon: 'Connection' }
    ]
  } catch (error) {
    ElMessage.error('加载指标数据失败')
  }
}

// 加载告警规则列表
const loadAlertRules = async () => {
  try {
    const response = await axios.get('/api/v1/monitoring/alert-rules')
    alertRules.value = response.data
  } catch (error) {
    ElMessage.error('加载告警规则失败')
  }
}

// 加载告警历史记录
const loadAlerts = async () => {
  try {
    const response = await axios.get('/api/v1/monitoring/alerts')
    alerts.value = response.data
  } catch (error) {
    ElMessage.error('加载告警历史失败')
  }
}

// 初始化图表
const initCharts = () => {
  if (cpuChart.value) {
    cpuChartInstance = echarts.init(cpuChart.value)
    cpuChartInstance.setOption(getChartOption('CPU使用率', '%'))
  }
  if (memoryChart.value) {
    memoryChartInstance = echarts.init(memoryChart.value)
    memoryChartInstance.setOption(getChartOption('内存使用率', '%'))
  }
  if (networkChart.value) {
    networkChartInstance = echarts.init(networkChart.value)
    networkChartInstance.setOption(getChartOption('网络流量', 'MB/s'))
  }
  if (diskChart.value) {
    diskChartInstance = echarts.init(diskChart.value)
    diskChartInstance.setOption(getChartOption('磁盘I/O', 'MB/s'))
  }
}

// 获取图表配置选项
const getChartOption = (title, unit) => {
  return {
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: [] },
    yAxis: { type: 'value', name: unit },
    series: [{ data: [], type: 'line', smooth: true }]
  }
}

// 显示创建规则对话框
const showCreateRule = () => {
  ruleForm.value = {
    name: '',
    metric_name: 'cpu_usage',
    operator: '>',
    threshold: 80,
    duration: 300,
    severity: 'warning'
  }
  createRuleVisible.value = true
}

// 创建告警规则
const createRule = async () => {
  creating.value = true
  try {
    await axios.post('/api/v1/monitoring/alert-rules', ruleForm.value)
    ElMessage.success('告警规则创建成功')
    createRuleVisible.value = false
    loadAlertRules()
  } catch (error) {
    ElMessage.error('告警规则创建失败')
  } finally {
    creating.value = false
  }
}

// 切换规则启用状态
const toggleRule = async (rule) => {
  try {
    await axios.put(`/api/v1/monitoring/alert-rules/${rule.id}`, {
      ...rule,
      enabled: !rule.enabled
    })
    rule.enabled = !rule.enabled
    ElMessage.success('规则状态更新成功')
  } catch (error) {
    ElMessage.error('规则状态更新失败')
  }
}

// 删除告警规则
const deleteRule = async (rule) => {
  try {
    await axios.delete(`/api/v1/monitoring/alert-rules/${rule.id}`)
    ElMessage.success('规则删除成功')
    loadAlertRules()
  } catch (error) {
    ElMessage.error('规则删除失败')
  }
}

// 确认告警
const acknowledgeAlert = async (alert) => {
  try {
    await axios.post(`/api/v1/monitoring/alerts/${alert.id}/acknowledge`)
    ElMessage.success('告警已确认')
    loadAlerts()
  } catch (error) {
    ElMessage.error('确认告警失败')
  }
}

// 解决告警
const resolveAlert = async (alert) => {
  try {
    await axios.post(`/api/v1/monitoring/alerts/${alert.id}/resolve`)
    ElMessage.success('告警已解决')
    loadAlerts()
  } catch (error) {
    ElMessage.error('解决告警失败')
  }
}

// 获取严重程度对应的标签类型
const getSeverityType = (severity) => {
  const types = { critical: 'danger', warning: 'warning', info: 'info' }
  return types[severity] || 'info'
}

// 获取状态对应的标签类型
const getStatusType = (status) => {
  const types = { firing: 'danger', acknowledged: 'warning', resolved: 'success' }
  return types[status] || 'info'
}

// 获取状态文本
const getStatusText = (status) => {
  const texts = { firing: '触发中', acknowledged: '已确认', resolved: '已解决' }
  return texts[status] || status
}

// 格式化日期
const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

// 组件挂载时初始化
onMounted(() => {
  loadMetrics()
  loadAlertRules()
  loadAlerts()
  setTimeout(initCharts, 100)
})

// 组件卸载前清理图表实例
onBeforeUnmount(() => {
  if (cpuChartInstance) cpuChartInstance.dispose()
  if (memoryChartInstance) memoryChartInstance.dispose()
  if (networkChartInstance) networkChartInstance.dispose()
  if (diskChartInstance) diskChartInstance.dispose()
})
</script>

<style scoped>
.monitoring-container {
  padding: 20px;
}

.metric-card {
  text-align: center;
  margin-bottom: 20px;
}

.metric-icon {
  margin: 20px 0;
}

.metric-name {
  font-size: 14px;
  color: #909399;
  margin-bottom: 10px;
}

.metric-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}

.charts-row {
  margin-top: 20px;
}

.chart-container {
  height: 300px;
}
</style>             <el-button size="small" @click="toggleRule(row)">
                {{ row.enabled ? '禁用' : '启用' }}
              </el-button>
              <el-button size="small" type="danger" @click="deleteRule(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 告警历史 -->
      <el-tab-pane label="告警历史" name="alerts">
        <el-table :data="alerts" style="width: 100%">
          <el-table-column label="级别" width="100">
            <template #default="{ row }">
              <el-tag :type="getSeverityType(row.severity)">
                {{ row.severity }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="rule_name" label="规则名称" width="200" />
          <el-table-column prop="message" label="告警信息" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag v-if="row.status === 'firing'" type="danger">触发中</el-tag>
              <el-tag v-else type="success">已解决</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="triggered_at" label="触发时间" width="180">
            <template #default="{ row }">
              {{ formatDate(row.triggered_at) }}
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- AI预测分析 -->
      <el-tab-pane label="AI预测分析" name="prediction">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>容量预测与优化建议</span>
              <el-button size="small" @click="runPrediction" :loading="predicting">
                <el-icon><MagicStick /></el-icon>
                运行分析
              </el-button>
            </div>
          </template>

          <div v-if="predictionResult">
            <el-alert
              :type="predictionResult.risk_level === 'high' ? 'error' : 'warning'"
              :closable="false"
              style="margin-bottom: 20px"
            >
              <template #title>
                风险等级: {{ predictionResult.risk_level }}
              </template>
              {{ predictionResult.summary }}
            </el-alert>

            <el-descriptions title="预测详情" :column="2" border>
              <el-descriptions-item label="预测时间范围">
                {{ predictionResult.prediction_window }}
              </el-descriptions-item>
              <el-descriptions-item label="置信度">
                {{ predictionResult.confidence }}%
              </el-descriptions-item>
              <el-descriptions-item label="CPU趋势">
                {{ predictionResult.cpu_trend }}
              </el-descriptions-item>
              <el-descriptions-item label="内存趋势">
                {{ predictionResult.memory_trend }}
              </el-descriptions-item>
            </el-descriptions>

            <div class="recommendations">
              <h4>优化建议</h4>
              <ul>
                <li v-for="(rec, index) in predictionResult.recommendations" :key="index">
                  {{ rec }}
                </li>
              </ul>
            </div>
          </div>

          <el-empty v-else description="点击运行分析按钮获取AI预测" />
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- 创建告警规则对话框 -->
    <el-dialog v-model="createRuleVisible" title="创建告警规则" width="500px">
      <el-form :model="ruleForm" label-width="100px">
        <el-form-item label="规则名称">
          <el-input v-model="ruleForm.name" />
        </el-form-item>
        <el-form-item label="监控指标">
          <el-select v-model="ruleForm.metric" style="width: 100%">
            <el-option label="CPU使用率" value="cpu_usage" />
            <el-option label="内存使用率" value="memory_usage" />
            <el-option label="磁盘使用率" value="disk_usage" />
            <el-option label="网络流量" value="network_traffic" />
          </el-select>
        </el-form-item>
        <el-form-item label="条件">
          <el-row :gutter="10">
            <el-col :span="8">
              <el-select v-model="ruleForm.operator">
                <el-option label="大于" value=">" />
                <el-option label="小于" value="<" />
                <el-option label="等于" value="=" />
              </el-select>
            </el-col>
            <el-col :span="16">
              <el-input-number v-model="ruleForm.threshold" :min="0" :max="100" />
            </el-col>
          </el-row>
        </el-form-item>
        <el-form-item label="持续时间">
          <el-input-number v-model="ruleForm.duration" :min="10" :max="3600" />
          <span style="margin-left: 10px">秒</span>
        </el-form-item>
        <el-form-item label="告警级别">
          <el-select v-model="ruleForm.severity" style="width: 100%">
            <el-option label="严重" value="critical" />
            <el-option label="警告" value="warning" />
            <el-option label="信息" value="info" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createRuleVisible = false">取消</el-button>
        <el-button type="primary" @click="createRule" :loading="creatingRule">
          创建
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import * as echarts from 'echarts'

const activeTab = ref('metrics')
const alertRules = ref([])
const alerts = ref([])
const createRuleVisible = ref(false)
const creatingRule = ref(false)
const predicting = ref(false)
const predictionResult = ref(null)

const systemMetrics = ref([
  { name: 'cpu', label: 'CPU使用率', value: '0%', icon: 'Cpu', color: '#409EFF' },
  { name: 'memory', label: '内存使用率', value: '0%', icon: 'Memo', color: '#67C23A' },
  { name: 'disk', label: '磁盘使用率', value: '0%', icon: 'FolderOpened', color: '#E6A23C' },
  { name: 'network', label: '网络流量', value: '0 MB/s', icon: 'Connection', color: '#F56C6C' }
])

const ruleForm = ref({
  name: '',
  metric: 'cpu_usage',
  operator: '>',
  threshold: 80,
  duration: 60,
  severity: 'warning'
})

const cpuChart = ref(null)
const memoryChart = ref(null)
const networkChart = ref(null)
const diskChart = ref(null)

let charts = []
let metricsInterval = null

// 加载告警规则
const loadAlertRules = async () => {
  try {
    const response = await axios.get('/api/monitoring/alert-rules')
    alertRules.value = response.data
  } catch (error) {
    ElMessage.error('加载告警规则失败')
  }
}

// 加载告警历史
const loadAlerts = async () => {
  try {
    const response = await axios.get('/api/monitoring/alerts')
    alerts.value = response.data
  } catch (error) {
    ElMessage.error('加载告警历史失败')
  }
}

// 加载系统指标
const loadMetrics = async () => {
  try {
    const response = await axios.get('/api/monitoring/metrics')
    const metrics = response.data

    // 更新指标卡片
    systemMetrics.value[0].value = `${metrics.cpu_usage?.toFixed(1) || 0}%`
    systemMetrics.value[1].value = `${metrics.memory_usage?.toFixed(1) || 0}%`
    systemMetrics.value[2].value = `${metrics.disk_usage?.toFixed(1) || 0}%`
    systemMetrics.value[3].value = `${(metrics.network_traffic / 1024 / 1024).toFixed(2)} MB/s`

    // 更新图表（这里简化处理，实际应该维护时间序列数据）
    updateCharts(metrics)
  } catch (error) {
    console.error('加载指标失败', error)
  }
}

// 初始化图表
const initCharts = () => {
  const option = {
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: [] },
    yAxis: { type: 'value', max: 100 },
    series: [{ data: [], type: 'line', smooth: true }]
  }

  if (cpuChart.value) {
    const chart = echarts.init(cpuChart.value)
    chart.setOption(option)
    charts.push(chart)
  }
  if (memoryChart.value) {
    const chart = echarts.init(memoryChart.value)
    chart.setOption(option)
    charts.push(chart)
  }
  if (networkChart.value) {
    const chart = echarts.init(networkChart.value)
    chart.setOption(option)
    charts.push(chart)
  }
  if (diskChart.value) {
    const chart = echarts.init(diskChart.value)
    chart.setOption(option)
    charts.push(chart)
  }
}

// 更新图表
const updateCharts = (metrics) => {
  // 简化实现，实际应该维护历史数据
  const time = new Date().toLocaleTimeString()
  
  charts.forEach((chart, index) => {
    const option = chart.getOption()
    const data = option.series[0].data
    const xData = option.xAxis[0].data
    
    xData.push(time)
    if (xData.length > 20) xData.shift()
    
    let value = 0
    switch(index) {
      case 0: value = metrics.cpu_usage || 0; break
      case 1: value = metrics.memory_usage || 0; break
      case 2: value = metrics.network_traffic / 1024 / 1024 || 0; break
      case 3: value = metrics.disk_usage || 0; break
    }
    
    data.push(value)
    if (data.length > 20) data.shift()
    
    chart.setOption({
      xAxis: { data: xData },
      series: [{ data: data }]
    })
  })
}

// 显示创建规则对话框
const showCreateRule = () => {
  ruleForm.value = {
    name: '',
    metric: 'cpu_usage',
    operator: '>',
    threshold: 80,
    duration: 60,
    severity: 'warning'
  }
  createRuleVisible.value = true
}

// 创建告警规则
const createRule = async () => {
  creatingRule.value = true
  try {
    await axios.post('/api/monitoring/alert-rules', ruleForm.value)
    ElMessage.success('告警规则创建成功')
    createRuleVisible.value = false
    loadAlertRules()
  } catch (error) {
    ElMessage.error('告警规则创建失败')
  } finally {
    creatingRule.value = false
  }
}

// 切换规则状态
const toggleRule = async (rule) => {
  try {
    await axios.put(`/api/monitoring/alert-rules/${rule.id}`, {
      enabled: !rule.enabled
    })
    rule.enabled = !rule.enabled
    ElMessage.success('规则状态更新成功')
  } catch (error) {
    ElMessage.error('规则状态更新失败')
  }
}

// 删除规则
const deleteRule = async (rule) => {
  try {
    await ElMessageBox.confirm('确定要删除此规则吗？', '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    
    await axios.delete(`/api/monitoring/alert-rules/${rule.id}`)
    ElMessage.success('规则删除成功')
    loadAlertRules()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('规则删除失败')
    }
  }
}

// 运行AI预测
const runPrediction = async () => {
  predicting.value = true
  try {
    const response = await axios.post('/api/monitoring/predict')
    predictionResult.value = response.data
    ElMessage.success('AI分析完成')
  } catch (error) {
    ElMessage.error('AI分析失败')
  } finally {
    predicting.value = false
  }
}

// 获取告警级别类型
const getSeverityType = (severity) => {
  const types = {
    critical: 'danger',
    warning: 'warning',
    info: 'info'
  }
  return types[severity] || 'info'
}

// 格式化日期
const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

onMounted(() => {
  loadAlertRules()
  loadAlerts()
  loadMetrics()
  
  // 初始化图表
  setTimeout(() => {
    initCharts()
  }, 100)
  
  // 定时刷新指标
  metricsInterval = setInterval(loadMetrics, 5000)
})

onBeforeUnmount(() => {
  if (metricsInterval) {
    clearInterval(metricsInterval)
  }
  charts.forEach(chart => chart.dispose())
})
</script>

<style scoped>
.monitoring-container {
  padding: 20px;
}

.metric-card {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.metric-icon {
  margin-right: 15px;
}

.metric-info {
  flex: 1;
}

.metric-name {
  font-size: 14px;
  color: #909399;
  margin-bottom: 5px;
}

.metric-value {
  font-size: 24px;
  font-weight: bold;
}

.charts-row {
  margin-top: 20px;
}

.chart-container {
  height: 300px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.recommendations {
  margin-top: 20px;
}

.recommendations h4 {
  margin-bottom: 10px;
}

.recommendations ul {
  padding-left: 20px;
}

.recommendations li {
  margin-bottom: 8px;
  line-height: 1.6;
}
</style>
