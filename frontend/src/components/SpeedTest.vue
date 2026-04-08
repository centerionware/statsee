<template>
  <div>
    <button @click="run">Run Speed Test</button>
    <canvas ref="canvas"></canvas>
    <div>
      Download: {{ state.speedtest.download.toFixed(2) }} MB/s |
      Upload: {{ state.speedtest.upload.toFixed(2) }} MB/s
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import Chart from 'chart.js/auto'
import { state, startSpeedTest } from '../ws'

const canvas = ref()
let chart

onMounted(() => {
  chart = new Chart(canvas.value, {
    type: 'doughnut',
    data: {
      labels: ['Download', 'Upload'],
      datasets: [{ data: [0, 0] }]
    }
  })
})

watch(() => state.speedtest, () => {
  chart.data.datasets[0].data = [
    state.speedtest.download,
    state.speedtest.upload
  ]
  chart.update()
}, { deep: true })

function run() {
  startSpeedTest()
}
</script>