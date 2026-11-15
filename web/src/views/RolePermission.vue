<template>
  <div>
    <el-card shadow="never" class="search-card">
      <el-form :inline="true" :model="query">
        <el-form-item label="关键字">
          <el-input
            v-model="query.keyword"
            placeholder="角色名称或编码"
            clearable
            style="width: 220px"
            @keyup.enter="onSearch"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="onSearch">查询</el-button>
          <el-button @click="onReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" border stripe>
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column prop="roleName" label="角色名称" width="160" />
        <el-table-column prop="roleCode" label="角色编码" width="160" />
        <el-table-column label="数据范围" width="120" align="center">
          <template #default="{ row }">{{ dataScopeText(row.dataScope) }}</template>
        </el-table-column>
        <el-table-column label="内置" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.isBuiltin === 1 ? 'warning' : 'info'" size="small">
              {{ row.isBuiltin === 1 ? '内置' : '自定义' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column label="操作" width="140" fixed="right" align="center">
          <template #default="{ row }">
            <el-button link type="primary" @click="openGrant(row)">分配权限</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        class="pager"
        @current-change="load"
      />
    </el-card>

    <!-- 授权抽屉：左侧勾选权限点，右侧预览已选清单 -->
    <el-drawer v-model="drawerVisible" :title="`分配权限 · ${current.roleName || ''}`" size="620px">
      <div v-loading="permLoading" class="grant-body">
        <el-alert
          type="info"
          :closable="false"
          show-icon
          title="内置角色的权限由系统维护，保存会整体替换该角色的权限集合（后端事务保证一致）"
          class="grant-tip"
        />

        <el-tree
          ref="treeRef"
          :data="treeData"
          node-key="key"
          show-checkbox
          default-expand-all
          :props="{ label: 'label', children: 'children' }"
          class="perm-tree"
          @check="onCheck"
        />

        <div class="grant-summary">
          已选 {{ checkedCount }} 个权限点
        </div>
      </div>

      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSave">保存授权</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { nextTick, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listPermissions } from '@/api/permission'
import { getRolePermissions, grantRolePermissions, listRoles } from '@/api/role'

const loading = ref(false)
const permLoading = ref(false)
const saving = ref(false)
const rows = ref([])
const total = ref(0)
const query = reactive({ page: 1, pageSize: 10, keyword: '' })

const drawerVisible = ref(false)
const current = ref({})
const treeRef = ref(null)
const treeData = ref([])
const checkedCount = ref(0)

function dataScopeText(v) {
  return { 0: '全部数据', 1: '本部门', 2: '本部门及下级', 3: '仅本人' }[v] || '未设置'
}

async function load() {
  loading.value = true
  try {
    const data = await listRoles({
      page: query.page,
      pageSize: query.pageSize,
      keyword: query.keyword || undefined
    })
    rows.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    rows.value = []
  } finally {
    loading.value = false
  }
}

function onSearch() {
  query.page = 1
  load()
}

function onReset() {
  query.keyword = ''
  query.page = 1
  load()
}

// 权限点按类型分三组：1 菜单、2 按钮、3 接口，叶子节点的 key 即权限点 id
function buildTree(list) {
  const groups = [
    { key: 'g-menu', label: '菜单权限', children: [] },
    { key: 'g-btn', label: '按钮权限', children: [] },
    { key: 'g-api', label: '接口权限', children: [] }
  ]
  for (const p of list) {
    const idx = p.permType === 1 ? 0 : p.permType === 2 ? 1 : 2
    groups[idx].children.push({
      key: `p-${p.id}`,
      id: p.id,
      label: `${p.permName}（${p.permCode}）`
    })
  }
  return groups.filter((g) => g.children.length > 0)
}

async function openGrant(row) {
  current.value = row
  drawerVisible.value = true
  permLoading.value = true
  try {
    // 权限点总量不大，一次取全部；角色已授权标识用于回显
    const [permPage, owned] = await Promise.all([
      listPermissions({ page: 1, pageSize: 200 }),
      getRolePermissions(row.id)
    ])
    treeData.value = buildTree(permPage.list || [])
    await nextTick()
    const keys = (owned || []).map((id) => `p-${id}`)
    treeRef.value?.setCheckedKeys(keys, false)
    checkedCount.value = keys.length
  } catch (e) {
    treeData.value = []
  } finally {
    permLoading.value = false
  }
}

function collectChecked() {
  // 分组节点只是展示用，真正提交的只有叶子上的权限点
  const all = [
    ...(treeRef.value?.getCheckedKeys(false) || []),
    ...(treeRef.value?.getHalfCheckedKeys() || [])
  ]
  return all
    .filter((k) => String(k).startsWith('p-'))
    .map((k) => Number(String(k).slice(2)))
}

// 勾选变化时实时更新已选数量
function onCheck() {
  checkedCount.value = collectChecked().length
}

async function onSave() {
  const permIds = collectChecked()
  saving.value = true
  try {
    await grantRolePermissions(current.value.id, permIds.length ? permIds : [])
    ElMessage.success(`已保存，共 ${permIds.length} 个权限点`)
    checkedCount.value = permIds.length
    drawerVisible.value = false
  } catch (e) {
    // 错误提示由拦截器统一处理（如内置角色禁止修改返回 403）
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.search-card {
  margin-bottom: 16px;
}

.pager {
  margin-top: 16px;
  justify-content: flex-end;
}

.grant-body {
  padding-bottom: 12px;
}

.grant-tip {
  margin-bottom: 12px;
}

.perm-tree {
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 8px;
  max-height: 420px;
  overflow: auto;
}

.grant-summary {
  margin-top: 12px;
  font-size: 13px;
  color: #606266;
}
</style>
