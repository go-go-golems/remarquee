// Deterministic logic probes against the actual app.js, no browser/cloud needed.
// The DOM is stubbed; this does NOT verify browser rendering or native drag APIs.
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');
const elements = new Map();
function element() {
  return { value: '', hidden: true, textContent: '', classList: {toggle() {}},
    listeners: {}, addEventListener(k, f) { this.listeners[k] = f; },
    append() {}, showModal() {}, close() {}, focus() {}, select() {}, click() {} };
}
const pending = [];
const ctx = vm.createContext({console, setTimeout, clearTimeout,
  location: { hash: '' }, window: {addEventListener() {}},
  document: { getElementById(id) { if (!elements.has(id)) elements.set(id, element()); return elements.get(id); },
    createElement: element, createTextNode: element, querySelectorAll() { return []; }, addEventListener() {} },
  fetch(url) { return new Promise(resolve => pending.push({url, resolve})); },
});
let source = fs.readFileSync('cmd/remarquee/cmds/serve/frontend/app.js', 'utf8');
// Disable network-starting boot only; execute all declarations and listener wiring.
source = source.slice(0, source.indexOf('(async function boot()'));
source += '\nglobalThis.review = {state, loadFiles, runSearch, currentHashPath, setHash}; renderCrumbs = () => {}; renderRows = () => {}; renderDetail = () => {};';
vm.runInContext(source, ctx);
function complete(req, data) { req.resolve({ok:true,status:200,json:async()=>data}); }
(async () => {
  const r = ctx.review;
  const a = r.loadFiles('/A'); const b = r.loadFiles('/B');
  complete(pending[1], {entries:[{id:'B',name:'B-entry'}]}); await b;
  complete(pending[0], {entries:[{id:'A',name:'A-entry'}]}); await a;
  console.log(`out-of-order folder responses: cwd=${r.state.cwd} rows=${r.state.entries[0].name}`);
  assert.equal(r.state.cwd, '/B'); assert.equal(r.state.entries[0].name, 'A-entry');
  pending.length = 0;
  const search = r.runSearch('proj'); complete(pending[0], {results:[{entry:{id:'doc',name:'Project'}}]}); await search;
  const nav = r.loadFiles('/Target'); complete(pending[1], {entries:[{id:'t',name:'Target-child'}]}); await nav;
  console.log(`navigate from search: cwd=${r.state.cwd} query=${r.state.query}`);
  assert.equal(r.state.query, 'proj');
  ctx.location.hash = '#/literal%20name';
  console.log(`literal percent-encoded-looking name parses as: ${r.currentHashPath()}`);
  assert.equal(r.currentHashPath(), '/literal name');
  const input = elements.get('upload-files'); input.value = 'old-selection.pdf';
  elements.get('upload-btn').listeners.click();
  console.log(`reopen upload: input value still=${input.value}`);
  assert.equal(input.value, 'old-selection.pdf');
  console.log('All probes reproduced current behavior; these assertions document defects, not a passing product contract.');
})().catch(err => { console.error(err); process.exitCode = 1; });
