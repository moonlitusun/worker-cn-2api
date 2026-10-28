package admin

// indexHTML is the embedded WebUI page.
// Design: editorial theme with light/dark variants — warm paper neutrals,
// ink primary, hairline borders, tabular numerals. No external dependencies.
// No emoji.
const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Worker-CN-2API 管理控制台</title>
<script>
try {
  var t = localStorage.getItem('worker-cn-2api-theme');
  var dark = t === 'dark' || (t !== 'light' && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light');
} catch (e) {}
</script>
<style>
:root {
  color-scheme: light;
  --bg: #FAFAF9;
  --surface: #FFFFFF;
  --surface-2: #F5F5F4;
  --surface-3: #F0F0EE;
  --border: #E7E5E4;
  --border-strong: #D6D3D1;
  --text: #1C1917;
  --text-2: #44403C;
  --muted: #78716C;
  --accent: #2563EB;
  --accent-hover: #1D4ED8;
  --accent-soft: #EFF6FF;
  --accent-border: #BFDBFE;
  --accent-glow: rgba(37,99,235,0.12);
  --success: #15803D;
  --success-soft: #F0FDF4;
  --success-border: #BBF7D0;
  --success-glow: rgba(21,128,61,0.12);
  --danger: #B91C1C;
  --danger-soft: #FEF2F2;
  --danger-border: #FECACA;
  --warning: #92400E;
  --warning-soft: #FFFBEB;
  --warning-border: #FDE68A;
  --mark-bg: #1C1917;
  --mark-fg: #FFFFFF;
  --btn-bg: #1C1917;
  --btn-fg: #FFFFFF;
  --btn-bg-hover: #3B3734;
  --nav-glass: rgba(255,255,255,0.92);
  --login-glow: #F1F0ED;
  --placeholder: #A8A29E;
  --radius: 14px;
  --radius-sm: 10px;
  --shadow: 0 1px 2px rgba(28,25,23,0.05);
  --shadow-md: 0 4px 12px -2px rgba(28,25,23,0.08);
  --shadow-lg: 0 16px 40px -12px rgba(28,25,23,0.16);
}

[data-theme="dark"] {
  color-scheme: dark;
  --bg: #131211;
  --surface: #1B1A18;
  --surface-2: #23211F;
  --surface-3: #2C2A27;
  --border: #2E2C29;
  --border-strong: #45413C;
  --text: #EDEAE6;
  --text-2: #C4BFB8;
  --muted: #8F8A82;
  --accent: #6D9CF0;
  --accent-hover: #8BB0F4;
  --accent-soft: rgba(109,156,240,0.14);
  --accent-border: rgba(109,156,240,0.45);
  --accent-glow: rgba(109,156,240,0.28);
  --success: #4ADE80;
  --success-soft: rgba(74,222,128,0.12);
  --success-border: rgba(74,222,128,0.4);
  --success-glow: rgba(74,222,128,0.22);
  --danger: #F87171;
  --danger-soft: rgba(248,113,113,0.12);
  --danger-border: rgba(248,113,113,0.4);
  --warning: #FBBF24;
  --warning-soft: rgba(251,191,36,0.12);
  --warning-border: rgba(251,191,36,0.4);
  --mark-bg: #EDEAE6;
  --mark-fg: #1C1917;
  --btn-bg: #EDEAE6;
  --btn-fg: #1C1917;
  --btn-bg-hover: #FBFAF8;
  --nav-glass: rgba(27,26,24,0.92);
  --login-glow: #1F1D1B;
  --placeholder: #6E6A64;
  --shadow: 0 1px 2px rgba(0,0,0,0.3);
  --shadow-md: 0 4px 12px -2px rgba(0,0,0,0.4);
  --shadow-lg: 0 16px 40px -12px rgba(0,0,0,0.6);
}

* { margin: 0; padding: 0; box-sizing: border-box; }

html { -webkit-text-size-adjust: 100%; }

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC",
    "Hiragino Sans GB", "Microsoft YaHei", "Helvetica Neue", Helvetica, Arial,
    sans-serif;
  background: var(--bg);
  color: var(--text);
  line-height: 1.55;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  font-size: 14px;
}

code, .mono, .num {
  font-family: "SF Mono", "Fira Code", "Roboto Mono", "Cascadia Mono",
    Consolas, "Courier New", monospace;
}

.num { font-variant-numeric: tabular-nums; }

.hidden { display: none !important; }

::selection { background: var(--accent-border); }

/* ---------- Login ---------- */

.login-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background:
    radial-gradient(1200px 600px at 50% -10%, var(--login-glow) 0%, transparent 60%),
    var(--bg);
}
.login-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 18px;
  box-shadow: var(--shadow-lg);
  padding: 44px 40px 36px;
  width: 100%;
  max-width: 380px;
}
.login-mark {
  width: 44px; height: 44px;
  background: var(--mark-bg);
  border-radius: 12px;
  display: flex; align-items: center; justify-content: center;
  margin: 0 auto 18px;
  color: var(--mark-fg);
  font-family: Georgia, "Songti SC", serif;
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.02em;
}
.login-title {
  text-align: center;
  font-size: 19px;
  font-weight: 650;
  letter-spacing: -0.015em;
}
.login-subtitle {
  text-align: center;
  font-size: 13px;
  color: var(--muted);
  margin: 6px 0 28px;
}
.login-input {
  width: 100%;
  padding: 11px 14px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  font-size: 14px;
  font-family: inherit;
  background: var(--surface);
  color: var(--text);
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.login-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-glow);
}
.login-btn {
  width: 100%;
  padding: 11px;
  background: var(--btn-bg);
  color: var(--btn-fg);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  margin-top: 16px;
  font-family: inherit;
  transition: background 0.15s, transform 0.1s;
}
.login-btn:hover { background: var(--btn-bg-hover); }
.login-btn:active { transform: translateY(1px); }
.login-error {
  color: var(--danger);
  font-size: 13px;
  text-align: center;
  margin-top: 12px;
  min-height: 18px;
}

/* ---------- Shell ---------- */

.shell { display: flex; min-height: 100vh; }

.sidebar {
  width: 224px;
  flex-shrink: 0;
  background: var(--surface);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  position: sticky;
  top: 0;
  height: 100vh;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 22px 20px 18px;
}
.brand-mark {
  width: 30px; height: 30px;
  background: var(--mark-bg);
  border-radius: 9px;
  display: flex; align-items: center; justify-content: center;
  color: var(--mark-fg);
  font-family: Georgia, "Songti SC", serif;
  font-size: 16px;
  font-weight: 700;
}
.brand-name {
  font-size: 15px;
  font-weight: 650;
  letter-spacing: -0.01em;
}
.brand-sub {
  font-size: 11px;
  color: var(--muted);
  letter-spacing: 0.02em;
}

.nav { flex: 1; padding: 8px 12px; }
.nav-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  padding: 0 10px 8px;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 10px;
  margin-bottom: 2px;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  font-size: 13.5px;
  font-weight: 500;
  color: var(--text-2);
  cursor: pointer;
  font-family: inherit;
  text-align: left;
  transition: background 0.12s, color 0.12s;
}
.nav-item svg { flex-shrink: 0; color: var(--muted); }
.nav-item:hover { background: var(--surface-2); color: var(--text); }
.nav-item.active { background: var(--surface-2); color: var(--text); font-weight: 600; }
.nav-item.active svg { color: var(--text); }

.sidebar-foot {
  padding: 14px 16px 18px;
  border-top: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.status {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: 12.5px;
  color: var(--muted);
}
.status-dot {
  width: 7px; height: 7px;
  border-radius: 50%;
  background: var(--success);
  box-shadow: 0 0 0 3px var(--success-glow);
}
.logout-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px;
  background: transparent;
  color: var(--muted);
  border: 1px solid var(--border);
  border-radius: 8px;
  font-size: 12.5px;
  font-weight: 500;
  cursor: pointer;
  font-family: inherit;
  transition: all 0.12s;
}
.logout-btn:hover {
  color: var(--danger);
  background: var(--danger-soft);
  border-color: var(--danger-border);
}

.main {
  flex: 1;
  min-width: 0;
  padding: 34px 40px 64px;
}
.main-inner { max-width: 1040px; margin: 0 auto; }

.page-head { margin-bottom: 24px; }
.page-title {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -0.015em;
}
.page-desc {
  font-size: 13px;
  color: var(--muted);
  margin-top: 4px;
}

/* ---------- Stat cards ---------- */

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin-bottom: 14px;
}
.stat-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 18px 20px;
  box-shadow: var(--shadow);
}
.stat-label {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 11.5px;
  font-weight: 600;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.07em;
  margin-bottom: 10px;
}
.stat-dot { width: 7px; height: 7px; border-radius: 50%; }
.stat-dot.total { background: var(--text); }
.stat-dot.success { background: var(--success); }
.stat-dot.failed { background: var(--danger); }
.stat-dot.rate { background: var(--accent); }
.stat-value {
  font-size: 27px;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.1;
}
.stat-sub {
  font-size: 12px;
  color: var(--muted);
  margin-top: 6px;
}

/* ---------- Cards ---------- */

.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  margin-bottom: 16px;
  overflow: hidden;
  box-shadow: var(--shadow);
}
.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}
.card-title { font-size: 14.5px; font-weight: 650; }
.card-title small {
  display: block;
  font-size: 12px;
  font-weight: 400;
  color: var(--muted);
  margin-top: 2px;
}
.card-body { padding: 20px; }
.card-body.flush { padding: 0; overflow-x: auto; }

/* ---------- Chart ---------- */

.chart-head { display: flex; align-items: baseline; justify-content: space-between; }
.chart-legend {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 12px;
  color: var(--muted);
}
.legend-item { display: inline-flex; align-items: center; gap: 6px; }
.legend-dot { width: 9px; height: 9px; border-radius: 2px; }
.legend-dot.ok { background: var(--accent); }
.legend-dot.fail { background: var(--danger); }
.chart-peak { font-size: 12px; color: var(--muted); }

.chart {
  overflow-x: auto;
}
.chart-plot {
  display: flex;
  align-items: stretch;
  gap: 6px;
  height: 150px;
  margin-top: 18px;
  padding-top: 8px;
  border-bottom: 1px solid var(--border);
}
.chart-col {
  flex: 1 1 0;
  min-width: 14px;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
}
.chart-bars {
  flex: 1;
  width: 100%;
  max-width: 18px;
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  gap: 1px;
}
.bar { width: 100%; border-radius: 2px 2px 0 0; }
.bar.ok { background: var(--accent); }
.bar.fail { background: var(--danger); opacity: 0.85; }
.chart-axis {
  display: flex;
  gap: 6px;
  padding-top: 6px;
}
.chart-axis-item {
  flex: 1 1 0;
  min-width: 14px;
  text-align: center;
  font-size: 10px;
  color: var(--muted);
  font-family: "SF Mono", Consolas, monospace;
  white-space: nowrap;
}

/* ---------- Table ---------- */

table { width: 100%; border-collapse: collapse; }
th {
  padding: 10px 20px;
  text-align: left;
  font-size: 11px;
  font-weight: 600;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.07em;
  background: var(--surface-2);
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
td {
  padding: 12px 20px;
  border-bottom: 1px solid var(--border);
  font-size: 13.5px;
  color: var(--text-2);
  vertical-align: middle;
}
tr:last-child td { border-bottom: none; }
tbody tr { transition: background 0.1s; }
tbody tr:hover { background: var(--surface-2); }
td .mono { font-size: 12.5px; }
td .muted { color: var(--muted); }
td.num, th.num { text-align: right; }

.rate-cell { display: flex; align-items: center; gap: 10px; justify-content: flex-end; }
.rate-bar {
  width: 72px;
  height: 5px;
  background: var(--surface-3);
  border-radius: 3px;
  overflow: hidden;
}
.rate-fill { height: 100%; border-radius: 3px; background: var(--accent); }
.rate-fill.warn { background: var(--warning); }
.rate-fill.bad { background: var(--danger); }
.rate-text { font-size: 12.5px; color: var(--text-2); min-width: 44px; text-align: right; }

.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
}
.badge.ok { background: var(--success-soft); color: var(--success); border: 1px solid var(--success-border); }
.badge.bad { background: var(--danger-soft); color: var(--danger); border: 1px solid var(--danger-border); }

/* ---------- Forms ---------- */

.row {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 14px;
}
.row > label {
  min-width: 88px;
  font-weight: 500;
  font-size: 13.5px;
  color: var(--text-2);
}
.row > .field { flex: 1; }

input[type="text"], input[type="number"], input[type="password"], input[type="search"] {
  width: 100%;
  padding: 10px 13px;
  background: var(--surface);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: 13.5px;
  font-family: inherit;
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}
input[type="text"]:focus, input[type="number"]:focus, input[type="password"]:focus, input[type="search"]:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-glow);
}
input::placeholder { color: var(--placeholder); }
input[readonly] {
  background: var(--surface-2);
  color: var(--muted);
  border-color: var(--border);
}

/* ---------- Buttons ---------- */

.btn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 10px 18px;
  background: var(--btn-bg);
  color: var(--btn-fg);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s;
  font-family: inherit;
  white-space: nowrap;
}
.btn:hover { background: var(--btn-bg-hover); }
.btn:active { transform: translateY(1px); }
.btn.secondary {
  background: var(--surface);
  color: var(--text-2);
  border: 1px solid var(--border-strong);
  font-weight: 500;
}
.btn.secondary:hover { background: var(--surface-2); color: var(--text); }
.btn.danger {
  background: var(--surface);
  color: var(--danger);
  border: 1px solid var(--danger-border);
  font-weight: 500;
}
.btn.danger:hover { background: var(--danger-soft); }
.btn.sm {
  padding: 5px 11px;
  font-size: 12.5px;
  font-weight: 500;
  border-radius: 8px;
}
.btn-row { display: flex; gap: 8px; }
.btn-row.right { justify-content: flex-end; }

/* ---------- Model grid ---------- */

.model-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
  gap: 10px;
}
.model-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 16px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 13.5px;
  font-weight: 500;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.model-item:hover {
  border-color: var(--border-strong);
  box-shadow: var(--shadow);
}

/* ---------- Hints ---------- */

.hint {
  font-size: 13px;
  color: var(--muted);
  margin-bottom: 16px;
}
.hint-warn {
  font-size: 13px;
  color: var(--warning);
  padding: 11px 14px;
  background: var(--warning-soft);
  border: 1px solid var(--warning-border);
  border-radius: var(--radius-sm);
  margin-top: 12px;
}
.empty-row {
  text-align: center;
  padding: 32px 16px;
  color: var(--muted);
  font-size: 13.5px;
}
.empty-row .empty-title { font-size: 14px; font-weight: 600; color: var(--text-2); margin-bottom: 4px; }

/* ---------- Toast ---------- */

.toast {
  position: fixed;
  bottom: 28px;
  left: 50%;
  transform: translateX(-50%) translateY(0);
  padding: 12px 22px;
  background: var(--btn-bg);
  color: var(--btn-fg);
  border-radius: 10px;
  font-size: 13.5px;
  font-weight: 500;
  z-index: 9999;
  box-shadow: var(--shadow-lg);
  transition: opacity 0.25s, transform 0.25s;
  pointer-events: none;
}
.toast.error { background: var(--danger); }
.toast.hidden { opacity: 0; transform: translateX(-50%) translateY(8px); }

/* ---------- Theme toggle ---------- */

.foot-actions { display: flex; align-items: center; gap: 8px; }
.theme-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 31px;
  height: 31px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--muted);
  cursor: pointer;
  flex-shrink: 0;
  transition: color 0.12s, background 0.12s;
}
.theme-btn:hover { color: var(--text); background: var(--surface-2); }

/* ---------- Responsive ---------- */

.mobile-nav {
  display: none;
  position: sticky;
  top: 0;
  z-index: 50;
  background: var(--nav-glass);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--border);
  padding: 10px 12px;
  gap: 8px;
  align-items: center;
}
.mobile-nav-items {
  display: flex;
  gap: 4px;
  flex: 1;
  overflow-x: auto;
}
.mobile-nav .nav-item { flex: 0 0 auto; width: auto; justify-content: center; padding: 8px 14px; }
.mobile-nav .nav-item span.mnav-label { font-size: 12.5px; }

@media (max-width: 1180px) {
  .stat-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 900px) {
  .shell { flex-direction: column; }
  .sidebar { display: none; }
  .main { padding: 24px 18px 48px; }
  .stat-grid { grid-template-columns: repeat(2, 1fr); }
  .mobile-nav { display: flex; }
}

@media (max-width: 560px) {
  .row { flex-direction: column; align-items: stretch; }
  .row > label { min-width: auto; }
  .stat-value { font-size: 22px; }
  .chart-col, .chart-axis-item { min-width: 12px; }
  /* Prevent iOS Safari auto-zoom on focus (requires >=16px). */
  input[type="text"], input[type="number"], input[type="password"], input[type="search"] { font-size: 16px; }
}

/* Provider diagnostics and model details */
#traeStatusHint:empty { display:none; }
.provider-meta { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:12px 20px; padding:16px; margin:12px 0; border-radius:var(--radius-sm); background:var(--surface-2); }
.provider-meta div { min-width:0; }
.provider-meta dt { color:var(--muted);font-size:12px;margin-bottom:4px; }
.provider-meta dd { margin:0;font-size:13px;overflow-wrap:anywhere; }
.provider-operation { padding:9px 0;border-bottom:1px solid var(--border);font-size:12px;overflow-wrap:anywhere;white-space:pre-wrap; }
.provider-operation:last-child { border-bottom:0; }
.provider-operation.failed { color:var(--danger); }
.provider-actions { display:flex;gap:8px;flex-wrap:wrap;margin:14px 0;align-items:center; }
.provider-actions .btn { white-space:normal; }
.provider-select, .field select { min-width:0;max-width:100%;padding:9px 12px;color:var(--text);background:var(--surface);border:1px solid var(--border-strong);border-radius:8px;font:inherit; }
.field select { width:100%; }
.card-head .badge { flex-shrink:0;white-space:nowrap; }
.badge.warn { background:var(--warning-soft);color:var(--warning);border:1px solid var(--warning-border); }
.model-grid { display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:12px; }
.model-item { display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px 14px;background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-sm);transition:border-color 0.15s,box-shadow 0.15s; }
.model-item:hover { border-color:var(--border-strong);box-shadow:var(--shadow); }
.model-card-content { min-width:0;flex:1; }
.model-card-title { overflow-wrap:anywhere;font-size:13.5px;font-weight:600;color:var(--text);line-height:1.4; }
.model-card-meta { display:flex;flex-wrap:wrap;gap:6px;align-items:center;margin-top:6px; }
.model-tag { display:inline-flex;align-items:center;font-size:11px;padding:2px 7px;border-radius:6px;background:var(--surface-2);color:var(--muted);border:1px solid var(--border);white-space:nowrap; }
.model-tag.ok { background:var(--success-soft);color:var(--success);border-color:var(--success-border); }
.model-toolbar { display:flex;align-items:center;gap:10px;margin-bottom:16px;padding:8px 12px;background:var(--surface-2);border:1px solid var(--border);border-radius:var(--radius-sm); }
.model-toolbar .provider-select { flex:0 0 130px;min-width:130px;padding:8px 12px;background:var(--surface);border:1px solid var(--border);border-radius:8px;color:var(--text);font-size:13px;cursor:pointer; }
.model-toolbar input[type="search"] { flex:1;padding:8px 14px;background:var(--surface);border:1px solid var(--border);border-radius:8px;color:var(--text);font-size:13px;box-sizing:border-box; }
.model-toolbar input[type="search"]:focus { border-color:var(--accent);box-shadow:0 0 0 2px var(--accent-glow); }
.model-count-badge { font-size:12px;color:var(--muted);white-space:nowrap;padding:0 6px; }
@media(max-width:600px) { .provider-meta { grid-template-columns:1fr; } .provider-actions .provider-select { flex-basis:100%; } .model-grid { grid-template-columns:minmax(0,1fr); } .provider-actions .btn { flex:1; } .model-toolbar { flex-wrap:wrap; } .model-toolbar .provider-select { flex:1 1 100%; } .model-toolbar input[type="search"] { flex:1 1 100%; } }

/* Advanced management: compact sections, shared status rows. */
.advanced-summary { cursor:pointer; list-style:none; }
.advanced-summary::-webkit-details-marker,.settings-panel>summary::-webkit-details-marker,.failure-summary::-webkit-details-marker { display:none; }
.advanced-summary-meta { display:flex;align-items:center;gap:14px;color:var(--muted);font-size:12px; }
.disclosure-chevron { display:inline-block;transition:transform .18s; }
details[open]>.advanced-summary .disclosure-chevron,details[open]>.advanced-heading .disclosure-chevron { transform:rotate(180deg); }
.advanced-body { display:grid;grid-template-columns:minmax(0,1fr);gap:16px;background:var(--surface-2); }
.advanced-panel { min-width:0;border:1px solid var(--border);border-radius:12px;background:var(--surface);padding:18px 20px; }
.advanced-heading { display:flex;align-items:center;justify-content:space-between;gap:12px;font-size:13px;font-weight:600;list-style:none; }
.settings-panel>summary { cursor:pointer; }
.advanced-grid { display:grid;grid-template-columns:1fr 1fr;gap:16px;align-items:start; }
.advanced-fields { margin-top:20px; }
.advanced-fields .row { flex-direction:column;align-items:stretch;gap:8px;margin-bottom:16px; }
.advanced-fields .row>label { font-size:12px;min-width:0;color:var(--muted); }
.advanced-fields .field { width:100%; }
.session-panel .provider-meta { grid-template-columns:repeat(4,minmax(0,1fr));margin:16px 0;padding:16px;gap:16px; }
.chat-test-actions { margin:16px 0 0; }
.chat-test-actions select { flex:1; }
.operation-row { display:grid;grid-template-columns:80px 64px minmax(0,1fr);align-items:center;gap:12px;padding:11px 0;border-top:1px solid var(--border);font-size:12px; }
.operation-label { color:var(--text); }
.operation-time { min-width:0;overflow-wrap:anywhere;text-align:right;color:var(--muted);font-variant-numeric:tabular-nums; }
.operation-badge { color:var(--muted);font-size:11px; }
.operation-badge.success { color:var(--success); }
.operation-badge.failed,.failure-kind { color:var(--danger); }
.failure-list { max-height:460px;overflow:auto; }
.failure-item+.failure-item { border-top:1px solid var(--border); }
.failure-summary { display:flex;align-items:center;gap:12px;padding:13px 0;cursor:pointer;list-style:none; }
.failure-title { font-size:12px;overflow-wrap:anywhere;flex:1;min-width:0; }
.failure-kind { font-size:11px;padding:3px 7px;border-radius:6px;background:var(--surface-2);white-space:nowrap; }
.failure-meta { font-size:11px;color:var(--muted);font-variant-numeric:tabular-nums; }
.failure-reason { grid-column:1/-1;margin:8px 0 12px;padding:12px;background:var(--surface-2);border-radius:8px;font:12px/1.7 ui-monospace,monospace;white-space:pre-wrap;overflow-wrap:anywhere;color:var(--muted); }
@media(max-width:700px) { .advanced-grid { grid-template-columns:1fr; } .session-panel .provider-meta { grid-template-columns:repeat(2,minmax(0,1fr)); } .advanced-panel { padding:16px; } .operation-row { grid-template-columns:70px 48px minmax(0,1fr);gap:8px; } .failure-summary { flex-wrap:wrap; } .failure-meta { width:100%; } }
</style>
</head>
<body>

<!-- Login -->
<div id="loginPage" class="login-wrap">
  <div class="login-card">
    <div class="login-mark">W</div>
    <div class="login-title">Worker-CN-2API</div>
    <div class="login-subtitle">请输入管理密码</div>
    <input type="password" id="loginPassword" class="login-input" placeholder="管理密码" autofocus autocomplete="current-password">
    <button class="login-btn" type="button" onclick="doLogin()">登 录</button>
    <div id="loginError" class="login-error"></div>
  </div>
</div>

<!-- Dashboard -->
<div id="dashboard" class="hidden">
<div class="shell">

  <!-- Desktop sidebar -->
  <aside class="sidebar">
    <div class="brand">
      <div class="brand-mark">W</div>
      <div>
        <div class="brand-name">Worker-CN-2API</div>
      </div>
    </div>
    <nav class="nav">
      <div class="nav-label">控制台</div>
      <button class="nav-item active" data-page="stats" onclick="switchPage('stats')">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><path d="M3 13V8"/><path d="M7 13V3"/><path d="M11 13v-3"/><path d="M15 13H1"/></svg>
        <span>统计</span>
      </button>
      <button class="nav-item" data-page="keys" onclick="switchPage('keys')">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="6" cy="10" r="3.5"/><path d="M8.8 7.2 14 2"/><path d="M11 4.5l2 2"/><path d="M8.2 6.8 9.8 8.4"/></svg>
        <span>密钥</span>
      </button>
      <button class="nav-item" data-page="token" onclick="switchPage('token')">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6.5V3.5h10v3"/><path d="M3.5 6.5h9l.5 7h-10l.5-7z"/><path d="M10.5 6.5v-2a2.5 2.5 0 0 0-5 0v2"/></svg>
        <span>渠道凭证</span>
      </button>
      <button class="nav-item" data-page="models" onclick="switchPage('models')">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2.5" y="2.5" width="4.5" height="4.5" rx="1"/><rect x="9" y="2.5" width="4.5" height="4.5" rx="1"/><rect x="2.5" y="9" width="4.5" height="4.5" rx="1"/><rect x="9" y="9" width="4.5" height="4.5" rx="1"/></svg>
        <span>模型</span>
      </button>
      <button class="nav-item" data-page="config" onclick="switchPage('config')">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
        <span>设置</span>
      </button>
    </nav>
    <div class="sidebar-foot">
      <span class="status"><span class="status-dot"></span>运行中</span>
      <div class="foot-actions">
        <button class="theme-btn" type="button" onclick="toggleTheme()" title="切换深浅模式" aria-label="切换深浅模式">
          <svg class="icon-moon" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
          <svg class="icon-sun hidden" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/></svg>
        </button>
        <button class="logout-btn" type="button" onclick="doLogout()">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10 2.5h2.5A1.5 1.5 0 0 1 14 4v8a1.5 1.5 0 0 1-1.5 1.5H10"/><path d="M6.5 5.5 9 8l-2.5 2.5"/><path d="M9 8H2"/></svg>
          退出
        </button>
      </div>
    </div>
  </aside>

  <!-- Mobile nav -->
  <div class="mobile-nav">
    <div class="mobile-nav-items" id="mobileNavItems"></div>
    <button class="theme-btn" type="button" onclick="toggleTheme()" title="切换深浅模式" aria-label="切换深浅模式">
      <svg class="icon-moon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
      <svg class="icon-sun hidden" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/></svg>
    </button>
  </div>

  <main class="main">
  <div class="main-inner">

    <!-- Stats -->
    <section id="page-stats" class="page">
      <div class="page-head">
        <div class="page-title">请求统计</div>

      </div>
      <div id="statsChannelTabs" class="provider-actions" style="margin-bottom:20px"></div>
      <p id="statsChannelHint" class="hint" hidden>渠道历史用量无法回溯。</p>


      <div class="stat-grid">
        <div class="stat-card">
          <div class="stat-label"><span class="stat-dot total"></span>总请求</div>
          <div class="stat-value num" id="statTotal">0</div>
          <div class="stat-sub num" id="statReqSub">成功率 —</div>
        </div>
        <div class="stat-card">
          <div class="stat-label"><span class="stat-dot success"></span>Token 总数</div>
          <div class="stat-value num" id="statTokens">0</div>
          <div class="stat-sub num" id="statCredits">Credits 消耗 0</div>
        </div>
        <div class="stat-card">
          <div class="stat-label"><span class="stat-dot rate"></span>总输入</div>
          <div class="stat-value num" id="statPrompt">0</div>
          <div class="stat-sub num" id="statCached">缓存命中 0</div>
        </div>
        <div class="stat-card">
          <div class="stat-label"><span class="stat-dot total"></span>总输出</div>
          <div class="stat-value num" id="statCompletion">0</div>
          <div class="stat-sub num" id="statAvgOut">平均 0 / 次</div>
        </div>
      </div>

      <div class="card">
        <div class="card-head">
          <div class="card-title">近 24 小时请求量</div>
          <div class="chart-legend">
            <span class="legend-item"><span class="legend-dot ok"></span>成功</span>
            <span class="legend-item"><span class="legend-dot fail"></span>失败</span>
            <span class="chart-peak num" id="chartPeak"></span>
          </div>
        </div>
        <div class="card-body">
          <div class="chart" id="chart"></div>
        </div>
      </div>

      <div class="card">
        <div class="card-head">
          <div class="card-title">按模型统计</div>
        </div>
        <div class="card-body flush">
          <table>
            <thead>
              <tr><th>模型</th><th class="num">请求</th><th class="num">成功</th><th class="num">失败</th><th class="num" style="width:200px">成功率</th></tr>
            </thead>
            <tbody id="statsTable"></tbody>
          </table>
        </div>
      </div>
      <div class="card">
        <div class="card-head"><div class="card-title">最近失败</div><span id="failureCount" class="model-count-badge">0 条</span></div>
        <div class="card-body failure-list" id="failureList" aria-live="polite"></div>
      </div>
    </section>

    <!-- Keys -->
    <section id="page-keys" class="page hidden">
      <div class="page-head">
        <div class="page-title">API 密钥</div>

      </div>

      <div class="card">
        <div class="card-head"><div class="card-title">创建密钥</div></div>
        <div class="card-body">
          <div class="row">
            <label>密钥</label>
            <div class="field"><input type="text" id="newKey" placeholder="留空则自动生成" autocomplete="off"></div>
          </div>
          <div class="row">
            <label>备注</label>
            <div class="field"><input type="text" id="newNote" placeholder="可选描述" autocomplete="off"></div>
          </div>
          <div class="btn-row right">
            <button class="btn secondary" type="button" onclick="addKey(true)">随机生成</button>
            <button class="btn" type="button" onclick="addKey(false)">创建</button>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-head"><div class="card-title">密钥列表</div></div>
        <div class="card-body flush">
          <table>
            <thead><tr><th>密钥</th><th>备注</th><th>创建时间</th><th class="num" style="width:170px">操作</th></tr></thead>
            <tbody id="keysTable"></tbody>
          </table>
        </div>
      </div>
    </section>

    <!-- Providers / Tokens -->
    <section id="page-token" class="page hidden">
      <div class="page-head">
        <div class="page-title">渠道凭证</div>

      </div>

      <div class="card">
        <div class="card-head"><div class="card-title">Qoder CN</div><span id="qoderBadge" class="badge">检查中...</span></div>
        <div class="card-body">
          <div class="row"><label for="qoderDisplay">当前凭据</label><div class="field"><input id="qoderDisplay" type="text" readonly placeholder="未配置"></div></div>
          <div class="row"><label for="qoderInput">更新凭据</label><div class="field"><input id="qoderInput" type="password" autocomplete="off" placeholder="PAT"></div></div>
          <div class="btn-row right"><button class="btn danger sm" type="button" onclick="clearQoder()">清空</button><button id="qoderTestButton" class="btn secondary sm" type="button" onclick="testChannel('qoder')">测试连接</button><button id="qoderSaveButton" class="btn" type="button" onclick="saveQoder()">保存并测试连接</button></div>
          <p id="qoderStatusHint" class="hint" role="status"></p>
        </div>
      </div>

      <!-- WorkBuddy 卡片 -->
      <div class="card">
        <div class="card-head">
          <div class="card-title">WorkBuddy CN</div>
          <span id="wbBadge" class="badge">检查中...</span>
        </div>
        <div class="card-body">
          <div class="row">
            <label>当前凭据</label>
            <div class="field"><input type="text" id="wbDisplay" readonly placeholder="未配置"></div>
          </div>
          <div class="row">
            <label>更新凭据</label>
            <div class="field"><input type="password" id="wbInput" placeholder="填入 accessToken 或旧版 auth_token" autocomplete="off"></div>
          </div>
          <div class="btn-row right">
            <button class="btn danger sm" type="button" onclick="clearWorkBuddy()">清空</button>
            <button id="workbuddyTestButton" class="btn secondary sm" type="button" onclick="testChannel('workbuddy')">测试连接</button>
            <button id="workbuddySaveButton" class="btn" type="button" onclick="saveWorkBuddy()">保存并测试连接</button>
          </div>
          <p id="workbuddyStatusHint" class="hint" role="status"></p>
        </div>
      </div>
      <div class="card">
        <div class="card-head">
          <div class="card-title">Trae CN</div>
          <span id="traeBadge" class="badge">检查中...</span>
        </div>
        <div class="card-body">
          <div class="row"><label for="traeDisplay">当前凭据</label><div class="field"><input id="traeDisplay" type="text" readonly placeholder="未配置"></div></div>
          <div class="row"><label for="traeInput">更新凭据</label><div class="field"><input id="traeInput" type="password" autocomplete="off" placeholder="个人版 JWT 或企业版 trae-lt- 登录令牌"></div></div>
          <div class="btn-row right">
            <button class="btn danger sm" type="button" onclick="clearTrae()">清空</button>
            <button id="traeConnectionButton" class="btn secondary sm" type="button" onclick="testTrae(false)">测试连接</button>
            <button id="traeSaveButton" class="btn" type="button" onclick="saveTrae()">保存并测试连接</button>
          </div>
          <p id="traeStatusHint" class="hint" role="status"></p>

        </div>
      </div>
      <details class="card advanced-management">
        <summary class="card-head advanced-summary"><span>高级管理</span><span class="advanced-summary-meta">Trae CN <span class="disclosure-chevron">⌄</span></span></summary>
        <div class="card-body advanced-body">
          <section class="advanced-panel session-panel">
            <div class="advanced-heading"><span>会话状态</span><button class="btn secondary sm" type="button" onclick="loadTraeStatus(false)">刷新</button></div>
          <dl class="provider-meta">
            <div><dt>当前账号</dt><dd id="traeCurrentAccount">当前会话（未命名）</dd></div>
            <div><dt>有效期</dt><dd id="traeExpiryStatus">检查中…</dd></div>
            <div><dt>自动续期</dt><dd id="traeRefreshStatus">检查中…</dd></div>
            <div><dt>正在处理 / 排队</dt><dd id="traeConcurrency">0 / 0</dd></div>
          </dl>
            <div id="traeOperations" class="operation-list" aria-live="polite"></div>
          </section>
          <section class="advanced-panel">
            <div class="advanced-heading"><span>对话测试</span><span class="model-tag">消耗少量额度</span></div>
            <div class="provider-actions chat-test-actions">
              <select id="traeTestModel" class="provider-select" aria-label="选择测试对话的模型"><option value="">先测试连接，再选择模型</option></select>
              <button id="traeChatButton" class="btn secondary sm" type="button" onclick="testTrae(true)">测试对话</button>
            </div>
          </section>
          <div class="advanced-grid">
            <details class="advanced-panel settings-panel">
              <summary class="advanced-heading"><span>凭据与续期</span><span class="disclosure-chevron">⌄</span></summary>
              <div class="advanced-fields">
            <div class="row"><label for="traeUserID">用户 ID</label><div class="field"><input id="traeUserID" type="text" autocomplete="off" placeholder="可选，账号 UserID"></div></div>
            <div class="row"><label for="traeClientID">产品类型</label><div class="field"><select id="traeClientID"><option value="ono9krqynydwx5">Trae Code CN</option><option value="en1oxy7wnw8j9n">TRAE SOLO CN</option></select></div></div>
            <div class="row"><label for="traeRefresh">刷新令牌</label><div class="field"><input id="traeRefresh" type="password" autocomplete="off" placeholder="可选，RefreshToken；保存时替换旧值"></div></div>
            <div class="row"><label for="traeExpires">过期时间</label><div class="field"><input id="traeExpires" type="text" placeholder="可选，ISO 8601；JWT 可自动识别"></div></div>

              </div>
            </details>
            <details class="advanced-panel settings-panel">
              <summary class="advanced-heading"><span>已保存账号</span><span class="disclosure-chevron">⌄</span></summary>
              <div class="advanced-fields">
            <div class="row"><label for="traeAccountSelect">账号</label><div class="field"><select id="traeAccountSelect" onchange="selectTraeProfile()"><option value="">当前会话（未保存）</option></select></div></div>
            <div class="row"><label for="traeAccountName">账号名称</label><div class="field"><input id="traeAccountName" type="text" maxlength="80" placeholder="例如：主账号、备用账号"></div></div>
            <div class="provider-actions">
              <button class="btn secondary sm" type="button" onclick="activateTraeAccount()">切换到所选账号</button>
              <button class="btn secondary sm" type="button" onclick="saveTraeAccount(true)">另存为新账号</button>
              <button class="btn secondary sm" type="button" onclick="saveTraeAccount(false)">更新并切换</button>
              <button class="btn danger sm" type="button" onclick="deleteTraeAccount()">删除所选账号</button>
            </div>
              </div>
            </details>
          </div>
        </div>
      </details>
    </section>

    <!-- Models -->
    <section id="page-models" class="page hidden">
      <div class="page-head">
        <div class="page-title">可用模型</div>

      </div>

      <div class="card">
        <div class="card-head"><div class="card-title">模型列表</div></div>
        <div class="card-body">
          <div id="modelChannelTabs" class="provider-actions" style="margin-bottom:16px"></div>
          <div class="model-toolbar">

            <input id="modelSearch" type="search" aria-label="搜索模型" placeholder="搜索模型名称或 ID…" oninput="renderModels()">
            <span id="modelCount" class="model-count-badge"></span>
            <button class="btn secondary sm" type="button" onclick="loadModels()">重新加载</button>
          </div>

          <div id="modelsList" class="model-grid"></div>
        </div>
      </div>
    </section>

    <!-- Settings -->
    <section id="page-config" class="page hidden">
      <div class="page-head">
        <div class="page-title">设置</div>

      </div>

      <div class="card">
        <div class="card-head"><div class="card-title">服务器配置</div></div>
        <div class="card-body">
          <p class="hint">修改后需重启。</p>
          <div class="row">
            <label>主机</label>
            <div class="field"><input type="text" id="cfgHost" placeholder="0.0.0.0" autocomplete="off"></div>
          </div>
          <div class="row">
            <label>端口</label>
            <div class="field"><input type="number" id="cfgPort" placeholder="10081"></div>
          </div>
          <div id="configHint" class="hint-warn hidden"></div>
        </div>
      </div>

      <div class="card">
        <div class="card-head"><div class="card-title">超时配置</div></div>
        <div class="card-body">
          <p class="hint">响应超时：等待开始返回。空闲超时：返回过程中无数据。单位：秒。</p>
          <div class="row">
            <label>响应超时</label>
            <div class="field"><input type="number" id="cfgChatTimeout" placeholder="120" min="1" max="3600"></div>
          </div>
          <div class="row">
            <label>空闲超时</label>
            <div class="field"><input type="number" id="cfgIdleTimeout" placeholder="300" min="1" max="3600"></div>
          </div>
          <div class="btn-row right">
            <button class="btn" type="button" onclick="saveConfig()">保存</button>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-head"><div class="card-title">修改密码</div></div>
        <div class="card-body">

          <div class="row">
            <label>当前密码</label>
            <div class="field"><input type="password" id="curPwd" placeholder="当前密码" autocomplete="current-password"></div>
          </div>
          <div class="row">
            <label>新密码</label>
            <div class="field"><input type="password" id="newPwd" placeholder="新密码" autocomplete="new-password"></div>
          </div>
          <div class="btn-row right">
            <button class="btn" type="button" onclick="savePassword()">修改密码</button>
          </div>
        </div>
      </div>
    </section>

  </div>
  </main>
</div>
</div>

<div id="toast" class="toast hidden"></div>

<script>
/* ---------- Helpers ---------- */

function api(path, opts) {
  opts = opts || {};
  return fetch('/admin/api' + path, {
    method: opts.method || 'GET',
    body: opts.body,
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin'
  }).then(function(res) {
    return res.json().catch(function() { return {}; }).then(function(data) {
      if (!res.ok) throw new Error(data.error || '请求失败 (' + res.status + ')');
      return data;
    });
  });
}

function showToast(msg, type) {
  var t = document.getElementById('toast');
  t.textContent = msg;
  t.className = 'toast ' + (type || '');
  clearTimeout(showToast._timer);
  showToast._timer = setTimeout(function() { t.className = 'toast hidden'; }, 2800);
}

function escapeHtml(s) {
  var d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML.replace(/'/g, '&#39;');
}

function fmt(n) {
  return Number(n || 0).toLocaleString('zh-CN');
}

function pct(rate) {
  if (rate === null || rate === undefined || isNaN(rate)) return '—';
  return (rate * 100).toFixed(1) + '%';
}

function copyText(text) {
  navigator.clipboard.writeText(text).then(function() { showToast('已复制到剪贴板'); });
}

/* ---------- Auth ---------- */

function checkAuth() {
  api('/auth').then(function(data) {
    if (data.authenticated) { showDashboard(); }
    else { showLogin(); }
  }).catch(function() { showLogin(); });
}

function showLogin() {
  loggedIn = false;
  document.getElementById('loginPage').classList.remove('hidden');
  document.getElementById('dashboard').classList.add('hidden');
}

function showDashboard() {
  loggedIn = true;
  document.getElementById('loginPage').classList.add('hidden');
  document.getElementById('dashboard').classList.remove('hidden');
  buildMobileNav();
  switchPage('stats');
}

function doLogin() {
  var pwd = document.getElementById('loginPassword').value;
  if (!pwd) return;
  api('/login', { method: 'POST', body: JSON.stringify({ password: pwd }) })
    .then(function() {
      document.getElementById('loginPassword').value = '';
      document.getElementById('loginError').textContent = '';
      showDashboard();
    })
    .catch(function(e) {
      document.getElementById('loginError').textContent = e.message;
    });
}

function doLogout() {
  api('/logout', { method: 'POST' })
    .then(function() { showLogin(); })
    .catch(function() { showLogin(); });
}

document.getElementById('loginPassword').addEventListener('keydown', function(e) {
  if (e.key === 'Enter') doLogin();
});

/* ---------- Navigation ---------- */

var NAV_ITEMS = [
  ['stats', '统计', 'M3 13V8M7 13V3M11 13v-3M15 13H1'],
  ['keys', '密钥', 'M8.8 7.2 14 2M11 4.5l2 2M8.2 6.8 9.8 8.4'],
  ['token', '令牌', 'M3 6.5V3.5h10v3M3.5 6.5h9l.5 7h-10l.5-7z'],
  ['models', '模型', ''],
  ['config', '设置', '']
];
var PAGES = ['stats', 'keys', 'token', 'models', 'config'];

function buildMobileNav() {
  var nav = document.getElementById('mobileNavItems');
  nav.innerHTML = '';
  NAV_ITEMS.forEach(function(item) {
    var b = document.createElement('button');
    b.className = 'nav-item';
    b.setAttribute('data-page', item[0]);
    b.setAttribute('onclick', "switchPage('" + item[0] + "')");
    b.innerHTML = '<span class="mnav-label">' + item[1] + '</span>';
    nav.appendChild(b);
  });
}

function switchPage(name) {
  currentPage = name;
  PAGES.forEach(function(p) {
    document.getElementById('page-' + p).classList.add('hidden');
  });
  document.getElementById('page-' + name).classList.remove('hidden');
  document.querySelectorAll('.nav-item').forEach(function(el) {
    el.classList.toggle('active', el.getAttribute('data-page') === name);
  });
  if (name === 'stats') loadStats();
  if (name === 'keys') loadKeys();
  if (name === 'token') loadProviders();
  if (name === 'models') loadModels();
  if (name === 'config') loadConfig();
}

var currentPage = 'stats';
var loggedIn = false;
setInterval(function() {
  if (loggedIn && currentPage === 'stats') loadStats();
}, 15000);

/* ---------- Theme ---------- */

function themeDark() {
  var saved = null;
  try { saved = localStorage.getItem('worker-cn-2api-theme'); } catch (e) {}
  if (saved === 'light' || saved === 'dark') return saved === 'dark';
  return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function applyTheme(dark) {
  document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'light');
  try { localStorage.setItem('worker-cn-2api-theme', dark ? 'dark' : 'light'); } catch (e) {}
  document.querySelectorAll('.icon-moon').forEach(function(el) { el.classList.toggle('hidden', dark); });
  document.querySelectorAll('.icon-sun').forEach(function(el) { el.classList.toggle('hidden', !dark); });
}

function toggleTheme() {
  applyTheme(!themeDark());
}

applyTheme(themeDark());

/* Shared channel filter: independent state for each mounted instance. */
var CHANNEL_OPTIONS = [
  { id: '', label: '全部' },
  { id: 'qoder', label: 'Qoder CN' },
  { id: 'workbuddy', label: 'WorkBuddy CN' },
  { id: 'trae', label: 'Trae CN' }
];
function createChannelTabs(container, label, onChange) {
  var value = '';
  container.setAttribute('role', 'group');
  container.setAttribute('aria-label', label);
  container.innerHTML = '';
  var buttons = CHANNEL_OPTIONS.map(function(option) {
    var button = document.createElement('button');
    button.type = 'button'; button.textContent = option.label;
    button.addEventListener('click', function() { select(option.id); });
    container.appendChild(button);
    return button;
  });
  function paint() {
    buttons.forEach(function(button, index) {
      var active = CHANNEL_OPTIONS[index].id === value;
      button.className = active ? 'btn sm' : 'btn secondary sm';
      button.setAttribute('aria-pressed', String(active));
    });
  }
  function select(next) {
    if (next === value || !CHANNEL_OPTIONS.some(function(option) { return option.id === next; })) return;
    value = next; paint(); onChange(value);
  }
  paint();
  return { select: select, value: function() { return value; } };
}

/* ---------- Stats ---------- */

var statsProvider = '';
var statsRequest = 0;
function selectStatsProvider(provider) {
  statsProvider = provider;
  document.getElementById('statsChannelHint').hidden = !provider;
  loadStats();
}
function loadStats() {
  var request = ++statsRequest;
  return api('/stats' + (statsProvider ? '?provider=' + encodeURIComponent(statsProvider) : '')).then(function(d) {
    if (request !== statsRequest) return;
    var tokens = (d.prompt_tokens || 0) + (d.completion_tokens || 0);
    document.getElementById('statTotal').textContent = fmt(d.total);
    document.getElementById('statReqSub').textContent = '成功率 ' + pct(d.success_rate);
    document.getElementById('statTokens').textContent = fmt(tokens);
    document.getElementById('statCredits').textContent = 'Credits 消耗 ' + fmtCredits(d.credits);
    document.getElementById('statPrompt').textContent = fmt(d.prompt_tokens);
    document.getElementById('statCached').textContent = '缓存命中 ' + fmt(d.cached_tokens);
    document.getElementById('statCompletion').textContent = fmt(d.completion_tokens);
    var avg = d.total > 0 ? Math.round((d.completion_tokens || 0) / d.total) : 0;
    document.getElementById('statAvgOut').textContent = '平均 ' + fmt(avg) + ' / 次';
    renderChart(d.hourly || []);
    renderStatsTable(d.by_model || []);
    renderFailures(d.failures || []);
  }).catch(function(e) { if (request === statsRequest) showToast(e.message, 'error'); });
}

function renderFailures(items) {
  var list = document.getElementById('failureList'); list.innerHTML = '';
  document.getElementById('failureCount').textContent = items.length + ' / 200 条';
  if (!items.length) { var empty = document.createElement('p'); empty.className = 'hint'; empty.textContent = '暂无失败记录。'; list.appendChild(empty); return; }
  var kinds = { timeout:'请求超时', queue_timeout:'排队超时', authentication:'认证失败', rate_limit:'请求限流', invalid_request:'参数错误', stream_error:'响应中断', client_cancelled:'客户端取消', upstream_error:'上游异常' };
  items.forEach(function(item) {
    var row = document.createElement('details'); row.className = 'failure-item';
    var summary = document.createElement('summary'); summary.className = 'failure-summary';
    var title = document.createElement('span'); title.className = 'failure-title'; title.textContent = item.model;
    var badge = document.createElement('span'); badge.className = 'failure-kind'; badge.textContent = (kinds[item.kind] || '请求失败') + ' · ' + item.status;
    var meta = document.createElement('span'); meta.className = 'failure-meta'; meta.textContent = timeLabel(item.at) + ' · ' + item.duration_ms + ' ms';
    summary.appendChild(title); summary.appendChild(badge); summary.appendChild(meta); row.appendChild(summary);
    var reason = document.createElement('pre'); reason.className = 'failure-reason'; reason.textContent = item.reason; row.appendChild(reason); list.appendChild(row);
  });
}

function fmtCredits(v) {
  var n = Number(v || 0);
  if (n === 0) return '0';
  if (n >= 1000) return n.toLocaleString('zh-CN', {maximumFractionDigits: 2});
  return n.toPrecision(6).replace(/\.?0+$/, '');
}

function renderChart(hourly) {
  var max = 0;
  hourly.forEach(function(h) { if (h.total > max) max = h.total; });
  document.getElementById('chartPeak').textContent = max > 0 ? '峰值 ' + fmt(max) : '暂无数据';

  var chart = document.getElementById('chart');
  chart.innerHTML = '';

  var plot = document.createElement('div');
  plot.className = 'chart-plot';
  var axis = document.createElement('div');
  axis.className = 'chart-axis';

  hourly.forEach(function(h, i) {
    var tip = h.label + ' · 总 ' + h.total + ' · 成功 ' + h.success + ' · 失败 ' + h.failed;

    var col = document.createElement('div');
    col.className = 'chart-col';
    col.title = tip;

    var bars = document.createElement('div');
    bars.className = 'chart-bars';
    if (h.total > 0 && max > 0) {
      var okH = Math.round(h.success / max * 100);
      var failH = Math.round(h.failed / max * 100);
      if (okH > 0) {
        var okBar = document.createElement('div');
        okBar.className = 'bar ok';
        okBar.style.height = okH + '%';
        bars.appendChild(okBar);
      }
      if (failH > 0) {
        var failBar = document.createElement('div');
        failBar.className = 'bar fail';
        failBar.style.height = failH + '%';
        bars.appendChild(failBar);
      }
    }
    col.appendChild(bars);
    plot.appendChild(col);

    var lab = document.createElement('div');
    lab.className = 'chart-axis-item';
    lab.textContent = (i % 4 === 0) ? h.label : '';
    lab.title = tip;
    axis.appendChild(lab);
  });

  chart.appendChild(plot);
  chart.appendChild(axis);
}

function renderStatsTable(rows) {
  var tb = document.getElementById('statsTable');
  tb.innerHTML = '';
  if (!rows || rows.length === 0) {
    tb.innerHTML = '<tr><td colspan="5"><div class="empty-row"><div class="empty-title">暂无统计数据</div>请求处理完成后会自动记录</div></td></tr>';
    return;
  }
  rows.forEach(function(r) {
    var tr = document.createElement('tr');
    var rate = r.success_rate;
    var cls = 'rate-fill';
    if (rate < 0.5) cls += ' bad';
    else if (rate < 0.9) cls += ' warn';
    tr.innerHTML =
      '<td><span class="mono">' + escapeHtml(r.model) + '</span></td>' +
      '<td class="num">' + fmt(r.total) + '</td>' +
      '<td class="num" style="color:var(--success)">' + fmt(r.success) + '</td>' +
      '<td class="num" style="color:var(--danger)">' + fmt(r.failed) + '</td>' +
      '<td><div class="rate-cell"><div class="rate-bar"><div class="' + cls + '" style="width:' + (rate * 100) + '%"></div></div>' +
      '<span class="rate-text num">' + pct(rate) + '</span></div></td>';
    tb.appendChild(tr);
  });
}

/* ---------- API Keys ---------- */

function loadKeys() {
  api('/keys').then(function(data) {
    var tb = document.getElementById('keysTable');
    tb.innerHTML = '';
    if (!data.keys || data.keys.length === 0) {
      tb.innerHTML = '<tr><td colspan="4"><div class="empty-row"><div class="empty-title">暂无密钥</div>在上方创建第一个密钥</div></td></tr>';
      return;
    }
    data.keys.forEach(function(k) {
      var tr = document.createElement('tr');
      var dt = new Date(k.created_at * 1000).toLocaleDateString('zh-CN');
      tr.innerHTML =
        '<td><span class="mono">' + escapeHtml(k.key) + '</span></td>' +
        '<td>' + (k.note ? escapeHtml(k.note) : '<span class="muted">—</span>') + '</td>' +
        '<td class="muted">' + dt + '</td>' +
        '<td><div class="btn-row" style="justify-content:flex-end"></div></td>';
      var btnRow = tr.querySelector('.btn-row');
      var copyBtn = document.createElement('button');
      copyBtn.className = 'btn secondary sm';
      copyBtn.type = 'button';
      copyBtn.textContent = '复制';
      copyBtn.addEventListener('click', function() { copyText(k.key); });
      var delBtn = document.createElement('button');
      delBtn.className = 'btn danger sm';
      delBtn.type = 'button';
      delBtn.textContent = '删除';
      delBtn.addEventListener('click', function() { deleteKey(k.id); });
      btnRow.appendChild(copyBtn);
      btnRow.appendChild(delBtn);
      tb.appendChild(tr);
    });
  }).catch(function(e) { showToast(e.message, 'error'); });
}

function addKey(gen) {
  var key = gen ? '' : document.getElementById('newKey').value.trim();
  var note = document.getElementById('newNote').value.trim();
  api('/keys', { method: 'POST', body: JSON.stringify({ key: key, note: note }) })
    .then(function() {
      document.getElementById('newKey').value = '';
      document.getElementById('newNote').value = '';
      showToast('密钥已创建');
      loadKeys();
    })
    .catch(function(e) { showToast(e.message, 'error'); });
}

function deleteKey(id) {
  if (!confirm('确定删除此密钥？')) return;
  api('/keys?id=' + encodeURIComponent(id), { method: 'DELETE' })
    .then(function() { showToast('密钥已删除'); loadKeys(); })
    .catch(function(e) { showToast(e.message, 'error'); });
}

/* ---------- WorkBuddy CN ---------- */

function loadProviders() {
  loadQoder();
  loadWorkBuddy();
  loadTrae();
}

function loadQoder() {
  return api('/qoder').then(function(data) {
    document.getElementById('qoderDisplay').value = data.token_masked || '';
    document.getElementById('qoderInput').value = '';
    var badge = document.getElementById('qoderBadge'); badge.textContent = data.configured ? '已配置 · 未验证' : '未配置'; badge.className = 'badge';
    document.getElementById('qoderStatusHint').textContent = '';
  }).catch(function(e) { showToast(e.message, 'error'); });
}
function saveQoder() {
  var pat = document.getElementById('qoderInput').value.trim(); if (!pat) { showToast('请输入 PAT', 'error'); return; }
  return saveChannel('qoder', { pat: pat }, loadQoder);
}
function clearQoder() {
  if (!confirm('清空 Qoder PAT？')) return;
  api('/qoder', { method: 'DELETE' }).then(function() { showToast('已清空'); loadQoder(); }).catch(function(e) { showToast(e.message, 'error'); });
}
function connectionTestMessage(result) {
  return '连接通过' + (typeof result.model_count === 'number' ? ' · ' + result.model_count + ' 个模型' : '');
}
var channelBusy = {};
function testChannel(provider) {
  if (channelBusy[provider]) return Promise.resolve();
  channelBusy[provider] = true;
  var save = document.getElementById(provider + 'SaveButton'), test = document.getElementById(provider + 'TestButton');
  var hint = document.getElementById(provider + 'StatusHint');
  var badge = document.getElementById(provider === 'workbuddy' ? 'wbBadge' : 'qoderBadge');
  save.disabled = test.disabled = true; test.textContent = '测试中…'; hint.textContent = '';
  return api('/channel/test', { method: 'POST', body: JSON.stringify({ provider: provider }) }).then(function(data) {
    badge.textContent = '最近测试通过'; badge.className = 'badge ok';
    hint.textContent = connectionTestMessage(data);
    showToast('连接测试通过');
  }).catch(function(e) {
    badge.textContent = '最近测试失败'; badge.className = 'badge bad';
    hint.textContent = '连接测试失败：' + e.message; showToast(e.message, 'error');
  }).finally(function() { channelBusy[provider] = false; save.disabled = test.disabled = false; test.textContent = '测试连接'; });
}
function saveChannel(provider, body, reload) {
  var button = document.getElementById(provider + 'SaveButton');
  if (button.disabled) return;
  button.disabled = true; button.textContent = '保存中…';
  return api('/' + provider, { method: 'POST', body: JSON.stringify(body) })
    .then(function() { return reload().then(function() { return testChannel(provider); }); })
    .catch(function(e) { showToast('保存失败：' + e.message, 'error'); })
    .finally(function() { button.disabled = false; button.textContent = '保存并测试连接'; });
}
function providerLabel(id) { return { 'qoder-cn':'Qoder CN', 'trae-cn':'Trae CN', 'workbuddy-cn':'WorkBuddy CN' }[id] || id; }

function loadWorkBuddy() {
  return api('/workbuddy').then(function(data) {
    document.getElementById('wbDisplay').value = data.token_masked || '';
    document.getElementById('wbInput').value = '';
    document.getElementById('workbuddyStatusHint').textContent = '';
    var b = document.getElementById('wbBadge');
    if (b) {
      if (data.configured) {
        b.className = 'badge';
        b.textContent = '已配置 · 未验证';
      } else {
        b.className = 'badge bad';
        b.textContent = '未配置';
      }
    }
  }).catch(function(e) { showToast(e.message, 'error'); });
}

function saveWorkBuddy() {
  var token = document.getElementById('wbInput').value.trim();
  if (!token) { showToast('WorkBuddy Token 不能为空', 'error'); return; }
  return saveChannel('workbuddy', { token: token }, loadWorkBuddy);
}

function clearWorkBuddy() {
  if (!confirm('确定清空 WorkBuddy 令牌？')) return;
  api('/workbuddy', { method: 'DELETE' })
    .then(function() { showToast('WorkBuddy 令牌已清空'); loadWorkBuddy(); })
    .catch(function(e) { showToast(e.message, 'error'); });
}

var traeProfiles = [];
var traeTesting = false;
function timeLabel(value) {
  if (!value) return '尚无记录';
  var date = new Date(value);
  return isNaN(date.getTime()) ? '时间未知' : date.toLocaleString('zh-CN', { hour12: false });
}
function traeErrorHint(message) {
  if (/Trae 企业版登录/.test(message)) return message;
  if (/HTTP 401|code["\s:]+1001|authenticate you/i.test(message)) return 'Trae 认证失败：请使用 Trae 中国区登录会话的访问令牌（Cloud-IDE-JWT），检查是否填错或已失效。';
  if (/HTTP 403/i.test(message)) return 'Trae 拒绝访问：请检查账号状态和所选模型权限。';
  if (/timeout|deadline exceeded/i.test(message)) return 'Trae 请求超时，请稍后重试并检查服务器网络。';
  return message;
}
function showTraeStatus(data) {
  var status = data.status || {};
  var seconds = status.expires_in_seconds;
  var badge = document.getElementById('traeBadge');
  var label = data.configured ? '已配置 · 未验证' : '未配置';
  var cls = data.configured ? 'badge' : 'badge bad';
  if (status.last_test && status.last_test.at) {
    label = status.last_test.success ? '最近测试通过' : '最近测试失败';
    cls = status.last_test.success ? 'badge ok' : 'badge bad';
  }
  if (data.configured && typeof seconds === 'number' && seconds <= 0) { label = '访问令牌已到期'; cls = 'badge warn'; }
  else if (data.configured && typeof seconds === 'number' && seconds < 86400) { label += ' · 即将到期'; if (cls !== 'badge bad') cls = 'badge warn'; }
  badge.textContent = label; badge.className = cls;
  var expiry = status.expires_at || data.expires_at;
  document.getElementById('traeExpiryStatus').textContent = !data.configured ? '未配置' : expiry ? timeLabel(expiry) + (seconds <= 0 ? '（已到期）' : seconds < 86400 ? '（24 小时内到期）' : '') : '上游未声明过期时间';
  document.getElementById('traeRefreshStatus').textContent = status.persistence_pending ? '会话已续期，等待重试保存' : status.refresh_available ? '已配置，到期前自动续期' : '未配置刷新凭据，需手动更新';
  document.getElementById('traeConcurrency').textContent = (status.active_requests || 0) + ' / ' + (status.queued_requests || 0);
  var hint = status.persistence_pending ? '会话保存失败，请检查配置文件权限。' : status.last_test && status.last_test.at ? (status.last_test.success ? connectionTestMessage(status.last_test) : '连接测试失败：' + traeErrorHint(status.last_test.error || '请重新测试')) : data.configured ? '令牌已保存，尚未验证连接。' : '尚未配置 Trae 令牌。';
  document.getElementById('traeStatusHint').textContent = hint;
  var container = document.getElementById('traeOperations'); container.innerHTML = '';
  [['连接测试', status.last_test], ['令牌续期', status.last_refresh], ['模型目录', status.last_catalog], ['模型调用', status.last_chat]].forEach(function(entry) {
    var op = entry[1] || {}, row = document.createElement('div'); row.className = 'operation-row';
    var label = document.createElement('span'); label.className = 'operation-label'; label.textContent = entry[0];
    var badge = document.createElement('span'); badge.className = 'operation-badge' + (op.at ? op.success ? ' success' : ' failed' : ''); badge.textContent = !op.at ? '暂无记录' : op.success ? '成功' : '失败';
    var time = document.createElement('span'); time.className = 'operation-time'; time.textContent = op.at ? timeLabel(op.at) + ' · ' + op.duration_ms + ' ms' : '—';
    row.appendChild(label); row.appendChild(badge); row.appendChild(time);
    if (op.error) {var error = document.createElement('pre'); error.className = 'failure-reason'; error.textContent = op.error; row.appendChild(error);}
    container.appendChild(row);
  });
}
function loadTraeStatus(silent) {
  return api('/trae').then(showTraeStatus).catch(function(e) { if (!silent) showToast(e.message, 'error'); });
}
function loadTrae() {
  return api('/trae').then(function(data) {
    document.getElementById('traeDisplay').value = data.token_masked || '';
    document.getElementById('traeInput').value = '';
    document.getElementById('traeUserID').value = data.user_id || '';
    document.getElementById('traeRefresh').value = '';
    document.getElementById('traeExpires').value = '';
    document.getElementById('traeClientID').value = data.client_id || 'ono9krqynydwx5';
    showTraeStatus(data);
    document.getElementById('traeTestModel').innerHTML = '<option value="">先测试连接，再选择模型</option>';
    return loadTraeAccounts();
  }).catch(function(e) { showToast(e.message, 'error'); });
}
function traeForm() {
  return { token: document.getElementById('traeInput').value.trim(), user_id: document.getElementById('traeUserID').value.trim(), refresh_token: document.getElementById('traeRefresh').value.trim(), expires_at: document.getElementById('traeExpires').value.trim(), client_id: document.getElementById('traeClientID').value };
}
function saveTrae() {
  if (traeTesting) return;
  var form = traeForm();
  if (!form.token) { showToast('请粘贴 Trae 访问令牌', 'error'); return; }
  var button = document.getElementById('traeSaveButton');
  button.disabled = true; button.textContent = '保存中…';
  return api('/trae', { method: 'POST', body: JSON.stringify(form) })
    .then(function() {
      document.getElementById('traeInput').value = '';
      document.getElementById('traeRefresh').value = '';
      return loadTrae().then(function() { return testTrae(false); });
    })
    .catch(function(e) { showToast('保存失败：' + e.message, 'error'); })
    .finally(function() { button.disabled = false; button.textContent = '保存并测试连接'; });
}
function clearTrae() {
  if (!confirm('确定清空当前 Trae 账号的令牌？已保存的其他账号不受影响。')) return;
  api('/trae', { method: 'DELETE' })
    .then(function() { showToast('当前 Trae 令牌已清空'); loadTrae(); })
    .catch(function(e) { showToast(e.message, 'error'); });
}
function testTrae(chat) {
  if (traeTesting) return Promise.resolve();
  var model = chat ? document.getElementById('traeTestModel').value : '';
  if (chat && !model) { showToast('请先测试连接，再选择测试模型', 'error'); return; }
  var connectionButton = document.getElementById('traeConnectionButton'), chatButton = document.getElementById('traeChatButton');
  traeTesting = true;
  connectionButton.disabled = true; chatButton.disabled = true;
  document.getElementById('traeSaveButton').disabled = true;
  var button = chat ? chatButton : connectionButton; var old = button.textContent; button.textContent = '测试中…';
  return api('/trae/test', { method: 'POST', body: JSON.stringify({ model: model }) }).then(function(result) {
    var select = document.getElementById('traeTestModel'); select.innerHTML = '<option value="">选择测试对话的模型</option>';
    (result.models || []).forEach(function(m) { var option = document.createElement('option'); option.value = m.id; option.textContent = m.name + ' · ' + m.id; select.appendChild(option); });
    if (model) select.value = model;
    showToast(result.chat_tested ? '对话测试通过' : '连接测试通过');
  }).catch(function(e) { showToast(traeErrorHint(e.message), 'error'); }).finally(function() {
    traeTesting = false;
    button.textContent = old; connectionButton.disabled = false; chatButton.disabled = false; document.getElementById('traeSaveButton').disabled = false; return loadTraeStatus(true);
  });
}
function loadTraeAccounts() {
  return api('/trae/accounts').then(function(data) {
    traeProfiles = data.accounts || []; var select = document.getElementById('traeAccountSelect');
    select.innerHTML = '<option value="">当前会话（未保存）</option>';
    var activeName = '当前会话（未命名）';
    traeProfiles.forEach(function(profile) { var option = document.createElement('option'); option.value = profile.id; option.textContent = profile.name + (profile.active ? ' · 当前使用' : '') + (!profile.configured ? ' · 无令牌' : ''); select.appendChild(option); if (profile.active) activeName = profile.name; });
    select.value = data.active_id || ''; document.getElementById('traeCurrentAccount').textContent = activeName; selectTraeProfile();
  });
}
function selectTraeProfile() {
  var id = document.getElementById('traeAccountSelect').value;
  var profile = traeProfiles.find(function(p) { return p.id === id; });
  document.getElementById('traeAccountName').value = profile ? profile.name : '';
}
function saveTraeAccount(asNew) {
  var form = traeForm(); form.action = 'save'; form.name = document.getElementById('traeAccountName').value.trim();
  form.id = asNew ? '' : document.getElementById('traeAccountSelect').value;
  if (!form.name) { showToast('请输入账号名称', 'error'); return; }
  if (!asNew && !form.id) { showToast('请选择已保存账号，或使用另存为新账号', 'error'); return; }
  api('/trae/accounts', { method: 'POST', body: JSON.stringify(form) }).then(function() { showToast('账号已保存并切换'); loadTrae(); }).catch(function(e) { showToast(e.message, 'error'); });
}
function activateTraeAccount() {
  var id = document.getElementById('traeAccountSelect').value;
  if (!id) { showToast('请选择已保存账号', 'error'); return; }
  api('/trae/accounts', { method: 'POST', body: JSON.stringify({ action: 'activate', id: id }) }).then(function() { showToast('账号已切换'); loadTrae(); }).catch(function(e) { showToast(e.message, 'error'); });
}
function deleteTraeAccount() {
  var id = document.getElementById('traeAccountSelect').value;
  if (!id) { showToast('请选择要删除的账号', 'error'); return; }
  if (!confirm('确定删除所选账号？删除当前账号会立即停用该渠道。')) return;
  api('/trae/accounts?id=' + encodeURIComponent(id), { method: 'DELETE' }).then(function() { showToast('账号已删除'); loadTrae(); }).catch(function(e) { showToast(e.message, 'error'); });
}
setInterval(function() { if (loggedIn && currentPage === 'token') loadTraeStatus(true); }, 15000);

/* ---------- Models ---------- */

var modelProvider = '';
var cachedModels = [];
function loadModels() {
  document.getElementById('modelCount').textContent = '加载中…';
  api('/models').then(function(data) {
    cachedModels = data.models || [];
    renderModels();
  }).catch(function(e) { document.getElementById('modelCount').textContent = '加载失败'; showToast(e.message, 'error'); });
}
function formatTokens(n) {
  if (!n) return '';
  if (n >= 1000000) return (n / 1000000).toFixed(0) + 'M';
  if (n >= 1000) return (n / 1000).toFixed(0) + 'K';
  return Number(n).toLocaleString('zh-CN');
}

function renderModels() {
  var provider = modelProvider ? modelProvider + '-cn' : '';
  var search = document.getElementById('modelSearch').value.trim().toLowerCase();
  var list = cachedModels.filter(function(item) {
    return (!provider || item.provider === provider) &&
           (!search || ((item.name || '') + ' ' + (item.display_name || '')).toLowerCase().includes(search));
  });
  var grid = document.getElementById('modelsList'); grid.innerHTML = '';
  document.getElementById('modelCount').textContent = list.length + ' / ' + cachedModels.length + ' 个模型';
  if (!list.length) { var empty = document.createElement('p'); empty.className = 'hint'; empty.textContent = '没有符合条件的模型'; grid.appendChild(empty); return; }
  list.forEach(function(item) {
    var name = item.name;
    var card = document.createElement('div'); card.className = 'model-item';
    var content = document.createElement('div'); content.className = 'model-card-content';
    var title = document.createElement('div'); title.className = 'model-card-title mono'; title.textContent = name; content.appendChild(title);

    var meta = document.createElement('div'); meta.className = 'model-card-meta';
    var provBadge = document.createElement('span'); provBadge.className = 'badge'; provBadge.textContent = providerLabel(item.provider); meta.appendChild(provBadge);

    var stripped = name.replace(/^[^/]+\//, '');
    if (item.display_name && item.display_name !== name && item.display_name !== stripped) {
      var nameTag = document.createElement('span'); nameTag.className = 'model-tag'; nameTag.textContent = item.display_name; meta.appendChild(nameTag);
    }
    if (item.context_window && item.context_window > 0) {
      var ctxTag = document.createElement('span'); ctxTag.className = 'model-tag'; ctxTag.textContent = '上下文 ' + formatTokens(item.context_window); meta.appendChild(ctxTag);
    }
    if (item.max_output_tokens && item.max_output_tokens > 0) {
      var outTag = document.createElement('span'); outTag.className = 'model-tag'; outTag.textContent = '输出 ' + formatTokens(item.max_output_tokens); meta.appendChild(outTag);
    }
    if (item.supports_tools === true) {
      var toolTag = document.createElement('span'); toolTag.className = 'model-tag ok'; toolTag.textContent = '工具调用'; meta.appendChild(toolTag);
    }
    if (item.provider === 'trae-cn') {
      var textTag = document.createElement('span'); textTag.className = 'model-tag'; textTag.textContent = '文本对话'; meta.appendChild(textTag);
    }

    content.appendChild(meta);
    card.appendChild(content);

    var copy = document.createElement('button'); copy.className = 'btn secondary sm'; copy.type = 'button'; copy.textContent = '复制'; copy.setAttribute('aria-label', '复制模型 ' + name); copy.addEventListener('click', function() { copyText(name); }); card.appendChild(copy);
    grid.appendChild(card);
  });
}

/* ---------- Config ---------- */

function loadConfig() {
  api('/config').then(function(data) {
    document.getElementById('cfgHost').value = data.host || '';
    document.getElementById('cfgPort').value = data.port || '';
    document.getElementById('cfgChatTimeout').value = data.chat_timeout_seconds || '';
    document.getElementById('cfgIdleTimeout').value = data.idle_timeout_seconds || '';
    document.getElementById('configHint').classList.add('hidden');
  }).catch(function(e) { showToast(e.message, 'error'); });
}

function saveConfig() {
  var host = document.getElementById('cfgHost').value.trim();
  var port = parseInt(document.getElementById('cfgPort').value, 10);
  if (!host) host = '0.0.0.0';
  if (!port || port < 1 || port > 65535) { showToast('端口无效', 'error'); return; }
  var body = { host: host, port: port };
  var ct = parseInt(document.getElementById('cfgChatTimeout').value, 10);
  var it = parseInt(document.getElementById('cfgIdleTimeout').value, 10);
  if (document.getElementById('cfgChatTimeout').value.trim() !== '') {
    if (!ct || ct < 1 || ct > 3600) { showToast('Chat 响应超时需在 1-3600 秒之间', 'error'); return; }
    body.chat_timeout_seconds = ct;
  }
  if (document.getElementById('cfgIdleTimeout').value.trim() !== '') {
    if (!it || it < 1 || it > 3600) { showToast('流空闲超时需在 1-3600 秒之间', 'error'); return; }
    body.idle_timeout_seconds = it;
  }
  api('/config', { method: 'POST', body: JSON.stringify(body) })
    .then(function(data) {
      showToast('配置已保存（超时配置即时生效）');
      var h = document.getElementById('configHint');
      h.textContent = data.restart || '修改主机或端口后需要重启服务才能生效。';
      h.classList.remove('hidden');
      loadConfig();
    })
    .catch(function(e) { showToast(e.message, 'error'); });
}

/* ---------- Password ---------- */

function savePassword() {
  var cur = document.getElementById('curPwd').value;
  var nw = document.getElementById('newPwd').value;
  if (!cur || !nw) { showToast('请填写完整', 'error'); return; }
  api('/password', { method: 'POST', body: JSON.stringify({ current_password: cur, new_password: nw }) })
    .then(function() {
      showToast('密码已修改');
      document.getElementById('curPwd').value = '';
      document.getElementById('newPwd').value = '';
    })
    .catch(function(e) { showToast(e.message, 'error'); });
}

/* ---------- Init ---------- */

var statsChannelTabs = createChannelTabs(document.getElementById('statsChannelTabs'), '统计渠道', selectStatsProvider);
var modelChannelTabs = createChannelTabs(document.getElementById('modelChannelTabs'), '模型渠道', function(provider) {
  modelProvider = provider; renderModels();
});
checkAuth();
</script>
</body>
</html>`
