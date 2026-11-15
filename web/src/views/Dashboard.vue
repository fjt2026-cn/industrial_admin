<template>
  <div>
    <el-row :gutter="16" class="cards">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="card-label">用户总数</div>
          <div class="card-value">{{ overview.userTotal }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="card-label">今日活跃</div>
          <div class="card-value">{{ overview.todayActive }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="card-label">今日操作</div>
          <div class="card-value">{{ overview.todayOps }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="card-label">失败操作</div>
          <div class="card-value danger">{{ overview.failOps }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <template #header>
        <div class="panel-head">
          <span>运营数据趋势</span>
          <el-radio-group v-model="days" size="small" @change="loadAll">
            <el-radio-button :value="7">近 7 天</el-radio-button>
            <el-radio-button :value="30">近 30 天</el-radio-button>
          </el-radio-group>
        </div>
      </template>

      <div class="chart-title">用户增长（柱：当日新增 / 线：累计用户）</div>
      <div ref="growthRef" class="chart"></div>

      <div class="chart-title">用户活跃度（活跃用户、登录次数、操作总量）</div>
      <div ref="activityRef" class="chart"></div>
    </el-card>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import * as echarts from 'echarts'
import { getOverview } from '@/api/dashboard'
import { getUserActivity, getUserGrowth } from '@/api/stats'

const days = ref(7)
const overview = reactive({ userTotal: 0, todayActive: 0, todayOps: 0, failOps: 0 })

const growthRef = ref(null)
const activityRef = ref(null)
let growthChart = null
let activityChart = null

// 概览卡片
async function loadOverview() {
  try {
    const data = await getOverview({ days: days.value })
    Object.assign(overview, {
      userTotal: data.userTotal,
      todayActive: data.todayActive,
      todayOps: data.todayOps,
      failOps: data.failOps
    })
  } catch (e) {
    // 错误提示由 axios 拦截器统一处理
  }
}

// 用户增长：柱状图（当天新增）+ 折线图（累计用户），共用同一 X 轴
async function loadGrowth() {
  const data = await getUserGrowth({ days: days.value, granularity: 'day' })
  growthChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['新增用户', '累计用户'] },
    grid: { left: 48, right: 24, top: 48, bottom: 32 },
    xAxis: { type: 'category', data: data.xAxis, axisLabel: { rotate: days.value > 15 ? 45 : 0 } },
    yAxis: [
      { type: 'value', name: '新增', minInterval: 1 },
      { type: 'value', name: '累计', minInterval: 1 }
    ],
    series: [
      {
        name: '新增用户',
        type: 'bar',
        barMaxWidth: 22,
        data: data.newUsers,
        itemStyle: { color: '#4E79A7' }
      },
      {
        name: '累计用户',
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        symbol: 'circle',
        symbolSize: 5,
        data: data.totalUsers,
        itemStyle: { color: '#F28E2B' }
      }
    ]
  })
}

// 用户活跃度：三条折线，分别对应去重活跃用户、登录次数与操作总量
async function loadActivity() {
  const data = await getUserActivity({ days: days.value })
  activityChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['活跃用户', '登录次数', '操作总量'] },
    grid: { left: 48, right: 24, top: 48, bottom: 32 },
    xAxis: { type: 'category', data: data.xAxis, axisLabel: { rotate: days.value > 15 ? 45 : 0 } },
    yAxis: { type: 'value', name: '次数', minInterval: 1 },
    series: [
      { name: '活跃用户', type: 'line', smooth: true, data: data.activeUsers, itemStyle: { color: '#59A14F' } },
      { name: '登录次数', type: 'line', smooth: true, data: data.loginCount, itemStyle: { color: '#4E79A7' } },
      { name: '操作总量', type: 'line', smooth: true, data: data.operationCount, itemStyle: { color: '#E15759' } }
    ]
  })
}

async function loadAll() {
  loadOverview()
  try {
    await Promise.all([loadGrowth(), loadActivity()])
  } catch (e) {
    // 接口失败时保留上一次图表
  }
}

// 容器尺寸变化时重算，避免侧边栏收起后图表被拉伸
function onResize() {
  growthChart?.resize()
  activityChart?.resize()
}

onMounted(() => {
  growthChart = echarts.init(growthRef.value)
  activityChart = echarts.init(activityRef.value)
  window.addEventListener('resize', onResize)
  loadAll()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  growthChart?.dispose()
  activityChart?.dispose()
})
</script>

<style scoped>
.cards {
  margin-bottom: 16px;
}

.card-label {
  font-size: 12px;
  color: #909399;
}

.card-value {
  font-size: 26px;
  font-weight: 600;
  color: #303133;
  margin-top: 8px;
}

.card-value.danger {
  color: #f56c6c;
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.chart-title {
  font-size: 13px;
  color: #606266;
  margin: 8px 0;
}

.chart {
  width: 100%;
  height: 320px;
}
</style>
