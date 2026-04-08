<template>
  <div style="padding:20px">
    <h1>StatSee Dashboard</h1>

    <div style="display:grid;grid-template-columns:1fr 1fr;gap:20px">
      <CpuChart style="height: 350px;"/>
      <RamChart style="height: 350px;"/>
      <DiskChart style="height: 350px;"/>
      <NetChart style="height: 350px;"/>
      <NetTotals style="height: 350px;"/>
    </div>

    <SpeedTest style="height: 350px;"/>
  </div>
</template>

<script setup>
import { initWebSocket } from './store.js';
import CpuChart from './components/CpuChart.vue';
import RamChart from './components/RamChart.vue';
import DiskChart from './components/DiskChart.vue';
import NetChart from './components/NetworkChart.vue';
import NetTotals from './components/NetworkTotals.vue';
import SpeedTest from './components/SpeedTest.vue';


initWebSocket();
</script>