<template>
  <div class="card">
    <h2>Disk I/O (MB)</h2>
    <select v-model="selectedDisk" class="mb-2">
      <option v-for="(d, key) in diskList" :key="key" :value="key">{{ key }} {{ d.label || '' }}</option>
    </select>
    <canvas ref="diskCanvas" style="width:100%;height:300px;"></canvas>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, nextTick } from 'vue';
import Chart from 'chart.js/auto';
import { state } from '../state.js';

const diskCanvas = ref(null);
let diskChart = null;
const selectedDisk = ref('');
const diskList = ref({});

watch(() => state.disk, (d) => {
  diskList.value = d;
  if (!selectedDisk.value && Object.keys(d).length > 0) {
    selectedDisk.value = Object.keys(d)[0];
  }
}, { deep: true });

watch(selectedDisk, () => {
  // reset chart when disk selection changes
  if (!diskChart) return;
  diskChart.data.labels = [];
  diskChart.data.datasets[0].data = [];
  diskChart.data.datasets[1].data = [];
});

onMounted(async () => {
  await nextTick();
  const ctx = diskCanvas.value.getContext('2d');
  diskChart = new Chart(ctx, {
    type: 'line',
    data: {
      labels: [],
      datasets: [
        { label: 'Read MB', data: [], borderColor: 'cyan', fill: false },
        { label: 'Write MB', data: [], borderColor: 'magenta', fill: false }
      ]
    },
    options: { animation: false, responsive: true, maintainAspectRatio: false, scales: { y: { min: 0 } } }
  });
});

watch([() => state.disk, selectedDisk], () => {
  if (!diskChart || !selectedDisk.value) return;
  const disk = state.disk[selectedDisk.value];
  if (!disk) return;

  const ts = new Date().toLocaleTimeString();
  const readMB = disk.readBytes / 1024 / 1024;
  const writeMB = disk.writeBytes / 1024 / 1024;

  diskChart.data.labels.push(ts);
  diskChart.data.datasets[0].data.push(readMB);
  diskChart.data.datasets[1].data.push(writeMB);

  if (diskChart.data.labels.length > 30) {
    diskChart.data.labels.shift();
    diskChart.data.datasets.forEach(d => d.data.shift());
  }

  diskChart.update();
}, { deep: true });
</script>