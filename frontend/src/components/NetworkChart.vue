<template>
  <div class="card">
    <h2>Network Traffic</h2>
    <div class="mb-2 text-gray-300" v-html="totalsHtml"></div>
    <canvas ref="netCanvas" style="width:100%;height:300px;"></canvas>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, nextTick } from 'vue';
import Chart from 'chart.js/auto';
import { store } from '../store.js';

const netCanvas = ref(null);
let netChart = null;
const totalsHtml = ref('Loading network totals...');

async function loadNetworkTotals() {
  try {
    const res = await fetch('/api/network-totals');
    const totals = await res.json();
    let txt = '';
    for(let iface in totals){
      txt += `<b>${iface}</b>: Today: ${totals[iface].daily_in.toFixed(2)}GB in / ${totals[iface].daily_out.toFixed(2)}GB out<br>`;
    }
    totalsHtml.value = txt;
  } catch(e){ console.error(e); }
}

onMounted(async () => {
  await nextTick();
  const ctx = netCanvas.value.getContext('2d');
  netChart = new Chart(ctx, {
    type: 'line',
    data: { labels: [], datasets: [
      { label: 'Ingress MB/s', data: [], borderColor: 'lime', fill: false },
      { label: 'Egress MB/s', data: [], borderColor: 'yellow', fill: false }
    ]},
    options: { animation: false, responsive: true, maintainAspectRatio: false, scales: { y: { min: 0 } } }
  });
  loadNetworkTotals();
  setInterval(loadNetworkTotals, 3000);
});

watch(() => store.stats, (stats) => {
  if(!stats || !netChart) return;
  const ts = new Date(stats.ts * 1000).toLocaleTimeString();
  let totalIn=0,totalOut=0;
  for(let k in stats.net){ totalIn+=stats.net[k].rate_recv; totalOut+=stats.net[k].rate_sent; }
  netChart.data.labels.push(ts);
  netChart.data.datasets[0].data.push(totalIn);
  netChart.data.datasets[1].data.push(totalOut);
  if(netChart.data.labels.length>30){ 
    netChart.data.labels.shift(); 
    netChart.data.datasets[0].data.shift(); 
    netChart.data.datasets[1].data.shift(); 
  }
  netChart.update();
});
</script>