<template>
  <div class="card">
    <h2>RAM Usage (MB)</h2>
    <canvas ref="ramCanvas" height="300"></canvas>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue';
import Chart from 'chart.js/auto';
import { store } from '../store.js';

const ramCanvas = ref(null);
let ramChart = null;

onMounted(() => {
  ramChart = new Chart(ramCanvas.value.getContext('2d'), {
    type: 'line',
    data: { labels: [], datasets: [
      { label: 'Used MB', data: [], borderColor: 'orange', fill: false },
      { label: 'Free MB', data: [], borderColor: 'green', fill: false }
    ] },
    options: { animation: false, scales: { y: { min: 0 } } }
  });
});

watch(() => store.stats, (stats) => {
  if(stats && ramChart) {
    const ts = new Date(stats.ts * 1000).toLocaleTimeString();
    ramChart.data.labels.push(ts);
    ramChart.data.datasets[0].data.push(stats.ram.used);
    ramChart.data.datasets[1].data.push(stats.ram.free);
    if(ramChart.data.labels.length > 30) {
      ramChart.data.labels.shift();
      ramChart.data.datasets[0].data.shift();
      ramChart.data.datasets[1].data.shift();
    }
    ramChart.update();
  }
});
</script>