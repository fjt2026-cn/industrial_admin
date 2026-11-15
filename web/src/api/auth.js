import request from '@/utils/request'

// 登录：成功返回 { token, expiresAt, user, permissions, menus }
export function login(data) {
  return request.post('/auth/login', data)
}

// 当前登录人信息（令牌有效时可用）
export function getProfile() {
  return request.get('/auth/profile')
}
