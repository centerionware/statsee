// frontend/src/store.js
import { reactive } from 'vue';

export const store = reactive({
  stats: null,      // CPU, RAM, Disk, Net
  speedtest: null,  // Download/Upload
});

export function initWebSocket() {
  const ws = new WebSocket(`ws://${location.host}/ws`);

  ws.onmessage = e => {
    const msg = JSON.parse(e.data);
    if(msg.type === 'stats') {
      store.stats = msg;
    } else if(msg.type === 'speedtest_update' || msg.type === 'speedtest_done') {
      store.speedtest = msg;
    }
  };

  return ws;
}