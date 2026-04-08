<template>
  <div class="card">
    <h2 class="text-xl font-semibold mb-2">RAM Usage (MB)</h2>
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
        data: {
          labels: [],
          datasets: [
            { label: 'Used MB', data: [], borderColor: 'orange', fill: false },
            { label: 'Free MB', data: [], borderColor: 'green', fill: false },
          ],
        },
        options: { animation: false, scales: { y: { min: 0 } } },
      });
    });

    watch(
      () => [state.ram.used, state.ram.free],
      () => {
        if (!chart) return;
        const ts = new Date().toLocaleTimeString();
        chart.data.labels.push(ts);
        chart.data.datasets[0].data.push(state.ram.used);
        chart.data.datasets[1].data.push(state.ram.free);
        if (chart.data.labels.length > 30) {
          chart.data.labels.shift();
          chart.data.datasets[0].data.shift();
          chart.data.datasets[1].data.shift();
        }
        chart.update();
      }
    );

    return { chartRef };
  },
};
</script>