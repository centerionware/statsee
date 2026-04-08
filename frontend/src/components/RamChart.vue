<template><canvas ref="canvas"></canvas></template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { state } from '../ws'
import { createMultiLineChart } from '../utils/chart'

const canvas = ref()
let chart

onMounted(() => {
  chart = createMultiLineChart(canvas.value, ['Used', 'Free'])
})

watch(() => state.ramUsed, () => {
  chart.data.labels = state.labels
  chart.data.datasets[0].data = state.ramUsed
  chart.data.datasets[1].data = state.ramFree
  chart.update()
}, { deep: true })
</script>