<template><canvas ref="canvas"></canvas></template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { state } from '../ws'
import { createLineChart } from '../utils/chart'

const canvas = ref()
let chart

onMounted(() => {
  chart = createLineChart(canvas.value, 'CPU %')
})

watch(() => state.cpu, () => {
  chart.data.labels = state.labels
  chart.data.datasets[0].data = state.cpu
  chart.update()
}, { deep: true })
</script>