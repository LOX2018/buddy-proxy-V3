package admin

import (
	"html"
	"strings"
)

// UIRevision is the admin console markup/script stamp. Bump it on every
// meaningful change to PageHTML so operators can tell the rebuilt binary
// carries the new UI (independent of the release tag injected via ldflags).
const UIRevision = "2026.10.04-oauth-inprivate"

func PageHTML() string {
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<meta name="cbp-ui-revision" content="` + UIRevision + `"/>
<title>CodeBuddy Proxy · Console</title>
<style>
:root{
  --bg:#0e1116;
  --surface:#151b24;
  --surface-2:#1a212c;
  --fg:#eef2f8;
  --fg-80:rgba(238,242,248,.86);
  --fg-70:rgba(238,242,248,.76);
  --fg-60:rgba(238,242,248,.62);
  --fg-40:rgba(238,242,248,.46);
  --fg-20:rgba(238,242,248,.24);
  --fg-16:rgba(238,242,248,.19);
  --fg-12:rgba(238,242,248,.16);
  --fg-10:rgba(238,242,248,.14);
  --fg-08:rgba(238,242,248,.11);
  --invert-bg:#3fc6b0;
  --invert-fg:#0c1216;
  --accent:#3fc6b0;
  --accent-soft:rgba(63,198,176,.14);
  --danger:#ff7a7a;
  --display:Georgia,"Iowan Old Style","Palatino Linotype",Palatino,"Songti SC","Noto Serif SC",serif;
  --sans:system-ui,-apple-system,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;
  --mono:ui-monospace,"SF Mono",Menlo,Consolas,monospace;
}
*{box-sizing:border-box}
::selection{background:rgba(63,198,176,.35);color:#fff}
html,body{margin:0;min-height:100%}
body{
  font-family:var(--sans);
  font-size:14px;line-height:1.625;
  color:var(--fg);
  background:radial-gradient(1100px 520px at 18% -8%, rgba(63,198,176,.07), transparent 60%), radial-gradient(900px 480px at 100% 0%, rgba(108,140,255,.06), transparent 55%), var(--bg);
  letter-spacing:-.01em;
}
a{color:inherit;text-decoration:none}
.shell{max-width:1200px;margin:0 auto;padding:0 24px 64px}

/* 顶栏 */
.topbar{
  position:sticky;top:0;z-index:50;
  display:flex;align-items:center;justify-content:center;gap:16px;flex-wrap:wrap;
  min-height:58px;padding:14px 0 16px;margin-bottom:24px;
  background:rgba(14,17,22,.86);backdrop-filter:blur(10px);
  border-bottom:1px solid var(--fg-10);
}
.brand{display:flex;align-items:baseline;gap:12px}
.brand strong{
  font-family:var(--display);font-size:18px;font-weight:400;
  letter-spacing:.2em;text-transform:uppercase;color:var(--fg);
}
.brand span{
  font-family:var(--sans);font-size:11px;
  letter-spacing:.15em;text-transform:uppercase;color:var(--fg-40);
}
.mark{display:none}
.pillrow{
  display:flex;align-items:center;gap:0;flex-wrap:nowrap;flex-shrink:0;
  font-family:var(--sans);font-size:14px;font-weight:500;letter-spacing:.08em;text-transform:uppercase;
  color:var(--fg-70);
}
.pill{display:inline-flex;align-items:center;gap:8px;flex:0 0 auto;white-space:nowrap}
.pill + .pill::before{content:"·";margin:0 12px;color:var(--fg-40)}
.pill .dot{
  width:7px;height:7px;border-radius:50%;background:var(--accent);
  box-shadow:0 0 6px rgba(63,198,176,.8);
}
.pill .dot.bad{background:transparent;border:1px solid var(--danger);box-shadow:none;border-color:var(--danger)}
.pill[hidden]{display:none}
.pill.upd{color:#ffc456;cursor:pointer;position:relative}
.pill.upd:hover{text-decoration:underline}

/* 核心切页导航条 (Editorial Chapter Navigation) */
.tabs-nav{
  display:flex;
  border:1px solid var(--fg-12);border-radius:10px;overflow-x:auto;
  background:var(--surface);box-shadow:0 2px 12px rgba(0,0,0,.25);
  margin-bottom:32px;
}
.tab-link{
  flex:1;min-width:140px;
  padding:14px 18px;background:transparent;border:none;
  border-right:1px solid var(--fg-08);
  font-family:var(--sans);font-size:13px;letter-spacing:.05em;
  color:var(--fg-60);cursor:pointer;text-align:left;
  transition:color 200ms ease,background-color 200ms ease;
  display:flex;align-items:baseline;gap:8px;
}
.tab-link:last-child{border-right:none}
.tab-link:hover{color:var(--fg);background:var(--accent-soft)}
.tab-link.active{
  color:var(--fg);font-weight:600;
  box-shadow:inset 0 -3px 0 var(--accent);
  background:var(--accent-soft);
}
.tab-idx{
  font-family:var(--display);font-style:italic;font-size:15px;color:var(--accent);
  opacity:.65;
}
.tab-link.active .tab-idx{color:var(--accent);opacity:1}

/* 切页容器 */
.tab-panel{display:none}
.tab-panel.active{
  display:block;
  animation:panelFade 250ms ease-out;
}
@keyframes panelFade{
  from{opacity:0;transform:translateY(4px)}
  to{opacity:1;transform:translateY(0)}
}

/* 卡片与面板 */
.panel{
  border:1px solid var(--fg-12);border-radius:12px;
  background:linear-gradient(180deg,rgba(255,255,255,.02),rgba(255,255,255,0) 60%),var(--surface);
  padding:24px;margin-bottom:24px;
  transition:border-color 200ms ease,transform 200ms ease;
}
.panel:hover{border-color:rgba(63,198,176,.55)}
.panel-inner{height:100%}
.eyebrow{
  font-size:11px;font-family:var(--sans);letter-spacing:.2em;text-transform:uppercase;
  color:var(--accent);margin-bottom:8px;font-weight:600;
}
h1{
  margin:0 0 8px;font-family:var(--display);font-weight:400;letter-spacing:-.02em;
  font-size:clamp(24px,3vw,32px);color:var(--fg);
}
.lede{margin:0;color:var(--fg-70);font-size:13.5px;line-height:1.7;max-width:56ch}
.checkin-hint{margin:12px 0 0;font-size:11.5px;line-height:1.6;color:var(--fg-60);max-width:60ch}
.checkin-detail{margin:12px 0 0;padding:12px 14px;border:1px solid var(--fg-12);border-radius:8px;background:rgba(0,0,0,.25);font-family:var(--mono);font-size:11.5px;line-height:1.6;color:var(--fg-80);white-space:pre-wrap}
.test-model-label{display:inline-flex;align-items:center;gap:8px;font-size:12px;color:var(--fg-60)}
.test-model-label select{min-width:160px;font-size:13px}
.metrics{
  display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:12px;margin-top:24px;
}
@media (max-width:980px){.metrics{grid-template-columns:repeat(3,minmax(0,1fr))}}
@media (max-width:720px){.metrics{grid-template-columns:repeat(2,minmax(0,1fr))}}
.metric{
  position:relative;overflow:hidden;
  padding:16px;border:1px solid var(--fg-12);border-radius:10px;background:var(--surface-2);
}
.metric::before{
  content:"";position:absolute;left:0;top:0;bottom:0;width:3px;
  background:var(--accent);opacity:.7;
}
.metric .k{font-size:11px;font-family:var(--sans);letter-spacing:.15em;text-transform:uppercase;color:var(--fg-60);font-weight:600}
.metric .v{
  margin-top:8px;font-family:var(--mono);font-size:24px;font-weight:400;
  letter-spacing:-.02em;color:var(--fg);text-align:right;
}
.metric .v.sm{font-size:13.5px;line-height:1.4;color:var(--fg-80)}
.section-head{
  display:flex;align-items:flex-end;justify-content:space-between;gap:12px;flex-wrap:wrap;
  margin:0 0 16px;padding-bottom:12px;border-bottom:1px solid var(--fg-10);
}
.section-head h2{
  margin:0;font-family:var(--display);font-size:18px;font-weight:400;
  letter-spacing:-.01em;color:var(--fg);
}
.section-head p{margin:4px 0 0;color:var(--fg-60);font-size:12px}
.actions{display:flex;gap:10px;flex-wrap:wrap}
button,.btn,.linkbtn{
  appearance:none;border:1px solid var(--fg-16);border-radius:8px;
  background:var(--surface-2);color:var(--fg);cursor:pointer;
  display:inline-flex;align-items:center;justify-content:center;gap:8px;
  padding:8px 16px;font-family:var(--sans);font-size:12px;
  letter-spacing:.06em;text-transform:uppercase;
  transition:color 200ms ease,border-color 200ms ease,background-color 200ms ease,box-shadow 200ms ease;
}
button:hover,.btn:hover,.linkbtn:hover{
  color:#fff;border-color:var(--accent);background:var(--accent-soft);
}
button:active,.btn:active,.linkbtn:active{opacity:.8}
button:focus-visible,.btn:focus-visible,.linkbtn:focus-visible,select:focus-visible,input:focus-visible{
  outline:2px solid var(--accent);outline-offset:2px;
}
button.primary,.btn.primary,button.teal{
  background:var(--invert-bg);color:var(--invert-fg);border-color:var(--invert-bg);font-weight:600;
  box-shadow:0 0 0 1px rgba(63,198,176,.3),0 4px 14px rgba(63,198,176,.18);
}
button.primary:hover,.btn.primary:hover,button.teal:hover{
  background:#55d8c3;color:var(--invert-fg);border-color:#55d8c3;opacity:1;
}
button.ghost,.linkbtn{background:transparent;color:var(--fg-80);border-color:var(--fg-12)}
button.danger{color:var(--danger);border-color:rgba(255,122,122,.4)}
button.danger:hover{background:rgba(255,122,122,.12);color:var(--danger);border-color:var(--danger)}
.field-grid{display:grid;grid-template-columns:1fr 1.4fr;gap:16px;margin:8px 0 16px}
@media (max-width:640px){.field-grid{grid-template-columns:1fr}}
label{
  display:block;font-size:11px;font-family:var(--sans);letter-spacing:.18em;
  text-transform:uppercase;color:var(--fg-60);margin-bottom:6px;font-weight:600;
}
select,input{
  width:100%;padding:8px 12px;border:1px solid var(--fg-16);border-radius:8px;
  background:rgba(0,0,0,.22);color:var(--fg);font-family:var(--sans);font-size:14px;
  outline:none;transition:border-color 200ms ease,box-shadow 200ms ease;
}
select:focus,input:focus{border-color:var(--accent);box-shadow:0 0 0 3px var(--accent-soft)}
textarea{
  width:100%;padding:10px 12px;border:1px solid var(--fg-16);border-radius:8px;
  background:rgba(0,0,0,.22);color:var(--fg);font-family:var(--mono);font-size:12px;line-height:1.6;
  outline:none;transition:border-color 200ms ease;resize:vertical;
}
textarea:focus{border-color:var(--accent)}
.policy-switch{display:flex;align-items:center;gap:16px;margin:4px 0 16px;flex-wrap:wrap}
.toggle{display:flex;align-items:center;gap:10px;margin:0;cursor:pointer;user-select:none}
.toggle input{width:auto;padding:0;border:none;accent-color:var(--accent);height:16px;width:16px;cursor:pointer}
.toggle span{
  font-size:11px;letter-spacing:.14em;text-transform:uppercase;color:var(--fg-80);margin:0;font-weight:600;
}
.policy-switch .meta-line{margin:0}
.act-group{border-bottom:1px solid var(--fg-08)}
.act-group:last-child{border-bottom:none}
.act-group-head{
  display:flex;align-items:center;gap:10px;padding:8px 14px;
  background:linear-gradient(90deg,var(--accent-soft),transparent 70%);
  font-family:var(--sans);
}
.act-group-head .gname{
  font-size:10.5px;letter-spacing:.18em;text-transform:uppercase;color:var(--fg);font-weight:700;
}
.act-group-head .gcount{
  font-family:var(--mono);font-size:11px;color:var(--accent);border:1px solid rgba(63,198,176,.45);
  padding:1px 7px;border-radius:99px;
}
.act-group-head .gtime{
  margin-left:auto;font-family:var(--mono);font-size:11px;color:var(--fg-60);
}
.act-group .act-item{padding:8px 14px;font-size:12.5px}
.oauth-status{
  margin-top:16px;padding:12px 14px;border:1px solid var(--fg-12);border-radius:8px;
  background:var(--surface-2);
}
.oauth-status .title{
  font-size:10.5px;letter-spacing:.18em;text-transform:uppercase;color:var(--fg-60);margin-bottom:6px;font-weight:600;
}
.oauth-status .msg{font-size:13px;line-height:1.5;color:var(--fg-80)}

.account{
  display:grid;gap:10px;padding:18px;border:1px solid var(--fg-12);border-radius:10px;
  background:var(--surface-2);
  transition:border-color 200ms ease,transform 200ms ease;margin-bottom:12px;
}
.account:hover{border-color:rgba(63,198,176,.5);transform:translateY(-1px)}
.account-top{display:flex;justify-content:space-between;gap:12px;flex-wrap:wrap;align-items:center}
.account-title{display:flex;flex-wrap:wrap;gap:8px;align-items:center}
.account-title strong{
  font-family:var(--display);font-size:16px;font-weight:400;color:var(--fg);letter-spacing:-.01em;
}
.badge{
  display:inline-flex;align-items:center;padding:2px 7px;border:1px solid var(--fg-16);border-radius:6px;
  font-size:10.5px;font-family:var(--sans);letter-spacing:.1em;text-transform:uppercase;
  color:var(--fg-70);background:rgba(238,242,248,.04);
}
.badge.site{border-color:rgba(63,198,176,.5);color:var(--accent);background:var(--accent-soft)}
.badge.muted{border-color:var(--fg-10);color:var(--fg-60);background:rgba(238,242,248,.03)}
.badge.on{border-color:var(--accent);color:#0c1216;background:var(--accent);font-weight:600}
.badge.off{border-color:var(--fg-16);color:var(--fg-60)}
.seg{display:inline-flex;gap:0;border:1px solid var(--fg-16);border-radius:9px;overflow:hidden;background:var(--surface-2)}
.seg button{
  border:none;border-right:1px solid var(--fg-12);background:transparent;color:var(--fg-60);
  padding:7px 15px;font-size:12px;letter-spacing:.08em;text-transform:uppercase;cursor:pointer;
  transition:background-color 200ms ease,color 200ms ease;
}
.seg button:last-child{border-right:none}
.seg button:hover{color:var(--fg);background:rgba(238,242,248,.06)}
.seg button.active{background:var(--invert-bg);color:var(--invert-fg);font-weight:600}
.seg-hint{margin:10px 0 16px;color:var(--fg-60);font-size:12px}
.meta{color:var(--fg-70);font-size:12.5px;line-height:1.5}
.meta code,.mono{font-family:var(--mono);font-size:12px}
.err{
  margin-top:4px;color:var(--danger);font-family:var(--mono);font-size:12px;
  border-left:2px solid var(--danger);padding:4px 0 4px 8px;
  background:rgba(255,122,122,.06);border-radius:0 6px 6px 0;
}
.chips{display:flex;flex-wrap:wrap;gap:8px}
.chip{
  padding:6px 12px;border:1px solid var(--fg-16);border-radius:99px;font-size:12px;font-family:var(--mono);
  color:var(--fg);background:var(--surface-2);transition:border-color 200ms ease,color 200ms ease,font-style 500ms ease;
}
.chip.btnish{cursor:pointer}
.chip.btnish:hover{border-color:var(--accent);color:var(--accent);font-style:italic}
.empty{
  padding:32px 16px;border:1px dashed var(--fg-20);border-radius:8px;text-align:center;color:var(--fg-60);
  font-family:var(--sans);font-size:13px;
}
.hidden{display:none!important}
details.raw{margin-top:12px;border:1px solid var(--fg-12);border-radius:8px;background:var(--surface-2)}
details.raw summary{
  cursor:pointer;list-style:none;padding:10px 14px;font-size:11px;letter-spacing:.18em;
  text-transform:uppercase;color:var(--fg-60);font-family:var(--sans);transition:color 200ms ease;
}
details.raw summary:hover{color:var(--accent)}
details.raw summary::-webkit-details-marker{display:none}
pre{
  margin:0;padding:0 14px 14px;white-space:pre-wrap;word-break:break-word;
  font-family:var(--mono);font-size:11.5px;line-height:1.6;color:var(--fg-80);
  max-height:280px;overflow:auto;
}
.copyline{display:grid;grid-template-columns:1fr auto;gap:12px;align-items:center}
.copyline input{font-family:var(--mono);font-size:12.5px}
.bound-keys{margin-top:18px;display:flex;flex-direction:column;gap:0}
.bound-key{
  display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap;
  padding:12px 0;border-top:1px solid var(--fg-10);
}
.bound-key .meta{display:flex;flex-wrap:wrap;gap:8px;align-items:center}
.bound-keys-new{margin-top:16px}
.bound-keys-new label{
  display:block;font-size:11px;letter-spacing:.12em;text-transform:uppercase;color:var(--fg-60);margin-bottom:8px;font-weight:600;
}
.bound-keys-new select{
  width:100%;padding:8px 12px;border:1px solid var(--fg-16);border-radius:8px;
  background:rgba(0,0,0,.22);font:inherit;color:var(--fg);
}
.secret-hint{margin-top:12px;font-size:12px;color:var(--fg-60);line-height:1.5}
.toast{
  position:fixed;right:24px;bottom:24px;z-index:100;min-width:180px;max-width:min(420px,92vw);
  padding:12px 20px;background:var(--invert-bg);color:var(--invert-fg);font-size:12px;
  letter-spacing:.08em;text-transform:uppercase;font-family:var(--sans);font-weight:600;
  border:1px solid var(--invert-bg);border-radius:8px;box-shadow:0 8px 24px rgba(0,0,0,.35);
  opacity:0;pointer-events:none;transition:opacity 300ms ease;
}
.toast.show{opacity:1}
.toast.err{background:var(--danger);color:#12060a;border-color:var(--danger)}
.modal-mask{
  position:fixed;inset:0;z-index:120;display:flex;align-items:center;justify-content:center;
  background:rgba(8,11,15,.72);backdrop-filter:blur(4px);padding:24px;
}
.modal-mask[hidden]{display:none}
.modal{
  width:min(520px,100%);max-height:78dvh;overflow:auto;
  background:var(--surface);border:1px solid var(--fg-16);border-radius:12px;
  box-shadow:0 24px 64px rgba(0,0,0,.5);
}
.modal .modal-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;flex-wrap:wrap;padding:22px 22px 0}
.modal .modal-head .eyebrow{margin:0}
.modal h2{margin:6px 0 0;font-family:var(--display);font-size:22px;font-weight:400;letter-spacing:-.02em}
.modal .modal-meta{padding-top:4px;font-size:12px;letter-spacing:.04em;color:var(--fg-60);text-align:right;line-height:1.6}
.modal .modal-body{
  margin:14px 22px 0;padding-top:14px;border-top:1px solid var(--fg-10);
  font-size:12.5px;line-height:1.7;color:var(--fg-80);
  white-space:pre-wrap;word-break:break-word;overflow-wrap:anywhere;
}
.modal .modal-actions{display:flex;justify-content:flex-end;gap:10px;padding:20px 22px 22px}
.modal .modal-actions a,.modal .modal-actions button{
  display:inline-flex;align-items:center;justify-content:center;min-width:96px;
  padding:9px 16px;font-size:11px;letter-spacing:.1em;text-transform:uppercase;font-weight:600;
  border-radius:6px;cursor:pointer;text-decoration:none;
}
.modal .modal-actions a{background:var(--accent);color:#08130f;border:1px solid var(--accent)}
.modal .modal-actions a:hover{filter:brightness(1.12)}
.modal .modal-actions button{background:transparent;color:var(--fg-70);border:1px solid var(--fg-16)}
.modal .modal-actions button:hover{color:var(--fg);border-color:var(--fg-40)}
.usage-line{margin-top:4px;font-size:12px}
.usage-line .pill{display:inline-flex;padding:2px 8px;font-family:var(--mono);font-size:11px;border:1px solid var(--fg-16);border-radius:99px;color:var(--fg)}
.usage-line .pill.good{border-color:var(--accent);color:var(--accent);font-weight:500}
.usage-line .pill.warn{border-color:rgba(255,196,86,.5);color:#ffc456;font-weight:500}
.usage-line .pill.bad{border-color:rgba(255,122,122,.4);color:var(--danger)}
.account .actions button{padding:4px 10px;font-size:11px;color:var(--fg-70);border-color:var(--fg-12)}
.account .actions button:hover{color:var(--accent);border-color:var(--accent)}
.usage-filters{display:flex;flex-wrap:wrap;gap:16px 24px;margin:16px 0 0;align-items:end}
.usage-filters label{display:flex;flex-direction:column;gap:6px;font-size:11px;letter-spacing:.12em;text-transform:uppercase;color:var(--fg-60);font-weight:600}
.usage-filters select{min-width:180px;font-size:13px;text-transform:none;letter-spacing:0}
.usage-models{margin-top:16px;overflow:auto;border:1px solid var(--fg-12);border-radius:8px;background:var(--surface-2)}
.usage-models table{width:100%;border-collapse:collapse;font-size:12px}
.usage-models th,.usage-models td{padding:8px 12px;border-bottom:1px solid var(--fg-08);text-align:left}
.usage-models th{font-size:11px;letter-spacing:.12em;text-transform:uppercase;color:var(--fg-60);font-weight:600;background:rgba(0,0,0,.22)}
.usage-models td.mono{font-family:var(--mono);font-size:11px}
.usage-models .low{color:var(--fg-60)}

/* 概览下方双栏注释卡片 */
.overview-notes{
  display:grid;grid-template-columns:1fr 1fr;gap:20px;margin-top:24px;
}
@media (max-width:800px){.overview-notes{grid-template-columns:1fr}}
.note-item{
  border-top:1px solid var(--fg-10);padding-top:14px;
}
.note-item h3{
  font-family:var(--display);font-size:16px;font-weight:400;margin-bottom:6px;
}
.note-item p{
  font-size:12.5px;color:var(--fg-70);line-height:1.6;margin:0;
}

.log-table-wrap{margin-top:16px;border:1px solid var(--fg-12);border-radius:8px;background:var(--surface-2);overflow:hidden}
.log-table-window{max-height:min(340px,42vh);overflow:auto}
.log-table{width:100%;border-collapse:collapse;font-size:12px;min-width:880px}
.log-pager{
  display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap;
  padding:10px 12px;border-top:1px solid var(--fg-10);font-size:12px;color:var(--fg-70);
}
.log-pager .actions button{padding:6px 12px;font-size:11px}
.log-table th,.log-table td{padding:10px 12px;border-bottom:1px solid var(--fg-08);text-align:left;vertical-align:top}
.log-table th{
  font-size:10px;letter-spacing:.15em;text-transform:uppercase;color:var(--fg-60);font-weight:600;
  position:sticky;top:0;background:var(--surface-2);
}
.log-table td.mono{font-family:var(--mono);font-size:11px}
.log-table tr:hover td{background:rgba(238,242,248,.04)}
.log-table .idbtn{
  border:none;background:transparent;padding:0;font-family:var(--mono);font-size:11px;
  color:var(--accent);cursor:pointer;text-decoration:underline;text-underline-offset:3px;
}
.log-table .idbtn:hover{color:#7fe3d3}

.usage-chart{
  margin-top:20px;padding:16px 12px 8px;border:1px solid var(--fg-12);border-radius:10px;background:var(--surface-2);
}
.usage-chart-head{
  display:flex;align-items:baseline;justify-content:space-between;gap:12px;flex-wrap:wrap;
  margin-bottom:12px;padding-bottom:8px;border-bottom:1px solid var(--fg-10);
}
.usage-chart-head h3{
  margin:0;font-family:var(--display);font-size:16px;font-weight:400;color:var(--fg);
}
.usage-chart-legend{
  display:flex;gap:16px;font-size:11px;letter-spacing:.12em;text-transform:uppercase;color:var(--fg-70);
}
.usage-chart-legend span{display:inline-flex;align-items:center;gap:6px}
.usage-chart-legend .swatch{width:18px;height:0;border-top:2px solid var(--accent)}
.usage-chart-legend .swatch.dash{border-top-style:dashed;border-top-color:var(--fg-40)}
.chart-empty{padding:32px 12px;text-align:center;color:var(--fg-60);font-size:13px}
.usage-chart svg{display:block;width:100%;height:auto;max-height:200px}
.usage-chart .axis{stroke:rgba(238,242,248,.15);stroke-width:1}
.usage-chart .line-tokens{fill:none;stroke:var(--accent);stroke-width:1.5}
.usage-chart .line-rate{fill:none;stroke:var(--fg-60);stroke-width:1.5;stroke-dasharray:5 4}
.usage-chart .lbl{fill:var(--fg-60);font-family:var(--sans);font-size:10px}

#statusBox,#oauthBox,#modelsBox{display:none}

.activity-list{border:1px solid var(--fg-12);border-radius:10px;overflow:hidden;background:var(--surface-2)}
.act-item{display:flex;flex-wrap:wrap;gap:4px 12px;align-items:baseline;padding:9px 14px;border-bottom:1px solid var(--fg-08);font-size:13px;font-family:var(--mono)}
.act-item:last-child{border-bottom:none}
.act-item:hover{background:rgba(238,242,248,.05)}
.act-time{color:var(--fg-60);white-space:nowrap;min-width:150px}
.act-kind{color:var(--accent);font-weight:600}
.act-name{color:var(--fg);font-weight:700}
.act-fields{color:var(--fg-70)}
@media (max-width:640px){.act-time{min-width:0}}
@media (prefers-reduced-motion: reduce){
  *,*::before,*::after{animation:none !important;transition:none !important}
}
</style>
</head>
<body>
<div class="shell">
  <header class="topbar">
    <div class="pillrow">
      <span class="pill"><span class="dot" id="healthDot"></span><span id="healthText">检查中</span></span>
      <span class="pill upd" id="pillUpdate" hidden title="点击查看上游发布页"><span id="pillUpdateText"></span></span>
    </div>
  </header>

  <!-- 章节目录切页导航 (Chapter Tabs) -->
  <nav class="tabs-nav" role="tablist">
    <button type="button" class="tab-link active" data-tab="tab-overview">
      <span class="tab-idx">01 /</span> 概览与监控
    </button>
    <button type="button" class="tab-link" data-tab="tab-pool">
      <span class="tab-idx">02 /</span> 账号池与授权
    </button>
    <button type="button" class="tab-link" data-tab="tab-client">
      <span class="tab-idx">03 /</span> 客户端接入
    </button>
    <button type="button" class="tab-link" data-tab="tab-models">
      <span class="tab-idx">04 /</span> 模型与快照
    </button>
    <button type="button" class="tab-link" data-tab="tab-checkin">
      <span class="tab-idx">05 /</span> 签到状态
    </button>
    <button type="button" class="tab-link" data-tab="tab-usage">
      <span class="tab-idx">06 /</span> 用量与明细
    </button>
    <button type="button" class="tab-link" data-tab="tab-system">
      <span class="tab-idx">07 /</span> 活动与系统
    </button>
  </nav>

  <!-- ========================================================
       TAB 01: 概览与监控 (专注运行态与统计，不堆叠表单)
       ======================================================== -->
  <div class="tab-panel active" id="tab-overview">
    <section class="panel">
      <div class="panel-inner">
        <div class="eyebrow">Gateway Console · Overview</div>
        <h1>账号、模型与网关状态</h1>
        <p class="lede">监控 OAuth 登录、账号池状态与请求健康度。</p>
        <div class="metrics">
          <div class="metric"><div class="k">登录态</div><div class="v sm" id="mLogin">未登录</div></div>
          <div class="metric"><div class="k">Credits 余额</div><div class="v sm" id="mCredits">—</div></div>
          <div class="metric"><div class="k">启用账号</div><div class="v" id="mEnabled">0</div></div>
          <div class="metric"><div class="k">成功 / 失败</div><div class="v sm" id="mSF">0 / 0</div></div>
          <div class="metric"><div class="k">总 Tokens</div><div class="v sm" id="mTokens">0</div></div>
        </div>
        <div class="actions" style="margin-top:24px">
          <button class="ghost" id="btnPoolTest" type="button" disabled aria-describedby="poolTestHint">批量测试</button>
          <label class="test-model-label" for="testModelSelect">测试模型
            <select id="testModelSelect" aria-label="账号测试使用的模型">
              <option value="">自动（最低倍率）</option>
            </select>
          </label>
        </div>
        <p class="checkin-hint" id="poolTestHint">对当前号池已启用账号发最小 chat，验证可用性与延迟；默认选最低倍率模型。</p>
        <pre class="checkin-detail" id="poolTestRaw" hidden></pre>
      </div>
    </section>

    <div class="overview-notes">
      <div class="note-item">
        <h3>429 限频隔离与自动熔断</h3>
        <p>同一会话钉在一个账号上。新会话只补缺失或超过 5 分钟的额度快照，再按剩余最大选号。纯 429 / rate limit 短冷却 2 分钟，松钉后并行刷新候选额度并选剩余最大者；配额类错误会查询官方余额并对齐至 CycleEnd / SlicePeriod 结束。111xx 策略码仍为 5 分钟，部分 5xx 为 30 秒。</p>
      </div>
      <div class="note-item">
        <h3>Reasoning 思考链透传</h3>
        <p>支持客户端 <code>reasoning_effort</code> / <code>reasoning</code> 入站映射；流式与非流式可回传 <code>reasoning_content</code>（视上游模型是否开启思考）。</p>
      </div>
    </div>
  </div>

  <!-- ========================================================
       TAB 02: 账号池与授权 (专注号池调度与 OAuth)
       ======================================================== -->
  <div class="tab-panel" id="tab-pool">
    <section class="panel">
      <div class="panel-inner">
        <div class="section-head">
          <div>
            <h2>账号池集群</h2>
            <p>一键切换国内 / 国际号池，以及 CodeBuddy / WorkBuddy 上游；token 共用，请求跟随当前选择。</p>
          </div>
          <div class="actions">
            <div class="seg" id="poolSiteSeg" role="group" aria-label="号池区域">
              <button type="button" data-site="domestic" id="btnPoolDomestic">国内</button>
              <button type="button" data-site="global" id="btnPoolGlobal">国际</button>
            </div>
            <div class="seg" id="poolProductSeg" role="group" aria-label="上游产品">
              <button type="button" data-product="codebuddy" id="btnProductCodeBuddy">CodeBuddy</button>
              <button type="button" data-product="workbuddy" id="btnProductWorkBuddy">WorkBuddy</button>
            </div>
            <button class="ghost" id="btnRefresh" type="button">刷新状态</button>
          </div>
        </div>
        <p class="seg-hint" id="poolSiteHint">当前号池：—</p>
        <div id="accounts"></div>
      </div>
    </section>

    <section class="panel" id="codebuddy">
      <div class="panel-inner">
        <div class="section-head">
          <div>
            <h2>OAuth 快速授权</h2>
            <p>选择站点并开始认证，完成后账号自动归入池中。</p>
          </div>
        </div>
        <div class="field-grid">
          <div>
            <label for="site">站点</label>
            <select id="site">
              <option value="domestic">国内站 · domestic</option>
              <option value="global">国际站 · global</option>
            </select>
          </div>
          <div>
            <label for="label">账号标签</label>
            <input id="label" placeholder="留空则自动取真实账号名（如昵称）" value=""/>
          </div>
        </div>
        <div class="actions">
          <button class="primary" id="btnStart" type="button">开始认证</button>
          <button class="teal" id="btnPoll" type="button">检查登录</button>
          <a class="linkbtn" id="launchLink" href="#" target="_blank" rel="noreferrer">打开登录页</a>
        </div>
        <div class="oauth-status">
          <div class="title">会话状态</div>
          <div class="msg" id="oauthMsg">OAuth 会话空闲（账号池登录态见上方「登录态」；这里只反映进行中的认证流程）</div>
        </div>
        <details class="raw">
          <summary>原始 OAuth 响应</summary>
          <pre id="oauthRaw">idle</pre>
        </details>
      </div>
    </section>
  </div>

  <!-- ========================================================
       TAB 03: 客户端接入配置 (专注 OpenAI 端点与验证)
       ======================================================== -->
  <div class="tab-panel" id="tab-client">
    <section class="panel" id="client-config">
      <div class="panel-inner">
        <div class="section-head">
          <div>
            <h2>OpenAI 兼容接入</h2>
            <p>给下游客户端填 Base URL + API Key；不会暴露 CodeBuddy OAuth token。</p>
          </div>
          <div class="actions">
            <button class="ghost" id="btnRefreshClient" type="button">刷新接入信息</button>
            <button class="teal" id="btnGenerateKey" type="button">生成 API Key</button>
          </div>
        </div>
        <div class="field-grid" style="grid-template-columns:1fr 1fr">
          <div>
            <label for="openAiBaseUrl">Base URL</label>
            <div class="copyline">
              <input id="openAiBaseUrl" readonly placeholder="加载中…"/>
              <button class="ghost" id="copyBaseUrl" type="button">复制</button>
            </div>
          </div>
          <div>
            <label for="openAiChatUrl">Chat Completions</label>
            <div class="copyline">
              <input id="openAiChatUrl" readonly placeholder="加载中…"/>
              <button class="ghost" id="copyChatUrl" type="button">复制</button>
            </div>
            <div class="secret-hint">ZCode / OpenCode / 多数 SDK 走这条。</div>
          </div>
          <div>
            <label for="openAiResponsesUrl">Responses</label>
            <div class="copyline">
              <input id="openAiResponsesUrl" readonly placeholder="加载中…"/>
              <button class="ghost" id="copyResponsesUrl" type="button">复制</button>
            </div>
            <div class="secret-hint">Codex CLI 走这条（wire_api = responses）。</div>
          </div>
          <div>
            <label for="openAiApiKey">API Key（网关层）</label>
            <div class="copyline">
              <input id="openAiApiKey" class="secret-input" type="password" readonly placeholder="未配置"/>
              <button class="ghost" id="copyApiKey" type="button">复制</button>
            </div>
          </div>
          <div>
            <label for="openAiModel">推荐模型</label>
            <div class="copyline">
              <input id="openAiModel" readonly value="auto"/>
              <button class="ghost" id="copyModel" type="button">复制</button>
            </div>
          </div>
        </div>
        <div class="bound-keys" id="boundApiKeys"></div>
        <div class="bound-keys-new">
          <label for="boundKeySite">新建绑定 Key</label>
          <div class="copyline">
            <select id="boundKeySite" aria-label="绑定区域">
              <option value="domestic">国内 domestic</option>
              <option value="global">国际 global</option>
            </select>
            <button class="teal" id="btnNewBoundKey" type="button">新建绑定 Key</button>
          </div>
        </div>
        <div class="secret-hint" id="clientConfigHint">API Key 点击复制时才会读取明文。Key 默认写入 ~/.codebuddy/proxy.env，与账号池同目录，换文件夹启动也会沿用同一把。绑定 Key 换区不改主 Key。</div>
      </div>
    </section>
  </div>

  <!-- ========================================================
       TAB 04: 模型与诊断 (专注模型芯片与底层 JSON)
       ======================================================== -->
  <div class="tab-panel" id="tab-models">
    <section class="panel">
      <div class="panel-inner">
        <div class="section-head">
          <div>
            <h2>可用模型编目</h2>
            <p>展示上游模型名（不含 codebuddy/ 前缀）；点击模型芯片可复制。</p>
          </div>
          <div class="actions">
            <button class="primary" id="btnModels" type="button">拉取模型</button>
          </div>
        </div>
        <div class="chips" id="modelChips"><div class="empty">点击「拉取模型」加载</div></div>
        <details class="raw">
          <summary>原始模型响应</summary>
          <pre id="modelsRaw">[]</pre>
        </details>
      </div>
    </section>

    <section class="panel">
      <div class="panel-inner">
        <div class="section-head">
          <div>
            <h2>运行快照诊断</h2>
            <p>完整 status JSON，便于排障。</p>
          </div>
        </div>
        <details class="raw" open>
          <summary>status payload</summary>
          <pre id="statusRaw">loading…</pre>
        </details>
      </div>
    </section>

    <section class="panel">
      <div class="panel-inner">
        <div class="section-head">
          <div>
            <h2>模型白名单</h2>
            <p>写入 <code>~/.codebuddy/proxy-modelpolicy.json</code>。开启后仅允许列表内模型：公开 <code>/v1/models</code> 只返回白名单模型，<code>/v1/chat/completions</code> 对白名单外模型直接拒绝。<code>auto</code> 始终可用；管理台仍显示完整编目。每行一个模型 ID。</p>
          </div>
        </div>
        <div class="policy-switch">
          <label class="toggle">
            <input type="checkbox" id="mpEnabled"/>
            <span>启用模型白名单</span>
          </label>
          <span class="meta-line" id="mpStatus">—</span>
        </div>
        <div class="field-grid">
          <div>
            <label for="mpAllow">允许列表（allow）</label>
            <textarea id="mpAllow" rows="6" spellcheck="false" placeholder="hy3&#10;hy4-preview&#10;"></textarea>
          </div>
          <div>
            <label for="mpDeny">禁用列表（deny，可选）</label>
            <textarea id="mpDeny" rows="6" spellcheck="false" placeholder="glm-5.3-flash"></textarea>
          </div>
        </div>
        <div class="actions">
          <button class="primary" id="btnSavePolicy" type="button">保存白名单</button>
          <button class="ghost" id="btnResetPolicy" type="button">重读</button>
        </div>
      </div>
    </section>
  </div>

  <!-- ========================================================
       TAB 05: 签到状态 (账号池每日签到)
       ======================================================== -->
  <div class="tab-panel" id="tab-checkin">
    <section class="panel">
      <div class="panel-inner">
        <div class="eyebrow">Check-in · Pool</div>
        <h1>每日签到状态</h1>
        <p class="lede" id="checkinNote">读取当前号池 <code>DOMESTIC/GLOBAL</code> 区域各账号的签到状态；可单独签到或对启用账号批量签到。</p>
        <div class="metrics">
          <div class="metric"><div class="k">启用账号</div><div class="v" id="ciTotal">0</div></div>
          <div class="metric"><div class="k">已签到</div><div class="v" id="ciCheckedIn">0</div></div>
          <div class="metric"><div class="k">待签到</div><div class="v" id="ciPending">0</div></div>
          <div class="metric"><div class="k">无签到活动</div><div class="v sm" id="ciInactive">0</div></div>
          <div class="metric"><div class="k">失败</div><div class="v sm" id="ciFailed">0</div></div>
        </div>
        <div class="section-head" style="margin-top:20px;border:none;padding:0">
          <div class="idx"><span class="k">账号明细</span></div>
          <div class="actions">
            <button class="ghost" id="btnRefreshCheckin" type="button">刷新状态</button>
            <button class="primary" id="btnCheckinAll" type="button">全部签到</button>
          </div>
        </div>
        <div class="log-table-wrap">
          <div class="log-table-window">
            <table class="log-table">
              <thead>
                <tr>
                  <th>账号</th>
                  <th>站点</th>
                  <th>今日签到</th>
                  <th>连续天数</th>
                  <th>每日奖励</th>
                  <th>今日奖励</th>
                  <th>说明</th>
                  <th></th>
                </tr>
              </thead>
              <tbody id="checkinRows">
                <tr><td colspan="8" class="empty">加载中…</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </section>
  </div>

  <!-- ========================================================
       TAB 06: 用量与明细
       ======================================================== -->
  <div class="tab-panel" id="tab-usage">
    <section class="panel">
      <div class="panel-inner">
        <div class="eyebrow">Usage · Request Log</div>
        <h1>Token 与缓存统计</h1>
        <div class="section-head" style="margin-top:20px;border:none;padding:0">
          <div class="seg" id="usageRangeSeg" role="group" aria-label="统计周期">
            <button type="button" data-range="day" class="active">今日</button>
            <button type="button" data-range="week">7 日</button>
            <button type="button" data-range="month">30 日</button>
          </div>
          <div class="actions">
            <button class="ghost" id="btnRefreshUsage" type="button">刷新明细</button>
          </div>
        </div>
        <div class="metrics">
          <div class="metric"><div class="k">请求数</div><div class="v" id="uRequests">0</div></div>
          <div class="metric"><div class="k">失败</div><div class="v" id="uFailed">0</div></div>
          <div class="metric"><div class="k">Token 总量</div><div class="v" id="uTotalTokens">0</div></div>
          <div class="metric"><div class="k">缓存命中率</div><div class="v sm" id="uCacheHit">—</div></div>
          <div class="metric"><div class="k">Credits Σ</div><div class="v sm" id="uCredits">—</div></div>
        </div>
        <div class="usage-filters">
          <label>账号
            <select id="usageAccountFilter" aria-label="按账号筛选">
              <option value="">全部账号</option>
            </select>
          </label>
          <label>模型
            <select id="usageModelFilter" aria-label="按模型筛选">
              <option value="">全部模型</option>
            </select>
          </label>
        </div>
        <div class="usage-models" id="usageByModel" hidden></div>
        <div class="usage-chart" id="usageChartBox" aria-hidden="false">
          <div class="usage-chart-head">
            <h3>趋势</h3>
            <div class="usage-chart-legend">
              <span><i class="swatch"></i> Token 总量</span>
              <span><i class="swatch dash"></i> 缓存命中率 %</span>
            </div>
          </div>
          <div id="usageChart"><div class="chart-empty">加载中…</div></div>
        </div>
        <p class="checkin-hint" id="usageCreditHint">命中率 = 缓存 token ÷ prompt token（所有模型同一公式）。按模型表用来对比 hy3 / DeepSeek 等差异。Credits 仅在上游 usage 返回 <code>credit</code> 时累加。</p>
        <div class="log-table-wrap">
          <div class="log-table-window">
            <table class="log-table" aria-label="请求明细">
              <thead>
                <tr>
                  <th>时间</th>
                  <th>请求 ID</th>
                  <th>上游 Request</th>
                  <th>会话</th>
                  <th>模型</th>
                  <th>Token 总量</th>
                  <th>缓存命中</th>
                  <th>Credit</th>
                  <th>耗时</th>
                  <th>账号</th>
                  <th>状态</th>
                </tr>
              </thead>
              <tbody id="usageRows">
                <tr><td colspan="11" class="empty" style="border:none">加载中…</td></tr>
              </tbody>
            </table>
          </div>
          <div class="log-pager">
            <span id="usagePagerMeta">—</span>
            <div class="actions">
              <button type="button" class="ghost" id="btnUsagePrev" disabled>上一页</button>
              <button type="button" class="ghost" id="btnUsageNext" disabled>下一页</button>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>

  <!-- ========================================================
       TAB 07: 活动与系统 (按业务类型分组的运行活动日志 + 本机配置目录)
       ======================================================== -->
  <div class="tab-panel" id="tab-system">
    <section class="panel">
      <div class="panel-inner">
        <div class="eyebrow">System · Activity</div>
        <h1>活动记录与本机配置目录</h1>

        <div class="section-head" style="margin-top:20px;border:none;padding:0">
          <div class="idx"><span class="k">运行活动日志</span></div>
          <div class="actions">
            <button class="ghost" id="btnRefreshActivity" type="button">刷新</button>
          </div>
        </div>
        <div class="seg" id="logSubSeg" role="group" aria-label="日志子页" style="margin-bottom:10px">
          <button type="button" data-sub="lifecycle" class="active">服务生命周期</button>
          <button type="button" data-sub="checkin">签到</button>
          <button type="button" data-sub="account">账号管理</button>
          <button type="button" data-sub="system">系统与配置</button>
          <button type="button" data-sub="error">报错日志</button>
        </div>
        <div class="meta-line" id="actMeta" style="font-size:12px;color:var(--fg-40);margin-bottom:10px">—</div>
        <div class="activity-list" id="activityList" style="max-height:480px;overflow:auto">
          <div class="empty">加载中…</div>
        </div>
        <div class="meta-line hidden" id="errMeta" style="font-size:12px;color:var(--fg-40);margin-bottom:10px">—</div>
        <div class="activity-list hidden" id="errorList" style="max-height:480px;overflow:auto;border-color:var(--fg-20)">
          <div class="empty">加载中…</div>
        </div>

        <div class="section-head" style="margin-top:28px;border:none;padding:0;margin-bottom:0">
          <div class="idx"><span class="k">本机配置目录</span></div>
        </div>
        <div class="configdir-row" style="display:flex;align-items:center;gap:14px;flex-wrap:wrap;margin-top:12px">
          <button class="primary" id="btnOpenConfigDir" type="button" style="white-space:nowrap">打开配置目录</button>
          <p class="lede" style="margin:0;max-width:none;flex:1 1 320px;min-width:320px">在运行本管理台的电脑上打开 <code>~/.codebuddy</code> 配置目录（含账号池、活动日志、模型策略等数据文件），便于按需备份与检查。</p>
        </div>
        <div class="meta-line" id="configDirLine" style="font-size:12px;color:var(--fg-40);margin-top:6px">—</div>
      </div>
    </section>
  </div>
</div>

<div class="modal-mask" id="updModal" hidden>
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="updTitle">
    <div class="modal-head">
      <div>
        <div class="eyebrow">Upstream Update</div>
        <h2 id="updTitle">发现新版本</h2>
      </div>
      <div class="modal-meta" id="updMeta"></div>
    </div>
    <div class="modal-body" id="updBody"></div>
    <div class="modal-actions">
      <a id="updLink" href="#" target="_blank" rel="noopener noreferrer">去下载</a>
      <button id="updDismiss" type="button">知道了</button>
    </div>
  </div>
</div>

<div class="toast" id="toast" role="status" aria-live="polite"></div>

<!-- hidden compatibility targets for existing helpers -->
<pre id="statusBox"></pre>
<pre id="oauthBox"></pre>
<pre id="modelsBox"></pre>

<script>
const $ = (id) => document.getElementById(id);

/* 切页逻辑 (Tab Switcher) */
function switchTab(tabId){
  document.querySelectorAll('.tab-link').forEach(function(b){ b.classList.remove('active'); });
  document.querySelectorAll('.tab-panel').forEach(function(p){ p.classList.remove('active'); });
  var btn = document.querySelector('.tab-link[data-tab="' + tabId + '"]');
  if (btn) btn.classList.add('active');
  var p = $(tabId);
  if (p) p.classList.add('active');
}
document.querySelectorAll('.tab-link').forEach(function(btn){
  btn.addEventListener('click', function(){
    const tab = btn.getAttribute('data-tab');
    switchTab(tab);
    if (tab === 'tab-usage') refreshUsage().catch(function(e){ showToast(e.message, 'error'); });
    if (tab === 'tab-checkin') refreshCheckin().catch(function(e){ showToast(e.message, 'error'); });
    if (tab === 'tab-system') { refreshActivity().catch(function(e){ showToast(e.message, 'error'); }); refreshErrors().catch(function(e){ showToast(e.message, 'error'); }); }
  });
});
if (window.location.hash === '#codebuddy') {
  switchTab('tab-pool');
} else if (window.location.hash === '#client-config') {
  switchTab('tab-client');
} else if (window.location.hash === '#usage') {
  switchTab('tab-usage');
} else if (window.location.hash === '#checkin') {
  switchTab('tab-checkin');
} else if (window.location.hash === '#activity') {
  switchTab('tab-system');
}
window.addEventListener('hashchange', function(){
  if (window.location.hash === '#codebuddy') switchTab('tab-pool');
  if (window.location.hash === '#client-config') switchTab('tab-client');
  if (window.location.hash === '#usage') switchTab('tab-usage');
  if (window.location.hash === '#checkin') switchTab('tab-checkin');
  if (window.location.hash === '#activity') switchTab('tab-system');
});

async function api(path, opts={}) {
  const headers = Object.assign({'content-type':'application/json'}, opts.headers||{});
  const res = await fetch(path, Object.assign({}, opts, {headers: headers, credentials:'same-origin'}));
  const text = await res.text();
  let data = {};
  try { data = text ? JSON.parse(text) : {}; } catch (e) { data = {raw:text}; }
  if (!res.ok) throw new Error((data && data.error && data.error.message) || data.message || data.error || ('HTTP '+res.status));
  return data;
}

function escapeHtml(s){
  return String(s||'').replace(/[&<>"']/g, function(c){
    return ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]);
  });
}

function fmtUptime(ms){
  ms = Number(ms||0);
  if (!ms || ms < 0) return '—';
  const s = Math.floor(ms/1000);
  const h = Math.floor(s/3600);
  const m = Math.floor((s%3600)/60);
  const r = s%60;
  if (h > 0) return h + 'h ' + m + 'm';
  if (m > 0) return m + 'm ' + r + 's';
  return r + 's';
}

function setHealth(ok, text){
  const dot = $('healthDot');
  dot.className = 'dot' + (ok ? '' : ' bad');
  $('healthText').textContent = text;
}

function formatResetAt(ms){
  if (!ms) return '-';
  try { return new Date(ms).toLocaleString(); } catch (e) { return String(ms); }
}

const usageByAccount = {};
function renderAccounts(summary, activeSite) {
  const box = $('accounts');
  const all = (summary && summary.accounts) || [];
  activeSite = normalizeSite(activeSite || (summary && summary.activeSite) || '');
  if (!all.length) {
    box.innerHTML = '<div class="empty">暂无账号。请先在下方完成 OAuth 登录。</div>';
    return;
  }
  const accounts = all.filter(function(a){ return normalizeSite(a.site) === activeSite; });
  if (!accounts.length) {
    box.innerHTML = '<div class="empty">当前号池没有账号，切到另一区域可管理其他号</div>';
    return;
  }
  box.innerHTML = accounts.map(function(a) {
    const name = escapeHtml(a.userNickname || a.userName || a.userId || '未命名用户');
    const rawLabel = (a.label || '').trim();
    const genericLabel = !rawLabel || /^CodeBuddy(\s+OAuth)?$/i.test(rawLabel);
    const customLabel = genericLabel ? (a.userNickname || a.userName || a.id) : a.label;
    const label = escapeHtml(customLabel || a.id);
    const logged = a.loggedIn && a.hasCredentials;
    const site = normalizeSite(a.site);
    const inPool = true;
    let poolStateHtml = '';
    if (a.quotaExhausted && a.quotaResetAt) {
      poolStateHtml = '<div class="usage-line"><span class="pill bad">配额耗尽 · ' + escapeHtml(formatResetAt(a.quotaResetAt)) + ' 恢复</span></div>';
    } else if (a.cooldownUntil && a.cooldownUntil > Date.now()) {
      poolStateHtml = '<div class="usage-line"><span class="pill warn">冷却中 · ' + escapeHtml(formatResetAt(a.cooldownUntil)) + ' 恢复</span></div>';
    } else if (a.quotaRemaining != null && Number(a.quotaRemaining) <= 0) {
      poolStateHtml = '<div class="usage-line"><span class="pill warn">配额可能已耗尽</span></div>';
    }
    const usage = usageByAccount[a.id];
    let usageHtml = '<div class="usage-line"><button class="ghost" data-act="usage" data-id="' + escapeHtml(a.id) + '" type="button">查余额</button></div>';
    if (usage && usage.error) {
      usageHtml = '<div class="usage-line"><span class="pill bad" title="' + escapeHtml(usage.error) + '">查询失败</span> ' +
        '<button class="ghost" data-act="usage" data-id="' + escapeHtml(a.id) + '" type="button">重试</button></div>';
    } else if (usage && usage.credits) {
      const c = usage.credits;
      const remaining = c.unlimited ? '不限量' : (c.remaining == null ? '-' : String(c.remaining));
      const total = c.unlimited ? '不限量' : (c.total == null ? '-' : String(c.total));
      let level = 'good';
      if (!c.unlimited && Number(c.total) > 0) {
        const ratio = Number(c.remaining) / Number(c.total);
        if (ratio <= 0.05 || Number(c.remaining) <= 0) level = 'bad';
        else if (ratio <= 0.2) level = 'warn';
      }
      if (usage.notify && usage.notify.level === 'bad') level = 'bad';
      else if (usage.notify && usage.notify.level === 'warn' && level === 'good') level = 'warn';
      usageHtml = '<div class="usage-line"><span class="pill ' + level + '">' + escapeHtml(remaining + ' / ' + total) + ' Credits</span> ' +
        '<button class="ghost" data-act="usage" data-id="' + escapeHtml(a.id) + '" type="button">刷新余额</button> ' +
        '<a href="' + escapeHtml(usage.officialUsageUrl || 'https://www.codebuddy.cn/profile/plan') + '" target="_blank" rel="noreferrer">官网套餐</a></div>';
    }
    return '<div class="account">' +
      '<div class="account-top">' +
        '<div class="account-title">' +
          '<strong>' + label + '</strong>' +
          '<span class="badge site">' + escapeHtml(siteLabel(site)) + '</span>' +
          '<span class="badge ' + (inPool?'on':'muted') + '">' + (inPool?'当前号池':'其他区域') + '</span>' +
          '<span class="badge ' + (logged?'on':'off') + '">' + (logged?'已登录':'未登录') + '</span>' +
          '<span class="badge ' + (a.enabled?'on':'off') + '">' + (a.enabled?'enabled':'disabled') + '</span>' +
        '</div>' +
        '<div class="actions">' +
          '<button class="ghost" data-act="test" data-id="' + escapeHtml(a.id) + '" type="button">测试</button>' +
          '<button class="ghost" data-act="usage" data-id="' + escapeHtml(a.id) + '" type="button">查余额</button>' +
          '<button class="ghost" data-act="toggle" data-id="' + escapeHtml(a.id) + '" data-enabled="' + (a.enabled?0:1) + '" type="button">' + (a.enabled?'禁用':'启用') + '</button>' +
          '<button class="ghost" data-act="refresh" data-id="' + escapeHtml(a.id) + '" type="button">刷新 Token</button>' +
          '<button class="danger" data-act="delete" data-id="' + escapeHtml(a.id) + '" type="button">删除</button>' +
        '</div>' +
      '</div>' +
      '<div class="meta">' + name +
        ' · <span class="mono">' + escapeHtml(a.authType||'') + '</span>' +
        ' · success ' + (a.successRequests||0) + ' / fail ' + (a.failedRequests||0) +
        (a.tokenExpired ? ' · <span class="err">token expired</span>' : '') +
      '</div>' +
      usageHtml +
      poolStateHtml +
      (a.lastError ? ('<div class="err">' + escapeHtml(a.lastError) + '</div>') : '') +
    '</div>';
  }).join('');
  box.querySelectorAll('button[data-act]').forEach(function(btn){ btn.addEventListener('click', onAccountAction); });
}

function bareModelId(raw){
  const id = String(raw||'').trim();
  if (!id) return 'auto';
  return id.replace(/^codebuddy[/:]/i, '') || 'auto';
}
function renderModels(data){
  const chips = $('modelChips');
  let list = Array.isArray(data) ? data
    : (Array.isArray(data && data.models) ? data.models
    : (Array.isArray(data && data.data) ? data.data : []));
  fillTestModelSelect(list);
  if (!list.length) {
    chips.innerHTML = '<div class="empty">暂无模型数据</div>';
    return;
  }
  const modelId = function(m){
    if (typeof m === 'string') return m;
    const raw = m.modelId || m.upstreamId || m.id || m.name || m.model || 'model';
    return String(raw);
  };
  const visible = list.filter(function(m){
    const id = bareModelId(modelId(m));
    if (id === 'auto') return true;
    return typeof m === 'object' && m && m.creditMultiplier != null;
  });
  if (!visible.length) {
    chips.innerHTML = '<div class="empty">暂无模型数据</div>';
    return;
  }
  const nameCount = {};
  visible.forEach(function(m){
    if (typeof m !== 'object' || !m) return;
    const nm = String(m.displayName || m.name || '').toLowerCase().trim();
    if (nm) nameCount[nm] = (nameCount[nm] || 0) + 1;
  });
  const sortKey = function(m){
    const mult = (typeof m === 'object' && m && m.creditMultiplier != null) ? Number(m.creditMultiplier) : 0;
    return isFinite(mult) ? mult : 0;
  };
  list = visible.slice().sort(function(a,b){ return sortKey(a) - sortKey(b); });
  chips.innerHTML = list.map(function(m){
    const raw = typeof m === 'string' ? m : (m.modelId || m.upstreamId || m.id || m.name || m.model || 'model');
    const id = bareModelId(raw);
    let baseLabel = typeof m === 'string' ? id : String(m.displayName || m.name || '') || id;
    if (typeof m === 'object') {
      const nm = String(m.displayName || m.name || '').toLowerCase().trim();
      if (nm && nameCount[nm] > 1) baseLabel = id;
    }
    const credits = (typeof m === 'object' && m && m.credits) ? String(m.credits) : '';
    const free = typeof m === 'object' && m && (m.free === true || /x0(\.0+)?\s*credits/i.test(credits));
    const mult = (typeof m === 'object' && m && m.creditMultiplier != null) ? m.creditMultiplier : null;
    const tip = [id, credits ? ('倍率 ' + credits) : '', (m && m.description) ? m.description : ''].filter(Boolean).join(' | ');
    return '<button type="button" class="chip btnish" data-copy="' + escapeHtml(id) + '" title="' + escapeHtml(tip || '点击复制') + '">' + escapeHtml(baseLabel) + (free ? ' <span class="badge on">免费</span>' : (mult != null && credits ? ' <span class="badge site">' + escapeHtml(String(mult) + 'x') + '</span>' : '')) + '</button>';
  }).join('');
  chips.querySelectorAll('[data-copy]').forEach(function(btn){
    btn.addEventListener('click', function(){ copyText(btn.getAttribute('data-copy'), '模型', btn); });
  });
}

function fillTestModelSelect(list){
  const sel = $('testModelSelect');
  if (!sel) return;
  const prev = sel.value;
  const opts = ['<option value="">自动（最低倍率）</option>'];
  const seen = {};
  (list || []).forEach(function(m){
    const raw = typeof m === 'string' ? m : (m && (m.id || m.modelId || m.upstreamId || m.name));
    const id = bareModelId(raw);
    if (!id || id === 'auto' || seen[id]) return;
    seen[id] = true;
    const credits = (typeof m === 'object' && m && m.credits) ? String(m.credits) : '';
    const label = credits ? (id + ' · ' + credits.replace(/\s*credits$/i,'')) : id;
    opts.push('<option value="' + escapeHtml(id) + '">' + escapeHtml(label) + '</option>');
  });
  sel.innerHTML = opts.join('');
  if (prev && seen[prev]) sel.value = prev;
}

function selectedTestModel(){
  const sel = $('testModelSelect');
  return sel ? String(sel.value || '').trim() : '';
}

function normalizeSite(site){
  site = String(site||'').toLowerCase().trim();
  if (site === 'domestic' || site === 'cn' || site === 'china' || site === 'internal') return 'domestic';
  return 'global';
}
function siteLabel(site){
  return normalizeSite(site) === 'domestic' ? '国内' : '国际';
}
function normalizeProduct(product){
  product = String(product||'').toLowerCase().trim();
  if (product === 'workbuddy' || product === 'wb' || product === 'ide') return 'workbuddy';
  return 'codebuddy';
}
function productLabel(product){
  return normalizeProduct(product) === 'workbuddy' ? 'WorkBuddy' : 'CodeBuddy';
}
// 签到按钮跟随当前激活号池；paintPoolSite 每次刷新状态时同步。
// 空字符串表示尚未从 /status 拿到号池，按钮保持 disabled，避免首屏误打国际站。
let activeCheckinSite = '';
let activeProduct = 'codebuddy';
let lastPoolAccounts = null;
let checkinBusy = false;
let poolTestBusy = false;
let usageRange = 'day';
let usagePage = 1;
let usageAccount = '';
let usageModel = '';
const usagePageSize = 20;

function formatTime(ms){
  if (!ms) return '—';
  try { return new Date(ms).toLocaleString(); } catch(e) { return String(ms); }
}
function shortId(id){
  id = String(id||'');
  if (id.length <= 10) return id || '—';
  return id.slice(0, 8) + '…';
}
function formatHitRate(rate){
  if (rate == null || rate < 0) return '—';
  return rate.toFixed(1) + '%';
}
function rowTotalTokens(row){
  const p = row.promptTokens || 0;
  const c = row.completionTokens || 0;
  return row.totalTokens || (p + c);
}
function rowHitRate(row){
  const p = row.promptTokens || 0;
  if (p <= 0) return -1;
  return (row.cachedTokens || 0) / p * 100;
}
function paintUsageChart(series, summary){
  const host = $('usageChart');
  if (!host) return;
  const pts = series || [];
  const hasTokens = pts.some(function(p){ return (p.totalTokens||0) > 0; });
  const hasRate = pts.some(function(p){ return p.cacheHitRate != null && p.cacheHitRate >= 0; });
  const hasRequests = summary && (summary.requests || 0) > 0;
  if (!pts.length) {
    host.innerHTML = '<div class="chart-empty">暂无趋势数据</div>';
    return;
  }
  if (!hasTokens && !hasRate && !hasRequests) {
    host.innerHTML = '<div class="chart-empty">暂无趋势数据</div>';
    return;
  }
  const W = 720, H = 168, pad = {l: 44, r: 44, t: 14, b: 30};
  const innerW = W - pad.l - pad.r;
  const innerH = H - pad.t - pad.b;
  const maxTok = Math.max(1, pts.reduce(function(m, p){ return Math.max(m, p.totalTokens||0); }, 0));
  const n = pts.length;
  function xAt(i){ return pad.l + (n <= 1 ? innerW / 2 : (i / (n - 1)) * innerW); }
  function yTok(v){ return pad.t + innerH - (v / maxTok) * innerH; }
  function yRate(r){ if (r < 0) r = 0; return pad.t + innerH - (r / 100) * innerH; }
  let pathTok = '';
  let pathRate = '';
  pts.forEach(function(p, i){
    const x = xAt(i);
    const yt = yTok(p.totalTokens || 0);
    const yr = yRate(p.cacheHitRate != null ? p.cacheHitRate : -1);
    pathTok += (i ? ' L' : 'M') + x + ' ' + yt;
    if (p.cacheHitRate != null && p.cacheHitRate >= 0) {
      pathRate += (pathRate ? ' L' : 'M') + x + ' ' + yr;
    }
  });
  const labelStep = n <= 8 ? 1 : Math.max(1, Math.ceil(n / 8));
  const labels = pts.map(function(p, i){
    if (i % labelStep !== 0 && i !== n - 1) return '';
    const x = xAt(i);
    const anchor = i === 0 ? 'start' : (i === n - 1 ? 'end' : 'middle');
    return '<text class="lbl" x="' + x + '" y="' + (H - 8) + '" text-anchor="' + anchor + '">' + escapeHtml(p.label || '') + '</text>';
  }).join('');
  host.innerHTML =
    '<svg viewBox="0 0 ' + W + ' ' + H + '" role="img" aria-label="Token 与缓存命中率趋势">' +
    '<line class="axis" x1="' + pad.l + '" y1="' + (pad.t + innerH) + '" x2="' + (W - pad.r) + '" y2="' + (pad.t + innerH) + '"/>' +
    '<path class="line-tokens" d="' + pathTok + '"/>' +
    (pathRate ? '<path class="line-rate" d="' + pathRate + '"/>' : '') +
    labels +
    '</svg>';
}
function paintUsage(data){
  const sum = data.summary || {};
  $('uRequests').textContent = String(sum.requests || 0);
  $('uFailed').textContent = String(sum.failed || 0);
  $('uTotalTokens').textContent = String(sum.totalTokens || 0);
  $('uCacheHit').textContent = formatHitRate(sum.cacheHitRate);
  if (sum.creditRows > 0) {
    $('uCredits').textContent = (sum.credits || 0).toFixed(4) + ' · ' + sum.creditRows + ' 条';
  } else {
    $('uCredits').textContent = '—';
  }
  fillUsageFilter($('usageAccountFilter'), data.accounts || [], usageAccount, '全部账号');
  fillUsageFilter($('usageModelFilter'), data.models || [], usageModel, '全部模型');
  paintByModel(data.byModel || []);
  paintUsageChart(data.series || [], data.summary || {});
  const rows = data.requests || [];
  const tbody = $('usageRows');
  if (!tbody) return;
  if (!rows.length) {
    tbody.innerHTML = '<tr><td colspan="11" style="color:var(--fg-40);padding:24px;text-align:center">暂无记录</td></tr>';
    return;
  }
  tbody.innerHTML = rows.map(function(row){
    const proxyId = row.proxyRequestId || '';
    const upReq = row.upstreamConversationRequestId || '';
    const credit = row.credit != null ? String(row.credit) : '—';
    const account = row.accountLabel || shortId(row.accountId);
    const status = row.ok
      ? '<span class="badge on">OK</span>'
      : '<span class="badge off" title="' + escapeHtml(row.error||'') + '">FAIL</span>';
    const session = escapeHtml(row.sessionLabel || shortId(row.sessionKey));
    const totalTok = rowTotalTokens(row);
    const hit = formatHitRate(rowHitRate(row));
    return '<tr>' +
      '<td class="mono">' + escapeHtml(formatTime(row.at)) + '</td>' +
      '<td class="mono"><button type="button" class="idbtn" data-copy="' + escapeHtml(proxyId) + '">' + escapeHtml(shortId(proxyId)) + '</button></td>' +
      '<td class="mono">' + (upReq ? '<button type="button" class="idbtn" data-copy="' + escapeHtml(upReq) + '">' + escapeHtml(shortId(upReq)) + '</button>' : '—') + '</td>' +
      '<td>' + session + '</td>' +
      '<td class="mono">' + escapeHtml(row.model || '—') + '</td>' +
      '<td class="mono">' + String(totalTok) + '</td>' +
      '<td class="mono">' + escapeHtml(hit) + '</td>' +
      '<td class="mono">' + escapeHtml(credit) + '</td>' +
      '<td class="mono">' + (row.durationMs != null ? (row.durationMs + 'ms') : '—') + '</td>' +
      '<td>' + escapeHtml(account || '—') + '</td>' +
      '<td>' + status + '</td>' +
      '</tr>';
  }).join('');
  tbody.querySelectorAll('[data-copy]').forEach(function(btn){
    btn.addEventListener('click', function(){ copyText(btn.getAttribute('data-copy'), 'ID', btn); });
  });
}
function paintUsagePager(data){
  const total = data.requestsTotal != null ? data.requestsTotal : (data.requests || []).length;
  const limit = data.limit || usagePageSize;
  const offset = data.offset != null ? data.offset : (usagePage - 1) * limit;
  const pages = Math.max(1, Math.ceil(total / limit) || 1);
  usagePage = Math.floor(offset / limit) + 1;
  if ($('usagePagerMeta')) {
    $('usagePagerMeta').textContent = total
      ? ('第 ' + usagePage + ' / ' + pages + ' 页 · 本页 ' + (data.requests || []).length + ' 条 · 共 ' + total + ' 条')
      : '暂无明细';
  }
  const prev = $('btnUsagePrev');
  const next = $('btnUsageNext');
  if (prev) prev.disabled = usagePage <= 1;
  if (next) next.disabled = usagePage >= pages || total === 0;
}
async function refreshUsage(){
  const offset = (usagePage - 1) * usagePageSize;
  let url = '/direct-admin/api/usage?range=' + encodeURIComponent(usageRange) +
    '&limit=' + usagePageSize + '&offset=' + offset;
  if (usageAccount) url += '&account=' + encodeURIComponent(usageAccount);
  if (usageModel) url += '&model=' + encodeURIComponent(usageModel);
  const data = await api(url);
  paintUsage(data);
  paintUsagePager(data);
  return data;
}
function fillUsageFilter(sel, values, current, allLabel){
  if (!sel) return;
  const keep = current || '';
  const opts = ['<option value="">' + allLabel + '</option>'];
  (values || []).forEach(function(v){
    const selected = v === keep ? ' selected' : '';
    opts.push('<option value="' + escapeHtml(v) + '"' + selected + '>' + escapeHtml(v) + '</option>');
  });
  if (keep && (values || []).indexOf(keep) < 0) {
    opts.push('<option value="' + escapeHtml(keep) + '" selected>' + escapeHtml(keep) + '</option>');
  }
  sel.innerHTML = opts.join('');
}
function paintByModel(rows){
  const host = $('usageByModel');
  if (!host) return;
  if (!rows || !rows.length) {
    host.hidden = true;
    host.innerHTML = '';
    return;
  }
  host.hidden = false;
  host.innerHTML = '<table aria-label="按模型缓存命中"><thead><tr>' +
    '<th>模型</th><th>请求</th><th>Token</th><th>缓存命中</th></tr></thead><tbody>' +
    rows.map(function(row){
      const rate = formatHitRate(row.cacheHitRate);
      const low = (row.cacheHitRate == null || row.cacheHitRate < 5) ? ' class="low"' : '';
      return '<tr>' +
        '<td class="mono">' + escapeHtml(row.model || '—') + '</td>' +
        '<td class="mono">' + String(row.requests || 0) + '</td>' +
        '<td class="mono">' + String(row.totalTokens || 0) + '</td>' +
        '<td class="mono"' + low + '>' + escapeHtml(rate) + '</td>' +
        '</tr>';
    }).join('') +
    '</tbody></table>';
}
function paintUsageRange(range){
  usageRange = range || 'day';
  usagePage = 1;
  const seg = $('usageRangeSeg');
  if (!seg) return;
  seg.querySelectorAll('button[data-range]').forEach(function(btn){
    btn.className = btn.getAttribute('data-range') === usageRange ? 'active' : '';
  });
}

function paintPoolHint(site, product, accounts){
  const domestic = accounts && accounts.domesticCount != null ? accounts.domesticCount : '—';
  const global = accounts && accounts.globalCount != null ? accounts.globalCount : '—';
  const activeEnabled = accounts && accounts.activeEnabledCount != null ? accounts.activeEnabledCount : '—';
  if ($('poolSiteHint')) {
    $('poolSiteHint').textContent = '当前号池：' + siteLabel(site) + ' · 上游：' + productLabel(product) + ' · 可用启用账号 ' + activeEnabled + ' · 国内账号 ' + domestic + ' / 国际账号 ' + global + '（仅当前区域会参与请求）';
  }
}

function paintPoolSite(site, accounts){
  site = normalizeSite(site);
  activeCheckinSite = site;
  if (accounts) lastPoolAccounts = accounts;
  const checkinBtn = $('btnCheckin');
  if (checkinBtn && !checkinBusy) checkinBtn.disabled = false;
  const poolTestBtn = $('btnPoolTest');
  if (poolTestBtn && !poolTestBusy) poolTestBtn.disabled = false;
  const domesticBtn = $('btnPoolDomestic');
  const globalBtn = $('btnPoolGlobal');
  if (domesticBtn) domesticBtn.className = site === 'domestic' ? 'active' : '';
  if (globalBtn) globalBtn.className = site === 'global' ? 'active' : '';
  paintPoolHint(site, activeProduct, lastPoolAccounts || accounts);
  // 切号池 / 刷新状态时清掉上一次的签明细，避免与新号池并存造成误导。
  // 签到进行中不清理：runPoolCheckin 会在 refreshStatus 之后重新写入。
  const checkinRaw = $('checkinRaw');
  if (checkinRaw && !checkinBusy) {
    checkinRaw.hidden = true;
    checkinRaw.textContent = '';
  }
  const poolTestRaw = $('poolTestRaw');
  if (poolTestRaw && !poolTestBusy) {
    poolTestRaw.hidden = true;
    poolTestRaw.textContent = '';
  }
  if ($('checkinHint')) {
    $('checkinHint').textContent = activeCheckinSite === 'global'
      ? '一键为国际号池全部已启用账号签到；国际站通常无签到活动，会提示不支持。账号多时整体耗时约 20s × 账号数（上限 5 分钟）。'
      : '一键为国内号池全部已启用账号签到（约 100 积分/天，连续第 7 天可达 1000）；账号多时整体耗时约 20s × 账号数（上限 5 分钟）。';
  }
}

async function switchPoolSite(site){
  site = normalizeSite(site);
  const data = await api('/direct-admin/api/pool-site', {method:'POST', body: JSON.stringify({site: site})});
  paintStatus(data);
  if (data.note) showToast(data.note);
  refreshModels().catch(function(){});
  return data;
}

async function switchPoolProduct(product){
  product = normalizeProduct(product);
  const data = await api('/direct-admin/api/pool-product', {method:'POST', body: JSON.stringify({product: product})});
  paintStatus(data);
  if (data.note) showToast(data.note);
  refreshModels().catch(function(){});
  return data;
}

function paintPoolProduct(product){
  product = normalizeProduct(product);
  activeProduct = product;
  const cbBtn = $('btnProductCodeBuddy');
  const wbBtn = $('btnProductWorkBuddy');
  if (cbBtn) cbBtn.className = product === 'codebuddy' ? 'active' : '';
  if (wbBtn) wbBtn.className = product === 'workbuddy' ? 'active' : '';
  if ($('pillProduct')) $('pillProduct').textContent = productLabel(product);
  paintPoolHint(activeCheckinSite || 'global', product, lastPoolAccounts);
}

function paintStatus(data){
  const stats = data.stats || {};
  const accounts = data.accounts || {};
  const cfg = data.config || {};
  const enabledCount = accounts.enabledCount != null ? accounts.enabledCount : ((accounts.accounts||[]).filter(function(a){return a.enabled;}).length);
  $('mEnabled').textContent = String(enabledCount);
  $('mSF').textContent = (stats.successRequests||0) + ' / ' + (stats.failedRequests||0);
  const totalTokens = stats.totalTokens || 0;
  const cachedTokens = stats.totalCachedTokens || 0;
  $('mTokens').textContent = cachedTokens > 0
    ? (totalTokens + ' · cache ' + cachedTokens)
    : String(totalTokens);
  const loggedIn = !!accounts.loggedIn;
  const primary = accounts.primary || ((accounts.accounts||[])[0] || null);
  $('mLogin').textContent = loggedIn
    ? ('已登录' + (primary && (primary.userNickname || primary.userName || primary.userId) ? (' · ' + (primary.userNickname || primary.userName || primary.userId)) : ''))
    : '未登录';
if ($('pillTransport')) $('pillTransport').textContent = data.transport || 'protocol_direct';
  if ($('pillVersion')) {
    const ver = data.version || (data.build && data.build.version) || 'dev';
    const ui = document.querySelector('meta[name="cbp-ui-revision"]');
    const uiRev = ui ? (ui.getAttribute('content') || '') : '';
    $('pillVersion').textContent = uiRev ? (ver + ' · ui ' + uiRev) : ver;
  }
  const poolSite = normalizeSite(data.poolSite || cfg.poolSite || cfg.site || 'global');
  const poolProduct = normalizeProduct(data.poolProduct || data.product || cfg.poolProduct || cfg.product || 'codebuddy');
  if ($('pillSite')) $('pillSite').textContent = siteLabel(poolSite);
  paintPoolSite(poolSite, accounts);
  paintPoolProduct(poolProduct);
  if ($('site') && !$('site').dataset.userTouched) {
    $('site').value = poolSite === 'domestic' ? 'domestic' : 'global';
  }
  const activeEnabled = accounts.activeEnabledCount != null ? accounts.activeEnabledCount : enabledCount;
  $('mEnabled').textContent = String(activeEnabled);
  const up = data.upstream || {};
  var healthOk = !!data.ok;
  var healthText = healthOk ? (loggedIn ? ('服务正常 · ' + siteLabel(poolSite) + '号池 · ' + productLabel(poolProduct)) : ('服务正常 · ' + siteLabel(poolSite) + '号池未登录')) : '状态异常';
  if (up.probed && up.ok === false) {
    healthOk = false;
    healthText = '上游不可达' + (up.message ? (' · ' + String(up.message).slice(0, 48)) : '');
  } else if (up.probed && up.ok) {
    healthText = '服务正常 · 上游可达 · ' + siteLabel(poolSite) + '号池 · ' + productLabel(poolProduct);
  }
  setHealth(healthOk, healthText);
  if (Array.isArray(data.accountUsages)) {
    data.accountUsages.forEach(function(item){
      if (!item || !item.accountId) return;
      usageByAccount[item.accountId] = item;
    });
  }
  if (primary && primary.id && primary.hasCredentials && !usageByAccount[primary.id] && !paintStatus._usageKick) {
    paintStatus._usageKick = true;
    fetchAccountUsage(primary.id, true).catch(function(){});
  }
  const usage = primary && usageByAccount[primary.id];
  if (usage && usage.credits) {
    const c = usage.credits;
    $('mCredits').textContent = c.unlimited ? '不限量' : ((c.remaining == null ? '-' : c.remaining) + ' / ' + (c.total == null ? '-' : c.total));
  } else if (usage && usage.error) {
    $('mCredits').textContent = '查询失败';
  } else {
    $('mCredits').textContent = loggedIn ? '点击查余额' : '—';
  }
  const raw = JSON.stringify(data, null, 2);
  $('statusBox').textContent = raw;
  $('statusRaw').textContent = raw;
  renderAccounts(accounts, poolSite);
}

function paintOAuth(data){
  const session = (data && data.session) || {};
  const login = (data && data.login) || {};
  const status = session.status || login.status || (data && data.ok ? 'ok' : 'idle');
  const err = session.error || login.message || '';
  let msg = '状态：' + status;
  if (session.label) msg += ' · ' + session.label;
  if (session.site) msg += ' · ' + session.site;
  if (err) msg += ' · ' + err;
  $('oauthMsg').textContent = msg;
  const raw = JSON.stringify(data, null, 2);
  $('oauthBox').textContent = raw;
  $('oauthRaw').textContent = raw;
}

let clientConfig = { apiKey: '', baseUrl: '', chatCompletionsUrl: '', recommendedModel: 'auto' };
function showToast(msg, kind){
  const el = $('toast');
  el.textContent = msg;
  el.className = 'toast show' + (kind === 'error' ? ' err' : '');
  clearTimeout(showToast._t);
  showToast._t = setTimeout(function(){ el.className = 'toast'; }, 2200);
}
async function copyText(text, label, button){
  const value = String(text||'');
  if (!value) { showToast((label||'内容') + '为空', 'error'); return; }
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(value);
    } else {
      const ta = document.createElement('textarea');
      ta.value = value; document.body.appendChild(ta); ta.select();
      document.execCommand('copy'); ta.remove();
    }
    const old = button ? button.textContent : '';
    if (button) button.textContent = '已复制';
    showToast((label||'内容') + ' 已复制');
    if (button) setTimeout(function(){ button.textContent = old; }, 1200);
  } catch (e) {
    showToast('复制失败：' + (e && e.message ? e.message : e), 'error');
  }
}
function paintClientConfig(cfg){
  clientConfig = cfg || clientConfig;
  const base = cfg.baseUrl || cfg.apiBase || '';
  const chat = cfg.chatCompletionsUrl || (base ? (base.replace(/\/$/,'') + '/chat/completions') : '');
  const responses = cfg.responsesUrl || (base ? (base.replace(/\/$/,'') + '/responses') : '');
  const model = bareModelId(cfg.recommendedModel || 'auto');
  $('openAiBaseUrl').value = base;
  $('openAiChatUrl').value = chat;
  if ($('openAiResponsesUrl')) $('openAiResponsesUrl').value = responses;
  $('openAiModel').value = model;
  const configured = !!cfg.apiKeyConfigured || !!cfg.apiKey;
  $('openAiApiKey').value = configured ? (cfg.apiKeyPreview || '已配置 · 点击复制') : '';
  $('openAiApiKey').placeholder = configured ? '' : 'API Key 未配置';
  $('copyApiKey').disabled = !configured;
  if (cfg.note) $('clientConfigHint').textContent = cfg.note;
  clientConfig.apiKey = cfg.apiKey || '';
  clientConfig.baseUrl = base;
  clientConfig.chatCompletionsUrl = chat;
  clientConfig.recommendedModel = model;
  renderBoundApiKeys(cfg.apiKeys || []);
}
function renderBoundApiKeys(list){
  const box = $('boundApiKeys');
  if (!box) return;
  if (!list.length) {
    box.innerHTML = '<div class="empty">还没有区域绑定 Key。主 Key 仍可用；新建一把即可按区域接入。</div>';
    return;
  }
  box.innerHTML = list.map(function(item){
    const site = normalizeSite(item.site);
    const preview = escapeHtml(item.preview || '已配置');
    return '<div class="bound-key">' +
      '<div class="meta"><span class="mono">' + preview + '</span>' +
        '<span class="badge site">' + escapeHtml(siteLabel(site)) + '</span></div>' +
      '<div class="actions">' +
        '<button class="ghost" data-act="copy-bound" data-site="' + escapeHtml(site) + '" type="button">复制</button>' +
        '<button class="danger" data-act="delete-bound" data-site="' + escapeHtml(site) + '" type="button">删除</button>' +
      '</div></div>';
  }).join('');
  box.querySelectorAll('button[data-act]').forEach(function(btn){
    btn.addEventListener('click', onBoundKeyAction);
  });
}
async function revealClientConfig(){
  return await api('/direct-admin/api/client-config/reveal', {method:'POST', body:'{}'});
}
async function onBoundKeyAction(ev){
  const btn = ev.currentTarget;
  const act = btn.getAttribute('data-act');
  const site = btn.getAttribute('data-site') || '';
  if (act === 'copy-bound') {
    if (!site) return;
    const revealed = await revealClientConfig();
    const item = (revealed.apiKeys || []).filter(function(x){ return x && x.site === site; }).pop();
    if (!item || !item.key) { showToast('未找到该区域的绑定 Key', 'error'); return; }
    return copyText(item.key, '绑定 Key', btn);
  }
  if (act === 'delete-bound') {
    if (!site) return;
    if (!confirm('删除该区域的绑定 Key？主 API Key 不会被删。')) return;
    const data = await api('/direct-admin/api/client-config/bound-keys', {method:'DELETE', body: JSON.stringify({site:site})});
    paintClientConfig(data);
    showToast('绑定 Key 已删除');
  }
}
async function generateBoundKey(){
  const site = $('boundKeySite') ? $('boundKeySite').value : 'global';
  const data = await api('/direct-admin/api/client-config/bound-keys', {method:'POST', body: JSON.stringify({site:site})});
  paintClientConfig(data);
  const created = data.created;
  if (created && created.key) await copyText(created.key, '绑定 Key', $('btnNewBoundKey'));
  if (data.note) showToast(data.note);
}
async function refreshClientConfig(){
  const data = await api('/direct-admin/api/client-config');
  paintClientConfig(data);
  return data;
}
async function generateApiKey(){
  if (!confirm('生成新的网关 API Key？\n\n旧 Key 会立即失效，新 Key 会写入 .env 并在重启后继续生效。\n请同步更新 ZCode / NewAPI 等客户端配置。')) return;
  const data = await api('/direct-admin/api/client-config/generate-key', {method:'POST', body:'{}'});
  paintClientConfig(data);
  const revealed = await revealClientConfig();
  if (revealed.apiKey) await copyText(revealed.apiKey, '新 API Key', $('btnGenerateKey'));
  if (data.note) showToast(data.note);
}

async function refreshStatus(fresh){
  const data = await api('/direct-admin/api/status' + (fresh ? '?fresh=1' : ''));
  paintStatus(data);
  if (fresh && data.creditsRefreshed) showToast('额度已从上游刷新');
}

function formatCheckinSummary(data){
  const s = data.summary || {};
  const parts = [];
  if (s.checkedIn > 0) parts.push('新签 ' + s.checkedIn);
  if (s.alreadyDone > 0) parts.push('已签 ' + s.alreadyDone);
  if (s.skipped > 0) parts.push('跳过 ' + s.skipped);
  if (s.unsupported > 0) parts.push('不支持 ' + s.unsupported);
  if (s.failed > 0) parts.push('失败 ' + s.failed);
  if (!parts.length) return data.note || '当前号池没有可签到的账号';
  return parts.join(' · ');
}

async function runPoolCheckin(){
  if (!activeCheckinSite) {
    showToast('号池状态尚未加载，请稍后再试', 'error');
    return;
  }
  const btn = $('btnCheckin');
  checkinBusy = true;
  if (btn) { btn.disabled = true; btn.textContent = '签到中…'; }
  try {
    const data = await api('/direct-admin/api/codebuddy/checkin', {method:'POST', body: JSON.stringify({site: activeCheckinSite})});
    const summary = formatCheckinSummary(data);
    showToast(summary, data.ok ? undefined : 'error');
    const results = data.results || [];
    for (const item of results) {
      if (item.ok && item.accountId && !item.alreadyCheckedIn) {
        await fetchAccountUsage(item.accountId, true);
      }
    }
    await refreshStatus();
    if (results.length && $('checkinRaw')) {
      const lines = results.map(function(item){
        const name = item.label || item.accountId || '账号';
        if (!item.supported) return name + '：' + (item.message || '不支持');
        if (item.ok && item.alreadyCheckedIn) return name + '：今日已签';
        if (item.ok) return name + '：' + (item.message || ('+' + (item.rewardCredits || 0) + ' 积分'));
        return name + '：' + (item.message || '失败');
      });
      $('checkinRaw').textContent = lines.join('\n');
      $('checkinRaw').hidden = false;
    }
  } catch (e) {
    showToast('签到失败：' + (e.message || e), 'error');
  } finally {
    checkinBusy = false;
    if (btn) { btn.disabled = false; btn.textContent = '每日签到'; }
  }
}

function formatPoolTestSummary(data){
  const s = data.summary || {};
  const parts = [];
  if (s.passed > 0) parts.push('通过 ' + s.passed);
  if (s.failed > 0) parts.push('失败 ' + s.failed);
  if (s.skipped > 0) parts.push('跳过 ' + s.skipped);
  const model = data.model ? (' · ' + data.model) : '';
  if (!parts.length) return (data.note || '当前号池没有可测试的账号') + model;
  return parts.join(' · ') + model;
}

async function runPoolChatTest(){
  if (!activeCheckinSite) {
    showToast('号池状态尚未加载，请稍后再试', 'error');
    return;
  }
  const btn = $('btnPoolTest');
  poolTestBusy = true;
  if (btn) { btn.disabled = true; btn.textContent = '测试中…'; }
  try {
    const body = {site: activeCheckinSite};
    const model = selectedTestModel();
    if (model) body.model = model;
    const data = await api('/direct-admin/api/codebuddy/test', {method:'POST', body: JSON.stringify(body)});
    showToast(formatPoolTestSummary(data), data.ok ? undefined : 'error');
    const results = data.results || [];
    if (results.length && $('poolTestRaw')) {
      const lines = results.map(function(item){
        const name = item.label || item.accountId || '账号';
        if (item.ok) return name + '：ok · ' + (item.latencyMs || 0) + 'ms · ' + (item.model || '');
        return name + '：' + (item.message || '失败');
      });
      $('poolTestRaw').textContent = lines.join('\n');
      $('poolTestRaw').hidden = false;
    }
  } catch (e) {
    showToast('批量测试失败：' + (e.message || e), 'error');
  } finally {
    poolTestBusy = false;
    if (btn) { btn.disabled = false; btn.textContent = '批量测试'; }
  }
}

async function testAccountChat(accountId){
  const body = {};
  const model = selectedTestModel();
  if (model) body.model = model;
  const data = await api('/direct-admin/api/codebuddy/accounts/'+encodeURIComponent(accountId)+'/test', {
    method:'POST',
    body: JSON.stringify(body)
  });
  if (data.ok) {
    showToast((data.label || data.accountId || '账号') + ' · ' + (data.latencyMs || 0) + 'ms · ' + (data.model || ''));
  } else {
    showToast('测试失败：' + (data.message || 'unknown'), 'error');
  }
  return data;
}

async function refreshModels(){
  const data = await api('/direct-admin/api/codebuddy/models?fresh=1');
  const raw = JSON.stringify(data, null, 2);
  $('modelsBox').textContent = raw;
  $('modelsRaw').textContent = raw;
  renderModels(data);
}

async function startOAuth(){
  $('oauthMsg').textContent = '正在创建登录会话…';
  const data = await api('/direct-admin/api/codebuddy/oauth/start', {
    method:'POST',
    body: JSON.stringify({site:$('site').value, label:$('label').value, reuseExisting:false})
  });
  paintOAuth(data);
  const session = (data && data.session) || {};
  const url = (data.login && data.login.url) || session.url || '#';
  const launch = (data.login && data.login.launchUrl) || session.launchUrl || url;
  $('launchLink').href = launch;
  // 服务端已经用无痕窗口拉起时不要再开普通标签页：那会带着浏览器里已登录的
  // CodeBuddy 会话，等于给同一个账号重复授权。
  if (url && url !== '#' && !session.openBrowser) window.open(url, '_blank', 'noopener');
}

async function pollOAuth(){
  $('oauthMsg').textContent = '正在检查登录…';
  const data = await api('/direct-admin/api/codebuddy/oauth/poll', {method:'POST', body:'{}'});
  paintOAuth(data);
  await refreshStatus();
}

async function fetchAccountUsage(accountId, silent){
  try {
    const result = await api('/direct-admin/api/codebuddy/accounts/'+encodeURIComponent(accountId)+'/usage', {method:'POST', body:'{}'});
    usageByAccount[accountId] = result;
    if (!silent) {
      const credits = result.credits || {};
      showToast(credits.display || credits.label || '余额已更新');
    }
  } catch (e) {
    usageByAccount[accountId] = { error: e.message || String(e) };
    if (!silent) showToast('查余额失败：' + (e.message || e), 'error');
  }
  await refreshStatus();
}
async function onAccountAction(ev){
  const btn = ev.currentTarget;
  const id = btn.getAttribute('data-id');
  const act = btn.getAttribute('data-act');
  if (act === 'usage') {
    await fetchAccountUsage(id, false);
    return;
  }
  if (act === 'test') {
    btn.disabled = true;
    try {
      await testAccountChat(id);
    } catch (e) {
      showToast('测试失败：' + (e.message || e), 'error');
    } finally {
      btn.disabled = false;
    }
    return;
  }
  if (act === 'delete') {
    if (!confirm('确认删除该账号？')) return;
    await api('/direct-admin/api/codebuddy/accounts/'+encodeURIComponent(id), {method:'DELETE'});
    delete usageByAccount[id];
  } else if (act === 'toggle') {
    const enabled = btn.getAttribute('data-enabled') === '1';
    await api('/direct-admin/api/codebuddy/accounts/'+encodeURIComponent(id)+'/'+(enabled?'enable':'disable'), {method:'POST', body:'{}'});
  } else if (act === 'refresh') {
    await api('/direct-admin/api/codebuddy/accounts/'+encodeURIComponent(id)+'/refresh-token', {method:'POST', body:'{}'});
  }
  await refreshStatus();
}

if ($('btnPoolDomestic')) $('btnPoolDomestic').onclick = function(){ switchPoolSite('domestic').catch(function(e){ showToast(e.message, 'error'); }); };
if ($('btnPoolGlobal')) $('btnPoolGlobal').onclick = function(){ switchPoolSite('global').catch(function(e){ showToast(e.message, 'error'); }); };
if ($('btnProductCodeBuddy')) $('btnProductCodeBuddy').onclick = function(){ switchPoolProduct('codebuddy').catch(function(e){ showToast(e.message, 'error'); }); };
if ($('btnProductWorkBuddy')) $('btnProductWorkBuddy').onclick = function(){ switchPoolProduct('workbuddy').catch(function(e){ showToast(e.message, 'error'); }); };
if ($('site')) $('site').addEventListener('change', function(){ $('site').dataset.userTouched = '1'; });
$('btnRefresh').onclick = function(){
  const btn = $('btnRefresh');
  btn.disabled = true;
  refreshStatus(true).catch(function(e){
    $('statusRaw').textContent = e.message;
    setHealth(false, '刷新失败');
  }).finally(function(){ btn.disabled = false; });
};
if ($('btnRefreshUsage')) $('btnRefreshUsage').onclick = function(){ refreshUsage().catch(function(e){ showToast(e.message, 'error'); }); };
if ($('usageRangeSeg')) $('usageRangeSeg').querySelectorAll('button[data-range]').forEach(function(btn){
  btn.addEventListener('click', function(){
    paintUsageRange(btn.getAttribute('data-range'));
    refreshUsage().catch(function(e){ showToast(e.message, 'error'); });
  });
});
if ($('btnUsagePrev')) $('btnUsagePrev').onclick = function(){
  if (usagePage <= 1) return;
  usagePage--;
  refreshUsage().catch(function(e){ showToast(e.message, 'error'); });
};
if ($('btnUsageNext')) $('btnUsageNext').onclick = function(){
  usagePage++;
  refreshUsage().catch(function(e){ showToast(e.message, 'error'); });
};
function onUsageFilterChange(){
  usageAccount = ($('usageAccountFilter') && $('usageAccountFilter').value) || '';
  usageModel = ($('usageModelFilter') && $('usageModelFilter').value) || '';
  usagePage = 1;
  refreshUsage().catch(function(e){ showToast(e.message, 'error'); });
}
if ($('usageAccountFilter')) $('usageAccountFilter').onchange = onUsageFilterChange;
if ($('usageModelFilter')) $('usageModelFilter').onchange = onUsageFilterChange;
if ($('btnCheckin')) $('btnCheckin').onclick = function(){ runPoolCheckin().catch(function(e){ showToast(e.message, 'error'); }); };
if ($('btnPoolTest')) $('btnPoolTest').onclick = function(){ runPoolChatTest().catch(function(e){ showToast(e.message, 'error'); }); };
$('btnModels').onclick = function(){ refreshModels().catch(function(e){ $('modelsRaw').textContent = e.message; $('modelChips').innerHTML = '<div class="empty">' + escapeHtml(e.message) + '</div>'; }); };
$('btnStart').onclick = function(){ startOAuth().catch(function(e){ $('oauthMsg').textContent = e.message; $('oauthRaw').textContent = e.message; }); };
$('btnPoll').onclick = function(){ pollOAuth().catch(function(e){ $('oauthMsg').textContent = e.message; $('oauthRaw').textContent = e.message; }); };
$('btnRefreshClient').onclick = function(){ refreshClientConfig().catch(function(e){ showToast(e.message, 'error'); }); };
$('btnGenerateKey').onclick = function(){ generateApiKey().catch(function(e){ showToast(e.message, 'error'); }); };
if ($('btnNewBoundKey')) $('btnNewBoundKey').onclick = function(){ generateBoundKey().catch(function(e){ showToast(e.message, 'error'); }); };
$('copyBaseUrl').onclick = function(){ copyText($('openAiBaseUrl').value, 'Base URL', $('copyBaseUrl')); };
$('copyChatUrl').onclick = function(){ copyText($('openAiChatUrl').value, 'Chat Completions', $('copyChatUrl')); };
if ($('copyResponsesUrl')) $('copyResponsesUrl').onclick = function(){ copyText($('openAiResponsesUrl').value, 'Responses', $('copyResponsesUrl')); };
$('copyModel').onclick = function(){ copyText($('openAiModel').value, '模型', $('copyModel')); };
$('copyApiKey').onclick = function(){
  refreshClientConfig().then(function(cfg){
    if (!cfg.apiKeyConfigured) { $('openAiApiKey').value = ''; showToast('API Key 未配置', 'error'); return; }
    $('openAiApiKey').value = cfg.apiKeyPreview || '已配置 · 点击复制';
    return revealClientConfig().then(function(full){
      if (!full.apiKey) { showToast('API Key 未配置', 'error'); return; }
      return copyText(full.apiKey, 'API Key', $('copyApiKey'));
    });
  }).catch(function(e){ showToast(e.message, 'error'); });
};
refreshStatus().catch(function(e){ $('statusRaw').textContent = e.message; setHealth(false, '无法连接'); });
refreshClientConfig().catch(function(e){ showToast(e.message, 'error'); });
refreshModels().catch(function(){});
loadModelPolicy().catch(function(){});
refreshCheckin().catch(function(){});
setInterval(function(){ refreshStatus().catch(function(){}); }, 15000);

/* 上游版本监控 (Upstream Update Check) */
let updShownTag = null;
try { updShownTag = localStorage.getItem('cbpUpdShownTag') || null; } catch(e){}
async function refreshUpdateCheck(){
  let d = null;
  try { d = await api('/direct-admin/api/update/check'); }
  catch(e){ d = null; }
  const pill = $('pillUpdate');
  if (!pill) return;
  if (d && d.updateAvailable && d.latestTag) {
    $('pillUpdateText').textContent = d.latestTag + ' 可更新';
    pill.hidden = false;
    pill.onclick = function(){ if (d && d.url) window.open(d.url, '_blank'); };
    if (updShownTag !== d.latestTag) {
      updShownTag = d.latestTag;
      try { localStorage.setItem('cbpUpdShownTag', d.latestTag); } catch(e){}
      showUpdateModal(d);
    }
  } else {
    pill.hidden = true;
  }
}
function showUpdateModal(d){
  const meta = [];
  if (d.publishedAt) {
    const t = new Date(d.publishedAt);
    if (!isNaN(t.getTime())) meta.push('发布于 ' + t.toLocaleDateString());
  }
  meta.push('当前 ' + (d.currentVersion || '未知版本'));
  if (d.repo) meta.push(d.repo);
  $('updTitle').textContent = '发现新版本 ' + (d.latestTag || '');
  $('updMeta').textContent = meta.join(' · ');
  $('updBody').textContent = d.notes || ('上游仓库已发布新版本，点击“去下载”前往 GitHub 查看发布说明。');
  $('updLink').href = d.url || (d.repo ? ('https://github.com/' + d.repo + '/releases/latest') : '#');
  $('updModal').hidden = false;
}
$('updDismiss').onclick = function(){ $('updModal').hidden = true; };
$('updModal').addEventListener('click', function(e){ if (e.target === this) this.hidden = true; });
refreshUpdateCheck();
setInterval(function(){ refreshUpdateCheck(); }, 30 * 60 * 1000);

/* 活动与系统 (Tab 07) */
function fmtTs(ts){
  if (!ts) return '—';
  var d = new Date(ts);
  if (isNaN(d.getTime())) return String(ts);
  return d.toLocaleString();
}
const ACTIVITY_KIND_LABEL = {
  'server-start':'服务启动','server-stop':'服务停止','server-error':'服务异常',
  'checkin':'签到',
  'account-switch':'账号切换','account-enabled':'启用账号','account-disabled':'禁用账号','account-removed':'删除账号',
  'model-policy-update':'更新模型白名单','api-key-generated':'生成 API Key','api-key-bound':'绑定站点 Key'
};
const ACTIVITY_GROUP = {
  'server-start':'lifecycle','server-stop':'lifecycle',
  'checkin':'checkin',
  'account-switch':'account','account-enabled':'account','account-disabled':'account','account-removed':'account',
  'model-policy-update':'system','api-key-generated':'system','api-key-bound':'system'
};
// 不参与业务分组展示的事件：error 仅出现在下方「报错日志」，open-config-dir 不再展示。
const ACTIVITY_HIDDEN = {'server-error':true,'open-config-dir':true};
const GROUP_LABEL = {'lifecycle':'服务生命周期','checkin':'签到','account':'账号管理','system':'系统与配置'};
function actGroupOf(e){
  if (ACTIVITY_HIDDEN[e.kind]) return null;
  return ACTIVITY_GROUP[e.kind] || 'system';
}
function kindName(k){ return ACTIVITY_KIND_LABEL[k] || escapeHtml(String(k||'未知')); }
function formatFields(fields){
  if (!fields) return '';
  var parts = [];
  Object.keys(fields).forEach(function(k){ parts.push(k + ': ' + escapeHtml(String(fields[k]))); });
  return '<span class="act-fields">' + parts.join(' · ') + '</span>';
}
// 账号管理 / 签到 日志格式：<名称>: 号池区域 · 国内/国际（+ 详情）
function formatActItem(e){
  const kind = e.kind;
  const special = (kind === 'account-switch' || kind === 'account-enabled' || kind === 'account-disabled' || kind === 'account-removed' || kind === 'checkin');
  if (!special) return '<span class="act-kind">' + kindName(kind) + '</span>' + formatFields(e.fields);
  const f = e.fields || {};
  const raw = f.site || (f.to === 'domestic' || f.to === 'global' ? f.to : '');
  const region = raw === 'domestic' ? '国内' : raw === 'global' ? '国际' : '';
  const name = f.account || f.id || f.from || '';
  const product = f.product || '';
  const parts = [];
  if (region) parts.push('号池区域 · ' + region);
  else if (product) parts.push('上游产品 · ' + escapeHtml(String(product)));
  else if (f.target) parts.push(escapeHtml(String(f.target)));
  if (kind === 'checkin'){
    const ok = (f.ok === true || f.ok === 'true');
    if (f.total != null && f.done != null && f.failed != null){
      parts.push('成功 ' + (f.ok == null ? 0 : f.ok) + '/' + f.total + ' · 已签到 ' + f.done + ' · 失败 ' + f.failed);
    } else if (!ok) {
      parts.push((f.already === true || f.already === 'true') ? '已签到' : '失败' + (f.message ? ' · ' + escapeHtml(String(f.message)) : ''));
    } else {
      parts.push('签到成功');
    }
  } else if (f.reason) {
    parts.push(escapeHtml(String(f.reason)));
  }
  let html = '<span class="act-kind">' + kindName(kind) + '</span>';
  if (name) html += ' <span class="act-name">' + escapeHtml(String(name)) + '</span>';
  if (parts.length) html += '<span class="act-fields">: ' + parts.join(' · ') + '</span>';
  return html;
}
const LOG_SUB_GROUP = {'lifecycle':'lifecycle','checkin':'checkin','account':'account','system':'system'};
async function refreshActivity(){
  const data = await api('/direct-admin/api/system/activity?limit=160');
  $('actMeta').textContent = data.path + (data.size ? ' · ' + data.size + ' bytes' : '');
  const box = $('activityList');
  const g = LOG_SUB_GROUP[logSubTab] || 'lifecycle';
  const list = [];
  const seen = {};
  (data.entries || []).forEach(function(e){
    if (actGroupOf(e) !== g) return;
    const key = e.kind + '\u0001' + e.ts + '\u0001' + JSON.stringify(e.fields || {});
    if (seen[key]) return;
    seen[key] = 1;
    list.push(e);
  });
  const top = list.slice(0, 3);
  if (!top.length){
    box.innerHTML = '<div class="empty">该业务类型暂无活动记录。</div>';
    return;
  }
  let html = '<div class="act-group"><div class="act-group-head">'
    + '<span class="gname">' + (GROUP_LABEL[g] || g) + '</span>'
    + '<span class="gcount">' + top.length + ' 条</span>'
    + '<span class="gtime">' + escapeHtml(fmtTs(top[0].ts)) + '</span>'
    + '</div>';
  html += top.map(function(e){
    return '<div class="act-item"><span class="act-time">' + escapeHtml(fmtTs(e.ts)) + '</span>'
      + formatActItem(e) + '</div>';
  }).join('');
  html += '</div>';
  box.innerHTML = html;
}
let logSubTab = 'lifecycle';
function switchLogSubTab(sub){
  logSubTab = sub || 'lifecycle';
  document.querySelectorAll('#logSubSeg button').forEach(function(b){ b.classList.toggle('active', b.getAttribute('data-sub') === logSubTab); });
  const isError = (logSubTab === 'error');
  $('actMeta').classList.toggle('hidden', isError);
  $('activityList').classList.toggle('hidden', isError);
  $('errMeta').classList.toggle('hidden', !isError);
  $('errorList').classList.toggle('hidden', !isError);
  if (isError) refreshErrors().catch(function(e){ showToast(e.message, 'error'); });
  else refreshActivity().catch(function(e){ showToast(e.message, 'error'); });
}
if ($('logSubSeg')) $('logSubSeg').onclick = function(ev){
  const b = ev.target.closest('button');
  if (b) switchLogSubTab(b.getAttribute('data-sub'));
};
$('btnRefreshActivity').onclick = function(){
  if (logSubTab === 'error') refreshErrors().catch(function(e){ showToast(e.message, 'error'); });
  else refreshActivity().catch(function(e){ showToast(e.message, 'error'); });
};
async function refreshErrors(){
  const data = await api('/direct-admin/api/system/activity?limit=200');
  $('errMeta').textContent = data.path + (data.size ? ' · ' + data.size + ' bytes' : '');
  const box = $('errorList');
  const list = [];
  const seen = {};
  (data.entries || []).forEach(function(e){
    if (e.kind !== 'server-error') return;
    const key = e.kind + '\u0001' + e.ts + '\u0001' + JSON.stringify(e.fields || {});
    if (seen[key]) return;
    seen[key] = 1;
    list.push(e);
  });
  const top = list.slice(0, 3);
  if (!top.length){
    box.innerHTML = '<div class="empty">暂无报错记录。服务异常事件会在此显示。</div>';
    return;
  }
  box.innerHTML = top.map(function(e){
    return '<div class="act-item"><span class="act-time">' + escapeHtml(fmtTs(e.ts)) + '</span>'
      + '<span class="act-kind">' + kindName(e.kind) + '</span>'
      + formatFields(e.fields) + '</div>';
  }).join('');
}
$('btnOpenConfigDir').onclick = function(){
  const btn = $('btnOpenConfigDir');
  btn.disabled = true;
  fetch('/direct-admin/api/system/open-config-dir', {method:'POST', credentials:'same-origin'})
    .then(function(res){ return res.json().catch(function(){ return {}; }).then(function(d){ if (!res.ok) throw new Error(d.error || ('HTTP ' + res.status)); return d; }); })
    .then(function(d){
      $('configDirLine').textContent = d.path || '';
      showToast('已打开配置目录: ' + (d.path || ''));
    })
    .catch(function(e){ showToast('打开失败: ' + e.message, 'error'); })
    .finally(function(){ btn.disabled = false; });
};
refreshActivity().catch(function(){});
refreshErrors().catch(function(){});

/* 模型白名单 (Tab 04) */
async function loadModelPolicy(){
  const data = await api('/direct-admin/api/system/model-policy');
  $('mpEnabled').checked = !!data.enabled;
  $('mpAllow').value = (data.allow || []).join('\n');
  $('mpDeny').value = (data.deny || []).join('\n');
  $('mpStatus').textContent = (data.path || '') + (data.enabled ? ' · 已启用' : ' · 未启用');
}
async function saveModelPolicy(){
  const body = {
    enabled: $('mpEnabled').checked,
    allow: $('mpAllow').value.split('\n').map(function(s){ return s.trim(); }).filter(Boolean),
    deny: $('mpDeny').value.split('\n').map(function(s){ return s.trim(); }).filter(Boolean)
  };
  const res = await fetch('/direct-admin/api/system/model-policy', {
    method:'PUT', credentials:'same-origin',
    headers:{'Content-Type':'application/json'},
    body: JSON.stringify(body)
  });
  const d = await res.json().catch(function(){ return {}; });
  if (!res.ok) throw new Error(d.error || ('HTTP ' + res.status));
  $('mpEnabled').checked = !!d.enabled;
  $('mpAllow').value = (d.allow || []).join('\n');
  $('mpDeny').value = (d.deny || []).join('\n');
  $('mpStatus').textContent = (d.path || '') + (d.enabled ? ' · 已启用' : ' · 未启用');
  showToast(d.enabled ? '白名单已保存并启用' : '白名单已保存（未启用）');
}
$('btnSavePolicy').onclick = function(){ saveModelPolicy().catch(function(e){ showToast(e.message, 'error'); }); };
$('btnResetPolicy').onclick = function(){ loadModelPolicy().catch(function(e){ showToast(e.message, 'error'); }); };

/* 签到状态 (Tab 05) */
async function refreshCheckin(){
  const data = await api('/direct-admin/api/codebuddy/checkin-status');
  if (data.note) $('checkinNote').textContent = data.note;
  const s = data.summary || {};
  $('ciTotal').textContent = s.total || 0;
  $('ciCheckedIn').textContent = s.checkedIn || 0;
  $('ciPending').textContent = s.pending || 0;
  $('ciInactive').textContent = s.inactive || 0;
  $('ciFailed').textContent = s.failed || 0;
  const rows = data.accounts || [];
  const tbody = $('checkinRows');
  if (!rows.length){
    tbody.innerHTML = '<tr><td colspan="8" class="empty">没有启用的可签到账号。</td></tr>';
    return;
  }
  tbody.innerHTML = rows.map(function(a){
    const rawLabel = (a.label || '').trim();
    const realName = a.userNickname || a.userName || a.userId || '';
    const genericLabel = !rawLabel || /^CodeBuddy(\s+OAuth)?$/i.test(rawLabel);
    const name = escapeHtml(realName || (genericLabel ? rawLabel : '') || a.accountId);
    const siteText = siteLabel(a.site);
    const state = a.error
      ? '<span class="act-kind" style="color:var(--fg-40)">失败</span>' + '<div class="meta-line">' + escapeHtml(a.error) + '</div>'
      : !a.active
        ? '<span class="act-kind" style="color:var(--fg-40)">无活动</span>'
        : a.todayCheckedIn
          ? '<span class="act-kind" style="color:var(--fg)">已签到</span>'
          : '<span class="act-kind" style="color:var(--fg-80)">待签到</span>';
    const btn = (a.error || !a.active || a.todayCheckedIn)
      ? ''
      : '<button type="button" class="ghost" data-ci="' + escapeHtml(a.accountId) + '" style="padding:4px 10px;font-size:11px">签到</button>';
    return '<tr>'
      + '<td><span class="idbtn" data-copy="' + escapeHtml(a.accountId) + '" title="点击复制 ID">' + name + '</span></td>'
      + '<td class="mono">' + escapeHtml(siteText) + '</td>'
      + '<td>' + state + '</td>'
      + '<td class="mono">' + (a.streakDays || 0) + ' 天</td>'
      + '<td class="mono">' + (a.dailyCredit || 0) + '</td>'
      + '<td class="mono">' + (a.todayCredit || 0) + '</td>'
      + '<td class="mono low">' + (a.isStreakDay ? '连签加成' : '—') + '</td>'
      + '<td>' + btn + '</td>'
      + '</tr>';
  }).join('');
}
async function checkinAccount(id){
  const res = await fetch('/direct-admin/api/codebuddy/accounts/' + encodeURIComponent(id) + '/checkin', {
    method:'POST', credentials:'same-origin'
  });
  const d = await res.json().catch(function(){ return {}; });
  if (!res.ok) throw new Error(d.error || ('HTTP ' + res.status));
  if (d.ok){ showToast('签到成功: ' + d.message); } else { showToast(d.message || '签到未生效', 'error'); }
}
async function checkinAllPool(){
  const site = (typeof activeCheckinSite !== 'undefined' && activeCheckinSite) || '';
  const res = await fetch('/direct-admin/api/codebuddy/checkin', {
    method:'POST', credentials:'same-origin',
    headers:{'Content-Type':'application/json'},
    body: JSON.stringify({site:site})
  });
  const d = await res.json().catch(function(){ return {}; });
  if (!res.ok) throw new Error(d.error || ('HTTP ' + res.status));
  let n = 0;
  (d.results || []).forEach(function(item){ if (item.ok && !item.alreadyCheckedIn) n++; });
  showToast(n ? ('签到完成：' + n + ' 个账号成功') : '没有待签到的账号');
  await refreshCheckin();
}
$('btnRefreshCheckin').onclick = function(){ refreshCheckin().catch(function(e){ showToast(e.message, 'error'); }); };
$('btnCheckinAll').onclick = function(){ checkinAllPool().catch(function(e){ showToast(e.message, 'error'); }); };
$('checkinRows').addEventListener('click', function(ev){
  const btn = ev.target.closest('button[data-ci]');
  if (!btn) return;
  btn.disabled = true;
  checkinAccount(btn.getAttribute('data-ci'))
    .then(function(){ return refreshCheckin(); })
    .catch(function(e){ showToast(e.message, 'error'); })
    .finally(function(){ btn.disabled = false; });
});
</script>
</body>
</html>`
}

func LaunchPage(message string, success bool) string {
	stateLabel := "认证未完成"
	if success {
		stateLabel = "认证成功"
	}
	return `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>CodeBuddy OAuth</title>
<style>
:root{
  --bg:#141416;
  --fg:#E8E5E0;
  --fg-60:rgba(232,229,224,0.60);
  --fg-20:rgba(232,229,224,0.20);
  --display:Georgia,"Iowan Old Style","Palatino Linotype",Palatino,"Songti SC","Noto Serif SC",serif;
  --sans:system-ui,-apple-system,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;
}
*{box-sizing:border-box}
body{
  margin:0;min-height:100dvh;display:grid;place-items:center;padding:24px;
  font-family:var(--sans);font-size:14px;line-height:1.625;color:var(--fg);
  background:var(--bg);
}
.card{
  width:min(520px,100%);padding:32px;
  background:transparent;border:1px solid var(--fg-20);
}
.meta{
  font-size:11px;letter-spacing:.2em;text-transform:uppercase;color:var(--fg-60);margin-bottom:12px;
}
h1{
  margin:0 0 12px;font-family:var(--display);font-size:26px;letter-spacing:-.02em;font-weight:400;color:var(--fg);
}
p{
  margin:0 0 24px;color:var(--fg-60);font-size:14px;line-height:1.6;
}
a{
  display:inline-flex;align-items:center;justify-content:center;
  padding:10px 20px;text-decoration:none;font-size:12px;letter-spacing:.08em;text-transform:uppercase;
  color:var(--bg);background:var(--fg);border:1px solid var(--fg);
  transition:background .2s, color .2s;
}
a:hover{background:var(--fg-20);color:var(--bg)}
a:focus-visible{outline:2px solid var(--fg);outline-offset:2px}
</style>
</head>
<body>
<div class="card">
  <div class="meta">CodeBuddy Proxy · OAuth</div>
  <h1>` + html.EscapeString(stateLabel) + `</h1>
  <p>` + html.EscapeString(message) + `</p>
  <a href="/direct-admin/#codebuddy">返回管理台</a>
</div>
</body>
</html>`
}

func Compact(value string) string { return strings.TrimSpace(value) }
