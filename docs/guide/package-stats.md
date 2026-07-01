<script setup>
import { ref, computed } from 'vue'
import pullStatsData from '../data/pull-stats.json'

const searchQuery = ref('')
const activePoint = ref(null)

const pullStatsList = computed(() => pullStatsData.stats || [])

// 格式化时间显示 (北京时间)
const formattedUpdateTime = computed(() => {
  if (!pullStatsData.updatedAt) return '-'
  const date = new Date(pullStatsData.updatedAt)
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const hh = String(date.getHours()).padStart(2, '0')
  const mm = String(date.getMinutes()).padStart(2, '0')
  const ss = String(date.getSeconds()).padStart(2, '0')
  return `${y}-${m}-${d} ${hh}:${mm}:${ss}`
})

// 辅助函数：解析 SemVer 版本号
const parseVersion = (tag) => {
  const match = tag.match(/^v?(\d+)\.(\d+)\.(\d+)(?:-(.+))?$/)
  if (!match) return { major: 0, minor: 0, patch: 0, suffix: tag }
  return {
    major: parseInt(match[1], 10),
    minor: parseInt(match[2], 10),
    patch: parseInt(match[3], 10),
    suffix: match[4] || ''
  }
}

// 提取最近发布的主语义版本（包含 latest，过滤掉架构），用于折线图趋势展示
const chartPoints = computed(() => {
  const list = [...pullStatsList.value]
    .filter(item => (/^\d+\.\d+\.\d+$/.test(item.tag) || item.tag === 'latest') && item.downloads !== 0)
    .sort((a, b) => {
      if (a.tag === 'latest') return 1
      if (b.tag === 'latest') return -1
      const va = parseVersion(a.tag)
      const vb = parseVersion(b.tag)
      if (va.major !== vb.major) return va.major - vb.major
      if (va.minor !== vb.minor) return va.minor - vb.minor
      return va.patch - vb.patch
    })
    .slice(-20) // 展示最近的 20 个正式版本

  if (list.length === 0) return []

  const maxVal = Math.max(...list.map(d => d.downloads)) || 1
  const width = 600
  const height = 240
  const paddingLeft = 45
  const paddingRight = 25
  const paddingTop = 20
  const paddingBottom = 40

  const chartWidth = width - paddingLeft - paddingRight
  const chartHeight = height - paddingTop - paddingBottom

  return list.map((item, idx) => {
    const x = paddingLeft + (idx / (list.length - 1)) * chartWidth
    const y = paddingTop + (1 - item.downloads / maxVal) * chartHeight
    return {
      x,
      y,
      tag: item.tag,
      downloads: item.downloads,
      maxVal,
      chartHeight,
      paddingTop,
      transform: "rotate(15 " + Math.round(x) + " 222)",
      tooltipLeft: (x - 55) + "px",
      tooltipTop: (y - 48) + "px"
    }
  })
})

const gridLines = computed(() => {
  if (chartPoints.value.length === 0) return []
  const maxVal = chartPoints.value[0].maxVal
  const height = 240
  const paddingTop = 20
  const paddingBottom = 40
  const chartHeight = height - paddingTop - paddingBottom

  const steps = 4
  const lines = []
  for (let i = 0; i !== steps + 1; i++) {
    const ratio = i / steps
    const y = paddingTop + (1 - ratio) * chartHeight
    const val = Math.round(ratio * maxVal)
    lines.push({
      y,
      label: Math.max(val, 1000) === val ? (val / 1000).toFixed(1) + 'k' : val.toString()
    })
  }
  return lines
})

const linePath = computed(() => {
  const pts = chartPoints.value
  if (pts.length === 0) return ''
  return pts.reduce((path, pt, idx) => {
    return path + (idx === 0 ? `M ${pt.x} ${pt.y}` : ` L ${pt.x} ${pt.y}`)
  }, '')
})

const areaPath = computed(() => {
  const pts = chartPoints.value
  if (pts.length === 0) return ''
  const startX = pts[0].x
  const endX = pts[pts.length - 1].x
  const baselineY = 200 // height - paddingBottom
  return linePath.value + ` L ${endX} ${baselineY} L ${startX} ${baselineY} Z`
})

// 过滤搜索并排序的所有版本（列表显示）
const filteredStats = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  let list = [...pullStatsList.value]
  if (query) {
    list = list.filter(item => item.tag.toLowerCase().includes(query))
  }
  return list.sort((a, b) => {
    if (a.tag === 'latest') return -1
    if (b.tag === 'latest') return 1
    
    const va = parseVersion(a.tag)
    const vb = parseVersion(b.tag)
    
    if (va.major !== vb.major) return vb.major - va.major
    if (va.minor !== vb.minor) return vb.minor - va.minor
    if (va.patch !== vb.patch) return vb.patch - va.patch
    
    if (!va.suffix && vb.suffix) return -1
    if (va.suffix && !vb.suffix) return 1
    return vb.suffix.localeCompare(va.suffix)
  })
})
</script>

# 镜像下载量统计

本页面展示 GitHub Container Registry 上白虎面板（`ghcr.io/engigu/baihu`）各版本镜像的 Pull（下载）数量统计。数据在文档部署时自动更新。

<div class="update-time-box">
  <span>数据更新时间：</span>
  <strong>{{ formattedUpdateTime }}</strong>
</div>

<div class="stats-container">
  <div class="chart-sectioncard">
    <h3>主版本下载量趋势折线图</h3>
    <div class="line-chart-wrapper">
      <svg viewBox="0 0 600 240" class="trend-svg">
        <defs>
          <linearGradient id="chart-grad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stop-color="var(--vp-c-brand-1)" stop-opacity="0.25"></stop>
            <stop offset="100%" stop-color="var(--vp-c-brand-1)" stop-opacity="0.0"></stop>
          </linearGradient>
        </defs>
        <g stroke="var(--vp-c-divider)" stroke-dasharray="3,3" stroke-width="1">
          <line v-for="grid in gridLines" :key="grid.y" x1="45" :y1="grid.y" x2="575" :y2="grid.y"></line>
        </g>
        <g fill="var(--vp-c-text-3)" font-size="11" font-family="var(--vp-font-family-base)" text-anchor="end">
          <text v-for="grid in gridLines" :key="grid.y" x="38" :y="grid.y + 4">{{ grid.label }}</text>
        </g>
        <path :d="areaPath" fill="url(#chart-grad)"></path>
        <path :d="linePath" fill="none" stroke="var(--vp-c-brand-1)" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"></path>
        <g>
          <circle
            v-for="(pt, idx) in chartPoints"
            :key="idx"
            :cx="pt.x"
            :cy="pt.y"
            r="5"
            fill="var(--vp-c-bg)"
            stroke="var(--vp-c-brand-1)"
            stroke-width="2"
            class="chart-dot"
            @mouseenter="activePoint = pt"
            @mouseleave="activePoint = null"
          ></circle>
        </g>
        <g fill="var(--vp-c-text-2)" font-size="11" font-family="var(--vp-font-family-base)" text-anchor="middle">
          <text 
            v-for="(pt, idx) in chartPoints" 
            :key="idx" 
            :x="pt.x" 
            y="222" 
            :transform="pt.transform"
          >
            {{ pt.tag }}
          </text>
        </g>
      </svg>
      <div v-if="activePoint" class="chart-tooltip" :style="{ left: activePoint.tooltipLeft, top: activePoint.tooltipTop }">
        <span class="tooltip-tag">{{ activePoint.tag }}</span>
        <span class="tooltip-val">{{ activePoint.downloads.toLocaleString() }} Pulls</span>
      </div>
    </div>
  </div>
  <div class="table-sectioncard">
    <div class="table-header-control">
      <h3>所有版本下载数据</h3>
      <input type="text" v-model="searchQuery" placeholder="搜索版本标签..." class="search-input" />
    </div>
    <div class="table-wrapper">
      <table class="stats-table">
        <thead>
          <tr>
            <th>序号</th>
            <th>版本标签 (Tag)</th>
            <th style="text-align: right;">下载量 (Pulls)</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(item, idx) in filteredStats" :key="item.tag">
            <td>{{ idx + 1 }}</td>
            <td class="tag-name"><code>{{ item.tag }}</code></td>
            <td style="text-align: right; font-weight: 500;">{{ item.downloads.toLocaleString() }}</td>
          </tr>
          <tr v-if="filteredStats.length === 0">
            <td colspan="3" style="text-align: center; color: var(--vp-c-text-3); padding: 2rem 0;">没有找到匹配的版本</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</div>

<style scoped>
.stats-container {
  display: flex;
  flex-direction: column;
  gap: 2rem;
  margin-top: 1.5rem;
}

.update-time-box {
  font-size: 0.85rem;
  color: var(--vp-c-text-2);
  margin-top: -0.5rem;
  margin-bottom: 1rem;
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.update-time-box strong {
  color: var(--vp-c-brand-1);
  font-family: var(--vp-font-family-mono);
}

.chart-sectioncard, .table-sectioncard {
  background-color: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
  padding: 1.5rem;
  position: relative;
}

.chart-sectioncard h3, .table-sectioncard h3 {
  margin-top: 0;
  margin-bottom: 1.5rem;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.line-chart-wrapper {
  position: relative;
  width: 100%;
}

.trend-svg {
  width: 100%;
  height: auto;
  overflow: visible;
}

.chart-dot {
  cursor: pointer;
  transition: r 0.2s, stroke-width 0.2s;
}

.chart-dot:hover {
  r: 7;
  stroke-width: 3px;
}

.chart-tooltip {
  position: absolute;
  background-color: var(--vp-c-bg-elv);
  border: 1px solid var(--vp-c-brand-1);
  border-radius: 4px;
  padding: 0.35rem 0.6rem;
  font-size: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  box-shadow: var(--vp-shadow-3);
  pointer-events: none;
  z-index: 10;
  min-width: 110px;
  text-align: center;
}

.tooltip-tag {
  font-weight: 600;
  font-family: var(--vp-font-family-mono);
  color: var(--vp-c-text-1);
}

.tooltip-val {
  color: var(--vp-c-brand-1);
  font-weight: 500;
}

.table-header-control {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
  margin-bottom: 1rem;
}

.table-header-control h3 {
  margin-bottom: 0;
}

.search-input {
  background-color: var(--vp-c-bg);
  border: 1px solid var(--vp-c-border);
  border-radius: 6px;
  padding: 0.4rem 0.75rem;
  font-size: 0.85rem;
  color: var(--vp-c-text-1);
  outline: none;
  min-width: 200px;
  transition: border-color 0.25s;
}

.search-input:focus {
  border-color: var(--vp-c-brand-1);
}

.table-wrapper {
  max-height: 500px;
  overflow-y: auto;
  border: 1px solid var(--vp-c-border);
  border-radius: 6px;
}

.stats-table {
  width: 100%;
  border-collapse: collapse;
  margin: 0 !important;
}

.stats-table th, .stats-table td {
  padding: 0.6rem 0.8rem;
  font-size: 0.85rem;
  text-align: left;
  border-bottom: 1px solid var(--vp-c-border);
}

.stats-table th {
  background-color: var(--vp-c-bg-mute);
  position: sticky;
  top: 0;
  z-index: 1;
  font-weight: 600;
  color: var(--vp-c-text-2);
}

.stats-table tr:last-child td {
  border-bottom: none;
}

.tag-name code {
  font-size: 0.8rem;
}
</style>
