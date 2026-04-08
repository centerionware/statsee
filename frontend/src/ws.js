// frontend/src/ws.js
import { state } from './state.js';

let ws;

export function initWS() {
  const protocol = location.protocol === 'https:' ? 'wss' : 'ws';
  ws = new WebSocket(`${protocol}://${location.host}/ws`);

  ws.onopen = () => console.log('[WS] connected');

  ws.onmessage = (e) => {
    const msg = JSON.parse(e.data);
    console.log('[WS] message:', msg);

    if (msg.type === 'stats') {
      state.cpu = msg.cpu;
      state.ram = msg.ram;
      state.disk = msg.disk;
      if (!state.selectedDisk) {
        // default to first disk
        state.selectedDisk = Object.keys(msg.disk)[0] || '';
      }
      state.net = { ...msg.net };
    }

    if (
      msg.type === 'speedtest_progress' ||
      msg.type === 'speedtest_done' ||
      msg.type === 'speedtest_start'
    ) {
      state.speedTest = { ...state.speedTest, ...msg };
    }
  };

  ws.onclose = () => {
    console.log('[WS] closed. reconnecting...');
    setTimeout(initWS, 2000);
  };
}

export function startSpeedTest() {
  if (ws && ws.readyState === WebSocket.OPEN) {
    console.log('[WS] sending speedtest start');
    ws.send(JSON.stringify({ type: 'speedtest' }));
  } else {
    console.warn('[WS] not connected');
  }
}