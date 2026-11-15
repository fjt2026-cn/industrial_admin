<template>
  <el-container class="layout">
    <el-header class="header">
      <div class="brand">工业软件企业后台管理系统</div>
      <div class="header-right">
        <span class="user-name">{{ user.realName || user.username || '未登录' }}</span>
        <el-button link type="primary" @click="onLogout">退出登录</el-button>
      </div>
    </el-header>

    <el-main>
      <el-alert
        title="前端基础环境已就绪：路由守卫、Axios 封装、令牌携带与统一错误提示均已接入"
        type="success"
        :closable="false"
        show-icon
        class="tip"
      />

      <el-row :gutter="16" class="cards">
        <el-col :span="6">
          <el-card shadow="hover">
            <div class="card-label">用户总数</div>
            <div class="card-value">{{ overview.userTotal }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <div class="card-label">今日活跃</div>
            <div class="card-value">{{ overview.todayActive }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <div class="card-label">今日操作</div>
            <div class="card-value">{{ overview.todayOps }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="hover">
            <div class="card-label">失败操作</div>
            <div class="card-value danger">{{ overview.failOps }}</div>
          </el-card>
        </el-col>
      </el-row>

      <el-card shadow="never" class="panel">
        <template #header>接口自检</template>
        <p class="panel-text">
          本页数据来自 <code>GET /api/v1/dashboard/overview</code>，请求由 Axios 拦截器自动带上 Bearer 令牌；
          若令牌失效，拦截器会清除本地缓存并跳回登录页。
        </p>
      </el-card>
    </el-main>
  </el-container>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'
import { getOverview } from '@/api/dashboard'

const router = useRouter()
const user = ref(JSON.parse(localStorage.getItem('user') || '{}'))
const overview = reactive({ userTotal: 0, todayActive: 0, todayOps: 0, failOps: 0 })

async function loadOverview() {
  try {
    const data = await getOverview({ days: 7 })
    Object.assign(overview, {
      userTotal: data.userTotal,
      todayActive: data.todayActive,
      todayOps: data.todayOps,
      failOps: data.failOps
    })
  } catch (e) {
    // 错误提示由拦截器统一处理
  }
}

async function onLogout() {
  await ElMessageBox.confirm('确认退出登录？', '提示', { type: 'warning' })
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  localStorage.removeItem('permissions')
  localStorage.removeItem('menus')
  router.push('/login')
}

onMounted(loadOverview)
</script>

<style scoped>
.layout {
  min-height: 100vh;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #1f2d3d;
  color: #fff;
}

.brand {
  font-size: 16px;
  font-weight: 600;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-name {
  font-size: 13px;
  color: #d0d4da;
}

.tip {
  margin-bottom: 16px;
}

.cards {
  margin-bottom: 8px;
}

.card-label {
  font-size: 12px;
  color: #909399;
}

.card-value {
  font-size: 26px;
  font-weight: 600;
  color: #303133;
  margin-top: 8px;
}

.card-value.danger {
  color: #f56c6c;
}

.panel-text {
  font-size: 13px;
  color: #606266;
  line-height: 1.8;
}
</style>
