// Dependency-free Canvas 2D chart helpers. The dashboard has no bundler, so
// this stays framework-free — plain functions operating on <canvas> nodes.

function chartsPrepareCanvas(canvas) {
  const ctx = canvas.getContext('2d');
  const dpr = window.devicePixelRatio || 1;
  const w = canvas.clientWidth || canvas.width;
  const h = canvas.clientHeight || canvas.height;
  canvas.width = Math.max(1, Math.round(w * dpr));
  canvas.height = Math.max(1, Math.round(h * dpr));
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, w, h);
  return { ctx, w, h };
}

// Compact single-series trend line, e.g. per-row latency sparkline.
function drawSparkline(canvas, series, opts = {}) {
  if (!canvas || typeof canvas.getContext !== 'function') return;
  const { ctx, w, h } = chartsPrepareCanvas(canvas);
  if (!Array.isArray(series) || series.length < 2) return;

  const min = Math.min(...series);
  const max = Math.max(...series);
  const range = max - min || 1;
  const pad = 2;
  const stepX = w / (series.length - 1);

  ctx.beginPath();
  series.forEach((v, i) => {
    const x = i * stepX;
    const y = h - pad - ((v - min) / range) * (h - pad * 2);
    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });
  ctx.strokeStyle = opts.color || '#ffb020';
  ctx.lineWidth = 1.5;
  ctx.lineJoin = 'round';
  ctx.lineCap = 'round';
  ctx.stroke();
}

// Multi-series line chart with a light horizontal grid, e.g. sent/received
// throughput over time for a tunnel.
function drawTimeSeries(canvas, seriesList, opts = {}) {
  if (!canvas || typeof canvas.getContext !== 'function') return;
  const { ctx, w, h } = chartsPrepareCanvas(canvas);

  const allValues = (seriesList || []).flatMap((s) => s.data || []);
  if (allValues.length < 2) return;

  const max = Math.max(...allValues, 1);
  const pad = 6;

  ctx.strokeStyle = opts.gridColor || 'rgba(140, 150, 165, 0.14)';
  ctx.lineWidth = 1;
  for (let i = 0; i <= 3; i++) {
    const y = pad + ((h - pad * 2) / 3) * i;
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(w, y);
    ctx.stroke();
  }

  seriesList.forEach(({ data, color }) => {
    if (!Array.isArray(data) || data.length < 2) return;
    const stepX = w / (data.length - 1);
    ctx.beginPath();
    data.forEach((v, i) => {
      const x = i * stepX;
      const y = h - pad - (v / max) * (h - pad * 2);
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.75;
    ctx.lineJoin = 'round';
    ctx.lineCap = 'round';
    ctx.stroke();
  });
}
