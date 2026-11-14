<template>
  <el-card class="feature-status" shadow="never">
    <el-empty :description="t(`capabilities.${feature}.title`)">
      <p>{{ t(`capabilities.${feature}.description`) }}</p>
      <p class="muted">{{ t('capabilities.unavailable') }}</p>
      <router-link to="/dashboard"><el-button type="primary">{{ t('menu.dashboard') }}</el-button></router-link>
    </el-empty>
  </el-card>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
defineProps({ feature: { type: String, required: true } })
const { t } = useI18n()
</script>

<style scoped>
.feature-status { max-width: 900px; margin: 24px auto; }
p { max-width: 660px; line-height: 1.8; margin: 0 0 16px; }
.muted { color: var(--el-text-color-secondary); }
</style>
