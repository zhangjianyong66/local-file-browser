const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const test = require('node:test');
const app = fs.readFileSync('web/app.js', 'utf8');
const login = fs.readFileSync('web/login.html', 'utf8').match(/<script>([\s\S]*?)<\/script>/)[1];

function harness(source = app, entry = '/') {
  const nodes = new Map();
  const pending = [];
  const events = {};
  const historyCalls = [];
  const opened = [];
  const location = {
    href: `http://localhost${entry}`, origin: 'http://localhost',
    assign(value) { this.redirect = value; }, replace(value) { this.redirect = value; },
  };
  const node = (selector) => {
    if (!nodes.has(selector)) nodes.set(selector, {
      innerHTML: '', textContent: '', classList: { add() {}, remove() {}, toggle() {} },
      addEventListener(type, listener) { this[type] = listener; },
      querySelectorAll() { return []; }, setAttribute() {}, removeAttribute() {}, focus() {},
    });
    return nodes.get(selector);
  };
  const context = vm.createContext({
    URL, location, document: { querySelector: node },
    FormData: class { append() {} },
    window: { open(...args) { opened.push(args); return null; }, addEventListener(type, listener) { events[type] = listener; } },
    history: Object.fromEntries(['pushState', 'replaceState'].map((method) => [method, (_, __, value) => {
      historyCalls.push([method, value]); location.href = new URL(value, location.origin).href;
    }])),
    fetch: (url) => new Promise((resolve) => pending.push({ url, resolve })),
  });
  vm.runInContext(source, context);
  return { context, node, pending, events, historyCalls, location, opened,
    run: (code) => vm.runInContext(code, context),
    async respond(index, data, status = 200) {
      pending[index].resolve({ status, ok: status >= 200 && status < 300, json: async () => data });
      await new Promise(setImmediate);
    },
  };
}

test('initial URL restores special characters and clears selection', async () => {
  const path = '中文 空格#%/子目录';
  const h = harness(app, `/?${new URLSearchParams({ path })}`);
  assert.equal(new URL(h.pending[0].url, h.location.origin).searchParams.get('path'), path);
  await h.respond(0, { path, entries: [] });
  assert.equal(h.run('current'), path);
  assert.equal(h.run('selected'), '');
  assert.match(h.node('#details').innerHTML, /选择一个文件/);
  assert.deepEqual(h.historyCalls.map(([method]) => method), ['replaceState']);
});

test('successful navigation pushes history; popstate and reload replace', async () => {
  const h = harness();
  await h.respond(0, { path: '', entries: [] });
  h.run("load('projects')");
  await h.respond(1, { path: 'projects', entries: [] });
  h.run("load('projects/docs')");
  await h.respond(2, { path: 'projects/docs', entries: [] });
  h.location.href = 'http://localhost/?path=projects';
  h.events.popstate();
  await h.respond(3, { path: 'projects', entries: [] });
  h.run("load('projects')");
  await h.respond(4, { path: 'projects', entries: [] });
  assert.deepEqual(h.historyCalls.map(([method]) => method), ['replaceState', 'pushState', 'pushState', 'replaceState', 'replaceState']);
  assert.equal(h.run('current'), 'projects');
});

test('failed directory keeps target address, retry target, and root action', async () => {
  const h = harness(app, '/?path=missing');
  await h.respond(0, { error: 'missing directory' }, 404);
  assert.equal(new URL(h.location.href).searchParams.get('path'), 'missing');
  assert.equal(h.node('#upload').disabled, true);
  h.node('#retry').click();
  assert.equal(h.pending[1].url, '/api/list?path=missing');
  await h.respond(1, { error: 'missing directory' }, 404);
  h.node('#return-root').click();
  await h.respond(2, { path: '', entries: [] });
  assert.equal(h.location.href, 'http://localhost/');
});

test('older directory and file requests cannot overwrite latest navigation', async () => {
  const h = harness();
  await h.respond(0, { path: '', entries: [] });
  h.run("show('old.txt', 'old.txt')");
  h.run("load('old')");
  h.run("load('latest')");
  await h.respond(3, { path: 'latest', entries: [] });
  await h.respond(2, { path: 'old', entries: [] });
  await h.respond(1, { name: 'old.txt', path: 'old.txt', preview: 'image' });
  assert.equal(h.run('current'), 'latest');
  assert.match(h.node('#details').innerHTML, /选择一个文件/);
  assert.equal(h.node('#page-title').textContent, 'latest');
});

test('API 401 carries pending target rather than previously loaded directory', async () => {
  const h = harness();
  await h.respond(0, { path: '', entries: [] });
  h.run("load('projects/docs')");
  await h.respond(1, {}, 401);
  const redirect = new URL(h.location.redirect, h.location.origin);
  assert.equal(redirect.pathname, '/login.html');
  assert.equal(new URL(redirect.searchParams.get('return'), h.location.origin).searchParams.get('path'), 'projects/docs');
});

test('stale directory and preview 401 responses do not interrupt current navigation', async () => {
  const h = harness();
  await h.respond(0, { path: '', entries: [] });
  h.run("show('old.txt', 'old.txt')");
  h.run("load('old')");
  h.run("load('latest')");
  await h.respond(3, { path: 'latest', entries: [] });
  await h.respond(2, {}, 401);
  await h.respond(1, {}, 401);
  assert.equal(h.location.redirect, undefined);
  assert.equal(h.run('current'), 'latest');
  assert.equal(h.node('#message').textContent, '');
});

test('upload refresh completion cannot add messages to a newer directory', async () => {
  const h = harness(app, '/?path=projects');
  await h.respond(0, { path: 'projects', entries: [] });
  const input = h.node('#upload');
  input.files = [{ name: 'new.txt' }];
  const uploading = input.change({ target: input });
  await h.respond(1, {});
  assert.equal(h.pending[2].url, '/api/list?path=projects');
  h.run("load('other')");
  await h.respond(3, { path: 'other', entries: [] });
  await h.respond(2, { path: 'projects', entries: [] });
  await uploading;
  assert.equal(h.run('current'), 'other');
  assert.equal(h.node('#message').textContent, '');
});

test('upload completion cannot navigate away from a newer directory', async () => {
  const h = harness(app, '/?path=projects');
  await h.respond(0, { path: 'projects', entries: [] });
  const input = h.node('#upload');
  input.files = [{ name: 'new.txt' }];
  const uploading = input.change({ target: input });
  assert.equal(h.pending[1].url, '/api/upload?path=projects');
  h.run("load('other')");
  await h.respond(2, { path: 'other', entries: [] });
  await h.respond(1, {});
  await uploading;
  assert.equal(h.run('current'), 'other');
  assert.equal(h.pending.length, 3);
  assert.equal(input.disabled, false);
  assert.equal(h.node('#message').textContent, '');
});

test('login only returns to local directory entries and rebuilds path', () => {
  for (const value of ['https://evil.test/', '//evil.test/', '/\\evil.test/', '/api/download?path=secret', '/markdown.html?path=x']) {
    const h = harness(login, `/login.html?${new URLSearchParams({ return: value })}`);
    assert.equal(h.run('returnDirectoryURL()'), '/');
  }
  const path = '中文 空格#%/子目录';
  const value = `/index.html?${new URLSearchParams({ path, unused: 'ignored' })}#fragment`;
  const h = harness(login, `/login.html?${new URLSearchParams({ return: value })}`);
  const returned = new URL(h.run('returnDirectoryURL()'), h.location.origin);
  assert.equal(returned.pathname, '/');
  assert.equal(returned.searchParams.get('path'), path);
  assert.equal(returned.searchParams.has('unused'), false);
  assert.equal(returned.hash, '');
});


test('HTML and Markdown clicks open isolated new windows with encoded paths', async () => {
  for (const [name, page] of [['PAGE.HTML', 'html'], ['page.HtM', 'html'], ['readme.MD', 'markdown']]) {
    const h = harness();
    await h.respond(0, { path: '', entries: [] });
    const path = `中文 空格#%?/子目录/${name}`;
    const row = {
      dataset: { path, dir: 'false' },
      querySelector: () => ({ textContent: name }),
      addEventListener(_, listener) { this.click = listener; },
    };
    h.node('#listing').querySelectorAll = () => [row];
    h.run('renderList([])');
    // A nonempty list registers the actual row click handler.
    h.run(`renderList([{ path: ${JSON.stringify(path)}, name: ${JSON.stringify(name)}, dir: false, size: 1 }])`);
    row.click({ preventDefault() {} });
    assert.equal(h.opened.length, 1);
    const [address, target, features] = h.opened[0];
    const url = new URL(address, h.location.origin);
    assert.equal(url.pathname, `/${page}.html`);
    assert.equal(url.searchParams.get('path'), path);
    assert.equal(target, '_blank');
    assert.equal(features, 'noopener,noreferrer');
    assert.equal(h.pending.length, 1, 'click must not fetch text preview');
    assert.equal(h.node('#message').textContent, '', 'noopener returning null is not a blocked popup');
  }
});
