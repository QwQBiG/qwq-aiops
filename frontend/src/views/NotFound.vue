<template>
  <el-empty :description="t('console.notFound')">
    <router-link to="/dashboard"><el-button type="primary">{{ t('console.goDashboard') }}</el-button></router-link>
  </el-empty>
</template>
<script setup>
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
</script>
