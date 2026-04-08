<template>
  <div style="background:#1f1f1f;border-radius:12px;padding:16px;">
    <h2 style="margin-bottom:10px">{{ title }}</h2>
    <slot />
  </div>
</template>

<script setup>
defineProps(['title'])
</script>