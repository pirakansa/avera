package report

// htmlReport uses explicitly escaped template values and fixed fragment destinations.
// It has no scripts, remote resources, active evidence URLs, or runtime dependencies.
const htmlReport = `<!doctype html>
<html lang="{{.Language | escape}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; img-src 'none'; base-uri 'none'; form-action 'none'">
<title>{{.Title | escape}}</title>
<style>
:root{color-scheme:light;--ink:#172c32;--muted:#52676c;--paper:#f5f6f2;--line:#d5e0db;--accent:#096c61}
*{box-sizing:border-box}html{scroll-behavior:smooth;scroll-padding-top:1rem}body{margin:0;background:var(--paper);color:var(--ink);font:16px/1.75 system-ui,-apple-system,"Noto Sans JP",sans-serif}a{color:var(--accent);text-underline-offset:.2em}a:focus-visible,summary:focus-visible{outline:3px solid #d28b23;outline-offset:4px}header{background:#133c3b;color:#fff;padding:3.5rem max(1.5rem,calc((100vw - 1160px)/2));border-bottom:6px solid #8dc4a4}header p{letter-spacing:.22em;color:#b8d8c9;font-size:.8rem}h1{font-size:clamp(1.7rem,4vw,2.8rem);line-height:1.3;margin:.6rem 0}h2{font-size:1.35rem;margin:0 0 1rem}main{max-width:1200px;margin:auto;padding:2rem 1.25rem 5rem}.notice{border-left:4px solid #bd852c;background:#fff6df;padding:1rem 1.3rem;border-radius:0 8px 8px 0;color:#624719}.metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:1rem;margin:1.5rem 0 2rem}.metric{background:white;border:1px solid var(--line);border-top:4px solid #467871;border-radius:10px;padding:1.2rem}.metric:nth-child(2){border-top-color:#328258}.metric:nth-child(3),.metric:nth-child(5),.metric:nth-child(6){border-top-color:#b57b25}.metric strong{display:block;font-size:2.2rem;line-height:1.3;font-weight:650;font-variant-numeric:tabular-nums}.metric span{font-size:.85rem;color:var(--muted)}nav{display:flex;gap:.6rem 1.5rem;flex-wrap:wrap;border-block:1px solid var(--line);padding:1rem 0;margin-bottom:2rem}section,details{background:#fff;border:1px solid var(--line);border-radius:12px;padding:1.5rem 1.8rem;margin:1rem 0;overflow-wrap:anywhere;box-shadow:0 3px 12px #173d3210}section[id^="item-"]{border-left:4px solid #467871}section:target,details:target{outline:3px solid #bd852c}section>p,details>p{color:var(--muted)}dl{margin:0}dl>div{display:grid;grid-template-columns:minmax(10rem,24%) minmax(0,1fr);gap:1.5rem;padding:.65rem 0;border-bottom:1px solid #edf0ed}dt{font-size:.85rem;font-weight:600;color:var(--muted)}dd{margin:0;white-space:pre-wrap;font-size:.95rem}ul{background:#f0f6f3;border-radius:8px;padding:1rem 1rem 1rem 2.5rem}li+li{margin-top:.7rem}.links{display:flex;gap:1rem;flex-wrap:wrap;margin-top:1rem}.links a{font-size:.9rem}summary{cursor:pointer;font-weight:650;font-size:1.1rem}details[open] summary{margin-bottom:1rem}footer{text-align:center;font-size:.8rem;color:var(--muted);padding:1rem}@media(max-width:650px){.metrics{grid-template-columns:repeat(2,minmax(0,1fr))}dl>div{display:block}dt{margin-bottom:.25rem}section,details{padding:1rem}header{padding:2rem 1.25rem}}@media print{body{background:white;font-size:10pt}header{background:white;color:#172c32;padding:0;border-bottom:2px solid}header p{color:#52676c}main{padding:1rem 0}nav{display:none}section,details{box-shadow:none;break-inside:avoid}.metrics{grid-template-columns:repeat(4,1fr)}a{color:inherit}}
.metric.metric-action{background:#fff0ee;border:2px solid #b73529;border-top:6px solid #b73529;color:#8c211a}.metric.metric-unreviewed{background:#fff6df;border:2px solid #aa6b08;border-top:6px solid #aa6b08;color:#754700}.metric.metric-action strong,.metric.metric-unreviewed strong{font-size:2.7rem;font-weight:750}.metric.metric-action span,.metric.metric-unreviewed span{color:inherit;font-weight:700}.metric .metric-notice{display:block;width:fit-content;margin-top:.65rem;padding:.15rem .6rem;border:1px solid currentColor;border-radius:4px;font-size:.8rem;font-weight:700}
` + NetworkMapStyles + `
</style>
</head>
<body>
<header><p>AVERA / REVIEW</p><h1>{{.Title | escape}}</h1></header>
<main>
<p class="notice">{{.Warning | escape}}</p>
<h2>{{.Summary | escape}}</h2>
<div class="metrics">{{range .Metrics}}<div class="metric{{if .Tone}} {{.Tone | escape}}{{end}}"><strong>{{.Count | escape}}</strong><span>{{.Label | escape}}</span>{{if .Notice}}<span class="metric-notice">{{.Notice | escape}}</span>{{end}}</div>{{end}}</div>
{{if .Network}}{{template "networkMap" .Network}}{{end}}
<nav aria-label="{{.Contents | escape}}">{{range .Blocks}}{{if or (eq .ID "run") (eq .ID "scope") (eq .ID "current") (eq .ID "historical") (eq .ID "local-reference") (eq .ID "routes") (eq .ID "unmatched") (eq .ID "evidence")}}<a href="#{{.ID | escape}}">{{.Title | escape}}</a>{{end}}{{end}}</nav>
{{range .Blocks}}
{{if .Technical}}<details id="{{.ID | escape}}"><summary>{{.Title | escape}} · {{$.Details | escape}}</summary>{{else}}<section id="{{.ID | escape}}"><h2>{{.Title | escape}}</h2>{{end}}
{{if .Note}}<p>{{.Note | escape}}</p>{{end}}
{{if .Fields}}<dl>{{range .Fields}}<div><dt>{{.Label | escape}}</dt><dd>{{.Value | escape}}</dd></div>{{end}}</dl>{{end}}
{{if .Questions}}<ul>{{range .Questions}}<li>{{. | escape}}</li>{{end}}</ul>{{end}}
{{if .Links}}<div class="links">{{range .Links}}<a href="#{{.ID | escape}}">{{.Label | escape}}</a>{{end}}</div>{{end}}
{{if .Technical}}</details>{{else}}</section>{{end}}
{{end}}
</main><footer>AVERA · {{.Title | escape}}</footer>
</body></html>
{{define "mapCard"}}<div class="map-card {{.Tone | escape}}">{{if .Anchor}}<a href="#{{.Anchor | escape}}">{{.Title | escape}}</a>{{else}}<strong>{{.Title | escape}}</strong>{{end}}<p class="map-state">{{.Decision | escape}}</p>{{if .Detail}}<p>{{.Detail | escape}}</p>{{end}}{{if .Purpose}}<p>{{.Purpose | escape}}</p>{{end}}{{if .Safeguards}}<p>{{.Safeguards | escape}}</p>{{end}}</div>{{end}}
{{define "networkMap"}}<section id="network-map" class="network-map"><h2>{{.Title | escape}}</h2><p>{{.Note | escape}}</p><div class="map-outside">{{.Outside | escape}}</div><div class="map-host"><h3>{{.Host | escape}}</h3>{{range .Rows}}<div class="map-row {{.Mode | escape}}">{{template "mapCard" .Source}}<div class="map-arrow">{{if ne .Mode "unknown"}}{{.Relation | escape}}{{end}}</div><div class="map-targets">{{range .Targets}}{{template "mapCard" .}}{{else}}<div class="map-unknown">{{.Relation | escape}}</div>{{end}}</div></div>{{else}}<p>{{.Empty | escape}}</p>{{end}}{{if .References}}<h3>{{.ReferencesLabel | escape}}</h3><div class="map-reference-grid">{{range .References}}{{template "mapCard" .}}{{end}}</div>{{end}}</div><p class="map-legend">{{.Legend | escape}}</p>{{if .LimitNote}}<p class="notice">{{.LimitNote | escape}}</p>{{end}}<ul>{{range .Coverage}}<li>{{. | escape}}</li>{{end}}</ul></section>{{end}}
`
