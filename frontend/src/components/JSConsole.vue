<template>
  <div class="card md:col-span-2">
    <div class="flex justify-between items-center mb-2">
      <h2>JS Console</h2>

      <div class="flex gap-2">
        <button @click="toggle">
          {{ paused ? 'Resume' : 'Pause' }}
        </button>

        <button @click="clearLogs">Clear</button>
      </div>
    </div>

    <div ref="consoleEl" class="console">
      <div
        v-for="(log, index) in logs"
        :key="index"
        :class="['log', log.type]"
        @click="copyLog(log)"
      >
        <span class="time">[{{ log.time }}]</span>
        <span class="type">[{{ log.type.toUpperCase() }}]</span>
        <span class="msg">{{ log.message }}</span>
      </div>
    </div>

    <div v-if="copied" class="toast">
      Copied!
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue';
import { consoleState, togglePause } from '../console.js';

const logs = consoleState.logs;
const paused = consoleState.paused;

const consoleEl = ref(null);
const copied = ref(false);

function clearLogs() {
  consoleState.logs.splice(0);
}

function toggle() {
  togglePause();
}

async function copyLog(log) {
  const text = `[${log.time}] [${log.type.toUpperCase()}] ${log.message}`;

  // ✅ Try modern API first
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text);
      showCopied();
      return;
    } catch (err) {
      console.warn('Clipboard API failed, falling back');
    }
  }

  // ✅ Fallback (works on mobile/http)
  try {
    const textarea = document.createElement('textarea');
    textarea.value = text;

    // prevent scroll jump
    textarea.style.position = 'fixed';
    textarea.style.top = '-9999px';

    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();

    const success = document.execCommand('copy');
    document.body.removeChild(textarea);

    if (success) {
      showCopied();
    } else {
      throw new Error('execCommand failed');
    }
  } catch (err) {
    console.error('Copy fallback failed:', err);
    alert('Copy failed — long press to select manually');
  }
}

function showCopied() {
  copied.value = true;
  setTimeout(() => {
    copied.value = false;
  }, 1000);
}

// auto-scroll when new logs come in (only if not paused)
watch(logs, async () => {
  if (consoleState.paused) return;

  await nextTick();

  const el = consoleEl.value;
  if (el) {
    el.scrollTop = 0; // because we're using unshift (new logs on top)
  }
}, { deep: true });
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
  margin-bottom: 6px;
  cursor: pointer;
}

.log:hover {
  background: rgba(255,255,255,0.05);
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
  white-space: pre-wrap;
}

.toast {
  position: absolute;
  bottom: 10px;
  right: 10px;
  background: #333;
  color: white;
  padding: 6px 10px;
  border-radius: 4px;
  font-size: 12px;
}
</style>