let current = '';
let selected = '';
let selectedName = '';
let isUploading = false;
let target = new URL(location.href).searchParams.get('path') || '';
let directoryRequest = 0;
let previewRequest = 0;
let directoryReady = false;
let currentEntries = [];
let sortMode = 'modified-desc';
let searchRequest = 0;
let searchController;
let searchActive = false;
let submittedQuery = '';
function directoryURL(path) {
  const url = new URL('/', location.origin);
  if (path) url.searchParams.set('path', path);
  return url.pathname + url.search;
}
function loginURL() {
  const url = new URL('/login.html', location.origin);
  url.searchParams.set('return', directoryURL(target));
  return url.pathname + url.search;
}
const q = (selector) => document.querySelector(selector);
const enc = encodeURIComponent;
const esc = (value) => String(value).replace(/[&<>"']/g, (char) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));
const fmtBytes = (bytes) => {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
};
const fmtDate = (value) => new Date(value).toLocaleString('zh-CN', {year:'numeric', month:'short', day:'numeric', hour:'2-digit', minute:'2-digit'});
const icon = (name) => {
  const paths = {folder:'<path d="M3 6.5A1.5 1.5 0 0 1 4.5 5h4l2 2h5A1.5 1.5 0 0 1 17 8.5v6A1.5 1.5 0 0 1 15.5 16h-11A1.5 1.5 0 0 1 3 14.5z"/>', file:'<path d="M6 3h6l4 4v10H6z"/><path d="M12 3v4h4"/>', back:'<path d="m15 6-6 6 6 6"/><path d="M9 12h9"/>'};
  return `<svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths[name] || paths.file}</svg>`;
};
async function api(url, options, isCurrent = () => true) {
  const response = await fetch(url, options);
  if (response.status === 401) { if (isCurrent()) location.assign(loginURL()); throw new Error('登录已过期'); }
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data.error || '请求失败，请稍后重试。');
  return data;
}
function setMessage(text, kind = '') {
  const node = q('#message');
  node.textContent = text;
  node.className = `message ${kind}`.trim();
}
function breadcrumbs(path = current) {
  let built = '';
  const parts = path ? path.split('/') : [];
  q('#crumb').innerHTML = `<a href="#" data-path="">根目录</a>${parts.map((part) => { built = built ? `${built}/${part}` : part; return ` <span aria-hidden="true">/</span> <a href="#" data-path="${esc(built)}">${esc(part)}</a>`; }).join('')}`;
  q('#crumb').querySelectorAll('a').forEach((link) => link.addEventListener('click', (event) => { event.preventDefault(); load(link.dataset.path); }));
  q('#page-title').textContent = path ? path.split('/').pop() : '根目录';
}
function showListLoading() {
  q('#listing').innerHTML = '<div class="loading-state"><span class="spinner" aria-hidden="true"></span><span>正在读取目录…</span></div>';
}
function setSearchEnabled(enabled) {
  ['#search-query', '#search-scope', '#search-submit'].forEach((selector) => { q(selector).disabled = !enabled; });
}
function resetDetails() {
  ++previewRequest;
  selected = ''; selectedName = '';
  q('#details').innerHTML = '<div class="details-empty"><div class="empty-icon" aria-hidden="true">◌</div><h2 id="details-title">选择一个文件</h2></div>';
  q('#workspace').classList.remove('mobile-detail');
  q('#details-panel').classList.remove('mobile-active');
}
function invalidateSearch() {
  ++searchRequest;
  searchController?.abort();
  searchController = undefined;
}
function clearSearch(restore = true) {
  invalidateSearch();
  searchActive = false;
  submittedQuery = '';
  q('#search-query').value = '';
  q('#search-clear').disabled = true;
  q('#listing-title').textContent = '当前目录';
  q('#listing-panel').classList.remove('search-results');
  q('#sort-order').disabled = !directoryReady;
  if (restore && directoryReady) {
    resetDetails();
    renderList(currentEntries);
    setMessage('');
  }
}
function openFile(path, name) {
  if (/\.(md|html?)$/i.test(name)) {
    const page = /\.md$/i.test(name) ? 'markdown' : 'html';
    window.open(`/${page}.html?path=${enc(path)}`, '_blank', 'noopener,noreferrer');
    return;
  }
  show(path, name);
}
function renderSearchResults(data) {
  const incomplete = data.truncated || data.skipped > 0;
  q('#entry-count').textContent = `${data.entries.length} 项${incomplete ? ' · 部分结果' : ''}`;
  const reasons = {result_limit: '已达到结果数量上限', scan_limit: '已达到扫描数量上限', time_limit: '已达到搜索时间上限'};
  const notices = [];
  if (data.truncated) notices.push(`${reasons[data.reason] || '搜索未完成'}，请缩小搜索范围。`);
  if (data.skipped > 0) notices.push(`有 ${data.skipped} 个文件或目录无法读取，结果可能不完整。`);
  const notice = notices.length ? `<div class="search-notice" role="status">${esc(notices.join(' '))}</div>` : '';
  if (!data.entries.length) {
    q('#listing').innerHTML = `${notice}<div class="empty-list"><strong>${incomplete ? '本次搜索暂未找到匹配文件' : '没有匹配的文件'}</strong></div>`;
    return;
  }
  q('#listing').innerHTML = notice + data.entries.map((entry) => `<div class="search-result"><a class="file-row" href="#" data-path="${esc(entry.path)}" data-name="${esc(entry.name)}" aria-current="false"><span class="search-result-name"><span class="file-name">${icon('file')}<span class="file-label">${esc(entry.name)}</span></span><span class="result-path">${esc(entry.path)}</span><span class="result-meta">${fmtBytes(entry.size)} · ${fmtDate(entry.modified)}</span></span></a><button class="button button-ghost open-parent" type="button" data-parent="${esc(entry.path.split('/').slice(0, -1).join('/'))}" title="打开所在目录" aria-label="打开 ${esc(entry.name)} 所在目录">${icon('folder')}</button></div>`).join('');
  q('#listing').querySelectorAll('.file-row').forEach((row) => row.addEventListener('click', (event) => {
    event.preventDefault(); openFile(row.dataset.path, row.dataset.name);
  }));
  q('#listing').querySelectorAll('.open-parent').forEach((button) => button.addEventListener('click', () => load(button.dataset.parent)));
}
async function searchFiles(query = q('#search-query').value.trim()) {
  if (!directoryReady) return;
  if (!query) { clearSearch(); return; }
  invalidateSearch();
  const request = searchRequest;
  searchController = new AbortController();
  const controller = searchController;
  submittedQuery = query;
  searchActive = true;
  q('#search-clear').disabled = false;
  q('#sort-order').disabled = true;
  q('#listing-title').textContent = '搜索结果';
  q('#listing-panel').classList.add('search-results');
  q('#entry-count').textContent = '';
  resetDetails();
  setMessage('');
  q('#listing').innerHTML = '<div class="loading-state"><span class="spinner" aria-hidden="true"></span><span>正在搜索文件…</span></div>';
  const scope = q('#search-scope').value === 'current' ? current : '';
  const params = new URLSearchParams({q: query, path: scope});
  try {
    const data = await api(`/api/search?${params}`, {signal: controller.signal}, () => request === searchRequest);
    if (request !== searchRequest) return;
    renderSearchResults(data);
  } catch (error) {
    if (request !== searchRequest || error.name === 'AbortError') return;
    q('#listing').innerHTML = `<div class="error-state"><strong>搜索失败</strong><span>${esc(error.message)}</span><button class="button button-secondary" type="button" id="retry-search">重试</button></div>`;
    q('#retry-search').addEventListener('click', () => searchFiles(submittedQuery));
  }
}
function sortedEntries(entries) {
  const [field, order] = sortMode.split('-');
  const direction = order === 'asc' ? 1 : -1;
  const byName = (a, b) => a.name.localeCompare(b.name, 'zh-CN') || (a.name < b.name ? -1 : a.name > b.name ? 1 : 0);
  return [...entries].sort((a, b) => {
    if (a.dir !== b.dir) return a.dir ? -1 : 1;
    if (field === 'size' && a.dir) return byName(a, b);
    const comparison = field === 'name' ? byName(a, b)
      : field === 'size' ? a.size - b.size
      : new Date(a.modified).getTime() - new Date(b.modified).getTime();
    return comparison * direction || byName(a, b);
  });
}
function renderList(entries) {
  entries = sortedEntries(entries);
  q('#entry-count').textContent = `${entries.length} 项`;
  if (!entries.length) { q('#listing').innerHTML = '<div class="empty-list"><strong>此目录为空</strong><span>可以从右上角上传文件。</span></div>'; return; }
  q('#listing').innerHTML = entries.map((entry) => `<a class="file-row ${selected === entry.path ? 'is-selected' : ''}" href="#" data-path="${esc(entry.path)}" data-dir="${entry.dir}" aria-current="${selected === entry.path ? 'true' : 'false'}"><span class="file-name">${icon(entry.dir ? 'folder' : 'file')}<span class="file-label">${esc(entry.name)}</span></span><span class="file-size">${entry.dir ? '—' : fmtBytes(entry.size)}</span><span class="file-date">${fmtDate(entry.modified)}</span><span class="file-mime">${esc(entry.dir ? '目录' : entry.mime)}</span></a>`).join('');
  q('#listing').querySelectorAll('.file-row').forEach((row) => row.addEventListener('click', (event) => {
    event.preventDefault();
    if (row.dataset.dir === 'true') { load(row.dataset.path); return; }
    const name = row.querySelector('.file-label').textContent;
    openFile(row.dataset.path, name);
  }));
}
async function load(path = '', navigation = 'push') {
  clearSearch(false);
  setSearchEnabled(false);
  const request = ++directoryRequest;
  ++previewRequest;
  target = path;
  directoryReady = false;
  currentEntries = [];
  q('#sort-order').disabled = true;
  selected = ''; selectedName = '';
  q('#upload').disabled = true;
  q('#entry-count').textContent = '';
  breadcrumbs(target);
  q('#details').innerHTML = '<div class="details-empty"><div class="empty-icon" aria-hidden="true">◌</div><h2 id="details-title">选择一个文件</h2><p>从左侧目录选择文件，在这里查看信息和预览。</p></div>';
  setMessage('');
  showListLoading();
  q('#workspace').classList.remove('mobile-detail');
  q('#details-panel').classList.remove('mobile-active');
  try {
    const data = await api(`/api/list?path=${enc(path)}`, undefined, () => request === directoryRequest);
    if (request !== directoryRequest) return;
    current = data.path;
    target = current;
    const url = directoryURL(current);
    const previous = directoryURL(new URL(location.href).searchParams.get('path') || '');
    if (navigation === 'push' && previous !== url) history.pushState(null, '', url);
    else history.replaceState(null, '', url);
    directoryReady = true;
    setSearchEnabled(true);
    currentEntries = data.entries;
    q('#sort-order').disabled = false;
    q('#upload').disabled = isUploading;
    breadcrumbs();
    renderList(currentEntries);
    q('#details').innerHTML = '<div class="details-empty"><div class="empty-icon" aria-hidden="true">◌</div><h2 id="details-title">选择一个文件</h2><p>从左侧目录选择文件，在这里查看信息和预览。</p></div>';
    setMessage('');
  } catch (error) {
    if (request !== directoryRequest) return;
    history.replaceState(null, '', directoryURL(target));
    q('#listing').innerHTML = `<div class="error-state"><strong>目录读取失败</strong><span>${esc(error.message)}</span><button class="button button-secondary" type="button" id="retry">重试</button><button class="button button-secondary" type="button" id="return-root">返回根目录</button></div>`;
    q('#retry').addEventListener('click', () => load(target, 'replace'));
    q('#return-root').addEventListener('click', () => load(''));
    setMessage(error.message, 'is-error');
  }
}
async function show(path, name) {
  const request = ++previewRequest;
  selected = path; selectedName = name;
  q('#workspace').classList.add('mobile-detail');
  q('#listing').querySelectorAll('.file-row').forEach((row) => { const active = row.dataset.path === path; row.classList.toggle('is-selected', active); row.setAttribute('aria-current', active ? 'true' : 'false'); });
  q('#details-panel').classList.add('mobile-active');
  q('#details').innerHTML = '<div class="loading-state details-loading"><span class="spinner" aria-hidden="true"></span><span>正在加载预览…</span></div>';
  try {
    const data = await api(`/api/info?path=${enc(path)}`, undefined, () => request === previewRequest);
    if (request !== previewRequest) return;
    let html = `<div class="detail-header"><button class="back-button" id="back-to-list" type="button">${icon('back')}<span>${searchActive ? '返回结果' : '返回目录'}</span></button><div><p class="eyebrow">文件详情</p><h2 id="details-title">${esc(data.name)}</h2></div></div><div class="file-meta"><div><span>位置</span><strong>${esc(data.path || '根目录')}</strong></div><div><span>大小</span><strong>${fmtBytes(data.size)}</strong></div><div><span>修改时间</span><strong>${fmtDate(data.modified)}</strong></div><div><span>类型</span><strong>${esc(data.mime)}</strong></div></div><a class="button button-secondary download-button" href="/api/download?path=${enc(path)}">↓ <span>下载文件</span></a><div class="preview-area">`;
    if (data.preview === 'text' || data.preview === 'markdown') {
      const preview = await api(`/api/preview?path=${enc(path)}`, undefined, () => request === previewRequest);
      if (request !== previewRequest) return;
      if (preview.truncated) html += '<div class="preview-notice"><strong>文件超过 5 MiB</strong><span>浏览器预览有大小限制，请下载文件查看完整内容。</span></div>';
      else if (data.preview === 'markdown') html += `<div class="preview-stats">${preview.characters.toLocaleString()} 字符 · ${preview.lines.toLocaleString()} 行</div><article class="markdown-body">${preview.html}</article><details><summary>查看 Markdown 原文</summary><pre>${esc(preview.content)}</pre></details>`;
      else html += `<div class="preview-stats">${preview.characters.toLocaleString()} 字符 · ${preview.lines.toLocaleString()} 行</div><pre class="text-preview">${esc(preview.content)}</pre>`;
    } else if (data.preview === 'image') html += `<img class="media-preview" src="/api/media?path=${enc(path)}" alt="${esc(data.name)}">`;
    else if (data.preview === 'video') html += `<video class="media-preview" controls preload="metadata" src="/api/media?path=${enc(path)}"></video>`;
    else if (data.preview === 'pdf') html += `<iframe class="pdf-preview" title="${esc(data.name)}" src="/api/media?path=${enc(path)}"></iframe>`;
    else html += '<div class="preview-notice"><strong>此文件类型不支持在线预览</strong><span>可以下载文件到本地查看。</span></div>';
    q('#details').innerHTML = `${html}</div>`;
    q('#back-to-list').addEventListener('click', () => { q('#details-panel').classList.remove('mobile-active'); q('#workspace').classList.remove('mobile-detail'); });
    q('#details-panel').focus({preventScroll: true});
  } catch (error) { if (request !== previewRequest) return; q('#details').innerHTML = `<div class="error-state"><strong>文件加载失败</strong><span>${esc(error.message)}</span><button class="button button-secondary" type="button" id="retry-file">重试</button></div>`; q('#retry-file')?.addEventListener('click', () => show(path, selectedName)); setMessage(error.message, 'is-error'); }
}
q('#sort-order').addEventListener('change', (event) => {
  if (!directoryReady || searchActive) return;
  sortMode = event.target.value;
  renderList(currentEntries);
});
q('#search-form').addEventListener('submit', (event) => { event.preventDefault(); searchFiles(); });
q('#search-clear').addEventListener('click', () => { clearSearch(); q('#search-query').focus(); });
q('#search-query').addEventListener('input', () => {
  q('#search-clear').disabled = !q('#search-query').value && !searchActive;
  if (!q('#search-query').value.trim() && searchActive) clearSearch();
});
q('#search-scope').addEventListener('change', () => { if (submittedQuery) searchFiles(submittedQuery); });
q('#upload').addEventListener('change', async (event) => {
  const file = event.target.files[0];
  if (!file || isUploading || !directoryReady) return;
  const uploadPath = current;
  const request = directoryRequest;
  isUploading = true;
  const label = q('.upload-button');
  event.target.disabled = true;
  label.classList.add('is-loading');
  label.setAttribute('aria-disabled', 'true');
  setMessage(`正在上传 ${file.name}…`, 'is-loading');
  const body = new FormData(); body.append('file', file);
  try {
    await api(`/api/upload?path=${enc(uploadPath)}`, {method:'POST', body}, () => request === directoryRequest);
    event.target.value = '';
    if (request === directoryRequest) {
      const refresh = load(uploadPath, 'replace');
      const refreshRequest = directoryRequest;
      await refresh;
      if (refreshRequest === directoryRequest && directoryReady) setMessage(`“${file.name}”上传成功`, 'is-success');
    }
  }
  catch (error) { if (request === directoryRequest) setMessage(error.message.includes('already exists') ? '上传失败：同名文件已存在，请更换文件名。' : `上传失败：${error.message}`, 'is-error'); }
  finally { isUploading = false; event.target.disabled = !directoryReady; label.classList.remove('is-loading'); label.removeAttribute('aria-disabled'); }
});
q('#logout').addEventListener('click', async () => { q('#logout').disabled = true; try { await api('/api/logout', {method:'POST'}); location.assign(loginURL()); } catch (error) { setMessage(error.message, 'is-error'); q('#logout').disabled = false; } });
window.addEventListener('popstate', () => load(new URL(location.href).searchParams.get('path') || '', 'replace'));
load(target, 'replace');
