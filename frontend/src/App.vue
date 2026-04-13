<template>
  <div style="padding:20px">
    <h1>StatSee Dashboard</h1>

    <div style="display:grid;grid-template-columns:1fr 1fr;gap:20px">
      <CpuChart style="height: 400px;padding: 35px; margin: 10px;"/>
      <RamChart style="height: 400px;padding: 35px; margin: 10px;"/>
      <DiskChart style="height: 400px;padding: 35px; margin: 10px;"/>
      <NetChart style="height: 400px;padding: 35px; margin: 10px;"/>
      <JSConsole style="height: 400px;padding: 35px; margin: 10px;"/>
      <SpeedTest style="height: 400px;padding: 35px; margin: 10px;"/>
      <Card title="Network Totals" style="padding: 35px; margin: 10px;">
        <NetTotals />
      </Card>
    </div>

    
  </div>
</template>

<script setup>
import { initWS } from './ws.js';
import { initConsoleCapture } from './console.js';

import CpuChart from './components/CpuChart.vue';
import RamChart from './components/RamChart.vue';
import DiskChart from './components/DiskChart.vue';
import NetChart from './components/NetworkChart.vue';
import NetTotals from './components/NetworkTotals.vue';
import SpeedTest from './components/SpeedTest.vue';
import JSConsole from './components/JSConsole.vue'
initConsoleCapture();
initWS();
</script>