import { reactive } from 'vue';

export const store = reactive({
  stats: null,
  networkTotals: {},
  speedtest: { download: 0, upload: 0 }
});

export let ws = null;

export function initWebSocket() {
  ws = new WebSocket(`ws://${location.host}/ws`);

  ws.onmessage = (e) => {
    const msg = JSON.parse(e.data);
    if(msg.type === 'stats') {
      store.stats = msg;
    } else if(msg.type === 'speedtest_update' || msg.type === 'speedtest_done') {
      store.speedtest = { download: msg.download, upload: msg.upload };
    }
  };
}