<template>
  <div>
    <!-- 查询区：关键字 + 状态，回车即查 -->
    <el-card shadow="never" class="search-card">
      <el-form :inline="true" :model="query">
        <el-form-item label="关键字">
          <el-input
            v-model="query.keyword"
            placeholder="账号或姓名"
            clearable
            style="width: 200px"
            @keyup.enter="onSearch"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" placeholder="全部" clearable style="width: 120px">
            <el-option label="启用" :value="1" />
            <el-option label="停用" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="onSearch">查询</el-button>
          <el-button @click="onReset">重置</el-button>
          <el-button type="success" @click="openCreate">新增用户</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 列表：部门名与角色名由后端聚合返回，前端只渲染 -->
    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" border stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column prop="username" label="账号" width="130" />
        <el-table-column prop="realName" label="姓名" width="110" />
        <el-table-column prop="deptName" label="所属部门" width="130" />
        <el-table-column label="角色" min-width="160">
          <template #default="{ row }">
            <el-tag v-for="r in row.roles" :key="r" size="small" class="role-tag">{{ r }}</el-tag>
            <span v-if="!row.roles || row.roles.length === 0" class="empty-text">未分配</span>
          </template>
        </el-table-column>
        <el-table-column prop="email" label="邮箱" min-width="170" show-overflow-tooltip />
        <el-table-column prop="phone" label="手机号" width="130" />
        <el-table-column label="状态" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="170">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="210" fixed="right" align="center">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="warning" @click="openReset(row)">重置口令</el-button>
            <el-button link type="danger" :disabled="row.username === 'admin'" @click="onDelete(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        class="pager"
        @current-change="load"
        @size-change="load"
      />
    </el-card>

    <!-- 新增 / 编辑弹窗：新增时显示账号与口令，编辑时显示状态 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑用户' : '新增用户'" width="560px">
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="90px">
        <el-form-item label="账号" prop="username">
          <el-input v-model="form.username" :disabled="isEdit" placeholder="字母数字，2-50 位" />
        </el-form-item>
        <el-form-item v-if="!isEdit" label="口令" prop="password">
          <el-input v-model="form.password" type="password" show-password placeholder="至少 6 位" />
        </el-form-item>
        <el-form-item label="姓名" prop="realName">
          <el-input v-model="form.realName" placeholder="真实姓名" />
        </el-form-item>
        <el-form-item label="所属部门" prop="deptId">
          <el-select v-model="form.deptId" placeholder="请选择部门" clearable style="width: 100%">
            <el-option
              v-for="d in deptOptions"
              :key="d.id"
              :label="d.label"
              :value="d.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="角色" prop="roleIds">
          <el-select v-model="form.roleIds" multiple placeholder="可多选" style="width: 100%">
            <el-option v-for="r in roleOptions" :key="r.id" :label="r.roleName" :value="r.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="选填" />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input v-model="form.phone" placeholder="选填" />
        </el-form-item>
        <el-form-item v-if="isEdit" label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">停用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" placeholder="选填" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onSubmit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 重置口令：单独一个小弹窗，避免与编辑表单混在一起 -->
    <el-dialog v-model="resetVisible" title="重置口令" width="420px">
      <el-form label-width="90px">
        <el-form-item label="账号">
          <span>{{ resetTarget.username }}</span>
        </el-form-item>
        <el-form-item label="新口令">
          <el-input v-model="resetPwd" type="password" show-password placeholder="至少 6 位" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="onResetSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getDeptTree } from '@/api/dept'
import { listRoles } from '@/api/role'
import { createUser, deleteUser, listUsers, resetPassword, updateUser } from '@/api/user'

const loading = ref(false)
const saving = ref(false)
const rows = ref([])
const total = ref(0)

const query = reactive({ page: 1, pageSize: 10, keyword: '', status: undefined })

const dialogVisible = ref(false)
const isEdit = ref(false)
const editingId = ref(null)
const formRef = ref(null)
const form = reactive({
  username: '', password: '', realName: '', email: '', phone: '',
  deptId: undefined, roleIds: [], status: 1, remark: ''
})

const resetVisible = ref(false)
const resetTarget = ref({})
const resetPwd = ref('')

const deptOptions = ref([])
const roleOptions = ref([])

const formRules = {
  username: [
    { required: true, message: '请输入账号', trigger: 'blur' },
    { min: 2, max: 50, message: '账号长度 2-50 位', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入口令', trigger: 'blur' },
    { min: 6, max: 32, message: '口令长度 6-32 位', trigger: 'blur' }
  ],
  realName: [{ required: true, message: '请输入姓名', trigger: 'blur' }]
}

// 时间格式化：后端返回 RFC3339，这里只取到分钟
function formatTime(v) {
  if (!v) return ''
  const d = new Date(v)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

async function load() {
  loading.value = true
  try {
    const data = await listUsers({
      page: query.page,
      pageSize: query.pageSize,
      keyword: query.keyword || undefined,
      status: query.status
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
  query.status = undefined
  query.page = 1
  load()
}

// 部门树拍平成下拉选项，用缩进体现层级
function flattenDept(nodes, level = 0, out = []) {
  for (const n of nodes) {
    out.push({ id: n.id, label: '　'.repeat(level) + n.name })
    if (n.children?.length) flattenDept(n.children, level + 1, out)
  }
  return out
}

async function loadOptions() {
  try {
    const tree = await getDeptTree()
    deptOptions.value = flattenDept(tree || [])
  } catch (e) {
    deptOptions.value = []
  }
  try {
    const data = await listRoles({ page: 1, pageSize: 50 })
    roleOptions.value = data.list || []
  } catch (e) {
    roleOptions.value = []
  }
}

function resetForm() {
  Object.assign(form, {
    username: '', password: '', realName: '', email: '', phone: '',
    deptId: undefined, roleIds: [], status: 1, remark: ''
  })
  formRef.value?.clearValidate()
}

function openCreate() {
  isEdit.value = false
  editingId.value = null
  resetForm()
  dialogVisible.value = true
}

function openEdit(row) {
  isEdit.value = true
  editingId.value = row.id
  Object.assign(form, {
    username: row.username,
    password: '',
    realName: row.realName,
    email: row.email,
    phone: row.phone,
    deptId: row.deptId ?? undefined,
    roleIds: [],
    status: row.status,
    remark: ''
  })
  // 编辑时回显已绑定角色（按角色名称匹配标识）
  fillRoleIds(row.roles || [])
  dialogVisible.value = true
  formRef.value?.clearValidate()
}

function fillRoleIds(roleNames) {
  const ids = roleOptions.value
    .filter((r) => roleNames.includes(r.roleName))
    .map((r) => r.id)
  form.roleIds = ids
  if (ids.length === 0) {
    // 角色列表可能尚未加载完成时，异步补齐
    loadOptions().then(() => {
      form.roleIds = roleOptions.value
        .filter((r) => roleNames.includes(r.roleName))
        .map((r) => r.id)
    })
  }
}

async function onSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    if (isEdit.value) {
      await updateUser(editingId.value, {
        realName: form.realName,
        email: form.email,
        phone: form.phone,
        deptId: form.deptId ?? null,
        status: form.status,
        remark: form.remark,
        roleIds: form.roleIds
      })
      ElMessage.success('修改成功')
    } else {
      await createUser({
        username: form.username.trim(),
        password: form.password,
        realName: form.realName,
        email: form.email,
        phone: form.phone,
        deptId: form.deptId ?? null,
        remark: form.remark,
        roleIds: form.roleIds
      })
      ElMessage.success('新增成功')
    }
    dialogVisible.value = false
    load()
  } catch (e) {
    // 错误提示由拦截器统一弹出（如账号已存在 409）
  } finally {
    saving.value = false
  }
}

async function onDelete(row) {
  await ElMessageBox.confirm(`确认删除用户「${row.realName || row.username}」？`, '提示', {
    type: 'warning'
  })
  try {
    await deleteUser(row.id)
    ElMessage.success('删除成功')
    load()
  } catch (e) {
    // 统一提示
  }
}

function openReset(row) {
  resetTarget.value = row
  resetPwd.value = ''
  resetVisible.value = true
}

async function onResetSubmit() {
  if (resetPwd.value.length < 6) {
    ElMessage.warning('新口令至少 6 位')
    return
  }
  saving.value = true
  try {
    await resetPassword(resetTarget.value.id, resetPwd.value)
    ElMessage.success('口令已重置')
    resetVisible.value = false
  } catch (e) {
    // 统一提示
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadOptions()
  load()
})
</script>

<style scoped>
.search-card {
  margin-bottom: 16px;
}

.pager {
  margin-top: 16px;
  justify-content: flex-end;
}

.role-tag {
  margin-right: 4px;
}

.empty-text {
  color: #c0c4cc;
  font-size: 12px;
}
</style>
