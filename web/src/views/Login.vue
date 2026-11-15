<template>
  <div class="login-page">
    <el-card class="login-card" shadow="always">
      <div class="login-title">工业软件企业后台管理系统</div>
      <div class="login-sub">数字化运营平台 · 统一登录入口</div>

      <el-form ref="formRef" :model="form" :rules="rules" size="large" @keyup.enter="onSubmit">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="请输入登录账号" clearable />
        </el-form-item>
        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="请输入登录口令"
            show-password
            clearable
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" class="login-btn" :loading="loading" @click="onSubmit">
            {{ loading ? '登录中…' : '登 录' }}
          </el-button>
        </el-form-item>
      </el-form>

      <div class="login-tip">默认管理员账号：admin / Admin@123</div>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { login } from '@/api/auth'

const router = useRouter()
const route = useRoute()

const formRef = ref(null)
const loading = ref(false)
const form = reactive({ username: '', password: '' })

// 前端只做必填与长度校验，后端仍会再校验一次，前端校验的目的是减少无效请求
const rules = {
  username: [
    { required: true, message: '请输入登录账号', trigger: 'blur' },
    { min: 2, max: 50, message: '账号长度应为 2 到 50 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入登录口令', trigger: 'blur' },
    { min: 6, max: 32, message: '口令长度应为 6 到 32 位', trigger: 'blur' }
  ]
}

async function onSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    const data = await login({
      username: form.username.trim(),
      password: form.password
    })

    // 令牌与用户信息落本地，后续请求由 axios 拦截器自动带上
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(data.user || {}))
    localStorage.setItem('permissions', JSON.stringify(data.permissions || []))
    localStorage.setItem('menus', JSON.stringify(data.menus || []))

    ElMessage.success('登录成功，欢迎 ' + (data.user?.realName || form.username))
    const redirect = route.query.redirect || '/home'
    router.push(String(redirect))
  } catch (e) {
    // 错误提示已由 axios 响应拦截器统一弹出，此处只结束加载状态
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1f2d3d 0%, #2f4f6f 100%);
}

.login-card {
  width: 380px;
  padding: 8px 12px 4px;
  border-radius: 8px;
}

.login-title {
  font-size: 18px;
  font-weight: 600;
  text-align: center;
  color: #1f2d3d;
}

.login-sub {
  font-size: 12px;
  color: #909399;
  text-align: center;
  margin: 6px 0 22px;
}

.login-btn {
  width: 100%;
}

.login-tip {
  font-size: 12px;
  color: #a8abb2;
  text-align: center;
  margin-top: 4px;
}
</style>
