<template>
  <div class="card">
    <h2>CPU Load</h2>
    <canvas ref="cpuCanvas" height="300"></canvas>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue';
import Chart from 'chart.js/auto';
import { store } from '../store.js';

const cpuCanvas = ref(null);
let cpuChart = null;

onMounted(() => {
  cpuChart = new Chart(cpuCanvas.value.getContext('2d'), {
    type: 'line',
    data: { labels: [], datasets: [{ label: 'CPU %', data: [], borderColor: 'red', fill: false }] },
    options: { animation: false, scales: { y: { min: 0, max: 100 } } }
  });
});

watch(() => store.stats, (stats) => {
  if(stats && cpuChart) {
    const ts = new Date(stats.ts * 1000).toLocaleTimeString();
    cpuChart.data.labels.push(ts);
    cpuChart.data.datasets[0].data.push(stats.cpu);
    if(cpuChart.data.labels.length > 30) {
      cpuChart.data.labels.shift();
      cpuChart.data.datasets[0].data.shift();
    }
    cpuChart.update();
  }
});
</script>