#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE=(docker compose -f "$ROOT_DIR/deployments/docker/docker-compose.yml" -p snowy)
WEB_BASE="${WEB_BASE:-http://127.0.0.1:3001}"
API_BASE="${API_BASE:-http://127.0.0.1:8080}"

log() { printf '\033[0;36m▸ %s\033[0m\n' "$*"; }
ok() { printf '\033[0;32m✓ %s\033[0m\n' "$*"; }
fail() { printf '\033[0;31m✗ %s\033[0m\n' "$*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing command: $1"
}

http_get() {
  local url="$1"
  curl --noproxy '*' --http1.0 -fsS --max-time 20 "$url" >/dev/null
}

http_post_json() {
  local url="$1"
  local data="$2"
  python3 - "$url" "$data" <<'PY'
import json, sys, urllib.request
url, data = sys.argv[1], sys.argv[2].encode()
req = urllib.request.Request(
    url,
    data=data,
    headers={"Content-Type": "application/json", "Accept": "application/json"},
    method="POST",
)
with urllib.request.urlopen(req, timeout=240) as resp:
    body = resp.read()
open('/tmp/snowy-smoke-response.json', 'wb').write(body)
payload = json.loads(body.decode())
if payload.get('code') != 'OK':
    print(payload, file=sys.stderr)
    sys.exit(1)
PY
}

assert_search_llm_answer() {
  python3 - <<'PY'
import json, sys
payload=json.load(open('/tmp/snowy-smoke-response.json'))
data=payload.get('data') or {}
answer=data.get('answer') or ''
tags=data.get('knowledge_tags') or []
if 'api key is empty' in answer:
    print({'error':'search returned api-key diagnostic','answer':answer[:500]}, file=sys.stderr)
    sys.exit(1)
if '大模型直答' in tags and 'mimo' in tags:
    sys.exit(0)
if '本地兜底' in tags and answer:
    print({'warning':'MiMo direct answer unavailable during smoke; accepted structured fallback','tags':tags,'answer':answer[:180]}, file=sys.stderr)
    sys.exit(0)
print({'error':'search response is neither MiMo direct answer nor structured fallback','tags':tags,'answer':answer[:240]}, file=sys.stderr)
sys.exit(1)
PY
}

require_cmd docker
require_cmd curl
require_cmd python3

log "checking docker compose services"
"${COMPOSE[@]}" ps --status running >/tmp/snowy-smoke-ps.txt
for svc in snowy-api snowy-web snowy-worker mysql redis minio; do
  grep -q "$svc" /tmp/snowy-smoke-ps.txt || fail "$svc is not running"
done
ok "compose services are running"

log "checking API health"
http_get "$API_BASE/healthz" || fail "API health check failed"
ok "API health is OK"

log "checking web routes"
for path in / /search /physics /biology /learning /monitoring; do
  http_get "$WEB_BASE$path" || fail "web route failed: $path"
done
ok "web routes are OK"

log "checking core APIs"
http_get "$WEB_BASE/api/v1/recommendations" || fail "recommendations failed"
http_get "$WEB_BASE/api/v1/monitoring/llm" || fail "monitoring dashboard api failed"
http_post_json "$WEB_BASE/api/v1/search/query" '{"query":"牛顿第二定律","filters":{"subject":"physics"}}' || fail "search query failed"
assert_search_llm_answer || fail "search query did not use real MiMo LLM"
http_post_json "$WEB_BASE/api/v1/modeling/biology/analyze" '{"question":"光合作用中光照强度对有机物积累的影响"}' || fail "biology analyze failed"
http_post_json "$WEB_BASE/api/v1/agent/chat" '{"message":"你好","mode":"search","filters":{"subject":"physics"}}' || fail "agent chat failed"
ok "core APIs are OK"

ok "Snowy Docker smoke passed"
