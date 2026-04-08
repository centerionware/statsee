export async function getNetworkTotals() {
  const res = await fetch('/api/network-totals')
  return await res.json()
}