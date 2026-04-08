<template>
  <div class="card">
    <h2>RAM Usage (MB)</h2>
    <canvas ref="ramCanvas" style="width:100%;height:300px;"></canvas>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, nextTick } from 'vue';
import Chart from 'chart.js/auto';
import { store } from '../store.js';

const ramCanvas = ref(null);
let ramChart = null;

onMounted(async () => {
  await nextTick();
  const ctx = ramCanvas.value.getContext('2d');
  ramChart = new Chart(ctx, {
    type: 'line',
    data: { labels: [], datasets: [
      { label: 'Used MB', data: [], borderColor: 'orange', fill: false },
      { label: 'Free MB', data: [], borderColor: 'green', fill: false }
    ]},
    options: { animation: false, responsive: true, maintainAspectRatio: false, scales: { y: { min: 0 } } }
  });
});

watch(() => store.stats, (stats) => {
  if(!stats || !ramChart) return;
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
});
</script>