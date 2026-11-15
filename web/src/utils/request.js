import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'

// 统一请求实例：baseURL 走 /api/v1，开发期由 vite 代理转发到后端
const service = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: { 'Content-Type': 'application/json;charset=utf-8' }
})

// 请求拦截：自动携带 token；登录接口本身不需要
service.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截：后端统一返回 {code, message, data}，这里只把 data 交给业务层
service.interceptors.response.use(
  (response) => {
    const body = response.data
    if (body.code === 0) {
      return body.data
    }

    // 业务错误：40100 未登录/令牌失效，40101 口令错误等，统一提示后拒绝
    ElMessage.error(body.message || '请求失败')
    if (body.code === 40100) {
      localStorage.removeItem('token')
      router.push('/login')
    }
    return Promise.reject(new Error(body.message || '请求失败'))
  },
  (error) => {
    // 网络层错误：超时、后端未启动、跨域被拒等
    let msg = '网络异常，请稍后重试'
    if (error.response) {
      const status = error.response.status
      const body = error.response.data || {}
      msg = body.message || `请求失败（HTTP ${status}）`
      if (status === 401) {
        localStorage.removeItem('token')
        router.push('/login')
      }
    } else if (error.code === 'ECONNABORTED') {
      msg = '请求超时，请检查后端服务是否已启动'
    }
    ElMessage.error(msg)
    return Promise.reject(error)
  }
)

export default service
