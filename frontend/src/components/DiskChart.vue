<template><canvas ref="canvas"></canvas></template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { state } from '../ws'
import { createMultiLineChart } from '../utils/chart'

const canvas = ref()
let chart

onMounted(() => {
  chart = createMultiLineChart(canvas.value, ['Read MB', 'Write MB'])
})

watch(() => state.diskRead, () => {
  chart.data.labels = state.labels
  chart.data.datasets[0].data = state.diskRead
  chart.data.datasets[1].data = state.diskWrite
  chart.update()
}, { deep: true })
</script>