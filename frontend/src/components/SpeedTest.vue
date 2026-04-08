<template>
  <div class="card" style="grid-column: span 2;">
    <h2>Speed Test</h2>
    <button @click="startSpeedTest">Run Speed Test</button>
    <canvas ref="speedCanvas" height="300" class="mt-4 w-full"></canvas>
    <div class="mt-2 text-lg">{{ resultText }}</div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue';
import Chart from 'chart.js/auto';
import { store, ws } from '../store.js';

const speedCanvas = ref(null);
let speedChart = null;
const resultText = ref('');

onMounted(() => {
  speedChart = new Chart(speedCanvas.value.getContext('2d'), {
    type: 'doughnut',
    data: { labels: ['Download','Upload'], datasets: [{ label: 'Speed Test', data: [0,0], backgroundColor: ['blue','red'] }] },
    options: { responsive: true, plugins: { legend: { position: 'bottom' } } }
  });
});

watch(() => store.speedtest, (speed) => {
  if(speedChart) {
    speedChart.data.datasets[0].data = [speed.download, speed.upload];
    speedChart.update();
    if(speed.download > 0) {
      resultText.value = `Download: ${speed.download.toFixed(2)} MB/s, Upload: ${speed.upload.toFixed(2)} MB/s`;
    }
  }
});

function startSpeedTest() {
  if(ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'speedtest' }));
  }
}
</script>