<template>
  <div style="padding:20px">
    <h1>StatSee Dashboard</h1>
    <div style="display:grid;grid-template-columns:1fr 1fr;gap:20px">
      <CpuChart />
      <RamChart />
      <DiskChart />
      <NetworkChart />
      <SpeedTest />
    </div>
  </div>
</template>

<script>
import CpuChart from './components/CpuChart.vue';
import RamChart from './components/RamChart.vue';
import DiskChart from './components/DiskChart.vue';
import NetworkChart from './components/NetworkChart.vue';
import SpeedTest from './components/SpeedTest.vue';
import { initWS } from './ws.js';

export default {
  components: { CpuChart, RamChart, DiskChart, NetworkChart, SpeedTest },
  setup() {
    initWS();
  },
};
</script>