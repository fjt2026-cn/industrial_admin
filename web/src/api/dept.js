import request from '@/utils/request'

// 部门树（含 children 递归）
export function getDeptTree() {
  return request.get('/depts/tree')
}

// 部门平铺分页列表
export function listDepts(params) {
  return request.get('/depts', { params })
}
