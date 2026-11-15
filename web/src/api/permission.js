import request from '@/utils/request'

// 权限点列表（分页）：permType 1 菜单 2 按钮 3 接口
export function listPermissions(params) {
  return request.get('/permissions', { params })
}
