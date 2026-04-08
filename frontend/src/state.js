// frontend/src/state.js
import { reactive } from 'vue';

export const state = reactive({
  cpu: 0,
  ram: { used: 0, free: 0 },
  disk: {},          // all disk info keyed by device name
  selectedDisk: '',  // currently selected disk for DiskChart
  net: {},
  speedTest: { download: 0, upload: 0, type: '', stage: '' },
});