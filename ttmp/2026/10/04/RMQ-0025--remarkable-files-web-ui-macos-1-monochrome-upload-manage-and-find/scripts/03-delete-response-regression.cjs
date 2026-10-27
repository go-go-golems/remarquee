// Offline regression tests for the actual embedded frontend functions.
// Run from repository root: node --test <this-file>
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { test } = require('node:test');
const root = path.resolve(__dirname, '../../../../../..');
const source = fs.readFileSync(path.join(root, 'cmd/remarquee/cmds/serve/frontend/app.js'), 'utf8');
function between(start, end) {
  const a = source.indexOf(start), b = source.indexOf(end, a);
  assert.ok(a >= 0 && b > a, `Missing frontend function boundary: ${start}`);
  return source.slice(a, b);
}
const functions = between('async function api(', 'function toast(') + '\n' +
  between('async function confirmDelete(', 'async function refreshAfterMutation(');

function harness(status, body, { confirmed = true, refreshError = null } = {}) {
  const toasts = [];
  let refreshes = 0, requests = 0;
  const sandbox = {
    fetch: async () => { requests++; return { ok: status >= 200 && status < 300, status, statusText: 'Bad Gateway', json: async () => body }; },
    confirmDialog: async () => confirmed,
    toast: (message, isError) => toasts.push({ message, isError: !!isError }),
    refreshAfterMutation: async () => { refreshes++; if (refreshError) throw new Error(refreshError); },
  };
  const { confirmDelete } = vm.runInNewContext(functions + '\n({confirmDelete});', sandbox);
  return { run: () => confirmDelete({ id: 'folder', name: 'Folder' }), toasts, refreshes: () => refreshes, requests: () => requests };
}

for (const [label, deleted, remaining] of [['partial removal', 1, 2], ['notification failure', 3, 0]]) {
  test(`${label} reports error and refreshes rather than claiming success`, async () => {
    const message = `delete removed ${deleted} record(s) (${remaining} remaining): cloud failed`;
    const h = harness(502, { deleted, remainingIds: [], error: { code: 'partial_delete', message } });
    await h.run();
    assert.deepEqual(h.toasts, [{ message: `Delete failed: ${message}`, isError: true }]);
    assert.equal(h.refreshes(), 1);
    assert.equal(h.requests(), 1);
  });
}
test('successful deletion still reports success and refreshes', async () => {
  const h = harness(200, { deleted: 3 });
  await h.run();
  assert.deepEqual(h.toasts, [{ message: 'Deleted Folder', isError: false }]);
  assert.equal(h.refreshes(), 1);
});
test('refresh failure preserves the deletion error and adds refresh diagnostic', async () => {
  const h = harness(502, { error: { message: 'partial removal' } }, { refreshError: 'list unavailable' });
  await h.run();
  assert.equal(h.toasts.at(-1).message, 'Delete failed: partial removal; refresh failed: list unavailable');
  assert.ok(h.toasts.every(t => t.isError));
  assert.equal(h.refreshes(), 1);
});
test('cancel does not write or refresh', async () => {
  const h = harness(200, { deleted: 3 }, { confirmed: false });
  await h.run();
  assert.equal(h.requests(), 0);
  assert.equal(h.refreshes(), 0);
  assert.equal(h.toasts.length, 0);
});
