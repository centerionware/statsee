<template>
  <div class="card md:col-span-2">
    <div class="flex justify-between items-center mb-2">
      <h2>JS Console</h2>
      <button @click="clearLogs">Clear</button>
    </div>

    <div class="console">
      <div
        v-for="(log, index) in logs"
        :key="index"
        :class="['log', log.type]"
      >
        <span class="time">[{{ log.time }}]</span>
        <span class="type">[{{ log.type.toUpperCase() }}]</span>
        <span class="msg">{{ log.message }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { consoleState } from '../console.js';

const logs = consoleState.logs;

function clearLogs() {
  consoleState.logs.splice(0);
}
</script>

<style scoped>
.console {
  background: #000;
  color: #0f0;
  font-family: monospace;
  font-size: 12px;
  height: 300px;
  overflow-y: auto;
  padding: 10px;
  border-radius: 6px;
}

.log {
  margin-bottom: 4px;
}

.log.warn {
  color: yellow;
}

.log.error {
  color: red;
}

.time {
  color: #888;
  margin-right: 6px;
}

.type {
  margin-right: 6px;
}

.msg {
  word-break: break-word;
}
</style>