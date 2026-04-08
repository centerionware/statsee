<template>
  <div class="card">
    <h2 class="text-xl font-semibold mb-2">CPU Load</h2>
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

    onMounted(() => {
      chart = new Chart(chartRef.value, {
        type: 'line',
        data: { labels: [], datasets: [{ label: 'CPU %', data: [], borderColor: 'red', fill: false }] },
        options: { animation: false, scales: { y: { min: 0, max: 100 } } },
      });
    });

    const chartRef = ref(null);

    watch(
      () => state.cpu,
      (val) => {
        if (!chart) return;
        const ts = new Date().toLocaleTimeString();
        chart.data.labels.push(ts);
        chart.data.datasets[0].data.push(val);
        if (chart.data.labels.length > 30) {
          chart.data.labels.shift();
          chart.data.datasets[0].data.shift();
        }
        chart.update();
      }
    );

    return { chartRef };
  },
};
</script>