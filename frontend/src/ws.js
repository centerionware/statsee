import { reactive } from 'vue'

export const state = reactive({
  labels: [],
  cpu: [],
  ramUsed: [],
  ramFree: [],
  diskRead: [],
  diskWrite: [],
  netIn: [],
  netOut: [],
  speedtest: {
    download: 0,
    upload: 0,
    running: false
  }
})

const ws = new WebSocket(`ws://${location.host}/ws`)

ws.onmessage = (e) => {
  const msg = JSON.parse(e.data)
  const ts = new Date().toLocaleTimeString()

  if (msg.type === 'stats') {
    state.labels.push(ts)

    state.cpu.push(msg.cpu)
    state.ramUsed.push(msg.ram.used)
    state.ramFree.push(msg.ram.free)

    let read = 0, write = 0
    for (let k in msg.disk) {
      read += msg.disk[k].ReadBytes / 1024 / 1024
      write += msg.disk[k].WriteBytes / 1024 / 1024
    }

    let netIn = 0, netOut = 0
    for (let k in msg.net) {
      netIn += msg.net[k].rate_recv
      netOut += msg.net[k].rate_sent
    }

    state.diskRead.push(read)
    state.diskWrite.push(write)
    state.netIn.push(netIn)
    state.netOut.push(netOut)

    if (state.labels.length > 30) {
      for (let key in state) {
        if (Array.isArray(state[key])) state[key].shift()
      }
    }
  }

  if (msg.type === 'speedtest_update') {
    state.speedtest.download = msg.download
    state.speedtest.upload = msg.upload
    state.speedtest.running = true
  }

  if (msg.type === 'speedtest_done') {
    state.speedtest.download = msg.download
    state.speedtest.upload = msg.upload
    state.speedtest.running = false
  }
}

export function startSpeedTest() {
  ws.send(JSON.stringify({ type: 'speedtest' }))
}