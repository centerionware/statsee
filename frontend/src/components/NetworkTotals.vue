<template>
  <div v-html="html"></div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getNetworkTotals } from '../api'

const html = ref('Loading...')

async function load() {
  const totals = await getNetworkTotals()
  let out = ''
  for (let iface in totals) {
    const t = totals[iface]
    out += `<b>${iface}</b>: Today ${t.daily_in.toFixed(2)}GB / ${t.daily_out.toFixed(2)}GB<br>`
  }
  html.value = out
}

onMounted(() => {
  load()
  setInterval(load, 3000)
})
</script>