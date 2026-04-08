<template>
  <div style="padding:20px">
    <h1>StatSee Dashboard</h1>

    <div style="display:grid;grid-template-columns:1fr 1fr;gap:20px">
      <CpuChart />
      <RamChart />
      <DiskChart />
      <NetChart />
    </div>

    <SpeedTest />
  </div>
</template>

<script setup>
import CpuChart from './components/CpuChart.vue';
import RamChart from './components/RamChart.vue';
import DiskChart from './components/DiskChart.vue';
import NetChart from './components/NetChart.vue';
import SpeedTest from './components/SpeedTest.vue';
import { initWebSocket } from './store.js';

initWebSocket();
</script>