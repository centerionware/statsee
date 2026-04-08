import { reactive } from 'vue';

export const consoleState = reactive({
  logs: [],
  paused: false
});

let buffer = [];

function formatMessage(args) {
  return args.map(a => {
    try {
      return typeof a === 'object'
        ? JSON.stringify(a, null, 2)
        : String(a);
    } catch {
      return String(a);
    }
  }).join(' ');
}

function pushLog(entry) {
  if (consoleState.paused) {
    buffer.push(entry);
    return;
  }

  consoleState.logs.unshift(entry);

  if (consoleState.logs.length > 200) {
    consoleState.logs.pop();
  }
}

export function togglePause() {
  consoleState.paused = !consoleState.paused;

  // flush buffer when resuming
  if (!consoleState.paused && buffer.length > 0) {
    consoleState.logs.unshift(...buffer.reverse());
    buffer = [];

    if (consoleState.logs.length > 200) {
      consoleState.logs.length = 200;
    }
  }
}

function addLog(type, args) {
  const entry = {
    type,
    message: formatMessage(args),
    time: new Date().toLocaleTimeString()
  };

  pushLog(entry);
}

export function initConsoleCapture() {
  const origLog = console.log;
  const origWarn = console.warn;
  const origError = console.error;

  console.log = (...args) => {
    addLog('log', args);
    origLog(...args);
  };

  console.warn = (...args) => {
    addLog('warn', args);
    origWarn(...args);
  };

  console.error = (...args) => {
    addLog('error', args);
    origError(...args);
  };

  window.addEventListener('error', (e) => {
    addLog('error', [e.message, e.filename, e.lineno]);
  });

  window.addEventListener('unhandledrejection', (e) => {
    addLog('error', ['Unhandled Promise:', e.reason]);
  });

  console.log('[ConsoleCapture] initialized');
}