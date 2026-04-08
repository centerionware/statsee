import { reactive } from 'vue';

export const consoleState = reactive({
  logs: []
});

function addLog(type, args) {
  const message = args.map(a => {
    try {
      return typeof a === 'object' ? JSON.stringify(a) : String(a);
    } catch {
      return String(a);
    }
  }).join(' ');

  consoleState.logs.unshift({
    type,
    message,
    time: new Date().toLocaleTimeString()
  });

  // keep last 200 logs
  if (consoleState.logs.length > 200) {
    consoleState.logs.pop();
  }
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

  // capture runtime errors
  window.addEventListener('error', (e) => {
    addLog('error', [e.message, e.filename, e.lineno]);
  });

  // capture promise errors
  window.addEventListener('unhandledrejection', (e) => {
    addLog('error', ['Unhandled Promise:', e.reason]);
  });

  console.log('[ConsoleCapture] initialized');
}