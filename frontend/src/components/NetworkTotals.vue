<template>
  <div class="wrap">
    <!-- TODAY -->
    <div class="section">
      <div class="totals">
        <div class="dl">
          ⬇ {{ live.daily_in.toFixed(2) }} GB
        </div>
        <div class="ul">
          ⬆ {{ live.daily_out.toFixed(2) }} GB
        </div>
      </div>

      <div class="progress">
        <div
          class="progress-inner"
          :style="{ width: todayPercent + '%' }"
        ></div>
      </div>
    </div>

    <!-- MONTH SELECTOR -->
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

    <!-- DAILY VIEW -->
    <div v-if="isCurrentMonth" class="section">
      <div class="days">
        <div
          v-for="d in filteredDays"
          :key="d.date"
          class="day"
          @click="openDay(d)"
        >
          <div class="date">
            {{ formatDate(d.date) }}
          </div>

          <div class="bar">
            <div
              class="bar-dl"
              :style="{ width: percent(d.in) + '%' }"
            ></div>
            <div
              class="bar-ul"
              :style="{ width: percent(d.out) + '%' }"
            ></div>
          </div>
        </div>
      </div>
    </div>

    <!-- MONTH VIEW -->
    <div v-else class="section">
      <div class="month-bar">
        <div
          class="bar-dl"
          :style="{ width: percent(selectedMonthData.in) + '%' }"
        ></div>
        <div
          class="bar-ul"
          :style="{ width: percent(selectedMonthData.out) + '%' }"
        ></div>
      </div>

      <div class="month-text">
        ⬇ {{ selectedMonthData.in.toFixed(2) }} GB<br />
        ⬆ {{ selectedMonthData.out.toFixed(2) }} GB
      </div>
    </div>

    <!-- BOTTOM SHEET -->
    <div v-if="selectedDay" class="sheet" @click.self="selectedDay = null">
      <div class="sheet-inner">
        <div class="sheet-title">
          {{ selectedDay.date }}
        </div>

        <div>⬇ {{ selectedDay.in.toFixed(2) }} GB</div>
        <div>⬆ {{ selectedDay.out.toFixed(2) }} GB</div>
        <div><b>Total:</b> {{ (selectedDay.in + selectedDay.out).toFixed(2) }} GB</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'

const live = ref({
  daily_in: 0,
  daily_out: 0,
  monthly_in: 0,
  monthly_out: 0,
})

const history = ref({
  daily: [],
  monthly: [],
})

const selectedMonth = ref('')
const selectedDay = ref(null)

async function loadLive() {
  const res = await fetch('/api/network-live')
  const json = await res.json()
  const iface = Object.keys(json)[0]
  live.value = json[iface]
}

async function loadHistory() {
  const res = await fetch('/api/network-history')
  history.value = await res.json()

  if (history.value.monthly.length) {
    selectedMonth.value =
      history.value.monthly[history.value.monthly.length - 1].month
  }
}

const isCurrentMonth = computed(() => {
  const now = new Date().toISOString().slice(0, 7)
  return selectedMonth.value === now
})

const filteredDays = computed(() => {
  return history.value.daily.filter(d =>
    d.date.startsWith(selectedMonth.value)
  )
})

const selectedMonthData = computed(() => {
  return (
    history.value.monthly.find(m => m.month === selectedMonth.value) || {
      in: 0,
      out: 0,
    }
  )
})

const maxDaily = computed(() => {
  return Math.max(
    ...history.value.daily.map(d => d.in + d.out),
    1
  )
})

function percent(val) {
  return (val / maxDaily.value) * 100
}

const todayPercent = computed(() => {
  const totalToday = live.value.daily_in + live.value.daily_out
  const totalMonth = live.value.monthly_in + live.value.monthly_out
  if (totalMonth === 0) return 0
  return (totalToday / totalMonth) * 100
})

function openDay(d) {
  selectedDay.value = d
}

function selectMonth(m) {
  selectedMonth.value = m
}

function formatDate(d) {
  return d.slice(5)
}

onMounted(() => {
  loadLive()
  loadHistory()
  setInterval(loadLive, 3000)
})
</script>

<style scoped>
.wrap {
  font-size: 14px;
}

.section {
  margin-bottom: 12px;
}

.totals {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.dl {
  color: #3b82f6;
}

.ul {
  color: #ef4444;
}

.progress {
  height: 10px;
  background: #333;
  border-radius: 10px;
  overflow: hidden;
}

.progress-inner {
  height: 100%;
  background: #3b82f6;
}

.month-scroll {
  display: flex;
  overflow-x: auto;
  gap: 8px;
  margin-bottom: 12px;
}

.month-pill {
  padding: 6px 10px;
  background: #222;
  border-radius: 20px;
  white-space: nowrap;
}

.month-pill.active {
  background: #3b82f6;
}

.days {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.day {
  display: flex;
  align-items: center;
  gap: 8px;
}

.date {
  width: 50px;
  font-size: 12px;
}

.bar {
  flex: 1;
  height: 20px;
  background: #333;
  border-radius: 10px;
  overflow: hidden;
  position: relative;
}

.bar-dl {
  height: 100%;
  background: #3b82f6;
}

.bar-ul {
  height: 100%;
  background: #ef4444;
  position: absolute;
  top: 0;
  opacity: 0.7;
}

.month-bar {
  height: 24px;
  background: #333;
  border-radius: 12px;
  overflow: hidden;
  position: relative;
  margin-bottom: 8px;
}

.sheet {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex;
  align-items: flex-end;
}

.sheet-inner {
  background: #111;
  width: 100%;
  padding: 16px;
  border-radius: 16px 16px 0 0;
}
</style>