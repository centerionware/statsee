<template>
  <div class="card md:col-span-2 flex flex-col items-center">
    <h2>Speed Test</h2>

    <button @click="runSpeedTest" class="mb-4">
      Run Speed Test
    </button>

    <canvas ref="canvas" width="300" height="180"></canvas>

    <div class="mt-4 text-xl font-bold">
      {{ speedLabel }}
    </div>

    <div class="mt-2 text-lg">
      {{ resultText }}
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue';
import { state } from '../state.js';
import { startSpeedTest } from '../ws.js';

const canvas = ref(null);
const speedLabel = ref('0.00 MB/s');
const resultText = ref('');

let ctx;
let currentSpeed = 0;
let targetSpeed = 0;
let mode = 'idle'; // download | upload

const MAX_SPEED = 1000; // adjust for needle scaling

function runSpeedTest() {
  console.log('[UI] start speedtest');
  startSpeedTest();
  resultText.value = 'Running speed test...';
  speedLabel.value = '0.00 MB/s';
  currentSpeed = 0;
  targetSpeed = 0;
}

function drawGauge() {
  ctx.clearRect(0, 0, 300, 180);

  // arc
  ctx.beginPath();
  ctx.arc(150, 150, 120, Math.PI, 2 * Math.PI);
  ctx.lineWidth = 15;
  ctx.strokeStyle = '#ddd';
  ctx.stroke();

  // needle
  const angle = Math.PI + (currentSpeed / MAX_SPEED) * Math.PI;
  ctx.beginPath();
  ctx.moveTo(150, 150);
  ctx.lineTo(
    150 + 100 * Math.cos(angle),
    150 + 100 * Math.sin(angle)
  );
  ctx.lineWidth = 4;
  ctx.strokeStyle = mode === 'upload' ? 'red' : 'blue';
  ctx.stroke();

  // center dot
  ctx.beginPath();
  ctx.arc(150, 150, 5, 0, 2 * Math.PI);
  ctx.fillStyle = '#000';
  ctx.fill();
}

function animate() {
  currentSpeed += (targetSpeed - currentSpeed) * 0.1;
  drawGauge();
  requestAnimationFrame(animate);
}

onMounted(() => {
  ctx = canvas.value.getContext('2d');
  animate();
});

watch(() => state.speedTest, (msg) => {
  if (!msg) return;

  console.log('[UI] speedtest update:', msg);

  if (msg.type === 'speedtest_start') {
    resultText.value = 'Running speed test...';
  }

  if (msg.type === 'speedtest_progress') {
    if (msg.stage === 'download') {
      mode = 'download';
      targetSpeed = msg.download || 0;
      speedLabel.value = `${targetSpeed.toFixed(2)} MB/s`;
      resultText.value = `Downloading...`;
    }

    if (msg.stage === 'upload') {
      mode = 'upload';
      targetSpeed = msg.upload || 0;
      speedLabel.value = `${targetSpeed.toFixed(2)} MB/s`;
      resultText.value = `Uploading...`;
    }
  }

  if (msg.type === 'speedtest_done') {
    targetSpeed = 0;
    resultText.value =
      `Download: ${(msg.download || 0).toFixed(2)} MB/s, ` +
      `Upload: ${(msg.upload || 0).toFixed(2)} MB/s`;
    speedLabel.value = '0.00 MB/s';
  }

}, { deep: true });
</script>