<template>
  <div style="padding:20px">
    <h1>StatSee Dashboard</h1>

    <div style="display:grid;grid-template-columns:1fr 1fr;gap:20px">

      <Card title="CPU"><CpuChart/></Card>
      <Card title="RAM"><RamChart/></Card>
      <Card title="Disk"><DiskChart/></Card>

      <Card title="Network">
        <NetworkTotals/>
        <NetworkChart/>
      </Card>

      <Card title="Speed Test" style="grid-column: span 2;">
        <SpeedTest/>
      </Card>

    </div>
  </div>
</template>

<script setup>
import Card from './components/Card.vue'
import CpuChart from './components/CpuChart.vue'
import RamChart from './components/RamChart.vue'
import DiskChart from './components/DiskChart.vue'
import NetworkChart from './components/NetworkChart.vue'
import NetworkTotals from './components/NetworkTotals.vue'
import SpeedTest from './components/SpeedTest.vue'
</script>