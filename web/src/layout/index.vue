<template>
  <el-container class="layout">
    <!-- 侧边栏：菜单项与路由表一一对应，点击即跳转（router 模式） -->
    <el-aside :width="collapsed ? '64px' : '200px'" class="aside">
      <div class="logo">{{ collapsed ? 'IMS' : '工业软件后台管理' }}</div>
      <el-menu
        :default-active="activeMenu"
        :collapse="collapsed"
        router
        background-color="#1f2d3d"
        text-color="#c0c4cc"
        active-text-color="#ffd04b"
      >
        <el-menu-item index="/dashboard">运营看板</el-menu-item>
        <el-sub-menu index="sys">
          <template #title>系统管理</template>
          <el-menu-item index="/users">用户管理</el-menu-item>
          <el-menu-item index="/roles">角色授权</el-menu-item>
        </el-sub-menu>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="header">
        <div class="header-left">
          <el-button link type="primary" @click="collapsed = !collapsed">
            {{ collapsed ? '展开' : '收起' }}
          </el-button>
          <span class="page-title">{{ route.meta.title }}</span>
        </div>
        <div class="header-right">
          <span class="user-name">{{ user.realName || user.username }}</span>
          <el-button link type="primary" @click="onLogout">退出登录</el-button>
        </div>
      </el-header>

      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const collapsed = ref(false)
const user = ref(JSON.parse(localStorage.getItem('user') || '{}'))

// 高亮当前菜单：取路由 path，子路由也能对上父级菜单
const activeMenu = computed(() => route.path)

async function onLogout() {
  await ElMessageBox.confirm('确认退出登录？', '提示', { type: 'warning' })
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  localStorage.removeItem('permissions')
  localStorage.removeItem('menus')
  router.push('/login')
}
</script>

<style scoped>
.layout {
  min-height: 100vh;
}

.aside {
  background: #1f2d3d;
  transition: width 0.2s;
  overflow-x: hidden;
}

.logo {
  height: 56px;
  line-height: 56px;
  text-align: center;
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  border-bottom: 1px solid #2c3e50;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #ebeef5;
  background: #fff;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-name {
  font-size: 13px;
  color: #606266;
}

.main {
  background: #f5f7fa;
  padding: 16px;
}
</style>
