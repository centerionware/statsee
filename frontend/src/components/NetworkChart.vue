<template>
  <div class="card">
    <h2>Network Traffic</h2>
    <div v-html="networkHtml" class="mb-2"></div>
    <canvas ref="netCanvas" height="300"></canvas>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue';
import Chart from 'chart.js/auto';
import { store } from '../store.js';

const netCanvas = ref(null);
let netChart = null;
const networkHtml = ref('Loading network totals...');

async function fetchTotals() {
  try {
    const res = await fetch('/api/network-totals');
    const totals = await res.json();
    store.networkTotals = totals;
    let txt = '';
    for(let iface in totals){
      txt += `<b>${iface}</b>: Today: ${totals[iface].daily_in.toFixed(2)}GB in / ${totals[iface].daily_out.toFixed(2)}GB out, Month: ${totals[iface].monthly_in.toFixed(2)}GB in / ${totals[iface].monthly_out.toFixed(2)}GB out<br>`;
    }
    networkHtml.value = txt;
  } catch(e){ console.error(e); }
}

setInterval(fetchTotals, 3000);
fetchTotals();

onMounted(() => {
  netChart = new Chart(netCanvas.value.getContext('2d'), {
    type: 'line',
    data: { labels: [], datasets: [
      { label: 'Ingress MB/s', data: [], borderColor: 'lime', fill: false },
      { label: 'Egress MB/s', data: [], borderColor: 'yellow', fill: false }
    ]},
    options: { animation: false, scales: { y: { min: 0 } } }
  });
});

watch(() => store.stats, (stats) => {
  if(stats && netChart) {
    const ts = new Date(stats.ts * 1000).toLocaleTimeString();
    let totalIn = 0, totalOut = 0;
    for(const k in stats.net) {
      totalIn += stats.net[k].rate_recv;
      totalOut += stats.net[k].rate_sent;
    }
    netChart.data.labels.push(ts);
    netChart.data.datasets[0].data.push(totalIn);
    netChart.data.datasets[1].data.push(totalOut);
    if(netChart.data.labels.length > 30) {
      netChart.data.labels.shift();
      netChart.data.datasets[0].data.shift();
      netChart.data.datasets[1].data.shift();
    }
    netChart.update();
  }
});
</script>