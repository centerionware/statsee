import { state } from './state.js';

let ws;

export function initWS() {
  ws = new WebSocket(`ws://${location.host}/ws`);

  ws.onopen = () => {
    console.log('WebSocket connected');
  };

  ws.onmessage = (e) => {
    const msg = JSON.parse(e.data);

    if (msg.type === 'stats') {
      state.cpu = msg.cpu;
      state.ram.used = msg.ram.used;
      state.ram.free = msg.ram.free;
      state.disk.read = Object.values(msg.disk).reduce((sum, d) => sum + d.ReadBytes / 1024 / 1024, 0);
      state.disk.write = Object.values(msg.disk).reduce((sum, d) => sum + d.WriteBytes / 1024 / 1024, 0);
      state.net = { ...msg.net }; // replace object for reactivity
    } else if (msg.type === 'speedtest_update' || msg.type === 'speedtest_done') {
      state.speedTest.download = msg.download;
      state.speedTest.upload = msg.upload;
    }
  };

  ws.onclose = () => {
    console.log('WebSocket closed. Attempting reconnect in 2s...');
    setTimeout(initWS, 2000);
  };
}

export function startSpeedTest() {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'speedtest' }));
  }
}