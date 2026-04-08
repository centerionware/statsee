<template><canvas ref="canvas"></canvas></template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { state } from '../ws'
import { createMultiLineChart } from '../utils/chart'

const canvas = ref()
let chart

onMounted(() => {
  chart = createMultiLineChart(canvas.value, ['Ingress', 'Egress'])
})

watch(() => state.netIn, () => {
  chart.data.labels = state.labels
  chart.data.datasets[0].data = state.netIn
  chart.data.datasets[1].data = state.netOut
  chart.update()
}, { deep: true })
</script>