import request from '@/utils/request'

// 运营看板概览：用户总数、今日活跃、今日操作、失败操作与趋势、模块分布
export function getOverview(params) {
  return request.get('/dashboard/overview', { params })
}
