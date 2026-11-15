import request from '@/utils/request'

// 用户列表（分页 + 关键字 + 状态）
export function listUsers(params) {
  return request.get('/users', { params })
}

// 新增用户：后端会同时写入角色关联
export function createUser(data) {
  return request.post('/users', data)
}

// 修改用户：roleIds 非空时整体替换角色
export function updateUser(id, data) {
  return request.put(`/users/${id}`, data)
}

// 删除用户（软删除，内置 admin 不可删）
export function deleteUser(id) {
  return request.delete(`/users/${id}`)
}

// 重置口令
export function resetPassword(id, password) {
  return request.post(`/users/${id}/password`, { password })
}

// 查询用户已绑定角色
export function getUserRoles(id) {
  return request.get(`/users/${id}/roles`)
}

// 绑定用户角色（事务整体替换）
export function bindUserRoles(id, roleIds) {
  return request.put(`/users/${id}/roles`, { roleIds })
}
