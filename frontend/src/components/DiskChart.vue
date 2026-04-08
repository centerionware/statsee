<template>
  <div class="card">
    <h2>Disk I/O (MB)</h2>
    <canvas ref="diskCanvas" style="width:100%;height:300px;"></canvas>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, nextTick } from 'vue';
import Chart from 'chart.js/auto';
import { store } from '../store.js';

const diskCanvas = ref(null);
let diskChart = null;

onMounted(async () => {
  await nextTick();
  const ctx = diskCanvas.value.getContext('2d');
  diskChart = new Chart(ctx, {
    type: 'line',
    data: { labels: [], datasets: [
      { label: 'Read MB', data: [], borderColor: 'cyan', fill: false },
      { label: 'Write MB', data: [], borderColor: 'magenta', fill: false }
    ]},
    options: { animation: false, responsive: true, maintainAspectRatio: false, scales: { y: { min: 0 } } }
  });
});

watch(() => store.stats, (stats) => {
  if(!stats || !diskChart) return;
  const ts = new Date(stats.ts * 1000).toLocaleTimeString();
  let totalRead=0, totalWrite=0;
  for(let k in stats.disk){totalRead+=stats.disk[k].ReadBytes/1024/1024; totalWrite+=stats.disk[k].WriteBytes/1024/1024;}
  diskChart.data.labels.push(ts);
  diskChart.data.datasets[0].data.push(totalRead);
  diskChart.data.datasets[1].data.push(totalWrite);
  if(diskChart.data.labels.length > 30) {
    diskChart.data.labels.shift();
    diskChart.data.datasets[0].data.shift();
    diskChart.data.datasets[1].data.shift();
  }
  diskChart.update();
});
</script>