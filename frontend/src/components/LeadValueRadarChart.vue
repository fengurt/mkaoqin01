<template>
  <div class="lvr" :style="{ width: `${diameter}px`, height: `${diameter + (compact ? 0 : 22)}px` }">
    <svg
      class="lvr-svg"
      :viewBox="`0 0 ${vb} ${vb}`"
      xmlns="http://www.w3.org/2000/svg"
      :aria-label="ariaLabel"
      role="img"
    >
      <g :transform="`translate(${cx} ${cy})`">
        <template v-for="ring in gridRings" :key="'g' + ring">
          <polygon :points="hexPoints(ring)" class="lvr-grid" />
        </template>
        <g v-for="(lab, i) in labelAngles" :key="'a' + i">
          <line :x1="0" :y1="0" :x2="lab.x2" :y2="lab.y2" class="lvr-axis" />
        </g>
        <polygon :points="dataPoints" class="lvr-area" />
        <polygon :points="dataPoints" class="lvr-stroke" fill="none" />
        <circle
          v-for="(pt, i) in dataVertices"
          :key="'d' + i"
          :cx="pt.x"
          :cy="pt.y"
          r="3.2"
          class="lvr-dot"
        />
      </g>
      <g v-if="!compact && labels.length">
        <text
          v-for="(t, i) in labelTexts"
          :key="'t' + i"
          :x="t.x"
          :y="t.y"
          class="lvr-lbl"
          text-anchor="middle"
          dominant-baseline="middle"
        >
          {{ t.text }}
        </text>
      </g>
    </svg>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { LEAD_RADAR_AXIS_KEYS } from '../lib/leadValuePotential'

const props = defineProps({
  /** 与 LEAD_RADAR_AXIS_KEYS 同序的 0–100 分 */
  scores: { type: Array, required: true },
  /** 与 scores 同序的短标签 */
  labels: { type: Array, default: () => [] },
  /** 外接圆直径（像素） */
  diameter: { type: Number, default: 220 },
  compact: { type: Boolean, default: false },
  ariaLabel: { type: String, default: 'Lead value radar' },
})

const n = LEAD_RADAR_AXIS_KEYS.length
const vb = 260
const cx = vb / 2
const cy = vb / 2
const maxR = 76

const gridRings = computed(() => (props.compact ? [0.45, 0.72, 1] : [0.34, 0.55, 0.76, 1]))

function hexPoints(radiusFrac) {
  const pts = []
  for (let i = 0; i < n; i += 1) {
    const ang = -Math.PI / 2 + (i * 2 * Math.PI) / n
    const r = maxR * radiusFrac
    pts.push(`${r * Math.cos(ang)},${r * Math.sin(ang)}`)
  }
  return pts.join(' ')
}

const labelAngles = computed(() => {
  const arr = []
  for (let i = 0; i < n; i += 1) {
    const ang = -Math.PI / 2 + (i * 2 * Math.PI) / n
    arr.push({
      x2: (maxR + 6) * Math.cos(ang),
      y2: (maxR + 6) * Math.sin(ang),
    })
  }
  return arr
})

const dataVertices = computed(() => {
  const vals = props.scores.map((v) => {
    const x = Number(v)
    if (!Number.isFinite(x)) return 0
    return Math.max(0, Math.min(100, x)) / 100
  })
  while (vals.length < n) vals.push(0)
  const arr = []
  for (let i = 0; i < n; i += 1) {
    const ang = -Math.PI / 2 + (i * 2 * Math.PI) / n
    const rf = vals[i]
    arr.push({ x: maxR * rf * Math.cos(ang), y: maxR * rf * Math.sin(ang) })
  }
  return arr
})

const dataPoints = computed(() => dataVertices.value.map((p) => `${p.x},${p.y}`).join(' '))

const labelTexts = computed(() => {
  if (props.compact || !props.labels?.length) return []
  const pad = 18
  return props.labels.slice(0, n).map((text, i) => {
    const ang = -Math.PI / 2 + (i * 2 * Math.PI) / n
    const r = maxR + pad
    return {
      x: cx + r * Math.cos(ang),
      y: cy + r * Math.sin(ang) + (i === 0 || i === 3 ? 0 : i < 3 ? -2 : 2),
      text: String(text),
    }
  })
})
</script>

<style scoped>
.lvr {
  margin: 0 auto;
  user-select: none;
}
.lvr-svg {
  width: 100%;
  height: auto;
  display: block;
}
.lvr-grid {
  fill: rgba(184, 149, 79, 0.04);
  stroke: rgba(26, 34, 48, 0.12);
  stroke-width: 1;
}
.lvr-axis {
  stroke: rgba(26, 34, 48, 0.1);
  stroke-width: 1;
}
.lvr-area {
  fill: rgba(184, 149, 79, 0.22);
  stroke: none;
}
.lvr-stroke {
  stroke: rgba(107, 90, 50, 0.85);
  stroke-width: 1.6;
  stroke-linejoin: round;
}
.lvr-dot {
  fill: #fdfbf7;
  stroke: rgba(107, 90, 50, 0.95);
  stroke-width: 1.2;
}
.lvr-lbl {
  font-size: 8.5px;
  font-weight: 700;
  fill: var(--brand-subtext, #6b6560);
  letter-spacing: 0.02em;
}
</style>
