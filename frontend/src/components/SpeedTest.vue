<template>
  <div class="card md:col-span-2 flex flex-col items-start">
    <h2>Speed Test</h2>
    <button @click="runSpeedTest" class="mb-2">Run Speed Test</button>
    <canvas ref="speedCanvas" style="width:100%;height:300px;" class="mt-4"></canvas>
    <div class="mt-2 text-lg">{{ resultText }}</div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, nextTick } from 'vue';
import Chart from 'chart.js/auto';
import { store } from '../store.js';

const speedCanvas = ref(null);
let speedChart = null;
const resultText = ref('');

function runSpeedTest() {
  if(window.ws) window.ws.send(JSON.stringify({type:'speedtest'}));
}

onMounted(async () => {
  await nextTick();
  const ctx = speedCanvas.value.getContext('2d');
  speedChart = new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels: ['Download','Upload'],
      datasets: [{
        label: 'Speed Test',
        data: [0,0],
        backgroundColor: ['blue','red']
      }]
    },
    options: {
      responsive: true,
      plugins: {
        legend: { position: 'bottom' }
      }
    }
  });
});

watch(() => store.speedtest, (msg) => {
  if(!msg || !speedChart) return;
  speedChart.data.datasets[0].data = [msg.download || 0, msg.upload || 0];
  speedChart.update();
  if(msg.type === 'speedtest_done') {
    resultText.value = `Download: ${(msg.download||0).toFixed(2)} MB/s, Upload: ${(msg.upload||0).toFixed(2)} MB/s`;
  }
});
</script>