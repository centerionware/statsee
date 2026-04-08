import { state } from './state.js';

let ws;

export function initWS() {
  ws = new WebSocket(`ws://${location.host}/ws`);

  ws.onopen = () => {
    console.log('[WS] connected');
  };

  ws.onmessage = (e) => {
    const msg = JSON.parse(e.data);
    console.log('[WS] message:', msg);

    if (msg.type === 'stats') {
      state.cpu = msg.cpu;
      state.ram.used = msg.ram.used;
      state.ram.free = msg.ram.free;
      state.disk.read = Object.values(msg.disk).reduce((sum, d) => sum + d.ReadBytes / 1024 / 1024, 0);
      state.disk.write = Object.values(msg.disk).reduce((sum, d) => sum + d.WriteBytes / 1024 / 1024, 0);
      state.net = { ...msg.net };
    }

    if (
      msg.type === 'speedtest_progress' ||
      msg.type === 'speedtest_done' ||
      msg.type === 'speedtest_start'
    ) {
      state.speedTest = {
        ...state.speedTest,
        ...msg,
      };
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