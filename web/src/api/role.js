import request from '@/utils/request'

// 角色列表（分页）
export function listRoles(params) {
  return request.get('/roles', { params })
}

export function createRole(data) {
  return request.post('/roles', data)
}

export function updateRole(id, data) {
  return request.put(`/roles/${id}`, data)
}

export function deleteRole(id) {
  return request.delete(`/roles/${id}`)
}

// 角色已拥有的权限点标识数组
export function getRolePermissions(id) {
  return request.get(`/roles/${id}/permissions`)
}

// 角色授权：整体替换权限集合（后端事务）
export function grantRolePermissions(id, permIds) {
  return request.put(`/roles/${id}/permissions`, { permIds })
}
