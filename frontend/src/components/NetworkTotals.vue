<template>
  <div class="wrap">
    <div class="label">Today</div>

    <div class="totals">
      <div class="dl">⬇ {{ live.daily_in?.toFixed(2) || 0 }} GB</div>
      <div class="ul">⬆ {{ live.daily_out?.toFixed(2) || 0 }} GB</div>
    </div>

    <div class="progress">
      <div class="progress-inner" :style="{ width: todayPercent + '%' }"></div>
    </div>

    <!-- FIXED: monthly now derived from daily history -->
    <div class="sub">
      Month:
      {{ currentMonthTotal.in.toFixed(2) }} /
      {{ currentMonthTotal.out.toFixed(2) }} GB
    </div>

    <div class="month-scroll">
      <div
        v-for="m in history.monthly"
        :key="m.month"
        class="month-pill"
        :class="{ active: selectedMonth === m.month }"
        @click="selectMonth(m.month)"
      >
        {{ m.month }}
      </div>
    </div>

    <div v-if="isCurrentMonth">
      <div class="label">Daily Usage</div>

      <div v-if="filteredDays.length === 0" class="empty">
        No data yet
      </div>

      <div class="days">
        <div
          v-for="d in filteredDays"
          :key="d.date"
          class="day"
          @click="selectedDay = d"
        >
          <div class="date">{{ d.date.slice(5) }}</div>

          <div class="bar">
            <div class="bar-dl" :style="{ width: percent(d.in) + '%' }"></div>
            <div class="bar-ul" :style="{ width: percent(d.out) + '%' }"></div>
          </div>
        </div>
      </div>
    </div>

    <div v-else>
      <div class="label">{{ selectedMonth }}</div>

      <div class="month-bar">
        <div
          class="bar-dl"
          :style="{ width: percent(currentMonthTotal.in) + '%' }"
        ></div>
        <div
          class="bar-ul"
          :style="{ width: percent(currentMonthTotal.out) + '%' }"
        ></div>
      </div>
    </div>

    <div v-if="selectedDay" class="sheet" @click.self="selectedDay = null">
      <div class="sheet-inner">
        <b>{{ selectedDay.date }}</b><br />
        ⬇ {{ selectedDay.in.toFixed(2) }} GB<br />
        ⬆ {{ selectedDay.out.toFixed(2) }} GB
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'

const live = ref({})
const history = ref({ daily: [], monthly: [] })

const selectedMonth = ref('')
const selectedDay = ref(null)

async function loadLive() {
  const r = await fetch('/api/network-live')
  const j = await r.json()
  live.value = j[Object.keys(j)[0]]
}

async function loadHistory() {
  const r = await fetch('/api/network-history')
  history.value = await r.json()

  if (history.value.monthly.length) {
    selectedMonth.value = history.value.monthly.at(-1).month
  }
}

const isCurrentMonth = computed(() =>
  selectedMonth.value === new Date().toISOString().slice(0, 7)
)

const filteredDays = computed(() =>
  history.value.daily.filter(d => d.date.startsWith(selectedMonth.value))
)

/**
 * 🔥 FIX: monthly is now computed from daily data (source of truth)
 */
const currentMonthTotal = computed(() => {
  const days = filteredDays.value

  return days.reduce(
    (acc, d) => {
      acc.in += d.in
      acc.out += d.out
      return acc
    },
    { in: 0, out: 0 }
  )
})

const maxDaily = computed(() =>
  Math.max(...history.value.daily.map(d => d.in + d.out), 1)
)

function percent(v) {
  return (v / maxDaily.value) * 100
}

const todayPercent = computed(() => {
  const t = (live.value.daily_in || 0) + (live.value.daily_out || 0)
  const m = currentMonthTotal.value.in + currentMonthTotal.value.out
  return m === 0 ? 0 : (t / m) * 100
})

function selectMonth(m) {
  selectedMonth.value = m
}

onMounted(() => {
  loadLive()
  loadHistory()
  setInterval(loadLive, 3000)
})
</script>

<style scoped>
.wrap { font-size:14px; }
.label { color:#aaa; font-size:12px; margin-bottom:6px; }

.totals { display:flex; justify-content:space-between; margin-bottom:6px; }
.dl { color:#3b82f6; }
.ul { color:#ef4444; }

.progress {
  height:10px;
  background:#333;
  border-radius:10px;
  overflow:hidden;
}

.progress-inner {
  height:100%;
  background:#3b82f6;
}

.sub { font-size:12px; color:#888; margin-top:6px; }

.month-scroll {
  display:flex;
  gap:6px;
  overflow:auto;
  margin:10px 0;
}

.month-pill {
  background:#222;
  padding:4px 8px;
  border-radius:12px;
}

.month-pill.active {
  background:#3b82f6;
}

.days {
  display:flex;
  flex-direction:column;
  gap:6px;
}

.day {
  display:flex;
  gap:6px;
  align-items:center;
}

.date {
  width:45px;
  font-size:12px;
}

.bar {
  flex:1;
  height:18px;
  background:#333;
  border-radius:8px;
  position:relative;
  overflow:hidden;
}

.bar-dl {
  height:100%;
  background:#3b82f6;
}

.bar-ul {
  position:absolute;
  top:0;
  height:100%;
  background:#ef4444;
  opacity:.7;
}

.month-bar {
  height:20px;
  background:#333;
  border-radius:10px;
  overflow:hidden;
  position:relative;
}

.empty {
  color:#777;
  font-size:12px;
}

.sheet {
  position:fixed;
  inset:0;
  background:rgba(0,0,0,.6);
  display:flex;
  align-items:flex-end;
}

.sheet-inner {
  background:#111;
  width:100%;
  padding:16px;
  border-radius:12px 12px 0 0;
}
</style>