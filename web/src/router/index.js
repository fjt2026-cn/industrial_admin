import { createRouter, createWebHistory } from 'vue-router'

// 路由表：登录页公开；业务页挂在主布局下，共用侧边栏
const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { public: true, title: '登录' }
  },
  {
    path: '/',
    component: () => import('@/layout/index.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '运营看板' }
      },
      {
        path: 'users',
        name: 'UserManage',
        component: () => import('@/views/UserManage.vue'),
        meta: { title: '用户管理' }
      },
      {
        path: 'roles',
        name: 'RolePermission',
        component: () => import('@/views/RolePermission.vue'),
        meta: { title: '角色授权' }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/dashboard'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 全局前置守卫：无 token 且目标页非公开时，跳回登录页
router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (!to.meta.public && !token) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.path === '/login' && token) {
    return { path: '/dashboard' }
  }
  return true
})

export default router
