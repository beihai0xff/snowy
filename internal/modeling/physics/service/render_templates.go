package service

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/beihai0xff/snowy/internal/modeling/physics/domain"
)

func (s *serviceImpl) generateTemplateArtifact(
	sceneSpec *domain.SceneSpec,
	mode domain.RenderMode,
) (*domain.RenderArtifact, error) {
	props := cloneNumberMap(sceneSpec.DefaultProps)
	if sceneSpec.SceneType == "physics_force_3d" {
		ensureForce3DProps(props)
	}
	if strings.HasPrefix(sceneSpec.SceneType, "biology_") {
		ensureBiologyProps(props)
	}
	css := basePreviewCSS()

	var js string
	switch sceneSpec.SceneType {
	case "physics_projectile_3d":
		js = projectile3DScript(props)
	case "physics_projectile_2d":
		js = projectile2DScript(props)
	case "physics_force_3d":
		js = force3DScript(props)
	case "physics_force_diagram":
		js = forceDiagramScript(props)
	case "physics_generic_3d":
		js = generic3DScript(props)
	case "biology_photosynthesis_3d", "biology_cell_process_3d", "biology_concept_flow":
		js = biologyShowcaseScript(props, sceneSpec.SceneType)
	default:
		js = motion2DScript(props)
	}

	dependencies := []string{"native-html", "canvas"}
	allowedAPIs := []string{"requestAnimationFrame", "setTimeout", "postMessage", "CanvasRenderingContext2D"}
	if sceneSpec.SceneType == "physics_force_3d" {
		dependencies = []string{"native-html", "webgl", "canvas"}
		allowedAPIs = []string{"requestAnimationFrame", "setTimeout", "postMessage", "CanvasRenderingContext2D", "WebGLRenderingContext", "WebGL2RenderingContext"}
	}
	if strings.HasPrefix(sceneSpec.SceneType, "biology_") {
		dependencies = []string{"native-html", "canvas"}
		allowedAPIs = []string{"requestAnimationFrame", "setTimeout", "postMessage", "CanvasRenderingContext2D"}
	}

	indexHTML := buildPreviewHTML(sceneSpec.Title, css, js)
	artifact := &domain.RenderArtifact{
		SceneType:  sceneSpec.SceneType,
		RenderMode: mode,
		RenderManifest: &domain.RenderManifest{
			Entry:         "index.html",
			Framework:     "vanilla",
			Sandbox:       "iframe",
			RenderMode:    mode,
			MountSelector: "#snowy-preview-root",
			Dependencies:  dependencies,
			AllowedAPIs:   allowedAPIs,
			BlockedAPIs:   []string{"fetch", "XMLHttpRequest", "localStorage", "sessionStorage", "indexedDB", "document.cookie", "WebSocket", "navigator.sendBeacon"},
			InitialProps:  props,
		},
		CodeBundle: map[string]string{
			"index.html": indexHTML,
			"styles.css": css,
			"app.js":     js,
		},
		ResultSummary: sceneSpec.Summary,
	}

	if err := s.validateArtifact(artifact); err != nil {
		return nil, err
	}

	return artifact, nil
}

func ensureForce3DProps(props map[string]float64) {
	if _, ok := props["m"]; !ok {
		props["m"] = 2
	}
	if _, ok := props["a"]; !ok {
		props["a"] = 3
	}
	if _, ok := props["view_dimension"]; !ok {
		props["view_dimension"] = 3
	}
	if _, ok := props["camera_yaw"]; !ok {
		props["camera_yaw"] = 0.55
	}
	if _, ok := props["camera_pitch"]; !ok {
		props["camera_pitch"] = 0.42
	}
}

func ensureBiologyProps(props map[string]float64) {
	if _, ok := props["concept_count"]; !ok {
		props["concept_count"] = 4
	}
	if _, ok := props["relation_count"]; !ok {
		props["relation_count"] = 3
	}
	if _, ok := props["step_count"]; !ok {
		props["step_count"] = 3
	}
	if _, ok := props["animation_speed"]; !ok {
		props["animation_speed"] = 1
	}
}

func buildPreviewHTML(title, css, js string) string {
	return "<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"/><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"/>" +
		"<meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src data:; font-src data:; connect-src 'none';\"/>" +
		"<title>" + html.EscapeString(title) + "</title><style>" + css + "</style></head><body>" +
		"<div class=\"app-shell\"><div class=\"toolbar\"><div class=\"toolbar-title\">" + html.EscapeString(title) + "</div><div id=\"status\" class=\"status-badge\">初始化中…</div></div><div id=\"snowy-preview-root\" class=\"preview-root\"></div><div id=\"summary\" class=\"summary-panel\">等待渲染…</div></div>" +
		"<script>" + js + "</script></body></html>"
}

func basePreviewCSS() string {
	return strings.TrimSpace(`
html, body {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
  font-family: Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  background: #0b1220;
  color: #e6eefc;
}
* { box-sizing: border-box; }
body {
  background:
    radial-gradient(circle at top left, rgba(59, 130, 246, 0.22), transparent 32%),
    radial-gradient(circle at top right, rgba(16, 185, 129, 0.18), transparent 28%),
    linear-gradient(180deg, #0b1220 0%, #101826 100%);
}
.app-shell {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 100vh;
  padding: 16px;
}
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.toolbar-title {
  font-size: 16px;
  font-weight: 700;
  color: #f8fbff;
}
.status-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 32px;
  padding: 0 12px;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.14);
  border: 1px solid rgba(148, 163, 184, 0.18);
  color: #cbd5e1;
  font-size: 12px;
}
.preview-root {
  position: relative;
  flex: 1;
  min-height: 360px;
  overflow: hidden;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.96), rgba(17, 24, 39, 0.96));
  box-shadow: 0 20px 45px rgba(15, 23, 42, 0.25);
}
.summary-panel {
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.14);
  background: rgba(15, 23, 42, 0.52);
  padding: 12px 14px;
  font-size: 13px;
  line-height: 1.7;
  color: #dbeafe;
}
canvas {
  display: block;
  width: 100%;
  height: 100%;
}

.force3d-shell {
  position: absolute;
  inset: 0;
  overflow: hidden;
}
.force3d-canvas,
.force3d-labels {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}
.force3d-canvas { z-index: 1; }
.force3d-labels { z-index: 2; cursor: grab; }
.force3d-labels:active { cursor: grabbing; }
.view-switch {
  position: absolute;
  top: 14px;
  right: 14px;
  z-index: 5;
  display: inline-flex;
  gap: 6px;
  padding: 5px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.76);
  border: 1px solid rgba(148, 163, 184, 0.22);
  backdrop-filter: blur(10px);
}
.view-switch button {
  border: 0;
  border-radius: 999px;
  padding: 6px 12px;
  color: #cbd5e1;
  background: transparent;
  font-weight: 700;
  cursor: pointer;
}
.view-switch button.active {
  color: #04111f;
  background: linear-gradient(135deg, #67e8f9, #34d399);
}
.force3d-hint {
  position: absolute;
  left: 16px;
  bottom: 14px;
  z-index: 4;
  padding: 6px 10px;
  border-radius: 999px;
  color: #dbeafe;
  background: rgba(15, 23, 42, 0.62);
  border: 1px solid rgba(148, 163, 184, 0.18);
  font-size: 12px;
}
.metric-text {
  fill: #dbeafe;
  font-size: 13px;
  font-family: Inter, sans-serif;
}
.grid-line {
  stroke: rgba(148, 163, 184, 0.18);
  stroke-width: 1;
}
.axis-line {
  stroke: rgba(226, 232, 240, 0.75);
  stroke-width: 1.5;
}
`)
}

func runtimeWrapper(props map[string]float64, renderBody string) string {
	propsJSON, _ := json.Marshal(props)
	template := `(() => {
  const INITIAL_PROPS = __INITIAL_PROPS__;
  let props = { ...INITIAL_PROPS };
  const root = document.getElementById('snowy-preview-root');
  const summaryEl = document.getElementById('summary');
  const statusEl = document.getElementById('status');
  function post(type, payload = {}) {
    try {
      parent.postMessage({ source: 'snowy-preview', type, ...payload }, '*');
    } catch (_) {}
  }
  function numberValue(name, fallback) {
    const raw = Number(props[name]);
    return Number.isFinite(raw) ? raw : fallback;
  }
  function setSummary(text) {
    if (summaryEl) summaryEl.textContent = text;
  }
  function setStatus(text) {
    if (statusEl) statusEl.textContent = text;
  }
  function handleError(error) {
    const message = error instanceof Error ? error.message : String(error);
    setStatus('渲染失败');
    setSummary(message);
    post('error', { message });
  }
  window.addEventListener('message', (event) => {
    if (!event.data || typeof event.data !== 'object') return;
    if (event.data.type === 'snowy:update-props' && event.data.props) {
      props = { ...props, ...event.data.props };
      try {
        renderScene();
        setStatus('参数已同步到预览');
        post('preview', { status: 'updated' });
      } catch (error) {
        handleError(error);
      }
      return;
    }
    if (event.data.type === 'snowy:ping') {
      post('preview', { status: 'ready' });
    }
  });
  window.addEventListener('resize', () => {
    try {
      renderScene();
    } catch (error) {
      handleError(error);
    }
  });
  __RENDER_BODY__
  try {
    renderScene();
    setStatus('浏览器渲染就绪');
    post('preview', { status: 'ready' });
  } catch (error) {
    handleError(error);
  }
})();`
	template = strings.ReplaceAll(template, "__INITIAL_PROPS__", string(propsJSON))
	template = strings.ReplaceAll(template, "__RENDER_BODY__", renderBody)
	return template
}

func projectile2DScript(props map[string]float64) string {
	body := `
function renderScene() {
  root.innerHTML = '<canvas id="scene"></canvas>';
  const canvas = document.getElementById('scene');
  const ctx = canvas.getContext('2d');
  const width = Math.max(root.clientWidth, 360);
  const height = Math.max(root.clientHeight, 380);
  const ratio = window.devicePixelRatio || 1;
  canvas.width = width * ratio;
  canvas.height = height * ratio;
  ctx.setTransform(ratio, 0, 0, ratio, 0, 0);

  const v0 = numberValue('v0', 20);
  const angleDeg = numberValue('angle_deg', 45);
  const totalT = Math.max(0.3, numberValue('t', 2));
  const g = numberValue('g', 9.8);
  const angle = angleDeg * Math.PI / 180;
  const points = [];
  for (let i = 0; i <= 80; i += 1) {
    const tt = totalT * (i / 80);
    const x = v0 * Math.cos(angle) * tt;
    const y = v0 * Math.sin(angle) * tt - 0.5 * g * tt * tt;
    points.push({ x, y });
  }
  const maxX = Math.max(...points.map((p) => p.x), 1);
  const maxY = Math.max(...points.map((p) => p.y), 1);
  const minY = Math.min(...points.map((p) => p.y), 0);
  const pad = 42;
  const scaleX = (width - pad * 2) / maxX;
  const scaleY = (height - pad * 2) / Math.max(maxY - minY, 1);

  ctx.clearRect(0, 0, width, height);
  ctx.fillStyle = '#0f172a';
  ctx.fillRect(0, 0, width, height);
  ctx.strokeStyle = 'rgba(148, 163, 184, 0.18)';
  ctx.lineWidth = 1;
  for (let i = 0; i < 6; i += 1) {
    const y = pad + ((height - pad * 2) / 5) * i;
    ctx.beginPath();
    ctx.moveTo(pad, y);
    ctx.lineTo(width - pad, y);
    ctx.stroke();
  }
  for (let i = 0; i < 6; i += 1) {
    const x = pad + ((width - pad * 2) / 5) * i;
    ctx.beginPath();
    ctx.moveTo(x, pad);
    ctx.lineTo(x, height - pad);
    ctx.stroke();
  }

  const baseline = height - pad - (0 - minY) * scaleY;
  ctx.strokeStyle = 'rgba(226, 232, 240, 0.8)';
  ctx.lineWidth = 1.5;
  ctx.beginPath();
  ctx.moveTo(pad, baseline);
  ctx.lineTo(width - pad + 8, baseline);
  ctx.stroke();
  ctx.beginPath();
  ctx.moveTo(pad, height - pad + 8);
  ctx.lineTo(pad, pad - 8);
  ctx.stroke();

  ctx.strokeStyle = '#38bdf8';
  ctx.lineWidth = 3;
  ctx.beginPath();
  points.forEach((point, index) => {
    const px = pad + point.x * scaleX;
    const py = baseline - point.y * scaleY;
    if (index === 0) ctx.moveTo(px, py);
    else ctx.lineTo(px, py);
  });
  ctx.stroke();

  const last = points[points.length - 1];
  const lastX = pad + last.x * scaleX;
  const lastY = baseline - last.y * scaleY;
  ctx.fillStyle = '#f97316';
  ctx.beginPath();
  ctx.arc(lastX, lastY, 6, 0, Math.PI * 2);
  ctx.fill();

  ctx.fillStyle = '#dbeafe';
  ctx.font = '13px Inter, sans-serif';
  ctx.fillText('x', width - pad + 12, baseline + 4);
  ctx.fillText('y', pad - 10, pad - 12);
  ctx.fillText('起点', pad + 6, baseline - 10);
  ctx.fillText('末位置', lastX + 8, lastY - 8);

  const peak = Math.max(...points.map((p) => p.y));
  setSummary('当前预览通过浏览器 Canvas 渲染抛体轨迹，可直接拖动参数观察变化。水平位移 ' + last.x.toFixed(2) + ' m，最高点 ' + peak.toFixed(2) + ' m。');
}
`
	return runtimeWrapper(props, body)
}

func projectile3DScript(props map[string]float64) string {
	body := `
function renderScene() {
  root.innerHTML = '<canvas id="scene"></canvas>';
  const canvas = document.getElementById('scene');
  const ctx = canvas.getContext('2d');
  const width = Math.max(root.clientWidth, 360);
  const height = Math.max(root.clientHeight, 380);
  const ratio = window.devicePixelRatio || 1;
  canvas.width = width * ratio;
  canvas.height = height * ratio;
  ctx.setTransform(ratio, 0, 0, ratio, 0, 0);

  const v0 = numberValue('v0', 20);
  const angleDeg = numberValue('angle_deg', 45);
  const totalT = Math.max(0.3, numberValue('t', 2));
  const g = numberValue('g', 9.8);
  const angle = angleDeg * Math.PI / 180;
  const rawPoints = [];
  for (let i = 0; i <= 72; i += 1) {
    const tt = totalT * (i / 72);
    const x = v0 * Math.cos(angle) * tt;
    const y = v0 * Math.sin(angle) * tt - 0.5 * g * tt * tt;
    const z = x * 0.35;
    rawPoints.push({ x, y, z });
  }
  const maxX = Math.max(...rawPoints.map((p) => p.x), 1);
  const maxY = Math.max(...rawPoints.map((p) => p.y), 1);
  const maxZ = Math.max(...rawPoints.map((p) => p.z), 1);
  const pad = 54;
  const scale = Math.min((width - pad * 2) / Math.max(maxX, 1), (height - pad * 2) / Math.max(maxY + maxZ * 0.4, 1));
  const originX = pad;
  const originY = height - pad;

  function project(point) {
    const px = originX + point.x * scale + point.z * scale * 0.55;
    const py = originY - point.y * scale - point.z * scale * 0.28;
    return { x: px, y: py };
  }

  ctx.clearRect(0, 0, width, height);
  ctx.fillStyle = '#0f172a';
  ctx.fillRect(0, 0, width, height);
  ctx.strokeStyle = 'rgba(148, 163, 184, 0.16)';
  ctx.lineWidth = 1;

  for (let i = 0; i <= 5; i += 1) {
    const gx = originX + (maxX / 5) * i * scale;
    ctx.beginPath();
    ctx.moveTo(gx, originY);
    ctx.lineTo(gx + maxZ * scale * 0.55, originY - maxZ * scale * 0.28);
    ctx.stroke();
  }

  ctx.strokeStyle = 'rgba(226, 232, 240, 0.85)';
  ctx.lineWidth = 1.6;
  const xAxisEnd = project({ x: maxX * 1.05, y: 0, z: 0 });
  const yAxisEnd = project({ x: 0, y: maxY * 1.2, z: 0 });
  const zAxisEnd = project({ x: 0, y: 0, z: maxZ * 1.2 });
  ctx.beginPath(); ctx.moveTo(originX, originY); ctx.lineTo(xAxisEnd.x, xAxisEnd.y); ctx.stroke();
  ctx.beginPath(); ctx.moveTo(originX, originY); ctx.lineTo(yAxisEnd.x, yAxisEnd.y); ctx.stroke();
  ctx.beginPath(); ctx.moveTo(originX, originY); ctx.lineTo(zAxisEnd.x, zAxisEnd.y); ctx.stroke();

  ctx.strokeStyle = '#22d3ee';
  ctx.lineWidth = 3;
  ctx.beginPath();
  rawPoints.forEach((point, index) => {
    const projected = project(point);
    if (index === 0) ctx.moveTo(projected.x, projected.y);
    else ctx.lineTo(projected.x, projected.y);
  });
  ctx.stroke();

  const last = rawPoints[rawPoints.length - 1];
  const projectedLast = project(last);
  ctx.fillStyle = '#f97316';
  ctx.beginPath();
  ctx.arc(projectedLast.x, projectedLast.y, 6, 0, Math.PI * 2);
  ctx.fill();

  ctx.fillStyle = '#e2e8f0';
  ctx.font = '13px Inter, sans-serif';
  ctx.fillText('X', xAxisEnd.x + 8, xAxisEnd.y + 4);
  ctx.fillText('Y', yAxisEnd.x - 8, yAxisEnd.y - 12);
  ctx.fillText('Z', zAxisEnd.x + 6, zAxisEnd.y - 4);
  setSummary('当前预览使用浏览器 Canvas 绘制轻量 3D 轨迹投影，用于观察平抛过程在空间坐标系中的走势。终点投影 x=' + last.x.toFixed(2) + ' m，z=' + last.z.toFixed(2) + ' m。');
}
`
	return runtimeWrapper(props, body)
}

func force3DScript(props map[string]float64) string {
	body := `
let force3dAnimationFrame = 0;
let force3dDrag = false;
let force3dLastX = 0;
let force3dLastY = 0;
let force3dYaw = numberValue('camera_yaw', 0.55);
let force3dPitch = numberValue('camera_pitch', 0.42);

function mat4Perspective(fovy, aspect, near, far) {
  const f = 1 / Math.tan(fovy / 2);
  const nf = 1 / (near - far);
  return [f / aspect,0,0,0, 0,f,0,0, 0,0,(far + near) * nf,-1, 0,0,(2 * far * near) * nf,0];
}
function mat4Multiply(a, b) {
  const out = new Array(16).fill(0);
  for (let col = 0; col < 4; col += 1) {
    for (let row = 0; row < 4; row += 1) {
      out[col * 4 + row] = a[0 * 4 + row] * b[col * 4 + 0] + a[1 * 4 + row] * b[col * 4 + 1] + a[2 * 4 + row] * b[col * 4 + 2] + a[3 * 4 + row] * b[col * 4 + 3];
    }
  }
  return out;
}
function mat4Translate(x, y, z) {
  return [1,0,0,0, 0,1,0,0, 0,0,1,0, x,y,z,1];
}
function mat4RotateY(rad) {
  const c = Math.cos(rad); const s = Math.sin(rad);
  return [c,0,-s,0, 0,1,0,0, s,0,c,0, 0,0,0,1];
}
function mat4RotateX(rad) {
  const c = Math.cos(rad); const s = Math.sin(rad);
  return [1,0,0,0, 0,c,s,0, 0,-s,c,0, 0,0,0,1];
}
function transformPoint(m, p) {
  const x = p[0], y = p[1], z = p[2], w = 1;
  const tx = x*m[0] + y*m[4] + z*m[8] + w*m[12];
  const ty = x*m[1] + y*m[5] + z*m[9] + w*m[13];
  const tz = x*m[2] + y*m[6] + z*m[10] + w*m[14];
  const tw = x*m[3] + y*m[7] + z*m[11] + w*m[15];
  return { x: tx / tw, y: ty / tw, z: tz / tw };
}
function compileShader(gl, type, source) {
  const shader = gl.createShader(type);
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) throw new Error(gl.getShaderInfoLog(shader) || 'WebGL shader compile failed');
  return shader;
}
function createProgram(gl) {
  const vs = compileShader(gl, gl.VERTEX_SHADER, 'attribute vec3 aPosition;attribute vec3 aColor;uniform mat4 uMatrix;varying vec3 vColor;void main(){gl_Position=uMatrix*vec4(aPosition,1.0);vColor=aColor;}');
  const fs = compileShader(gl, gl.FRAGMENT_SHADER, 'precision mediump float;varying vec3 vColor;void main(){gl_FragColor=vec4(vColor,1.0);}');
  const program = gl.createProgram();
  gl.attachShader(program, vs); gl.attachShader(program, fs); gl.linkProgram(program);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) throw new Error(gl.getProgramInfoLog(program) || 'WebGL program link failed');
  return program;
}
function cubeVertices(scale) {
  const sx = 1.15 * scale, sy = 0.5 * scale, sz = 0.7 * scale;
  const c1 = [0.18,0.45,0.95], c2 = [0.10,0.28,0.66], c3 = [0.28,0.65,1.0];
  const p = [[-sx,-sy,-sz],[sx,-sy,-sz],[sx,sy,-sz],[-sx,sy,-sz],[-sx,-sy,sz],[sx,-sy,sz],[sx,sy,sz],[-sx,sy,sz]];
  const faces = [[0,1,2,0,2,3,c2],[4,6,5,4,7,6,c3],[0,4,5,0,5,1,c1],[3,2,6,3,6,7,c3],[1,5,6,1,6,2,c1],[0,3,7,0,7,4,c2]];
  const out = [];
  faces.forEach((f) => { for (let i = 0; i < 6; i += 1) out.push(...p[f[i]], ...f[6]); });
  return out;
}
function lineVertices(m, a) {
  const force = Math.abs(m * a);
  const fLen = Math.min(4.3, 1.25 + force / 10);
  const aLen = Math.min(3.8, 1 + Math.abs(a) / 12);
  const grid = [];
  for (let i = -5; i <= 5; i += 1) {
    grid.push(-5, -0.52, i, 0.22,0.31,0.46, 5, -0.52, i, 0.22,0.31,0.46);
    grid.push(i, -0.52, -5, 0.22,0.31,0.46, i, -0.52, 5, 0.22,0.31,0.46);
  }
  const axes = [0,-0.5,0, 0.93,0.25,0.25, 4.6,-0.5,0, 0.93,0.25,0.25, 0,-0.5,0, 0.30,0.84,0.42, 0,2.8,0, 0.30,0.84,0.42, 0,-0.5,0, 0.38,0.66,1.0, 0,-0.5,4.2, 0.38,0.66,1.0];
  const dir = a >= 0 ? 1 : -1;
  const fEnd = dir * fLen;
  const fx = 1.15 + fEnd;
  const ay = 0.86 + aLen;
  const arrows = [
    1.15,0.18,0, 1.0,0.82,0.20, fx,0.18,0, 1.0,0.82,0.20,
    fx,0.18,0, 1.0,0.82,0.20, fx - dir*0.35,0.34,0.18, 1.0,0.82,0.20,
    fx,0.18,0, 1.0,0.82,0.20, fx - dir*0.35,0.02,-0.18, 1.0,0.82,0.20,
    0,0.86,0, 0.16,0.75,1.0, 0,ay,0, 0.16,0.75,1.0,
    0,ay,0, 0.16,0.75,1.0, 0.20,ay-0.35,0.16, 0.16,0.75,1.0,
    0,ay,0, 0.16,0.75,1.0, -0.20,ay-0.35,-0.16, 0.16,0.75,1.0
  ];
  return grid.concat(axes, arrows);
}
function draw2D(labelCanvas, w, h, m, a) {
  const ctx = labelCanvas.getContext('2d');
  ctx.clearRect(0,0,w,h);
  ctx.fillStyle = '#0f172a'; ctx.fillRect(0,0,w,h);
  const ground = h * 0.68;
  ctx.strokeStyle = 'rgba(148,163,184,.25)'; ctx.lineWidth = 1;
  for (let x = 40; x < w; x += 40) { ctx.beginPath(); ctx.moveTo(x, ground); ctx.lineTo(x + 18, h - 28); ctx.stroke(); }
  ctx.strokeStyle = 'rgba(226,232,240,.65)'; ctx.lineWidth = 2; ctx.beginPath(); ctx.moveTo(34, ground); ctx.lineTo(w - 34, ground); ctx.stroke();
  const bw = 150, bh = 82, bx = w * 0.28, by = ground - bh;
  const g = ctx.createLinearGradient(bx, by, bx + bw, by + bh); g.addColorStop(0, '#60a5fa'); g.addColorStop(1, '#1d4ed8');
  ctx.fillStyle = g; ctx.fillRect(bx, by, bw, bh); ctx.strokeStyle = '#bfdbfe'; ctx.strokeRect(bx, by, bw, bh);
  ctx.fillStyle = '#eff6ff'; ctx.font = 'bold 17px Inter, sans-serif'; ctx.fillText('m=' + m.toFixed(1) + ' kg', bx + 26, by + 47);
  function arrow(x1,y1,x2,y2,color,text){ const ang=Math.atan2(y2-y1,x2-x1); ctx.strokeStyle=color;ctx.fillStyle=color;ctx.lineWidth=5;ctx.beginPath();ctx.moveTo(x1,y1);ctx.lineTo(x2,y2);ctx.stroke();ctx.beginPath();ctx.moveTo(x2,y2);ctx.lineTo(x2-14*Math.cos(ang-Math.PI/6),y2-14*Math.sin(ang-Math.PI/6));ctx.lineTo(x2-14*Math.cos(ang+Math.PI/6),y2-14*Math.sin(ang+Math.PI/6));ctx.closePath();ctx.fill();ctx.font='13px Inter, sans-serif';ctx.fillText(text,x2+8,y2-6); }
  const f = m * a;
  const dir = f >= 0 ? 1 : -1;
  const startX = dir > 0 ? bx + bw : bx;
  arrow(startX, by+bh/2, startX+Math.max(48, Math.min(210, Math.abs(f)*9))*dir, by+bh/2, '#f59e0b', 'F=' + f.toFixed(1) + ' N');
  arrow(bx+bw/2, by, bx+bw/2, by-Math.max(54, Math.min(150, Math.abs(a)*8)), '#22d3ee', 'a=' + a.toFixed(1) + ' m/s²');
  ctx.fillStyle = '#cbd5e1'; ctx.font = '13px Inter, sans-serif'; ctx.fillText('2D fallback：参数更新仍通过 postMessage 同步。', 18, 28);
}
function renderScene() {
  if (force3dAnimationFrame) cancelAnimationFrame(force3dAnimationFrame);
  root.innerHTML = '<div class="force3d-shell"><canvas id="force3dCanvas" class="force3d-canvas"></canvas><canvas id="force3dLabels" class="force3d-labels"></canvas><div class="view-switch"><button data-view="3">3D</button><button data-view="2">2D</button></div><div class="force3d-hint">拖拽旋转视角 · 参数滑块实时同步</div></div>';
  const canvas = document.getElementById('force3dCanvas');
  const labels = document.getElementById('force3dLabels');
  const width = Math.max(root.clientWidth, 360); const height = Math.max(root.clientHeight, 380);
  const ratio = window.devicePixelRatio || 1;
  [canvas, labels].forEach((el) => { el.width = width * ratio; el.height = height * ratio; el.style.width = width + 'px'; el.style.height = height + 'px'; });
  const labelCtx = labels.getContext('2d'); labelCtx.setTransform(ratio, 0, 0, ratio, 0, 0);
  const m = Math.max(0.1, numberValue('m', 2)); const a = numberValue('a', 3); const force = m * a;
  const view = numberValue('view_dimension', 3) >= 2.5 ? 3 : 2;
  document.querySelectorAll('.view-switch button').forEach((btn) => { btn.classList.toggle('active', Number(btn.dataset.view) === view); btn.onclick = () => { props.view_dimension = Number(btn.dataset.view); renderScene(); post('preview', { status: 'updated' }); }; });
  labels.onpointerdown = (event) => { force3dDrag = true; force3dLastX = event.clientX; force3dLastY = event.clientY; labels.setPointerCapture(event.pointerId); };
  labels.onpointermove = (event) => { if (!force3dDrag || view !== 3) return; force3dYaw += (event.clientX - force3dLastX) * 0.01; force3dPitch = Math.max(-0.15, Math.min(1.1, force3dPitch + (event.clientY - force3dLastY) * 0.008)); force3dLastX = event.clientX; force3dLastY = event.clientY; };
  labels.onpointerup = () => { force3dDrag = false; };
  labels.onpointercancel = () => { force3dDrag = false; };
  if (view === 2) { canvas.style.display = 'none'; draw2D(labels, width, height, m, a); setSummary('2D fallback 视图：F = ma = ' + force.toFixed(2) + ' N。点击 3D 可切回 WebGL 模型。'); return; }
  canvas.style.display = 'block';
  const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
  if (!gl) { props.view_dimension = 2; canvas.style.display = 'none'; draw2D(labels, width, height, m, a); setStatus('WebGL 不可用，已切 2D'); return; }
  gl.viewport(0, 0, width * ratio, height * ratio);
  const program = createProgram(gl); gl.useProgram(program);
  const posLoc = gl.getAttribLocation(program, 'aPosition'); const colorLoc = gl.getAttribLocation(program, 'aColor'); const matrixLoc = gl.getUniformLocation(program, 'uMatrix');
  const cubeBuffer = gl.createBuffer(); const lineBuffer = gl.createBuffer();
  const cubeData = new Float32Array(cubeVertices(Math.min(1.35, Math.max(0.75, m / 3)))); const lineData = new Float32Array(lineVertices(m, a));
  const perspective = mat4Perspective(Math.PI / 4, width / height, 0.1, 100);
  function bind(buffer, data) { gl.bindBuffer(gl.ARRAY_BUFFER, buffer); gl.bufferData(gl.ARRAY_BUFFER, data, gl.STATIC_DRAW); gl.enableVertexAttribArray(posLoc); gl.vertexAttribPointer(posLoc, 3, gl.FLOAT, false, 24, 0); gl.enableVertexAttribArray(colorLoc); gl.vertexAttribPointer(colorLoc, 3, gl.FLOAT, false, 24, 12); }
  function drawLabels(matrix) {
    labelCtx.clearRect(0,0,width,height); labelCtx.fillStyle = '#dbeafe'; labelCtx.font = '13px Inter, sans-serif';
    const project = (p) => { const q = transformPoint(matrix, p); return { x: (q.x * 0.5 + 0.5) * width, y: (-q.y * 0.5 + 0.5) * height }; };
    const pF = project([Math.min(5.2, 1.25 + Math.abs(force) / 8) * (force >= 0 ? 1 : -1), 0.35, 0]);
    const pA = project([0, Math.min(4.7, 1.6 + Math.abs(a) / 10), 0]); const pM = project([0,0.2,0]);
    labelCtx.fillStyle = '#fbbf24'; labelCtx.fillText('F=' + force.toFixed(1) + ' N', pF.x + 8, pF.y);
    labelCtx.fillStyle = '#67e8f9'; labelCtx.fillText('a=' + a.toFixed(1) + ' m/s²', pA.x + 8, pA.y);
    labelCtx.fillStyle = '#eff6ff'; labelCtx.font = 'bold 15px Inter, sans-serif'; labelCtx.fillText('m=' + m.toFixed(1) + ' kg', pM.x - 42, pM.y);
    labelCtx.fillStyle = '#cbd5e1'; labelCtx.font = '12px Inter, sans-serif'; labelCtx.fillText('X', project([4.8,-0.5,0]).x, project([4.8,-0.5,0]).y); labelCtx.fillText('Y', project([0,3.0,0]).x, project([0,3.0,0]).y); labelCtx.fillText('Z', project([0,-0.5,4.4]).x, project([0,-0.5,4.4]).y);
  }
  function frame() {
    const camera = mat4Multiply(mat4Translate(0, -0.35, -9.5), mat4Multiply(mat4RotateX(force3dPitch), mat4RotateY(force3dYaw)));
    const matrix = mat4Multiply(perspective, camera);
    gl.clearColor(0.04, 0.07, 0.12, 1); gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT); gl.enable(gl.DEPTH_TEST);
    gl.uniformMatrix4fv(matrixLoc, false, new Float32Array(matrix)); bind(lineBuffer, lineData); gl.lineWidth(2); gl.drawArrays(gl.LINES, 0, lineData.length / 6);
    const cubeMatrix = mat4Multiply(matrix, mat4Translate(0, 0, 0)); gl.uniformMatrix4fv(matrixLoc, false, new Float32Array(cubeMatrix)); bind(cubeBuffer, cubeData); gl.drawArrays(gl.TRIANGLES, 0, cubeData.length / 6);
    drawLabels(matrix); force3dAnimationFrame = requestAnimationFrame(frame);
  }
  frame();
  setStatus('3D WebGL 模型就绪');
  setSummary('3D WebGL 视图：蓝色方块表示物体，地面网格和 XYZ 坐标轴提供空间参照；黄色为合力 F=ma=' + force.toFixed(2) + ' N，青色为加速度矢量。可拖拽旋转视角，或切换 2D fallback。');
}
`
	return runtimeWrapper(props, body)
}

func biologyShowcaseScript(props map[string]float64, sceneType string) string {
	body := fmt.Sprintf(`
let biologyAnimationFrame = 0;
let biologyPaused = false;
const biologySceneType = %q;
function renderScene() {
  if (biologyAnimationFrame) cancelAnimationFrame(biologyAnimationFrame);
  root.innerHTML = '<div class="force3d-shell"><canvas id="biologyStage" class="force3d-canvas"></canvas><div class="view-switch"><button data-action="toggle">暂停</button></div><div class="force3d-hint">粒子流动 · 阶段演示 · 概念标签</div></div>';
  const canvas = document.getElementById('biologyStage');
  const ctx = canvas.getContext('2d');
  const width = Math.max(root.clientWidth, 360);
  const height = Math.max(root.clientHeight, 380);
  const ratio = window.devicePixelRatio || 1;
  canvas.width = width * ratio;
  canvas.height = height * ratio;
  canvas.style.width = width + 'px';
  canvas.style.height = height + 'px';
  ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
  const conceptCount = Math.max(3, numberValue('concept_count', 4));
  const relationCount = Math.max(2, numberValue('relation_count', 3));
  const stepCount = Math.max(2, numberValue('step_count', 3));
  const speed = Math.max(0.2, numberValue('animation_speed', 1));
  const labels = biologySceneType === 'biology_photosynthesis_3d'
    ? ['光子', '叶绿体', 'CO₂', 'H₂O', 'O₂', '糖分子']
    : biologySceneType === 'biology_cell_process_3d'
      ? ['细胞膜', '通道蛋白', '线粒体', '底物', '能量', '产物']
      : ['核心概念', '影响因素', '过程阶段', '结果表现', '反馈调节', '知识网络'];
  const particles = Array.from({ length: 42 }, (_, i) => ({ seed: i * 17.31, lane: i %% Math.min(6, labels.length) }));
  const toggle = document.querySelector('[data-action="toggle"]');
  toggle.onclick = () => { biologyPaused = !biologyPaused; toggle.textContent = biologyPaused ? '播放' : '暂停'; };
  let time = 0;
  function glowCircle(x, y, r, color) {
    const g = ctx.createRadialGradient(x, y, 0, x, y, r * 2.8);
    g.addColorStop(0, color);
    g.addColorStop(0.42, color.replace('0.95', '0.28'));
    g.addColorStop(1, 'rgba(15,23,42,0)');
    ctx.fillStyle = g; ctx.beginPath(); ctx.arc(x, y, r * 2.8, 0, Math.PI * 2); ctx.fill();
    ctx.fillStyle = color; ctx.beginPath(); ctx.arc(x, y, r, 0, Math.PI * 2); ctx.fill();
  }
  function drawArrow(x1, y1, x2, y2, color) {
    const a = Math.atan2(y2 - y1, x2 - x1);
    ctx.strokeStyle = color; ctx.lineWidth = 2.4; ctx.beginPath(); ctx.moveTo(x1, y1); ctx.lineTo(x2, y2); ctx.stroke();
    ctx.fillStyle = color; ctx.beginPath(); ctx.moveTo(x2, y2); ctx.lineTo(x2 - 10 * Math.cos(a - Math.PI / 6), y2 - 10 * Math.sin(a - Math.PI / 6)); ctx.lineTo(x2 - 10 * Math.cos(a + Math.PI / 6), y2 - 10 * Math.sin(a + Math.PI / 6)); ctx.closePath(); ctx.fill();
  }
  function frame() {
    if (!biologyPaused) time += 0.016 * speed;
    ctx.clearRect(0, 0, width, height);
    ctx.fillStyle = '#07111f'; ctx.fillRect(0, 0, width, height);
    const bg = ctx.createRadialGradient(width * 0.5, height * 0.42, 20, width * 0.5, height * 0.42, Math.max(width, height) * 0.7);
    bg.addColorStop(0, 'rgba(20,184,166,0.32)'); bg.addColorStop(0.45, 'rgba(59,130,246,0.14)'); bg.addColorStop(1, 'rgba(2,6,23,0.96)');
    ctx.fillStyle = bg; ctx.fillRect(0, 0, width, height);
    ctx.strokeStyle = 'rgba(148,163,184,0.13)'; ctx.lineWidth = 1;
    for (let x = -40 + ((time * 28) %% 40); x < width + 40; x += 40) { ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x + height * 0.25, height); ctx.stroke(); }
    const cx = width * 0.5, cy = height * 0.46;
    ctx.save(); ctx.translate(cx, cy); ctx.rotate(Math.sin(time * 0.35) * 0.04);
    ctx.strokeStyle = 'rgba(45,212,191,0.76)'; ctx.lineWidth = 3;
    ctx.fillStyle = biologySceneType === 'biology_photosynthesis_3d' ? 'rgba(34,197,94,0.22)' : 'rgba(96,165,250,0.20)';
    ctx.beginPath(); ctx.ellipse(0, 0, width * 0.22, height * 0.18, -0.22, 0, Math.PI * 2); ctx.fill(); ctx.stroke();
    ctx.strokeStyle = 'rgba(167,243,208,0.35)'; ctx.lineWidth = 2;
    for (let i = -2; i <= 2; i += 1) { ctx.beginPath(); ctx.ellipse(i * width * 0.055, 0, width * 0.055, height * 0.13, -0.22, 0, Math.PI * 2); ctx.stroke(); }
    ctx.restore();
    const nodeRadius = Math.min(width, height) * 0.034;
    const nodes = labels.slice(0, Math.min(labels.length, Math.max(4, conceptCount))).map((label, i, arr) => {
      const angle = -Math.PI * 0.82 + (Math.PI * 1.64 * i) / Math.max(1, arr.length - 1);
      return { label, x: cx + Math.cos(angle) * width * 0.34, y: cy + Math.sin(angle) * height * 0.30 };
    });
    nodes.forEach((node, i) => { if (i < nodes.length - 1) drawArrow(node.x, node.y, nodes[i+1].x, nodes[i+1].y, 'rgba(125,211,252,0.58)'); });
    particles.forEach((p) => {
      const from = nodes[p.lane %% nodes.length], to = nodes[(p.lane + 1) %% nodes.length];
      const phase = (time * 0.18 + p.seed * 0.017) %% 1;
      const x = from.x + (to.x - from.x) * phase + Math.sin(time + p.seed) * 8;
      const y = from.y + (to.y - from.y) * phase + Math.cos(time * 0.7 + p.seed) * 8;
      glowCircle(x, y, 3.2 + (p.lane %% 3), p.lane %% 2 === 0 ? 'rgba(250,204,21,0.95)' : 'rgba(45,212,191,0.95)');
    });
    nodes.forEach((node, i) => {
      glowCircle(node.x, node.y, nodeRadius, i %% 2 === 0 ? 'rgba(34,197,94,0.95)' : 'rgba(96,165,250,0.95)');
      ctx.fillStyle = '#ecfeff'; ctx.font = 'bold 13px Inter, sans-serif'; ctx.textAlign = 'center'; ctx.fillText(node.label, node.x, node.y + nodeRadius + 18);
    });
    const activeStep = Math.floor(time * 0.8) %% Math.max(1, stepCount);
    ctx.textAlign = 'left'; ctx.fillStyle = 'rgba(15,23,42,0.72)'; ctx.fillRect(18, 18, Math.min(360, width - 36), 112);
    ctx.strokeStyle = 'rgba(125,211,252,0.34)'; ctx.strokeRect(18, 18, Math.min(360, width - 36), 112);
    ctx.fillStyle = '#e0f2fe'; ctx.font = 'bold 16px Inter, sans-serif'; ctx.fillText(biologySceneType === 'biology_photosynthesis_3d' ? '光合作用能量转换' : biologySceneType === 'biology_cell_process_3d' ? '细胞过程动态演示' : '概念关系动态网络', 34, 48);
    ctx.font = '13px Inter, sans-serif'; ctx.fillStyle = '#bae6fd'; ctx.fillText('阶段 ' + (activeStep + 1) + '/' + stepCount + ' · 概念 ' + conceptCount + ' · 关系 ' + relationCount, 34, 76);
    ctx.fillStyle = '#a7f3d0'; ctx.fillText('粒子表示物质/能量流，箭头表示转化或影响路径', 34, 102);
    biologyAnimationFrame = requestAnimationFrame(frame);
  }
  frame();
  setStatus('生物动态演示就绪');
  setSummary('当前预览使用浏览器 Canvas 生成炫酷生物教学演示：粒子流表示物质或能量迁移，发光节点表示关键概念，阶段面板随动画展示过程拆解。');
}
`, sceneType)
	return runtimeWrapper(props, body)
}

func forceDiagramScript(props map[string]float64) string {
	body := `
function renderScene() {
  root.innerHTML = '<canvas id="scene"></canvas>';
  const canvas = document.getElementById('scene');
  const ctx = canvas.getContext('2d');
  const width = Math.max(root.clientWidth, 360);
  const height = Math.max(root.clientHeight, 360);
  const ratio = window.devicePixelRatio || 1;
  canvas.width = width * ratio;
  canvas.height = height * ratio;
  ctx.setTransform(ratio, 0, 0, ratio, 0, 0);

  const m = Math.max(0.1, numberValue('m', 2));
  const a = numberValue('a', 3);
  const force = m * a;
  const blockWidth = 120;
  const blockHeight = 74;
  const blockX = width * 0.28;
  const blockY = height * 0.52;
  const arrowScale = Math.min(150, Math.max(36, force * 8));

  ctx.clearRect(0, 0, width, height);
  ctx.fillStyle = '#0f172a';
  ctx.fillRect(0, 0, width, height);

  ctx.strokeStyle = 'rgba(148, 163, 184, 0.2)';
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(32, blockY + blockHeight + 30);
  ctx.lineTo(width - 32, blockY + blockHeight + 30);
  ctx.stroke();

  ctx.fillStyle = '#1d4ed8';
  ctx.fillRect(blockX, blockY, blockWidth, blockHeight);
  ctx.fillStyle = '#eff6ff';
  ctx.font = 'bold 18px Inter, sans-serif';
  ctx.fillText('m=' + m.toFixed(1) + 'kg', blockX + 24, blockY + 42);

  function arrow(x1, y1, x2, y2, color, label) {
    const angle = Math.atan2(y2 - y1, x2 - x1);
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    ctx.lineWidth = 4;
    ctx.beginPath();
    ctx.moveTo(x1, y1);
    ctx.lineTo(x2, y2);
    ctx.stroke();
    ctx.beginPath();
    ctx.moveTo(x2, y2);
    ctx.lineTo(x2 - 12 * Math.cos(angle - Math.PI / 6), y2 - 12 * Math.sin(angle - Math.PI / 6));
    ctx.lineTo(x2 - 12 * Math.cos(angle + Math.PI / 6), y2 - 12 * Math.sin(angle + Math.PI / 6));
    ctx.closePath();
    ctx.fill();
    ctx.font = '13px Inter, sans-serif';
    ctx.fillText(label, x2 + 8, y2 - 4);
  }

  arrow(blockX + blockWidth, blockY + blockHeight / 2, blockX + blockWidth + arrowScale, blockY + blockHeight / 2, '#22c55e', 'F=' + force.toFixed(1) + 'N');
  arrow(blockX + blockWidth / 2, blockY, blockX + blockWidth / 2, blockY - 90, '#38bdf8', 'a=' + a.toFixed(1) + 'm/s²');
  arrow(blockX + blockWidth / 2, blockY + blockHeight, blockX + blockWidth / 2, blockY + blockHeight + 90, '#fb7185', 'G≈' + (m * 9.8).toFixed(1) + 'N');

  setSummary('当前预览使用浏览器 Canvas 直接绘制受力示意图，合力 F = ma = ' + force.toFixed(2) + ' N，可通过滑块实时改变箭头长度。');
}
`
	return runtimeWrapper(props, body)
}

func motion2DScript(props map[string]float64) string {
	body := `
function renderScene() {
  root.innerHTML = '<canvas id="scene"></canvas>';
  const canvas = document.getElementById('scene');
  const ctx = canvas.getContext('2d');
  const width = Math.max(root.clientWidth, 360);
  const height = Math.max(root.clientHeight, 360);
  const ratio = window.devicePixelRatio || 1;
  canvas.width = width * ratio;
  canvas.height = height * ratio;
  ctx.setTransform(ratio, 0, 0, ratio, 0, 0);

  const x0 = numberValue('x0', 0);
  const v = numberValue('v', numberValue('v0', 5));
  const a = numberValue('a', 0);
  const t = Math.max(0.5, numberValue('t', 5));
  const x = x0 + v * t + 0.5 * a * t * t;
  const vmax = Math.max(Math.abs(x0), Math.abs(x), 10);
  const pad = 40;
  const lineY = height * 0.55;
  const px = (value) => pad + ((value + vmax) / (vmax * 2)) * (width - pad * 2);

  ctx.clearRect(0, 0, width, height);
  ctx.fillStyle = '#0f172a';
  ctx.fillRect(0, 0, width, height);

  ctx.strokeStyle = 'rgba(148, 163, 184, 0.22)';
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(pad, lineY);
  ctx.lineTo(width - pad, lineY);
  ctx.stroke();

  for (let i = 0; i <= 4; i += 1) {
    const tx = pad + ((width - pad * 2) / 4) * i;
    ctx.beginPath();
    ctx.moveTo(tx, lineY - 10);
    ctx.lineTo(tx, lineY + 10);
    ctx.stroke();
  }

  const startX = px(x0);
  const endX = px(x);
  ctx.strokeStyle = '#38bdf8';
  ctx.lineWidth = 6;
  ctx.beginPath();
  ctx.moveTo(startX, lineY);
  ctx.lineTo(endX, lineY);
  ctx.stroke();

  ctx.fillStyle = '#f97316';
  ctx.beginPath();
  ctx.arc(endX, lineY, 12, 0, Math.PI * 2);
  ctx.fill();

  ctx.fillStyle = '#e2e8f0';
  ctx.font = '13px Inter, sans-serif';
  ctx.fillText('x0=' + x0.toFixed(1) + 'm', startX - 24, lineY - 20);
  ctx.fillText('x=' + x.toFixed(1) + 'm', endX - 20, lineY - 24);
  ctx.fillText('v=' + v.toFixed(1) + 'm/s', pad, 36);
  ctx.fillText('a=' + a.toFixed(1) + 'm/s²', pad + 130, 36);
  ctx.fillText('t=' + t.toFixed(1) + 's', pad + 270, 36);

  setSummary('当前预览使用浏览器 Canvas 渲染一维运动场景。位移 x = ' + x.toFixed(2) + ' m，可直接通过参数滑块更新预览。');
}
`
	return runtimeWrapper(props, body)
}

func generic3DScript(props map[string]float64) string {
	body := `
let animationFrameId = 0;
function renderScene() {
  if (animationFrameId) cancelAnimationFrame(animationFrameId);
  root.innerHTML = '<canvas id="scene"></canvas>';
  const canvas = document.getElementById('scene');
  const ctx = canvas.getContext('2d');
  const width = Math.max(root.clientWidth, 360);
  const height = Math.max(root.clientHeight, 360);
  const ratio = window.devicePixelRatio || 1;
  canvas.width = width * ratio;
  canvas.height = height * ratio;
  ctx.setTransform(ratio, 0, 0, ratio, 0, 0);

  const size = Math.max(40, numberValue('size', 110));
  const speed = numberValue('rotation_speed', 0.018);
  const amplitude = Math.max(0.2, numberValue('t', 4));
  let angle = 0;
  const vertices = [
    [-1, -1, -1], [1, -1, -1], [1, 1, -1], [-1, 1, -1],
    [-1, -1, 1], [1, -1, 1], [1, 1, 1], [-1, 1, 1]
  ];
  const edges = [[0,1],[1,2],[2,3],[3,0],[4,5],[5,6],[6,7],[7,4],[0,4],[1,5],[2,6],[3,7]];

  function rotate([x, y, z], ry, rx) {
    const cosY = Math.cos(ry); const sinY = Math.sin(ry);
    const cosX = Math.cos(rx); const sinX = Math.sin(rx);
    let dx = x * cosY - z * sinY;
    let dz = x * sinY + z * cosY;
    let dy = y * cosX - dz * sinX;
    dz = y * sinX + dz * cosX;
    return [dx, dy, dz];
  }

  function project(vertex) {
    const [x, y, z] = vertex;
    const depth = 3.8 + z + amplitude * 0.2;
    return {
      x: width / 2 + (x * size * 1.8) / depth,
      y: height / 2 + (y * size * 1.8) / depth
    };
  }

  function frame() {
    angle += speed;
    ctx.clearRect(0, 0, width, height);
    ctx.fillStyle = '#0f172a';
    ctx.fillRect(0, 0, width, height);

    const transformed = vertices.map((vertex) => rotate(vertex, angle, angle * 0.65));
    const projected = transformed.map(project);

    ctx.strokeStyle = 'rgba(56, 189, 248, 0.85)';
    ctx.lineWidth = 2;
    edges.forEach(([a, b]) => {
      ctx.beginPath();
      ctx.moveTo(projected[a].x, projected[a].y);
      ctx.lineTo(projected[b].x, projected[b].y);
      ctx.stroke();
    });

    projected.forEach((point, index) => {
      ctx.fillStyle = index % 2 === 0 ? '#f97316' : '#38bdf8';
      ctx.beginPath();
      ctx.arc(point.x, point.y, 4.5, 0, Math.PI * 2);
      ctx.fill();
    });

    ctx.fillStyle = '#e2e8f0';
    ctx.font = '13px Inter, sans-serif';
    ctx.fillText('3D Scene', 18, 26);
    ctx.fillText('size=' + size.toFixed(0), 18, 46);
    ctx.fillText('rotation=' + speed.toFixed(3), 18, 66);

    animationFrameId = requestAnimationFrame(frame);
  }

  frame();
  setSummary('当前预览使用浏览器 Canvas 绘制轻量 3D 线框场景，并通过 requestAnimationFrame 实时旋转。可用于承接 3D / 空间关系建模需求。');
}
`
	return runtimeWrapper(props, body)
}

func previewPropsMarkdown(props map[string]float64) string {
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%.2f", key, props[key]))
	}

	return strings.Join(parts, ", ")
}
