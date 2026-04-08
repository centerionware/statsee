<template>
  <div class="card">
    <h2>CPU Load</h2>
    <canvas ref="cpuCanvas" style="width:100%;height:300px;"></canvas>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, nextTick } from 'vue';
import Chart from 'chart.js/auto';
import { store } from '../store.js';

const cpuCanvas = ref(null);
let cpuChart = null;

onMounted(async () => {
  await nextTick();
  const ctx = cpuCanvas.value.getContext('2d');
  cpuChart = new Chart(ctx, {
    type: 'line',
    data: { labels: [], datasets: [{ label: 'CPU %', data: [], borderColor: 'red', fill: false }] },
    options: { animation: false, responsive: true, maintainAspectRatio: false, scales: { y: { min: 0, max: 100 } } }
  });
});

watch(() => store.stats, (stats) => {
  if(!stats || !cpuChart) return;
  const ts = new Date(stats.ts * 1000).toLocaleTimeString();
  cpuChart.data.labels.push(ts);
  cpuChart.data.datasets[0].data.push(stats.cpu);
  if(cpuChart.data.labels.length > 30) {
    cpuChart.data.labels.shift();
    cpuChart.data.datasets[0].data.shift();
  }
  cpuChart.update();
});
</script>