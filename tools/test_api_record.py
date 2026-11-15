# -*- coding: utf-8 -*-
"""接口测试执行脚本：对 industrial_admin 后端逐条发送真实请求，记录实际结果。

输出 test_results.json（每条用例的请求、预期、实际摘要、判定），供生成测试记录表使用。
"""
import json
import os
import urllib.error
import urllib.request

BASE = 'http://127.0.0.1:18080/api/v1'
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'test_results.json')

RESULTS = []


def call(method, path, token=None, body=None, raw_auth=None):
    url = BASE + path
    data = json.dumps(body, ensure_ascii=False).encode('utf-8') if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    if data is not None:
        req.add_header('Content-Type', 'application/json')
    if raw_auth:
        req.add_header('Authorization', raw_auth)
    elif token:
        req.add_header('Authorization', 'Bearer ' + token)
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return r.status, json.loads(r.read().decode('utf-8'))
    except urllib.error.HTTPError as e:
        txt = e.read().decode('utf-8')
        try:
            return e.code, json.loads(txt)
        except Exception:
            return e.code, {'raw': txt}
    except Exception as e:  # 连接失败等
        return 0, {'raw': repr(e)}


def add(module, api, method, path, inp, expect, status, resp, check):
    """check: (bool_ok, actual_summary)"""
    ok, summary = check(status, resp)
    RESULTS.append({
        'module': module,
        'api': api,
        'method': method,
        'path': path,
        'input': inp,
        'expect': expect,
        'http_status': status,
        'actual': summary,
        'pass': bool(ok),
    })


def code_of(resp):
    return resp.get('code')


# ---------- 登录 ----------
module = '登录鉴权'

s, r = call('POST', '/auth/login', body={'username': 'admin', 'password': 'Admin@123'})
admin_token = (r.get('data') or {}).get('token') or ''
add(module, '用户登录', 'POST', '/auth/login',
    '{"username":"admin","password":"Admin@123"}',
    'code=0，返回 token 与用户信息',
    s, r, lambda st, rs: (code_of(rs) == 0 and bool(admin_token),
                          'HTTP %d，code=%s，token 长度 %d，user.realName=%s' % (
                              st, code_of(rs), len(admin_token),
                              ((rs.get('data') or {}).get('user') or {}).get('realName'))))

s, r = call('POST', '/auth/login', body={'username': 'admin', 'password': 'wrong-pass'})
add(module, '用户登录', 'POST', '/auth/login',
    'admin / wrong-pass（错误口令）',
    'code=40100，提示“账号或口令错误”，不返回 token',
    s, r, lambda st, rs: (code_of(rs) == 40100,
                          'HTTP %d，code=%s，message=%s，无 token 返回=%s' % (
                              st, code_of(rs), rs.get('message'), 'token' not in str(rs.get('data')))))

s, r = call('POST', '/auth/login', body={'username': 'no_such_user', 'password': 'Any@1234'})
add(module, '用户登录', 'POST', '/auth/login',
    '不存在的账号 no_such_user',
    'code=40100，账号不存在（不暴露账号是否存在细节）',
    s, r, lambda st, rs: (code_of(rs) == 40100,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('POST', '/auth/login', body={'username': '', 'password': 'Admin@123'})
add(module, '登录参数校验', 'POST', '/auth/login',
    '{"username":"","password":"Admin@123"}（账号为空）',
    'code=40001，参数校验失败',
    s, r, lambda st, rs: (code_of(rs) == 40001,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('POST', '/auth/login', body={'username': 'a' * 60, 'password': 'Admin@123'})
add(module, '登录参数校验', 'POST', '/auth/login',
    '账号 60 字符（超出 50 上限）',
    'code=40001，账号长度超限被拒',
    s, r, lambda st, rs: (code_of(rs) == 40001,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('GET', '/auth/profile', token=admin_token)
add(module, '登录鉴权', 'GET', '/auth/profile',
    'Authorization: Bearer <admin token>',
    'code=0，返回当前用户信息与权限点',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，username=%s，权限点 %d 个' % (
                              st, code_of(rs), ((rs.get('data') or {}).get('user') or {}).get('username'),
                              len((rs.get('data') or {}).get('permissions') or []))))

s, r = call('GET', '/users')
add(module, '登录鉴权', 'GET', '/users',
    '不带任何令牌',
    'code=40100，HTTP 401，提示未登录',
    s, r, lambda st, rs: (st == 401 and code_of(rs) == 40100,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('GET', '/users', raw_auth='Bearer not-a-real-token')
add(module, '登录鉴权', 'GET', '/users',
    'Authorization: Bearer not-a-real-token',
    'code=40100，令牌非法被拒',
    s, r, lambda st, rs: (st == 401 and code_of(rs) == 40100,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

# ---------- 用户管理 ----------
module = '用户管理'

s, r = call('GET', '/users?page=1&pageSize=10', token=admin_token)
add(module, '用户列表', 'GET', '/users?page=1&pageSize=10',
    'page=1, pageSize=10',
    'code=0，返回分页结构 {list,total,page,pageSize}',
    s, r, lambda st, rs: (code_of(rs) == 0 and 'list' in (rs.get('data') or {}),
                          'HTTP %d，code=%s，total=%s，当前页 %d 条' % (
                              st, code_of(rs), (rs.get('data') or {}).get('total'),
                              len((rs.get('data') or {}).get('list') or []))))

s, r = call('GET', '/users?page=1&pageSize=10&keyword=test', token=admin_token)
add(module, '用户列表-条件查询', 'GET', '/users?keyword=test',
    'keyword=test',
    'code=0，只返回匹配关键字的用户',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，total=%s，命中账号=%s' % (
                              st, code_of(rs), (rs.get('data') or {}).get('total'),
                              [u.get('username') for u in ((rs.get('data') or {}).get('list') or [])])))

new_user = {'username': 'qa_temp01', 'password': 'Qa@12345', 'realName': '测试临时用户',
            'email': 'qa_temp01@example.com', 'phone': '13000000001', 'deptId': 3,
            'remark': '接口测试，用后即删', 'roleIds': [6]}
s, r = call('POST', '/users', token=admin_token, body=new_user)
temp_id = (r.get('data') or {}).get('id')
add(module, '新增用户', 'POST', '/users',
    '{"username":"qa_temp01","password":"Qa@12345","realName":"测试临时用户","deptId":3,"roleIds":[6]}',
    'code=0，返回新用户 id，口令以 bcrypt 入库',
    s, r, lambda st, rs: (code_of(rs) == 0 and temp_id,
                          'HTTP %d，code=%s，新建 id=%s' % (st, code_of(rs), temp_id)))

s, r = call('POST', '/users', token=admin_token, body=new_user)
add(module, '新增用户-重复账号', 'POST', '/users',
    '同上，再次提交同一 username',
    'code=40900，账号重复被拒',
    s, r, lambda st, rs: (code_of(rs) == 40900,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('POST', '/users', token=admin_token,
            body={'username': 'qa_temp02', 'password': 'Qa@12345'})
add(module, '新增用户-缺参校验', 'POST', '/users',
    '缺少 realName 必填项',
    'code=40001，提示账号、口令、姓名为必填',
    s, r, lambda st, rs: (code_of(rs) == 40001,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('PUT', '/users/%s' % temp_id, token=admin_token,
            body={'realName': '测试临时用户改', 'email': 'qa_temp01b@example.com',
                  'phone': '13000000001', 'deptId': 3, 'status': 1,
                  'remark': '接口测试-已修改', 'roleIds': [6]})
add(module, '编辑用户', 'PUT', '/users/%s' % temp_id,
    '{"realName":"测试临时用户改","status":1,"roleIds":[6]}',
    'code=0，修改成功',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('POST', '/users/%s/password' % temp_id, token=admin_token, body={'password': 'Qa@54321'})
add(module, '重置口令', 'POST', '/users/%s/password' % temp_id,
    '{"password":"Qa@54321"}',
    'code=0，口令被重置并重新哈希',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('POST', '/auth/login', body={'username': 'qa_temp01', 'password': 'Qa@54321'})
add(module, '重置口令-登录验证', 'POST', '/auth/login',
    '用重置后的新口令登录 qa_temp01',
    'code=0，可用新口令登录',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，新口令登录成功' % (st, code_of(rs))))

s, r = call('DELETE', '/users/1', token=admin_token)
add(module, '删除用户-内置管理员', 'DELETE', '/users/1',
    'DELETE /users/1（admin）',
    'code=40300，内置管理员不可删除',
    s, r, lambda st, rs: (code_of(rs) == 40300,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('DELETE', '/users/%s' % temp_id, token=admin_token)
add(module, '删除用户', 'DELETE', '/users/%s' % temp_id,
    'DELETE 临时用户',
    'code=0，删除成功（含 user_role 关联清理）',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('GET', '/users?keyword=qa_temp01', token=admin_token)
add(module, '删除用户-落库校验', 'GET', '/users?keyword=qa_temp01',
    '删除后按关键字回查',
    'code=0，total=0，列表为空',
    s, r, lambda st, rs: (code_of(rs) == 0 and (rs.get('data') or {}).get('total') == 0,
                          'HTTP %d，code=%s，total=%s' % (st, code_of(rs), (rs.get('data') or {}).get('total'))))

# 普通员工令牌（用于越权测试）
s, r = call('POST', '/auth/login', body={'username': 'test01', 'password': 'Test@1234'})
emp_token = (r.get('data') or {}).get('token') or ''

s, r = call('POST', '/users', token=emp_token,
            body={'username': 'qa_temp03', 'password': 'Qa@12345', 'realName': '越权测试'})
add(module, '新增用户-越权', 'POST', '/users',
    '普通员工令牌新增用户',
    'code=40300，HTTP 403，权限不足',
    s, r, lambda st, rs: (st == 403 and code_of(rs) == 40300,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

# ---------- 角色授权 ----------
module = '角色授权'

s, r = call('GET', '/roles?page=1&pageSize=10', token=admin_token)
role_total = (r.get('data') or {}).get('total')
add(module, '角色列表', 'GET', '/roles?page=1&pageSize=10',
    'page=1, pageSize=10',
    'code=0，返回 6 个角色及分页信息',
    s, r, lambda st, rs: (code_of(rs) == 0 and role_total == 6,
                          'HTTP %d，code=%s，total=%s' % (st, code_of(rs), role_total)))

s, r = call('GET', '/permissions?page=1&pageSize=200', token=admin_token)
perm_total = (r.get('data') or {}).get('total')
add(module, '权限点列表', 'GET', '/permissions?page=1&pageSize=200',
    'page=1, pageSize=200',
    'code=0，返回全部权限点（菜单/按钮/接口）',
    s, r, lambda st, rs: (code_of(rs) == 0 and perm_total >= 20,
                          'HTTP %d，code=%s，total=%s' % (st, code_of(rs), perm_total)))

s, r = call('GET', '/roles/6/permissions', token=admin_token)
owned = (r.get('data') or [])
add(module, '角色已授权查询', 'GET', '/roles/6/permissions',
    'roleId=6（普通员工）',
    'code=0，返回该角色已有的权限点 id 列表',
    s, r, lambda st, rs: (code_of(rs) == 0 and isinstance(rs.get('data'), list),
                          'HTTP %d，code=%s，已授权 %s' % (st, code_of(rs), rs.get('data'))))

s, r = call('PUT', '/roles/6/permissions', token=admin_token, body={'permIds': [9999]})
s2, r2 = call('GET', '/roles/6/permissions', token=admin_token)
after_bad = (r2.get('data') or []) if code_of(r2) == 0 else None
add(module, '角色授权-非法权限点', 'PUT', '/roles/6/permissions',
    '{"permIds":[9999]}（不存在的权限点）',
    'code=40001，参数非法被拒，事务回滚，角色权限保持原值',
    s, r, lambda st, rs: (code_of(rs) == 40001 and after_bad == owned,
                          'HTTP %d，code=%s，message=%s；回查仍为 %s（与授权前一致）' % (
                              st, code_of(rs), rs.get('message'), after_bad)))

keep = sorted(owned)
s, r = call('PUT', '/roles/6/permissions', token=admin_token, body={'permIds': keep})
add(module, '角色授权-整体替换', 'PUT', '/roles/6/permissions',
    '{"permIds":%s}（原集合，验证替换事务）' % json.dumps(keep),
    'code=0，权限集合被整体替换',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('GET', '/roles/6/permissions', token=admin_token)
add(module, '角色授权-回查一致', 'GET', '/roles/6/permissions',
    '授权后再查一次',
    'code=0，返回集合与提交内容一致',
    s, r, lambda st, rs: (code_of(rs) == 0 and sorted(rs.get('data') or []) == keep,
                          'HTTP %d，code=%s，返回 %s，与提交一致=%s' % (
                              st, code_of(rs), rs.get('data'), sorted(rs.get('data') or []) == keep)))

s, r = call('PUT', '/roles/1/permissions', token=admin_token, body={'permIds': [1]})
add(module, '角色授权-内置角色', 'PUT', '/roles/1/permissions',
    '对内置角色 super_admin 授权',
    'code=403/400，内置角色禁止修改',
    s, r, lambda st, rs: (code_of(rs) in (400, 403, 40300) and code_of(rs) != 0,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('GET', '/roles/6/permissions', token=emp_token)
add(module, '角色授权-越权', 'GET', '/roles/6/permissions',
    '普通员工令牌访问授权接口',
    'code=40300，HTTP 403',
    s, r, lambda st, rs: (st == 403 and code_of(rs) == 40300,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

# ---------- 部门管理 ----------
module = '部门管理'

s, r = call('GET', '/depts/tree', token=admin_token)
add(module, '部门树查询', 'GET', '/depts/tree',
    '无参数',
    'code=0，返回递归 children 的部门树',
    s, r, lambda st, rs: (code_of(rs) == 0 and isinstance(rs.get('data'), list),
                          'HTTP %d，code=%s，根节点 %d 个，子节点数=%s' % (
                              st, code_of(rs), len(rs.get('data') or []),
                             [len(x.get('children') or []) for x in (rs.get('data') or [])])))

s, r = call('GET', '/depts?page=1&pageSize=10', token=admin_token)
add(module, '部门列表', 'GET', '/depts?page=1&pageSize=10',
    'page=1, pageSize=10',
    'code=0，返回分页列表',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，total=%s' % (st, code_of(rs), (rs.get('data') or {}).get('total'))))

import datetime

STAMP = datetime.datetime.now().strftime('%H%M%S')
DEPT_CODE = 'QA_TMP_' + STAMP

new_dept = {'parentId': 1, 'deptName': '接口测试部' + STAMP, 'deptCode': DEPT_CODE, 'leader': '测试员',
            'phone': '13000000002', 'sortNo': 99, 'status': 1}
s, r = call('POST', '/depts', token=admin_token, body=new_dept)
dept_id = (r.get('data') or {}).get('id')
add(module, '新增部门', 'POST', '/depts',
    '{"parentId":1,"deptName":"接口测试部%s","deptCode":"%s"}' % (STAMP, DEPT_CODE),
    'code=0，返回新部门 id',
    s, r, lambda st, rs: (code_of(rs) == 0 and dept_id,
                          'HTTP %d，code=%s，新建 id=%s' % (st, code_of(rs), dept_id)))

s, r = call('POST', '/depts', token=admin_token, body=new_dept)
add(module, '新增部门-编码重复', 'POST', '/depts',
    '重复提交同一 deptCode=%s' % DEPT_CODE,
    'code=40900，编码唯一校验生效',
    s, r, lambda st, rs: (code_of(rs) == 40900,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('PUT', '/depts/%s' % dept_id, token=admin_token,
            body={'parentId': dept_id, 'deptName': '接口测试部' + STAMP, 'deptCode': DEPT_CODE,
                  'leader': '测试员', 'phone': '13000000002', 'sortNo': 99, 'status': 1})
add(module, '编辑部门-自身上级', 'PUT', '/depts/%s' % dept_id,
    'parentId 指向自身',
    'code=40001，禁止把上级设为自己',
    s, r, lambda st, rs: (code_of(rs) == 40001,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('PUT', '/depts/1', token=admin_token,
            body={'parentId': dept_id, 'deptName': '总公司', 'deptCode': 'HQ',
                  'leader': '管理员', 'phone': '13000000000', 'sortNo': 1, 'status': 1})
add(module, '编辑部门-成环校验', 'PUT', '/depts/1',
    '把根节点上级设为它的子部门（成环）',
    'code=40001，检测到循环引用并拒绝',
    s, r, lambda st, rs: (code_of(rs) == 40001,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('DELETE', '/depts/%s' % dept_id, token=admin_token)
add(module, '删除部门', 'DELETE', '/depts/%s' % dept_id,
    'DELETE 临时部门',
    'code=0，删除成功',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('DELETE', '/depts/1', token=admin_token)
add(module, '删除部门-有子节点', 'DELETE', '/depts/1',
    'DELETE 根部门（下含子部门）',
    'code=40900，存在下级时禁止删除',
    s, r, lambda st, rs: (code_of(rs) == 40900,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

s, r = call('GET', '/configs', token=admin_token)
add(module, '参数读取', 'GET', '/configs',
    '无参数',
    'code=0，返回 10 条系统参数',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，条数=%d' % (
                              st, code_of(rs), len(rs.get('data') or []) if isinstance(rs.get('data'), list) else -1)))

# ---------- 日志查询 ----------
module = '日志查询'

s, r = call('GET', '/logs?page=1&pageSize=10', token=admin_token)
log_total = (r.get('data') or {}).get('total')
add(module, '操作日志分页', 'GET', '/logs?page=1&pageSize=10',
    'page=1, pageSize=10',
    'code=0，返回分页日志，total>0',
    s, r, lambda st, rs: (code_of(rs) == 0 and (log_total or 0) > 0,
                          'HTTP %d，code=%s，total=%s，本页 %d 条' % (
                              st, code_of(rs), log_total, len((rs.get('data') or {}).get('list') or []))))

s, r = call('GET', '/logs?page=1&pageSize=10&module=user', token=admin_token)
add(module, '操作日志-条件查询', 'GET', '/logs?module=user',
    'module=user（按模块过滤）',
    'code=0，只返回用户模块日志',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，total=%s' % (st, code_of(rs), (rs.get('data') or {}).get('total'))))

s, r = call('GET', '/logs?page=1&pageSize=10&result=0', token=admin_token)
add(module, '操作日志-结果过滤', 'GET', '/logs?result=0',
    'result=0（只查失败操作）',
    'code=0，返回失败日志（含登录失败记录）',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，total=%s' % (st, code_of(rs), (rs.get('data') or {}).get('total'))))

s, r = call('GET', '/logs', token=emp_token)
add(module, '操作日志-越权', 'GET', '/logs',
    '普通员工令牌访问日志查询',
    'code=40300，HTTP 403',
    s, r, lambda st, rs: (st == 403 and code_of(rs) == 40300,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

# ---------- 统计看板 ----------
module = '统计看板'

s, r = call('GET', '/dashboard/overview', token=admin_token)
add(module, '看板概览', 'GET', '/dashboard/overview',
    '无参数',
    'code=0，返回用户总数/活跃/操作数等指标',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，userTotal=%s，todayOps=%s' % (
                              st, code_of(rs), (rs.get('data') or {}).get('userTotal'),
                              (rs.get('data') or {}).get('todayOps'))))

s, r = call('GET', '/stats/users/growth?days=7&granularity=day', token=admin_token)
add(module, '用户增长统计', 'GET', '/stats/users/growth?days=7',
    'days=7, granularity=day',
    'code=0，返回 7 个日期点与新增/累计数组',
    s, r, lambda st, rs: (code_of(rs) == 0 and len((rs.get('data') or {}).get('xAxis') or []) == 7,
                          'HTTP %d，code=%s，xAxis 点数=%d，sumNew=%s' % (
                              st, code_of(rs), len((rs.get('data') or {}).get('xAxis') or []),
                              (rs.get('data') or {}).get('sumNew'))))

s, r = call('GET', '/stats/users/growth?days=999', token=admin_token)
add(module, '用户增长统计-窗口上限', 'GET', '/stats/users/growth?days=999',
    'days=999（超过上限 90）',
    'code=0，窗口回落为 90 天，返回 90 个点',
    s, r, lambda st, rs: (code_of(rs) == 0 and len((rs.get('data') or {}).get('xAxis') or []) == 90,
                          'HTTP %d，code=%s，点数=%d（回落 90）' % (
                              st, code_of(rs), len((rs.get('data') or {}).get('xAxis') or []))))

s, r = call('GET', '/stats/users/growth?days=abc', token=admin_token)
add(module, '用户增长统计-非法参数', 'GET', '/stats/users/growth?days=abc',
    'days=abc（非数字）',
    'code=0，非法参数回落默认 30 天',
    s, r, lambda st, rs: (code_of(rs) == 0 and len((rs.get('data') or {}).get('xAxis') or []) == 30,
                          'HTTP %d，code=%s，点数=%d（回落 30）' % (
                              st, code_of(rs), len((rs.get('data') or {}).get('xAxis') or []))))

s, r = call('GET', '/stats/users/activity?days=7', token=admin_token)
add(module, '用户活跃度统计', 'GET', '/stats/users/activity?days=7',
    'days=7',
    'code=0，返回 activeUsers/loginCount/operationCount 三条曲线',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，点数=%d，今日活跃=%s' % (
                              st, code_of(rs), len((rs.get('data') or {}).get('xAxis') or []),
                              ((rs.get('data') or {}).get('activeUsers') or [None])[-1])))

s, r = call('GET', '/stats/logs/frequency?days=7', token=admin_token)
add(module, '日志频次统计', 'GET', '/stats/logs/frequency?days=7',
    'days=7',
    'code=0，返回 total/success/failed 与模块分布',
    s, r, lambda st, rs: (code_of(rs) == 0,
                          'HTTP %d，code=%s，点数=%d，模块分布 %d 项' % (
                              st, code_of(rs), len((rs.get('data') or {}).get('xAxis') or []),
                              len((rs.get('data') or {}).get('moduleDist') or []))))

s, r = call('GET', '/stats/users/growth?days=7', token=emp_token)
add(module, '统计看板-越权', 'GET', '/stats/users/growth?days=7',
    '普通员工令牌访问统计接口',
    'code=40300，HTTP 403',
    s, r, lambda st, rs: (st == 403 and code_of(rs) == 40300,
                          'HTTP %d，code=%s，message=%s' % (st, code_of(rs), rs.get('message'))))

with open(OUT, 'w', encoding='utf-8') as f:
    json.dump(RESULTS, f, ensure_ascii=False, indent=2)

total = len(RESULTS)
passed = sum(1 for x in RESULTS if x['pass'])
print('cases:', total, 'passed:', passed, 'failed:', total - passed)
for x in RESULTS:
    if not x['pass']:
        print('FAIL:', x['module'], x['api'], x['http_status'], x['actual'])
