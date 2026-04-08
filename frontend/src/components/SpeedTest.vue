<template>
  <div class="card">
    <h2 class="text-xl font-semibold mb-2">Speed Test</h2>
    <button @click="startTest">Run Speed Test</button>
    <canvas ref="chart" height="300" class="mt-4 w-full"></canvas>
    <div class="mt-2 text-lg">{{ result }}</div>
  </div>
</template>

<script>
import { onMounted, ref, watch } from 'vue';
import Chart from 'chart.js/auto';
import { state } from '../state.js';
import { startSpeedTest } from '../ws.js';

export default {
  setup() {
    const chartRef = ref(null);
    let chart;
    const result = ref('');

    onMounted(() => {
      chart = new Chart(chartRef.value, {
        type: 'doughnut',
        data: { labels: ['Download', 'Upload'], datasets: [{ label: 'Speed Test', data: [0, 0], backgroundColor: ['blue', 'red'] }] },
        options: { responsive: true, plugins: { legend: { position: 'bottom' } } },
      });
    });

    watch(
      () => [state.speedTest.download, state.speedTest.upload],
      () => {
        if (!chart) return;
        chart.data.datasets[0].data = [state.speedTest.download, state.speedTest.upload];
        chart.update();
        result.value = `Download: ${state.speedTest.download.toFixed(2)} MB/s, Upload: ${state.speedTest.upload.toFixed(2)} MB/s`;
      }
    );

    const startTest = () => startSpeedTest();

    return { chartRef, result, startTest };
  },
};
</script>