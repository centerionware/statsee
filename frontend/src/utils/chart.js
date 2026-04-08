import Chart from 'chart.js/auto'

export function createLineChart(ctx, label) {
  return new Chart(ctx, {
    type: 'line',
    data: {
      labels: [],
      datasets: [{ label, data: [] }]
    },
    options: {
      animation: false,
      responsive: true
    }
  })
}

export function createMultiLineChart(ctx, labels) {
  return new Chart(ctx, {
    type: 'line',
    data: {
      labels: [],
      datasets: labels.map(l => ({ label: l, data: [] }))
    },
    options: {
      animation: false,
      responsive: true
    }
  })
}