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
    /* ⚠️ If embedding in a flex parent, use min-height:0 — see gotcha #10 */
    #network-container { flex: 1; min-height: 0; }
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

## Multi-tab architecture (for multiple diagrams in one file)

When building a file with multiple diagrams (e.g. one tab per service), use lazy initialization:

```js
// Store all diagram data as plain objects
const ALL_DIAGRAMS = {
  serviceA: { nodeData: {...}, edgeList: [...], defaultPositions: {...}, storageKey: 'a-v1', subtitle: '...' },
  serviceB: { ... },
};

// Per-tab runtime state
const instances = {}; // tabId → { network, nodes, edges, edgeInfo }
let activeTab = 'serviceA';

// Lazy init: called on first tab activation
function initDiagram(id) {
  if (instances[id]) return;
  const D = ALL_DIAGRAMS[id];
  const edgeInfo = {};

  const nodes = new vis.DataSet(Object.entries(D.nodeData).map(([nid, d]) => ({
    id: nid, label: d.label, ...nodeStyle(d.group),
  })));

  const edges = new vis.DataSet(D.edgeList.map(({ id, from, to, label, async: isAsync, desc }) => {
    edgeInfo[id] = { desc, async: isAsync };
    return buildEdge(id, from, to, label, isAsync);
  }));

  // dynamic height, inject positions, create network...
  const container = document.getElementById(`pane-${id}`).querySelector('.network-container');
  const network = new vis.Network(container, { nodes, edges }, { /* options */ });
  instances[id] = { network, nodes, edges, edgeInfo };
}

function switchTab(id) {
  activeTab = id;
  document.querySelectorAll('.tab-pane').forEach(p => p.classList.toggle('active', p.id === `pane-${id}`));
  initDiagram(id);
  updateZoomLabel();
}

// Toolbar functions act on the active network
function getActive() { return instances[activeTab]; }
function cycleRouting() {
  const inst = getActive(); if (!inst) return;
  // ... update inst.network
}
```

**CSS for multi-tab flex layout** — all ancestors of the container must have `min-height: 0` (see gotcha #10):

```css
body        { display: flex; flex-direction: column; height: 100vh; margin: 0; }
#main       { flex: 1; display: flex; overflow: hidden; min-height: 0; }
#content    { flex: 1; display: flex; flex-direction: column; overflow: hidden; min-height: 0; }
.tab-pane   { display: none; flex: 1; min-height: 0; overflow: hidden; }
.tab-pane.active { display: flex; }
.network-container { flex: 1; min-height: 0; }
```

**Bootstrap the first tab** inside `requestAnimationFrame` so layout is settled before vis.js measures the container:

```js
requestAnimationFrame(() => initDiagram('serviceA'));
```

After `initDiagram`, call `network.redraw()` + `network.fit()` as a safety net:

```js
instances[id] = { network, nodes, edges, edgeInfo };
setTimeout(() => { network.redraw(); network.fit(); }, 50);
```

**Reset layout** for multi-tab: clear only the active tab's localStorage key, then delete the instance and re-init:

```js
function resetLayout() {
  localStorage.removeItem(ALL_DIAGRAMS[activeTab].storageKey);
  delete instances[activeTab];
  document.getElementById(`pane-${activeTab}`).innerHTML = '<div class="network-container"></div>';
  switchTab(activeTab);
}
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

### 9. Cross-service queue naming (documentation trap)

Queue names don't always reflect the consuming service. Legacy queues may use a different service's prefix even though they're owned/consumed by another service. Example: `tapir-email-notifications-production.fifo` is owned by Tapir but consumed by Albatross — it was renamed before ownership was transferred. Always document this explicitly in the node's `purpose` field.

### 10. Multiple edges between the same two nodes — only one renders

vis.js renders only one edge when two edges share the same `from`/`to` pair. The second edge is silently dropped.

**Attempted workaround:** setting `smooth: { type: 'curvedCW', roundness: 0.25 }` on one edge and `curvedCCW` on the other to force them apart — this does not work because per-edge `smooth` overrides the global `setOptions` toggle (see pitfall #2), making the routing buttons stop working.

**Recommended approach:** merge parallel edges into a single edge. Use `\n` in the `label` to list both endpoints, and describe both in `desc`:

```js
{ id:'e21', from:'hyena', to:'albatross',
  label:'POST /emarsys/send\nGET /emarsys/is-subscribed',
  desc:'Hyena calls Albatross HTTP for two operations:\n• POST /emarsys/send — newsletter signup\n• GET /emarsys/is-subscribed — subscription status check' }
```

### 11. vis.js in a flex container produces a blank canvas

`vis.Network` reads `container.clientHeight` at init time. If the container is inside a flex chain and any ancestor is missing `min-height: 0`, the flex item defaults to `min-height: auto`, which can compute to `0` — vis.js then creates a zero-height canvas that renders as blank.

**Fix:** add `min-height: 0` to every element in the flex chain from `<body>` down to `.network-container`:

```css
body               { display: flex; flex-direction: column; height: 100vh; margin: 0; }
#wrapper           { flex: 1; display: flex; overflow: hidden; min-height: 0; }
#content           { flex: 1; display: flex; flex-direction: column; min-height: 0; }
.network-container { flex: 1; min-height: 0; }
```

Also wrap the first `initDiagram()` call in `requestAnimationFrame` so layout is computed before vis.js measures the container, and call `network.redraw()` + `network.fit()` after init as a safety net:

```js
// ✅ defer until layout is settled
requestAnimationFrame(() => initDiagram('serviceA'));

// inside initDiagram(), after instances[id] = ...
setTimeout(() => { network.redraw(); network.fit(); }, 50);
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

### Versioning layouts in git — "Copy layout" button

`localStorage` is ephemeral and per-browser. For diagrams that live in a git repo, `defaultPositions` in the source file is the **versioned layout**. The pattern for keeping them in sync:

1. Add a **📋 Copy layout** toolbar button.
2. On click, call `network.getPositions()`, format as a JS object literal, and write it to the clipboard.
3. The user pastes it over the `defaultPositions` block in the HTML source and commits.

```js
function copyLayout() {
  const pos = network.getPositions();
  const lines = Object.entries(pos)
    .map(([id, { x, y }]) => `    ${id}: { x: ${Math.round(x)}, y: ${Math.round(y)} },`);
  const text = `  defaultPositions: {\n${lines.join('\n')}\n  },`;
  navigator.clipboard.writeText(text)
    .then(() => showToast('Layout copied — paste over defaultPositions in the HTML'))
    .catch(() => {
      // Fallback for file:// origins where clipboard API is blocked
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.style.cssText = 'position:fixed;top:20px;left:50%;transform:translateX(-50%);width:500px;height:200px;z-index:9999;font-family:monospace;font-size:11px;';
      document.body.appendChild(ta);
      ta.select();
      showToast('Copy from the box, then click elsewhere to close');
      ta.addEventListener('blur', () => ta.remove());
    });
}
```

Always **bump `storageKey`** (e.g. `v2` → `v3`) when restructuring a diagram so that any stale positions saved in localStorage don't silently override your new `defaultPositions`.

---

## Container node (visual grouping box)

vis.js has no native compound/nested nodes. The workaround: insert a large, styled `box` node **first** in `nodeData` (so vis.js draws it behind everything else), then position regular nodes visually inside it.

```js
// In nodeData — MUST be first so it renders behind sub-nodes
handler_bg: {
  label: 'Handler',
  group: 'container',
  vis: {
    widthConstraint:  { minimum: 190, maximum: 190 },
    heightConstraint: { minimum: 460 },
    font: { vadjust: -210 },  // push label toward top of the box
  },
},
```

**Style the container group** with a transparent fill and dashed border:

```js
container: {
  shape: 'box',
  color: { background: 'rgba(13, 110, 253, 0.06)', border: '#0969da',
           highlight: { background: 'rgba(13, 110, 253, 0.1)', border: '#0550ae' } },
  font:  { color: '#0550ae', size: 13 },
  shapeProperties: { borderDashes: [6, 4] },
},
```

**Support per-node `vis` overrides in `initDiagram`** — spread after `nodeStyle()` so they take precedence:

```js
const nodes = new vis.DataSet(Object.entries(D.nodeData).map(([nid, d]) => ({
  id: nid, label: d.label, ...nodeStyle(d.group), ...(d.vis || {}),
})));
```

**Re-apply `heightConstraint` after the dynamic height update** — the dynamic update sets every node to `28 + 10 * degree`. A container node has degree 0, so it would be reset to 28px:

```js
// After the dynamic height update loop:
nodes.update(nodes.get().map(n => {
  const d = D.nodeData[n.id];
  if (d?.vis?.heightConstraint) return { id: n.id, heightConstraint: d.vis.heightConstraint };
  return null;
}).filter(Boolean));
```

**Label at the top** — use `font.vadjust` to shift the label up from the node center. For a 460px-tall box, `vadjust: -210` places the label near the top edge with a small margin.

**Known limitation:** dragging the container box does not move its sub-nodes. They are independent vis.js nodes. Inform the user, or mark the container `fixed: true` to prevent accidental drags (at the cost of it not being repositionable without editing source).

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

**Resizable sidebar:** add a 5px drag handle div between the canvas and the sidebar. The handle intercepts `mousedown` and adjusts the sidebar's `width` on `mousemove`:

```html
<!-- In HTML: handle goes between #content and #sidebar -->
<div id="sidebar-handle" title="Drag to resize sidebar"></div>
<aside id="sidebar">...</aside>
```

```css
#sidebar { width: 280px; min-width: 160px; max-width: 600px; border-left: none;
           overflow-y: auto; flex-shrink: 0; }
#sidebar-handle { width: 5px; background: transparent; cursor: col-resize; flex-shrink: 0;
                  border-left: 1px solid #d0d7de; transition: background 0.15s; }
#sidebar-handle:hover,
#sidebar-handle.dragging { background: #0969da; border-color: #0969da; }
```

```js
(function () {
  const handle  = document.getElementById('sidebar-handle');
  const sidebar = document.getElementById('sidebar');
  let dragging = false, startX = 0, startW = 0;

  handle.addEventListener('mousedown', e => {
    dragging = true; startX = e.clientX; startW = sidebar.offsetWidth;
    handle.classList.add('dragging');
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  });
  document.addEventListener('mousemove', e => {
    if (!dragging) return;
    const delta = startX - e.clientX;   // drag left → wider
    const min = parseInt(getComputedStyle(sidebar).minWidth);
    const max = parseInt(getComputedStyle(sidebar).maxWidth);
    sidebar.style.width = Math.min(max, Math.max(min, startW + delta)) + 'px';
  });
  document.addEventListener('mouseup', () => {
    if (!dragging) return;
    dragging = false; handle.classList.remove('dragging');
    document.body.style.cursor = ''; document.body.style.userSelect = '';
  });
})();
```

Key points:
- **Drag direction is inverted** (`startX - e.clientX`) because the handle is on the *left* edge of the sidebar — dragging left widens it.
- Set `border-left: none` on `#sidebar` and own the border on `#sidebar-handle` instead — avoids a double border when the handle is visible.
- `userSelect: none` on `<body>` during drag prevents text selection while resizing.

---

## Hub topology: group nodes

When a service has 15+ downstream connections (pure hub), individual nodes become unreadable. Replace them with **domain-group nodes** — each group node represents a cluster of related services and expands into a member table in the sidebar on click.

**Data structure:** add a `members` array to the node's `info`:

```js
g_catalog: { label: 'Catalog & Search', group: 'internal',
  info: { type: 'Service group · 5 services',
          purpose: 'Product catalog, search, translations, and recommendations.',
          members: [
            { name: 'Otter',    type: 'Internal', purpose: 'Product search.' },
            { name: 'Magpie',   type: 'Internal', purpose: 'Catalog data.' },
            { name: 'Anmitsu',  type: 'Internal', purpose: 'Translations.' },
            { name: 'Mink',     type: 'Internal', purpose: 'Recommendations.' },
            { name: 'FactFinder', type: 'External SaaS', purpose: 'Faceted search.' },
          ]}},
```

**Sidebar rendering:** after `sidebar.innerHTML = html`, append the member table if `i.members` is set:

```js
if (i.members && i.members.length) {
  const rows = i.members.map(m => `
    <tr>
      <td><strong>${m.name}</strong></td>
      <td><span style="font-size:11px;color:${m.type==='Internal'?'#0969da':'#e16f24'}">${m.type}</span></td>
      <td style="color:#57606a">${m.purpose}</td>
    </tr>`).join('');
  sidebar.innerHTML += `
    <p class="sidebar-section">MEMBER SERVICES (${i.members.length})</p>
    <table class="detail-table" style="width:100%">
      <thead><tr><th>Service</th><th>Type</th><th>Role</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>`;
}
```

**Edge labels** on group edges should be short summaries of the whole cluster's role (e.g. `'cart · orders\npricing'`) rather than a single service's label. Put the full breakdown in `desc`.

**When to use groups vs individual nodes:** use groups when a hub has ≥ 10 outbound edges that all fan out to the right — the label pile-up at the source node makes individual nodes unreadable regardless of layout. 6 group nodes with 1 edge each reads far better than 25 nodes with 25 crossing edges.

**Bump `storageKey`** when restructuring a diagram (e.g. `hyena-layout-v1` → `hyena-layout-v2`) so stale per-node positions from the old structure don't confuse the new layout.

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
