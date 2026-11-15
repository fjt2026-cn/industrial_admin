import request from '@/utils/request'

// 用户增长：{ granularity, xAxis, newUsers, totalUsers, sumNew }
export function getUserGrowth(params) {
  return request.get('/stats/users/growth', { params })
}

// 用户活跃度：{ xAxis, activeUsers, loginCount, operationCount }
export function getUserActivity(params) {
  return request.get('/stats/users/activity', { params })
}

// 日志频次：{ xAxis, total, success, failed, moduleDist }
export function getLogFrequency(params) {
  return request.get('/stats/logs/frequency', { params })
}
