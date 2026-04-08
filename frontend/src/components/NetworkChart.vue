<template>
  <div class="card">
    <h2 class="text-xl font-semibold mb-2">Network Traffic</h2>
    <div class="mb-2 text-lg font-bold text-gray-300">
      <div v-for="(iface, name) in state.net" :key="name">
        <b>{{ name }}</b>: Rx {{ iface.rate_recv.toFixed(2) }} MB/s / Tx {{ iface.rate_sent.toFixed(2) }} MB/s
      </div>
    </div>
    <canvas ref="chart" height="300"></canvas>
  </div>
</template>

<script>
import { onMounted, watch } from 'vue';
import Chart from 'chart.js/auto';
import { state } from '../state.js';

export default {
  setup() {
    let chart;
    const chartRef = ref(null);

    onMounted(() => {
      chart = new Chart(chartRef.value, {
        type: 'line',
        data: { labels: [], datasets: [
          { label: 'Ingress MB/s', data: [], borderColor: 'lime', fill: false },
          { label: 'Egress MB/s', data: [], borderColor: 'yellow', fill: false },
        ]},
        options: { animation: false, scales: { y: { min: 0 } } },
      });
    });

    watch(
      () => state.net,
      () => {
        if (!chart) return;
        const ts = new Date().toLocaleTimeString();
        const totalIn = Object.values(state.net).reduce((sum, n) => sum + n.rate_recv, 0);
        const totalOut = Object.values(state.net).reduce((sum, n) => sum + n.rate_sent, 0);
        chart.data.labels.push(ts);
        chart.data.datasets[0].data.push(totalIn);
        chart.data.datasets[1].data.push(totalOut);
        if(chart.data.labels.length > 30){
          chart.data.labels.shift();
          chart.data.datasets[0].data.shift();
          chart.data.datasets[1].data.shift();
        }
        chart.update();
      },
      { deep: true }
    );

    return { chartRef, state };
  },
};
</script>