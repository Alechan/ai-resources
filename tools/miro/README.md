# Miro — Board Data Extraction

How to export the full content of a Miro board (shapes + connectors) as structured JSON using the official REST API.

---

## Prerequisites

- A Miro account (free tier is fine)
- The board must be **in an account where you can create an app** (apps cannot be created in managed/enterprise accounts with OAuth restrictions — copy the board to your personal account if needed)

---

## Step 1 — Get an API token

1. Go to [https://miro.com/app/settings/user-profile/apps](https://miro.com/app/settings/user-profile/apps)
2. Click **"Create new app"**
3. Give it any name (e.g. `board-exporter`)
4. Under **"Board content"** → enable **`boards:read`**
5. Click **"Install app and get OAuth token"**
6. Select the team that owns your board
7. Copy the **access token** (starts with `eyJtaXJv...`)

> ⚠️ Treat this token as a secret. **Revoke it** after use from the same settings page.

---

## Step 2 — Find your board ID

From the board URL:

```
https://miro.com/app/board/uXjVHfpbulc=/
                             ^^^^^^^^^^^^
                             This is your board ID (URL-encode the `=` as `%3D`)
```

Board ID for API calls: `uXjVHfpbulc%3D`

---

## Step 3 — Fetch all board items (shapes, text, frames)

```bash
TOKEN="your-access-token"
BOARD="uXjVHfpbulc%3D"  # URL-encoded board ID

# First page
curl -s "https://api.miro.com/v2/boards/${BOARD}/items?limit=50" \
  -H "Authorization: Bearer ${TOKEN}" \
  > /tmp/miro-items-page1.json

# Check total
jq '.total, (.data | length)' /tmp/miro-items-page1.json
```

If `total > 50`, paginate using the `cursor` field:

```bash
CURSOR=$(jq -r '.cursor' /tmp/miro-items-page1.json)

curl -s "https://api.miro.com/v2/boards/${BOARD}/items?limit=50&cursor=${CURSOR}" \
  -H "Authorization: Bearer ${TOKEN}" \
  >> /tmp/miro-items-page2.json
```

Or use this Python script to fetch all pages at once:

```python
import urllib.request, json, time

TOKEN = "your-access-token"
BOARD = "uXjVHfpbulc%3D"

all_items = []
cursor = None
while True:
    url = f"https://api.miro.com/v2/boards/{BOARD}/items?limit=50"
    if cursor:
        url += f"&cursor={cursor}"
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {TOKEN}"})
    with urllib.request.urlopen(req) as r:
        d = json.loads(r.read())
    all_items.extend(d['data'])
    cursor = d.get('cursor')
    if not cursor or len(all_items) >= d['total']:
        break
    time.sleep(0.3)  # be polite

with open('/tmp/miro-board.json', 'w') as f:
    json.dump(all_items, f, indent=2)

print(f"Saved {len(all_items)} items")
```

---

## Step 4 — Fetch all connectors

Connectors (the lines/arrows between shapes) are a separate endpoint:

```python
import urllib.request, json, time

TOKEN = "your-access-token"
BOARD = "uXjVHfpbulc%3D"

all_connectors = []
cursor = None
while True:
    url = f"https://api.miro.com/v2/boards/{BOARD}/connectors?limit=50"
    if cursor:
        url += f"&cursor={cursor}"
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {TOKEN}"})
    with urllib.request.urlopen(req) as r:
        d = json.loads(r.read())
    all_connectors.extend(d['data'])
    cursor = d.get('cursor')
    if not cursor or len(all_connectors) >= d['total']:
        break
    time.sleep(0.3)

with open('/tmp/miro-connectors.json', 'w') as f:
    json.dump(all_connectors, f, indent=2)

print(f"Saved {len(all_connectors)} connectors")
```

Each connector looks like:

```json
{
  "id": "3458764...",
  "startItem": { "id": "3458764...AAA" },
  "endItem":   { "id": "3458764...BBB" },
  "style": {
    "startStrokeCap": "none",
    "endStrokeCap": "open_arrow",
    "strokeColor": "#1a1a2e"
  }
}
```

- `endStrokeCap: "open_arrow"` → directed arrow (A → B)
- Both caps `"none"` → undirected line

---

## Step 5 — Build a connection graph

```python
import json, re
from collections import defaultdict

items = json.load(open('/tmp/miro-board.json'))
connectors = json.load(open('/tmp/miro-connectors.json'))

def strip_html(html):
    if not html:
        return ''
    text = re.sub(r'<[^>]+>', ' ', html)
    text = text.replace('&amp;', '&').replace('&nbsp;', ' ').replace('&#xFEFF;', '')
    return ' '.join(text.split())

# Build ID → label map
id_to_label = {}
for item in items:
    if item['type'] == 'shape':
        label = strip_html(item['data'].get('content', ''))
        id_to_label[item['id']] = label

# Build graph
graph = defaultdict(list)
for conn in connectors:
    start_id = conn.get('startItem', {}).get('id')
    end_id = conn.get('endItem', {}).get('id')
    start_label = id_to_label.get(start_id, f'[{start_id}]')
    end_label = id_to_label.get(end_id, f'[{end_id}]')
    end_cap = conn['style'].get('endStrokeCap', 'none')
    arrow = '→' if end_cap != 'none' else '—'
    graph[start_label].append((arrow, end_label))

for node in sorted(graph):
    for arrow, target in sorted(graph[node]):
        print(f"{node} {arrow} {target}")
```

---

## Notes

- The **clipboard `text/html` format** (`miro-data-v1`) is a custom binary encoding — not useful for parsing.
- The **HAR file** from Chrome DevTools does not capture WebSocket frames (where board sync happens) — not useful either.
- The **REST API is the only reliable extraction path** for structured data.
- Shape `style.fillColor` encodes technology type if the board uses a colour legend — map it manually from the diagram's legend.
- Some connectors may link to non-service shapes (legend boxes, group labels, sub-items). Filter by whether both endpoints resolve to known service names.

---

## Revoking the token

After extraction, revoke the token at:  
[https://miro.com/app/settings/user-profile/apps](https://miro.com/app/settings/user-profile/apps)  
→ find the app → **Delete app** (or rotate the token).
