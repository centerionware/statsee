<template>
  <div class="card" style="height:100%">
    <h2>Network Traffic</h2>
    <canvas ref="netCanvas" style="width:100%;height:100%;"></canvas>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, nextTick } from 'vue';
import Chart from 'chart.js/auto';
import { store } from '../store.js';

const netCanvas = ref(null);
let netChart = null;

onMounted(async () => {
  await nextTick();

  const ctx = netCanvas.value.getContext('2d');

  netChart = new Chart(ctx, {
    type: 'line',
    data: {
      labels: [],
      datasets: [
        { label: 'Ingress MB/s', data: [], borderColor: 'lime', fill: false },
        { label: 'Egress MB/s', data: [], borderColor: 'yellow', fill: false }
      ]
    },
    options: {
      animation: false,
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        y: { min: 0 }
      }
    }
  });
});

watch(() => store.stats, (stats) => {
  if (!stats || !netChart) return;

  const ts = new Date(stats.ts * 1000).toLocaleTimeString();

  let totalIn = 0;
  let totalOut = 0;

  for (let k in stats.net) {
    totalIn += stats.net[k].rate_recv || 0;
    totalOut += stats.net[k].rate_sent || 0;
  }

  netChart.data.labels.push(ts);
  netChart.data.datasets[0].data.push(totalIn);
  netChart.data.datasets[1].data.push(totalOut);

  if (netChart.data.labels.length > 30) {
    netChart.data.labels.shift();
    netChart.data.datasets.forEach(d => d.data.shift());
  }

  netChart.update();
});
</script>