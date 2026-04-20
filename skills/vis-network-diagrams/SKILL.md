---
name: vis-network-diagrams
description: Build interactive HTML network/graph diagrams using vis.js (vis-network). Use when asked to create architecture diagrams, connection maps, dependency graphs, or any node-edge diagram as a self-contained HTML file.
---

# vis-network-diagrams

## Purpose

Build polished, interactive, self-contained HTML diagrams using [vis-network](https://visjs.github.io/vis-network/docs/network/) (vis.js). The output is a single `.html` file the user can open in a browser — no build step, no framework, no server required (see caveat below).

## When To Use

- Architecture / service connection maps
- Dependency graphs
- Data flow diagrams
- Any node-edge graph where the user wants to pan, zoom, drag nodes, and click for details

---

## Quick-start template

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>My Diagram</title>
  <script src="https://unpkg.com/vis-network@9.1.9/standalone/umd/vis-network.min.js"></script>
  <style>
    /* ... styles ... */
    #toolbar { padding: 7px 16px; background: #fff; border-bottom: 1px solid #d0d7de; display: flex; gap: 6px; }
    #network-container { flex: 1; }
    #toast { position: fixed; bottom: 16px; left: 50%; transform: translateX(-50%);
             background: #24292f; color: #fff; font-size: 12px; padding: 6px 14px;
             border-radius: 6px; z-index: 100; opacity: 0; transition: opacity 0.2s; pointer-events: none; }
    #toast.show { opacity: 1; }
  </style>
</head>
<body>

<!-- IMPORTANT: toolbar and toast MUST be outside #network-container -->
<div id="toolbar">
  <button onclick="network.fit({animation:true})">⊙ Fit</button>
  <button id="routing-btn" onclick="cycleRouting()">⤡ Routing: Straight</button>
  <button onclick="changeZoom(-0.15)">−</button>
  <span id="zoom-label">100%</span>
  <button onclick="changeZoom(+0.15)">+</button>
  <button onclick="resetLayout()">↺ Reset layout</button>
</div>
<div id="toast"></div>

<div id="network-container" style="height: 600px;"></div>

<script>
const nodes = new vis.DataSet([/* ... */]);
const edges = new vis.DataSet([/* ... */]);

// ─── Dynamic node height ───────────────────────────────────────────────────
const inDeg = {}, outDeg = {};
edges.get().forEach(e => {
  inDeg[e.to]    = (inDeg[e.to]    || 0) + 1;
  outDeg[e.from] = (outDeg[e.from] || 0) + 1;
});
nodes.update(nodes.get().map(n => ({
  id: n.id,
  heightConstraint: { minimum: 28 + 10 * Math.max(inDeg[n.id] || 0, outDeg[n.id] || 0) },
})));

// ─── Layout persistence ────────────────────────────────────────────────────
const STORAGE_KEY = 'my-diagram-layout-v1';
const DEFAULT_POSITIONS = { /* node_id: { x, y }, ... */ };

function loadPositions() {
  try { return JSON.parse(localStorage.getItem(STORAGE_KEY)) || DEFAULT_POSITIONS; }
  catch { return DEFAULT_POSITIONS; }
}
function savePositions() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(network.getPositions()));
  showToast('Layout saved');
}
function resetLayout() { localStorage.removeItem(STORAGE_KEY); location.reload(); }

let toastTimer;
function showToast(msg) {
  const t = document.getElementById('toast');
  t.textContent = msg; t.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => t.classList.remove('show'), 1800);
}

// Inject saved positions into nodes BEFORE creating the network
const positions = loadPositions();
nodes.forEach(n => { const p = positions[n.id]; if (p) nodes.update({ id: n.id, x: p.x, y: p.y }); });

// ─── Network ───────────────────────────────────────────────────────────────
const network = new vis.Network(
  document.getElementById('network-container'),
  { nodes, edges },
  {
    layout: { hierarchical: { enabled: false } }, // false = free drag
    physics:     { enabled: false },
    interaction: { hover: true, dragNodes: true },
    edges:       { smooth: { enabled: false } },  // straight by default
  }
);

network.on('dragEnd', ({ nodes: moved }) => { if (moved.length > 0) savePositions(); });

// ─── Routing toggle ────────────────────────────────────────────────────────
const ROUTING_MODES = [
  { label: 'Straight',  smooth: { enabled: false } },
  { label: 'Auto',      smooth: { type: 'continuous', roundness: 0.5 } },
  { label: 'Curved ↻', smooth: { type: 'curvedCW',   roundness: 0.25 } },
  { label: 'Curved ↺', smooth: { type: 'curvedCCW',  roundness: 0.25 } },
];
let routingIndex = 0;
function cycleRouting() {
  routingIndex = (routingIndex + 1) % ROUTING_MODES.length;
  const mode = ROUTING_MODES[routingIndex];
  network.setOptions({ edges: { smooth: mode.smooth } });
  document.getElementById('routing-btn').textContent = `⤡ Routing: ${mode.label}`;
}

// ─── Zoom control ──────────────────────────────────────────────────────────
function updateZoomLabel() {
  document.getElementById('zoom-label').textContent = Math.round(network.getScale() * 100) + '%';
}
function changeZoom(delta) {
  const next = Math.max(0.05, Math.min(3, network.getScale() + delta));
  network.moveTo({ scale: next, animation: { duration: 200, easingFunction: 'easeInOutQuad' } });
  setTimeout(updateZoomLabel, 220);
}
network.on('zoom', updateZoomLabel);
network.once('afterDrawing', updateZoomLabel);
</script>
</body>
</html>
```

---

## Critical gotchas (learned the hard way)

### 1. vis.js clears everything inside its container on init

`new vis.Network(container, ...)` **deletes all children** of `container`. Any HTML you put inside `#network-container` (buttons, toast divs, etc.) will be gone after the network is created.

**Fix:** put all UI elements (toolbar, toast) **outside** `#network-container`. Use `position: fixed` for the toast.

### 2. Per-edge `smooth` in the DataSet overrides `setOptions` permanently

If you set `smooth` on an edge object when building the DataSet (e.g. in an `edge()` helper), vis.js stores it as a per-edge override. Later calls to `network.setOptions({ edges: { smooth: ... } })` only update the **default** for new edges — existing edges keep their stored value.

**Fix:** do **not** put `smooth` on individual edge objects. Leave it out entirely. Then `setOptions` controls all edges globally.

```js
// ✅ correct — no smooth property on the edge
function edge(id, from, to, label) {
  return { id, from, to, label, /* no smooth here */ };
}

// ✅ routing toggle works because there's no per-edge override
function cycleRouting() {
  network.setOptions({ edges: { smooth: ROUTING_MODES[routingIndex].smooth } });
}

// ❌ wrong — per-edge smooth blocks setOptions
function edge(...) {
  return { ..., smooth: { type: 'straightCross', roundness: 0 } };
}
```

### 3. `smooth: false` is invalid syntax

vis.js does not accept `smooth: false`. It silently ignores it or crashes.

```js
// ❌
smooth: false

// ✅
smooth: { enabled: false }
```

### 4. `smooth: { type: 'dynamic' }` requires physics

`dynamic` uses physics-computed control points. With `physics: { enabled: false }`, dynamic edges are visually identical to straight lines. Use `continuous`, `curvedCW`, or `curvedCCW` instead.

| Smooth type       | Needs physics? | Visual without physics |
|-------------------|---------------|------------------------|
| `dynamic`         | ✅ yes        | Straight (invisible curve) |
| `continuous`      | ❌ no         | Auto-curved by direction |
| `curvedCW`        | ❌ no         | Clockwise arc |
| `curvedCCW`       | ❌ no         | Counter-clockwise arc |
| `{ enabled: false }` | ❌ no    | Straight line |

### 5. Label position depends on node shape

Some shapes render the label **outside/below** the shape, not inside the fill:

| Shape    | Label position |
|----------|---------------|
| `box`    | Inside (always readable) ✅ |
| `ellipse`| Inside ✅ |
| `diamond`| Below the shape ⚠️ |
| `dot`    | Below the shape ⚠️ |
| `star`   | Below the shape ⚠️ |
| `triangle`| Below the shape ⚠️ |

Use `box` or `ellipse` when label readability matters.

### 6. `layout.hierarchical` constrains dragging to within columns

When `layout.hierarchical` is enabled, drag is locked to within the same level/column. For free drag, use:
```js
layout: { hierarchical: { enabled: false } }
```
And provide initial positions via `nodes.update()` **before** calling `new vis.Network()`.

### 7. `file://` CORS restriction

Browsers block `fetch()` and some external script loads on `file://` origins. Always serve the file from a local HTTP server:
```bash
python3 -m http.server 8080
# then open http://localhost:8080/diagram.html
```

### 8. `dot` shape label renders below the node — leave it empty for waypoints

`dot` nodes render their label text **below** the circle, outside the fill. If you set `label: '⤡'` on a waypoint node it will appear as a floating arrow icon that moves with the node and looks like a stray cursor.

**Fix:** set `label: ''` and use `title` for the hover tooltip instead.

```js
// ❌ label appears as a floating symbol below the dot
nodes.add({ id: 'wp', label: '⤡', shape: 'dot', title: 'Drag to reroute' });

// ✅ clean — tooltip still works on hover
nodes.add({ id: 'wp', label: '', shape: 'dot', title: 'Drag to reroute' });
```

---

## Waypoint nodes for back-edges

When a back-edge (e.g. A → B where B is to the left of A) would visually slice through other nodes, route it via a small draggable waypoint node placed above or below the main flow.

```js
// 1. Add a tiny dot node — no label, just a tooltip
nodes.add({
  id: 'wp_a_to_b',
  label: '',
  title: 'Routing waypoint — drag to reroute the A → B edge',
  shape: 'dot',
  size: 7,
  color: { background: '#d0d7de', border: '#8c959f',
           highlight: { background: '#e8ebef', border: '#57606a' } },
});

// 2. Split the logical edge into two segments through the waypoint
edges.add([
  { id: 'e_a_wp',  from: 'a', to: 'wp_a_to_b', label: 'your label', ...edgeStyle },
  { id: 'e_wp_b',  from: 'wp_a_to_b', to: 'b', label: 'your label', ...edgeStyle },
]);

// 3. Give it an initial position above/below the flow
DEFAULT_POSITIONS['wp_a_to_b'] = { x: midX, y: -600 };
```

Repeat the same label and description on both segments so clicking either edge shows the same info.

Handle waypoint clicks in your sidebar to avoid showing a broken node detail panel:
```js
network.on('click', ({ nodes: ns }) => {
  if (ns[0] === 'wp_a_to_b') {
    sidebar.innerHTML = '<p>Routing waypoint — drag to reroute.</p>';
  } else {
    showNode(ns[0]);
  }
});
```

---

## Layout persistence pattern

Inject saved positions **before** creating the network — not after:

```js
// ✅ correct order
const positions = loadPositions();
nodes.forEach(n => { const p = positions[n.id]; if (p) nodes.update({ id: n.id, x: p.x, y: p.y }); });
const network = new vis.Network(...);  // network reads x/y from DataSet

// ❌ wrong — network already placed nodes, update has no visual effect
const network = new vis.Network(...);
nodes.update({ id: 'x', x: 100, y: 100 });
```

**Important:** saved `localStorage` positions silently override `DEFAULT_POSITIONS`. If the user changes the defaults and wonders why nothing moved, they need to clear storage. Always provide a Reset button:

```js
function resetLayout() { localStorage.removeItem(STORAGE_KEY); location.reload(); }
```

---

## Dynamic node height

Scale node height to the number of connections so high-degree nodes stand out visually:

```js
const inDeg = {}, outDeg = {};
edges.get().forEach(e => {
  inDeg[e.to]    = (inDeg[e.to]    || 0) + 1;
  outDeg[e.from] = (outDeg[e.from] || 0) + 1;
});
nodes.update(nodes.get().map(n => ({
  id: n.id,
  heightConstraint: { minimum: 28 + 10 * Math.max(inDeg[n.id] || 0, outDeg[n.id] || 0) },
})));
```

Call this **after** the edges DataSet is built but **before** creating the network.

---

## Sidebar / detail panel pattern

```js
network.on('click', ({ nodes: clicked, edges: clickedEdges }) => {
  if (clicked.length > 0) showNode(clicked[0]);
  else if (clickedEdges.length > 0) showEdge(clickedEdges[0]);
});
```

Use a separate `nodeData` / `edgeInfo` object (keyed by id) to store rich metadata that won't bloat the vis.js DataSet.

---

## Recommended vis-network CDN

```html
<script src="https://unpkg.com/vis-network@9.1.9/standalone/umd/vis-network.min.js"></script>
```

Pin the version. The standalone UMD bundle includes everything — no need for separate `vis-data`.

---

## Debugging: routing toggle has no visual effect

If a routing button calls `setOptions` but edges don't visibly change, run this in the browser console:

```js
// 1. Check if edges have a stored per-edge smooth value
edges.get().slice(0, 3).forEach(e => console.log(e.id, e.smooth));
// If you see smooth values here (e.g. {type:'straightCross'}), they're blocking setOptions.

// 2. Confirm setOptions IS cycling (index changes, mode is correct)
console.log(routingIndex, ROUTING_MODES[routingIndex]);
cycleRouting();
console.log(routingIndex, ROUTING_MODES[routingIndex]);

// 3. Force-apply as a test — if THIS works, the fix is to remove smooth from edge definitions
edges.update(edges.get().map(e => ({ id: e.id, smooth: ROUTING_MODES[routingIndex].smooth })));
```

If step 3 works but step 2 doesn't, per-edge `smooth` in the DataSet is the culprit — remove it from the edge builder function (see gotcha #2 above).
