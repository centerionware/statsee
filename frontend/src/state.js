import { reactive } from 'vue';

export const state = reactive({
  cpu: 0,
  ram: { used: 0, free: 0 },
  disk: { read: 0, write: 0 },
  net: {},
  speedTest: { download: 0, upload: 0 },
});