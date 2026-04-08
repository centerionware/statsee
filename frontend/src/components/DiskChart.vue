<template>
  <div class="card">
    <h2 class="text-xl font-semibold mb-2">Disk I/O (MB)</h2>
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
            { label: 'Read MB', data: [], borderColor: 'cyan', fill: false },
            { label: 'Write MB', data: [], borderColor: 'magenta', fill: false },
          ],
        },
        options: { animation: false, scales: { y: { min: 0 } } },
      });
    });

    watch(
      () => [state.disk.read, state.disk.write],
      () => {
        if (!chart) return;
        const ts = new Date().toLocaleTimeString();
        chart.data.labels.push(ts);
        chart.data.datasets[0].data.push(state.disk.read);
        chart.data.datasets[1].data.push(state.disk.write);
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