<template>
  <el-container class="layout">
    <!-- Header -->
    <el-header class="header">
      <div class="header-left">
        <div class="logo">
          <span class="logo-icon">◐</span>
          <span class="logo-text">SysMonitor</span>
        </div>
      </div>

      <div class="header-right">
        <div class="icon-notify" :class="{ 'has-unread': hasUnread }" @click="onNotifyClick">
          <svg viewBox="0 0 24 24" width="18" height="18"><path d="M12 22c1.1 0 2-.9 2-2h-4a2 2 0 0 0 2 2zM18 16v-5c0-3.07-1.63-5.64-4.5-6.32V4a1.5 1.5 0 0 0-3 0v.68C7.63 5.36 6 7.92 6 11v5l-1.99 2H20l-2-2z" fill="currentColor"/></svg>
          <span v-if="hasUnread" class="dot"></span>
        </div>

        <div class="icon-user">
          <img src="" alt="avatar" v-if="false" />
          <svg viewBox="0 0 24 24" width="18" height="18"><path d="M12 12c2.7 0 5-2.3 5-5s-2.3-5-5-5-5 2.3-5 5 2.3 5 5 5zm0 2c-3.3 0-10 1.7-10 5v3h20v-3c0-3.3-6.7-5-10-5z" fill="currentColor"/></svg>
        </div>
      </div>
    </el-header>

    <el-container class="body">
      <!-- Left Sidebar -->
      <el-aside :width="collapsed ? '64px' : '220px'" class="aside" :class="{ collapsed }">
        <div class="sidebar-top">
          <div class="module-title" v-if="!collapsed">监测模块</div>
        </div>

        <nav class="menu">
          <div v-for="(m, idx) in menu" :key="m.key"
               :class="['menu-item', { active: active === m.key }]"
               @click="active = m.key"
          >
            <div class="menu-left">
              <svg viewBox="0 0 24 24" width="20" height="20"><path :d="m.icon" fill="currentColor"/></svg>
            </div>
            <div class="menu-text" v-if="!collapsed">{{ m.title }}</div>
            <div class="menu-indicator" v-if="active === m.key && !collapsed"></div>
          </div>
        </nav>

        <div class="collapse-toggle" @click="toggleCollapse">
          <svg v-if="!collapsed" viewBox="0 0 24 24" width="14" height="14"><path d="M8 5v14l11-7z" fill="currentColor"/></svg>
          <svg v-else viewBox="0 0 24 24" width="14" height="14"><path d="M16 19V5L5 12z" fill="currentColor"/></svg>
        </div>
      </el-aside>

      <!-- Main Content -->
      <el-main class="main">
        <!-- 设备监测总览 -->
        <template v-if="active === 'device'">
          <!-- KPI row (2 items with ring charts) -->
          <section class="kpi-row-2">
            <div class="kpi-large card" v-for="(k, i) in kpisLarge" :key="i">
              <div class="kpi-large-title">{{ k.title }}</div>
              <div class="kpi-large-body">
                <div class="gauge" :ref="k.ref" style="width:160px;height:120px;"></div>
                <div class="kpi-large-meta">
                  <div class="value-big">{{ formatSig(k.value) }}%</div>
                  <div class="meta-sub">实时</div>
                </div>
              </div>
            </div>
          </section>

          <!-- Flow -->
          <section class="charts flow-section">
            <div class="chart-left card flow-card" style="width:100%;">
              <div class="chart-header">
                <div class="chart-title">流量监测（最近 120s）</div>
                <div class="flow-meta">
                  <div class="flow-meta-item">
                    <span>当前流量</span>
                    <strong>{{ formatSig(currentFlow) }} KB/s</strong>
                  </div>
                  <div class="flow-meta-item">
                    <span>峰值</span>
                    <strong>{{ formatSig(flowPeak) }} KB/s</strong>
                  </div>
                </div>
              </div>
              <div ref="flowChartRef" class="chart-area flow-chart"></div>
            </div>
          </section>
        </template>

        <!-- 性能统计 -->
        <template v-else-if="active === 'perf'">
          <section class="perf-section card">
            <div class="perf-title">性能统计</div>
            <div class="perf-grid">
              <div class="perf-block">
                <h4>CPU</h4>
                <div class="perf-row"><span>使用率</span><strong>{{ formatSig(dashboard.cpu.usage) }}%</strong></div>
                <div class="perf-row"><span>1 / 5 / 15 分钟负载</span>
                  <strong>{{ formatSig(dashboard.cpu.load1) }} / {{ formatSig(dashboard.cpu.load5) }} / {{ formatSig(dashboard.cpu.load15) }}</strong>
                </div>
                <div class="perf-row"><span>核心数</span><strong>{{ dashboard.cpu.cores || '-' }}</strong></div>
                <div class="perf-row"><span>温度</span>
                  <strong>{{ dashboard.perf && dashboard.perf.cpu_temp ? formatSig(dashboard.perf.cpu_temp) + ' °C' : '-' }}</strong>
                </div>
              </div>

              <div class="perf-block">
                <h4>内存与交换</h4>
                <div class="perf-row"><span>内存使用率</span><strong>{{ formatSig(dashboard.memory.used_percent) }}%</strong></div>
                <div class="perf-row"><span>已用 / 总计</span>
                  <strong>{{ formatSig(dashboard.memory.used / 1024 / 1024 / 1024) }} / {{ formatSig(dashboard.memory.total / 1024 / 1024 / 1024) }} GB</strong>
                </div>
                <div class="perf-row"><span>Swap 已用 / 总计</span>
                  <strong>{{ formatSig(dashboard.perf ? dashboard.perf.swap_used / 1024 / 1024 / 1024 : 0) }} / {{ formatSig(dashboard.perf ? dashboard.perf.swap_total / 1024 / 1024 / 1024 : 0) }} GB</strong>
                </div>
              </div>

              <div class="perf-block">
                <h4>磁盘 IO</h4>
                <div class="perf-row"><span>总读速</span>
                  <strong>{{ dashboard.perf ? formatSig(dashboard.perf.disk_read_kbps) : '-' }} KB/s</strong>
                </div>
                <div class="perf-row"><span>总写速</span>
                  <strong>{{ dashboard.perf ? formatSig(dashboard.perf.disk_write_kbps) : '-' }} KB/s</strong>
                </div>
              </div>

              <div class="perf-block">
                <h4>网络</h4>
                <div class="perf-row"><span>总下行</span>
                  <strong>{{ dashboard.perf ? formatSig(dashboard.perf.net_rx_kbps) : '-' }} KB/s</strong>
                </div>
                <div class="perf-row"><span>总上行</span>
                  <strong>{{ dashboard.perf ? formatSig(dashboard.perf.net_tx_kbps) : '-' }} KB/s</strong>
                </div>
              </div>
            </div>

            <!-- 网络日志 -->
            <div class="netlog-block">
              <div class="netlog-header">
                <span>网络流量日志</span>
                <span class="netlog-sub">最近 {{ dashboard.net_log?.length || 0 }} 条 · 单位 KB/s</span>
              </div>
              <div class="netlog-table">
                <div class="netlog-row netlog-row--head">
                  <span>时间</span>
                  <span>下行 Rx</span>
                  <span>上行 Tx</span>
                  <span>详细接口</span>
                </div>
                <div
                  v-for="(item, idx) in (dashboard.net_log || []).slice().reverse()"
                  :key="idx"
                  class="netlog-row"
                >
                  <span>{{ item.time }}</span>
                  <span>{{ formatSig(item.rx) }}</span>
                  <span>{{ formatSig(item.tx) }}</span>
                  <span class="netlog-detail">
                    <template v-if="item.interfaces && item.interfaces.length">
                      <span
                        v-for="(ni, j) in item.interfaces.slice(0,3)"
                        :key="j"
                        class="netlog-if"
                      >
                        {{ ni.interface }}: ↓{{ formatSig(ni.rx) }} / ↑{{ formatSig(ni.tx) }}
                      </span>
                      <span v-if="item.interfaces.length > 3" class="netlog-more">
                        … 等 {{ item.interfaces.length }} 个接口
                      </span>
                    </template>
                    <template v-else>
                      -
                    </template>
                  </span>
                </div>
                <div v-if="!dashboard.net_log || dashboard.net_log.length === 0" class="netlog-empty">
                  暂无网络日志
                </div>
              </div>
            </div>
          </section>
        </template>

        <!-- 告警中心（日志形式） -->
        <template v-else-if="active === 'alert'">
          <section class="alert-center card">
            <div class="quick-title">告警中心</div>
            <div v-if="alertsHistory.length === 0" class="alert-empty">暂无历史告警</div>
            <div v-else class="alert-list">
              <div v-for="(a, i) in alertsHistory" :key="i" class="alert-item card-alert">
                <div class="status" :style="{ background: statusColor(a.level) }"></div>
                <div class="alert-body">
                  <div class="alert-text">{{ a.text }}</div>
                  <div class="alert-time">{{ a.time }}</div>
                </div>
              </div>
            </div>
          </section>
        </template>
      </el-main>

      <!-- Right Quick Monitor -->
      <el-aside width="300px" class="quick">
        <div class="quick-card card">
          <div class="quick-title">实时告警</div>
          <div class="alerts">
            <div v-if="currentAlerts.length===0" class="alert-empty">当前无告警</div>
            <div v-for="(a, i) in currentAlerts" :key="i" class="alert-item card-alert" @click="openAlert(a)">
              <div class="status" :style="{ background: statusColor(a.level) }"></div>
              <div class="alert-body">
                <div class="alert-text">{{ a.text }}</div>
                <div class="alert-time">{{ a.time }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="quick-card card" style="margin-top:12px;">
          <div class="quick-title">系统信息</div>
          <div class="sys-grid">
            <div class="sys-row"><span>主机名</span><strong>{{ dashboard.system.hostname || '-' }}</strong></div>
            <div class="sys-row"><span>操作系统</span><strong>{{ dashboard.system.os || '-' }}</strong></div>
            <div class="sys-row"><span>平台</span><strong>{{ dashboard.system.platform || '-' }}</strong></div>
            <div class="sys-row"><span>启动时间</span><strong>{{ formatBootTime(dashboard.system.boot_time) }}</strong></div>
          </div>
        </div>
      </el-aside>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, onMounted, reactive, watch, nextTick } from 'vue'
import axios from 'axios'
import * as echarts from 'echarts'

// --- state ---
const collapsed = ref(false)
const active = ref('device')
const menu = [
  { key: 'device', title: '设备监测', icon: 'M3 3h18v18H3z' },
  { key: 'perf', title: '性能统计', icon: 'M12 3L21 21H3L12 3z' },
  { key: 'alert', title: '告警中心', icon: 'M12 22c5.5 0 10-4.5 10-10S17.5 2 12 2 2 6.5 2 12s4.5 10 10 10z' }
]
const hasUnread = ref(false)
// 告警日志 + 当前实时告警
const alertsHistory = ref([])
const currentAlerts = ref([])

const dashboard = reactive({
  cpu: { usage: 0, per_core: [] },
  memory: { used_percent: 0 },
  disk: [],
  network: [],
  system: {},
  perf: {},
  net_log: []
})

// KPI placeholders (will be updated each fetch)
const kpisLarge = reactive([
  { title: 'CPU 使用率', value: 0, delta: 0, ref: 'cpuGauge' },
  { title: '内存占用', value: 0, delta: 0, ref: 'memGauge' }
])

// history arrays
const flowHistory = ref(Array(120).fill(0))
const flowLabels = ref(Array(120).fill(''))
const currentFlow = ref(0)
const flowPeak = ref(0)

// chart instances
let gaugeCpu, gaugeMem, flowChart

// refs
const cpuGauge = ref(null)
const memGauge = ref(null)
const flowChartRef = ref(null)

// helper: format to 2 significant digits (returns string)
function formatSig(v) {
  if (v === null || v === undefined || Number.isNaN(Number(v))) return '-'
  const n = Number(v)
  // use toPrecision(2), then remove trailing zeros if decimal
  const s = n.toPrecision(2)
  // convert to Number to strip unnecessary trailing zeros (but keep decimals when needed)
  const asNum = Number(s)
  return asNum.toString()
}

// format boot time
function formatBootTime(ts) {
  return ts ? new Date(ts * 1000).toLocaleString() : '-'
}

function statusColor(level) {
  if (level === 'ok') return '#36D399'
  if (level === 'warn') return '#FFAB00'
  return '#F87272'
}

function toggleCollapse() {
  collapsed.value = !collapsed.value
}

// open alert (placeholder)
function openAlert(a) {
  // future: show detail modal
  console.log('open alert', a)
}

// fetch data from backend
async function fetchData() {
  try {
    const res = await axios.get('/api/dashboard')
    const d = res.data

    // defensively map backend fields to our dashboard structure
    dashboard.cpu = d.cpu || dashboard.cpu
    dashboard.memory = d.memory || dashboard.memory
    dashboard.disk = d.disk || dashboard.disk
    dashboard.network = d.network || dashboard.network
    dashboard.system = d.system || dashboard.system
    dashboard.perf = d.perf || dashboard.perf
    dashboard.net_log = d.net_log || dashboard.net_log

    kpisLarge[0].value = dashboard.cpu.usage || 0
    kpisLarge[1].value = dashboard.memory.used_percent || 0

    // total network (KB/s)
    const totalNet = (dashboard.network || []).reduce((s, n) => s + (n.rx || 0) + (n.tx || 0), 0)
    const normalizedNet = Number(totalNet)
    flowHistory.value.push(normalizedNet)
    if (flowHistory.value.length > 120) flowHistory.value.shift()
    const timeLabel = new Date().toLocaleTimeString('zh-CN', { hour12: false })
    flowLabels.value.push(timeLabel)
    if (flowLabels.value.length > 120) flowLabels.value.shift()
    currentFlow.value = normalizedNet
    flowPeak.value = Math.max(flowPeak.value, normalizedNet)

    // alerts mapping: d.alerts 为历史日志, d.current_alerts 为当前周期告警
    alertsHistory.value = d.alerts || []
    currentAlerts.value = d.current_alerts || []

    // update charts
    updateGauges()
    updateFlowChart()
  } catch (e) {
    // do not spam console in production
    console.warn('fetch error', e)
  }
}

function initGauges() {
  if (cpuGauge.value) {
    // 如果已存在实例，先销毁，防止重复初始化绑定到旧 DOM
    if (gaugeCpu) {
      gaugeCpu.dispose()
    }
    gaugeCpu = echarts.init(cpuGauge.value)
    gaugeCpu.setOption({
      animationDuration: 800,
      animationDurationUpdate: 800,
      animationEasing: 'cubicOut',
      graphic: [{
        type: 'circle',
        left: 'center',
        top: 'middle',
        shape: { r: 40 },
        style: {
          fill: new echarts.graphic.RadialGradient(0.5, 0.5, 1, [
            { offset: 0, color: 'rgba(22,93,255,0.10)' },
            { offset: 1, color: 'rgba(22,93,255,0.02)' }
          ])
        },
        silent: true,
        z: 0,
        keyframeAnimation: {
          duration: 2400,
          loop: true,
          keyframes: [
            { percent: 0, shape: { r: 38 }, style: { opacity: 0.16 } },
            { percent: 0.5, shape: { r: 42 }, style: { opacity: 0.28 } },
            { percent: 1, shape: { r: 38 }, style: { opacity: 0.16 } }
          ]
        }
      }],
      series: [
        {
          id: 'cpuBase',
          type: 'gauge',
          radius: '92%',
          startAngle: 90,
          endAngle: -269.9999,
          pointer: { show: false },
          progress: { show: true, roundCap: true, itemStyle: { color: 'rgba(22,93,255,0.12)' } },
          axisLine: { lineStyle: { width: 14, color: [[1, '#EEF4FF']] } },
          axisTick: { show: false },
          splitLine: { show: false },
          axisLabel: { show: false },
          title: { show: false },
          detail: { show: false },
          data: [{ value: 100 }],
          animation: false,
          z: 1
        },
        {
          id: 'cpuValue',
          type: 'gauge',
          radius: '90%',
          startAngle: 90,
          endAngle: -269.9999,
          pointer: { show: false },
          anchor: { show: true, size: 8, itemStyle: { color: '#165DFF', shadowBlur: 8, shadowColor: 'rgba(22,93,255,0.35)' } },
          progress: {
            show: true,
            roundCap: true,
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                { offset: 0, color: '#40A9FF' },
                { offset: 1, color: '#165DFF' }
              ])
            }
          },
          axisLine: { lineStyle: { width: 12, color: [[1, '#E6F0FF']] } },
          axisTick: { show: false },
          splitLine: { show: false },
          axisLabel: { show: false },
          title: { show: false },
          detail: { show: false },
          data: [{ value: 0 }],
          z: 2
        }
      ]
    })
  }

  if (memGauge.value) {
    if (gaugeMem) {
      gaugeMem.dispose()
    }
    gaugeMem = echarts.init(memGauge.value)
    gaugeMem.setOption({
      animationDuration: 800,
      animationDurationUpdate: 800,
      animationEasing: 'cubicOut',
      graphic: [{
        type: 'circle',
        left: 'center',
        top: 'middle',
        shape: { r: 40 },
        style: {
          fill: new echarts.graphic.RadialGradient(0.5, 0.5, 1, [
            { offset: 0, color: 'rgba(54,211,153,0.10)' },
            { offset: 1, color: 'rgba(54,211,153,0.02)' }
          ])
        },
        silent: true,
        z: 0,
        keyframeAnimation: {
          duration: 2400,
          loop: true,
          keyframes: [
            { percent: 0, shape: { r: 38 }, style: { opacity: 0.16 } },
            { percent: 0.5, shape: { r: 42 }, style: { opacity: 0.28 } },
            { percent: 1, shape: { r: 38 }, style: { opacity: 0.16 } }
          ]
        }
      }],
      series: [
        {
          id: 'memBase',
          type: 'gauge',
          radius: '92%',
          startAngle: 90,
          endAngle: -269.9999,
          pointer: { show: false },
          progress: { show: true, roundCap: true, itemStyle: { color: 'rgba(54,211,153,0.12)' } },
          axisLine: { lineStyle: { width: 14, color: [[1, '#ECFFF6']] } },
          axisTick: { show: false },
          splitLine: { show: false },
          axisLabel: { show: false },
          title: { show: false },
          detail: { show: false },
          data: [{ value: 100 }],
          animation: false,
          z: 1
        },
        {
          id: 'memValue',
          type: 'gauge',
          radius: '90%',
          startAngle: 90,
          endAngle: -269.9999,
          pointer: { show: false },
          anchor: { show: true, size: 8, itemStyle: { color: '#36D399', shadowBlur: 8, shadowColor: 'rgba(54,211,153,0.35)' } },
          progress: {
            show: true,
            roundCap: true,
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
                { offset: 0, color: '#79E2B5' },
                { offset: 1, color: '#36D399' }
              ])
            }
          },
          axisLine: { lineStyle: { width: 12, color: [[1, '#ECFFF6']] } },
          axisTick: { show: false },
          splitLine: { show: false },
          axisLabel: { show: false },
          title: { show: false },
          detail: { show: false },
          data: [{ value: 0 }],
          z: 2
        }
      ]
    })
  }
}

function updateGauges() {
  if (gaugeCpu) {
    gaugeCpu.setOption({ series: [{ id: 'cpuValue', data: [{ value: Number(kpisLarge[0].value || 0) }] }] })
  }
  if (gaugeMem) {
    gaugeMem.setOption({ series: [{ id: 'memValue', data: [{ value: Number(kpisLarge[1].value || 0) }] }] })
  }
}

function initFlowChart() {
  if (flowChartRef.value) {
    if (flowChart) {
      flowChart.dispose()
    }
    flowChart = echarts.init(flowChartRef.value)
    flowChart.setOption({
      tooltip: {
        trigger: 'axis',
        formatter: params => {
          if (!params || !params.length) return ''
          const p = params[0]
          return `流量: ${formatSig(p.data)} KB/s`
        }
      },
      grid: { left: '4%', right: '2%', bottom: '6%', top: '10%' },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: flowLabels.value,
        axisLine: { lineStyle: { color: '#E5E7EB' } },
        axisLabel: { show: true, color: '#6B7280', interval: 19 },
        axisTick: { show: false }
      },
      yAxis: {
        type: 'value',
        axisLine: { lineStyle: { color: '#E5E7EB' } },
        axisLabel: { color: '#6B7280' },
        splitLine: { lineStyle: { color: '#F3F4F6' } }
      },
      series: [{
        name: 'Traffic',
        type: 'line',
        smooth: true,
        showSymbol: false,
        lineStyle: {
          width: 3,
          color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: '#40A9FF' },
            { offset: 1, color: '#165DFF' }
          ])
        },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(64,169,255,0.35)' },
            { offset: 1, color: 'rgba(22,93,255,0.05)' }
          ])
        },
        data: flowHistory.value,
        animationDuration: 800,
        animationEasing: 'cubicOut'
      }]
    })
  }
}

function updateFlowChart() {
  if (!flowChart) return
  flowChart.setOption({
    xAxis: { data: flowLabels.value },
    series: [{ data: flowHistory.value }]
  })
}

// 重新查找设备监测区域内的图表 DOM，并初始化图表
function setupDeviceCharts() {
  // 通过 class 查找当前 DOM 中的仪表盘容器
  const gauges = document.querySelectorAll('.kpi-large .gauge')
  if (gauges.length >= 2) {
    cpuGauge.value = gauges[0]
    memGauge.value = gauges[1]
  }

  // 初始化 / 重新绑定图表
  initGauges()
  initFlowChart()

  // 切回设备监测时，立即用当前数据刷新一次
  updateGauges()
  updateFlowChart()
}

// init all charts
onMounted(() => {
  setupDeviceCharts()

  // initial fetch + start polling
  fetchData()
  setInterval(fetchData, 1000)

  // window resize -> charts resize
  window.addEventListener('resize', () => {
    gaugeCpu && gaugeCpu.resize && gaugeCpu.resize()
    gaugeMem && gaugeMem.resize && gaugeMem.resize()
    flowChart && flowChart.resize && flowChart.resize()
  })
})

// 当菜单切换回“设备监测”时，重新挂载并刷新图表，避免切换后空白
watch(active, async (val) => {
  if (val === 'device') {
    await nextTick()
    setupDeviceCharts()
  }
})
</script>

<style scoped>
/* color variables - light blue theme */
:root{
  --primary:#165DFF;
  --success:#22C55E;
  --warn:#FACC15;
  --danger:#F97373;
  --bg:#FFFFFF;
  --card-bg:#F9FAFB;
  --text-main:#111827;
  --text-sub:#6B7280;
  --border:#E5E7EB;
  --muted: rgba(15,23,42,0.04);
}

/* layout */
.layout {
  height:100vh;
  background:
    radial-gradient(circle at top left, rgba(191,219,254,0.55) 0, transparent 55%),
    radial-gradient(circle at bottom right, rgba(219,234,254,0.85) 0, transparent 60%),
    #F3F4F6;
  color:var(--text-main);
  font-family: Inter, "Noto Sans SC", "Microsoft YaHei", sans-serif;
}
.header {
  height:60px;
  display:flex;
  align-items:center;
  justify-content:space-between;
  padding:0 20px;
  border-bottom:1px solid var(--border);
  background:rgba(255,255,255,0.96);
  backdrop-filter: blur(10px);
  box-shadow:0 4px 18px rgba(15,23,42,0.06);
  position:relative;
  z-index:10;
}
.header-left .logo { display:flex; align-items:center; gap:8px; }
.logo-icon { width:32px; height:32px; border-radius:6px; display:inline-flex; align-items:center; justify-content:center; background:linear-gradient(180deg,var(--primary),#2a6bff); color:#fff; font-weight:700; font-size:14px }
.logo-text { font-weight:600; font-size:16px; color:var(--text-main) }

/* header right icons */
.header-right { display:flex; align-items:center; gap:14px; }
.icon-notify, .icon-user {
  position:relative;
  width:36px;
  height:36px;
  display:flex;
  align-items:center;
  justify-content:center;
  color:var(--text-sub);
  border-radius:999px;
  cursor:pointer;
  background:#FFFFFF;
  box-shadow:0 4px 10px rgba(15,23,42,0.08);
  border:1px solid rgba(209,213,219,0.9);
}
.icon-notify:hover, .icon-user:hover {
  color:var(--primary);
  background:#EFF6FF;
  box-shadow:0 8px 18px rgba(37,99,235,0.18);
}
.icon-notify .dot { position:absolute; top:8px; right:10px; width:8px;height:8px;border-radius:50%; background:var(--danger) }

/* body */
.body {
  height: calc(100vh - 60px);
  display:flex;
  padding:12px 16px 16px;
  box-sizing:border-box;
  gap:12px;
}

/* aside */
.aside {
  border-right:1px solid var(--border);
  background:linear-gradient(180deg,#FFFFFF,#F3F4F6);
  padding:16px 12px;
  display:flex;
  flex-direction:column;
  justify-content:space-between;
  box-shadow:0 8px 24px rgba(15,23,42,0.06);
}
.aside.collapsed { width:64px !important; }
.module-title { font-weight:600; color:var(--text-main); font-size:14px; margin-bottom:12px }

/* menu */
.menu { display:flex; flex-direction:column; gap:6px }
.menu-item {
  display:flex;
  align-items:center;
  gap:12px;
  height:48px;
  padding:0 8px;
  border-radius:999px;
  color:var(--text-sub);
  cursor:pointer;
  position:relative;
  transition:background .15s ease,color .15s ease, transform .15s ease;
}
.menu-item:hover { background:#F3F4F6; color:var(--text-main); transform:translateX(2px); }
.menu-item.active {
  background:#EEF4FF;
  color:var(--primary);
  border-left:3px solid var(--primary);
  padding-left:5px;
  box-shadow:0 6px 18px rgba(37,99,235,0.25);
}
.menu-left { width:28px; display:flex; align-items:center; justify-content:center }
.menu-text { font-size:14px }

/* collapse toggle */
.collapse-toggle { display:flex; align-items:center; justify-content:center; margin-top:12px; color:var(--text-sub); cursor:pointer }

/* main */
.main {
  padding:16px 18px 18px;
  overflow:auto;
  background: linear-gradient(180deg, #FFFFFF, #F9FAFB);
  border-radius:12px;
  box-shadow:0 10px 30px rgba(15,23,42,0.08);
}

/* cards */
.card {
  background:var(--card-bg);
  border-radius:10px;
  border:1px solid rgba(148,163,184,0.25);
  padding:14px;
  box-sizing:border-box;
  box-shadow:0 8px 22px rgba(15,23,42,0.06);
  transition: box-shadow .18s ease, transform .18s ease, border-color .18s ease;
}
.card:hover {
  box-shadow:0 14px 32px rgba(37,99,235,0.16);
  transform:translateY(-1px);
  border-color:rgba(37,99,235,0.35);
}

/* KPI rows */
.kpi-row { display:flex; gap:16px; margin-bottom:16px }
.kpi { flex:1; height:120px; display:flex; flex-direction:column; justify-content:space-between; padding:16px }
.kpi-title { color:var(--text-sub); font-size:14px }
.kpi-value { font-size:24px; font-weight:700; color:var(--text-main) }
.kpi-trend { display:flex; align-items:center; gap:6px; font-size:12px; color:var(--text-sub) }
.trend-up { color:var(--primary) }
.trend-down { color:var(--danger) }
.trend-icon { width:14px;height:14px }

/* large KPI with gauge */
.kpi-row-2 { display:flex; gap:16px; margin-bottom:16px }
.kpi-large { flex:1; height:140px; padding:12px; display:flex; flex-direction:column; justify-content:center; align-items:flex-start }
.kpi-large-body { display:flex; align-items:center; gap:12px }
.value-big { font-size:22px; font-weight:700; color:var(--text-main) }
.meta-sub { color:var(--text-sub); font-size:12px }

/* charts area */
.charts { display:flex; gap:16px; margin-top:12px }
.chart-left { flex:2; min-height:320px; padding:12px }
.chart-area { width:100%; height:260px }
.flow-section .chart-left { padding:18px; min-height:360px }
.chart-header { display:flex; justify-content:space-between; align-items:flex-start; gap:24px; margin-bottom:12px }
.flow-card .chart-title { margin-bottom:0 }
.flow-chart { width:100%; height:280px }
.flow-meta { display:flex; gap:24px; color:var(--text-sub); font-size:13px }
.flow-meta-item { display:flex; flex-direction:column; gap:4px }
.flow-meta-item strong { color:var(--text-main); font-size:18px; font-weight:600 }

/* perf & alert center */
.perf-section { margin-top:8px; }
.perf-title { font-weight:600; font-size:16px; margin-bottom:12px; color:var(--text-main); }
.perf-grid { display:grid; grid-template-columns:repeat(auto-fit, minmax(220px, 1fr)); gap:16px; }
.perf-block { padding:8px 4px; }
.perf-block h4 { margin:0 0 8px; font-size:14px; color:var(--text-main); }
.perf-row { display:flex; justify-content:space-between; align-items:center; margin-bottom:6px; font-size:13px; color:var(--text-sub); }
.perf-row strong { color:var(--text-main); }
.netlog-block { margin-top:16px; border-top:1px dashed var(--border); padding-top:12px; }
.netlog-header { display:flex; justify-content:space-between; align-items:center; font-size:13px; color:var(--text-sub); margin-bottom:6px; }
.netlog-sub { font-size:12px; }
.netlog-table { max-height:220px; overflow:auto; border-radius:8px; border:1px solid var(--border); background:#FFFFFF; }
.netlog-row { display:grid; grid-template-columns:90px 80px 80px 1fr; padding:6px 10px; font-size:12px; color:var(--text-sub); column-gap:6px; align-items:flex-start; }
.netlog-row span:last-child { text-align:right; }
.netlog-row--head { background:#F3F4F6; font-weight:600; color:var(--text-main); position:sticky; top:0; z-index:1; }
.netlog-row:nth-child(odd):not(.netlog-row--head) { background:#F9FAFB; }
.netlog-detail { display:flex; flex-wrap:wrap; gap:4px 8px; justify-content:flex-end; }
.netlog-if { white-space:nowrap; }
.netlog-more { white-space:nowrap; color:var(--text-sub); }
.netlog-empty { padding:10px; text-align:center; font-size:12px; color:var(--text-sub); }
.alert-center { margin-top:8px; }
.alert-list { display:flex; flex-direction:column; gap:8px; }

/* right quick */
.quick {
  background:linear-gradient(180deg,#FFFFFF,#F3F4F6);
  padding:16px;
  border-left:1px solid var(--border);
  box-shadow:0 8px 24px rgba(15,23,42,0.06);
  border-radius:12px 0 0 12px;
}
.quick-card { padding:12px }
.quick-title { font-weight:600; color:var(--text-main); font-size:16px; margin-bottom:8px }
.alerts { display:flex; flex-direction:column; gap:8px; }
.alert-empty { color:var(--text-sub); padding:12px; text-align:center }
.card-alert { display:flex; gap:12px; align-items:center; padding:8px; border-radius:6px; background:var(--bg); border:1px solid var(--border); cursor:pointer }
.status { width:10px;height:40px;border-radius:4px }
.alert-body { flex:1; display:flex; flex-direction:column; gap:4px }
.alert-text { color:var(--text-main); font-size:14px }
.alert-time { color:var(--text-sub); font-size:12px }

/* system info */
.sys-grid { display:flex; flex-direction:column; gap:8px; margin-top:8px }
.sys-row { display:flex; justify-content:space-between; color:var(--text-sub) }

/* responsive */
@media (max-width: 1200px) {
  .aside { width:64px !important }
  .quick { display:none }
  .kpi-row { flex-direction:column }
  .kpi-row-2 { flex-direction:column }
  .charts { flex-direction:column }
  .flow-meta { flex-direction:column }
}
</style>
