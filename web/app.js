let current = '';
let selected = '';
let selectedName = '';
let isUploading = false;
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
async function api(url, options) {
  const response = await fetch(url, options);
  if (response.status === 401) { location = '/login.html'; throw new Error('登录已过期'); }
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data.error || '请求失败，请稍后重试。');
  return data;
}
function setMessage(text, kind = '') {
  const node = q('#message');
  node.textContent = text;
  node.className = `message ${kind}`.trim();
}
function breadcrumbs() {
  let built = '';
  const parts = current ? current.split('/') : [];
  q('#crumb').innerHTML = `<a href="#" data-path="">根目录</a>${parts.map((part) => { built = built ? `${built}/${part}` : part; return ` <span aria-hidden="true">/</span> <a href="#" data-path="${esc(built)}">${esc(part)}</a>`; }).join('')}`;
  q('#crumb').querySelectorAll('a').forEach((link) => link.addEventListener('click', (event) => { event.preventDefault(); load(link.dataset.path); }));
  q('#page-title').textContent = current ? current.split('/').pop() : '根目录';
}
function showListLoading() {
  q('#listing').innerHTML = '<div class="loading-state"><span class="spinner" aria-hidden="true"></span><span>正在读取目录…</span></div>';
}
function renderList(entries) {
  q('#entry-count').textContent = `${entries.length} 项`;
  if (!entries.length) { q('#listing').innerHTML = '<div class="empty-list"><strong>此目录为空</strong><span>可以从右上角上传文件。</span></div>'; return; }
  q('#listing').innerHTML = entries.map((entry) => `<a class="file-row ${selected === entry.path ? 'is-selected' : ''}" href="#" data-path="${esc(entry.path)}" data-dir="${entry.dir}" aria-current="${selected === entry.path ? 'true' : 'false'}"><span class="file-name">${icon(entry.dir ? 'folder' : 'file')}<span class="file-label">${esc(entry.name)}</span></span><span class="file-size">${entry.dir ? '—' : fmtBytes(entry.size)}</span><span class="file-date">${fmtDate(entry.modified)}</span><span class="file-mime">${esc(entry.dir ? '目录' : entry.mime)}</span></a>`).join('');
  q('#listing').querySelectorAll('.file-row').forEach((row) => row.addEventListener('click', (event) => {
    event.preventDefault();
    if (row.dataset.dir === 'true') { load(row.dataset.path); return; }
    const name = row.querySelector('.file-label').textContent;
    if (name.toLowerCase().endsWith('.md')) {
      const previewWindow = window.open(`/markdown.html?path=${enc(row.dataset.path)}`, '_blank', 'noopener,noreferrer');
      if (!previewWindow) setMessage('浏览器阻止了新窗口，请允许弹出窗口后重试。', 'is-error');
      return;
    }
    show(row.dataset.path, name);
  }));
}
async function load(path = '') {
  showListLoading();
  q('#workspace').classList.remove('mobile-detail');
  q('#details-panel').classList.remove('mobile-active');
  try {
    const data = await api(`/api/list?path=${enc(path)}`);
    current = data.path;
    selected = '';
    breadcrumbs();
    renderList(data.entries);
    q('#details').innerHTML = '<div class="details-empty"><div class="empty-icon" aria-hidden="true">◌</div><h2 id="details-title">选择一个文件</h2><p>从左侧目录选择文件，在这里查看信息和预览。</p></div>';
    setMessage('');
  } catch (error) { q('#listing').innerHTML = `<div class="error-state"><strong>目录读取失败</strong><span>${esc(error.message)}</span><button class="button button-secondary" type="button" id="retry">重试</button></div>`; q('#retry')?.addEventListener('click', () => load(current)); setMessage(error.message, 'is-error'); }
}
async function show(path, name) {
  selected = path; selectedName = name;
  q('#workspace').classList.add('mobile-detail');
  q('#listing').querySelectorAll('.file-row').forEach((row) => { const active = row.dataset.path === path; row.classList.toggle('is-selected', active); row.setAttribute('aria-current', active ? 'true' : 'false'); });
  q('#details-panel').classList.add('mobile-active');
  q('#details').innerHTML = '<div class="loading-state details-loading"><span class="spinner" aria-hidden="true"></span><span>正在加载预览…</span></div>';
  try {
    const data = await api(`/api/info?path=${enc(path)}`);
    let html = `<div class="detail-header"><button class="back-button" id="back-to-list" type="button">${icon('back')}<span>返回目录</span></button><div><p class="eyebrow">文件详情</p><h2 id="details-title">${esc(data.name)}</h2></div></div><div class="file-meta"><div><span>位置</span><strong>${esc(data.path || '根目录')}</strong></div><div><span>大小</span><strong>${fmtBytes(data.size)}</strong></div><div><span>修改时间</span><strong>${fmtDate(data.modified)}</strong></div><div><span>类型</span><strong>${esc(data.mime)}</strong></div></div><a class="button button-secondary download-button" href="/api/download?path=${enc(path)}">↓ <span>下载文件</span></a><div class="preview-area">`;
    if (data.preview === 'text' || data.preview === 'markdown') {
      const preview = await api(`/api/preview?path=${enc(path)}`);
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
  } catch (error) { q('#details').innerHTML = `<div class="error-state"><strong>文件加载失败</strong><span>${esc(error.message)}</span><button class="button button-secondary" type="button" id="retry-file">重试</button></div>`; q('#retry-file')?.addEventListener('click', () => show(path, selectedName)); setMessage(error.message, 'is-error'); }
}
q('#upload').addEventListener('change', async (event) => {
  const file = event.target.files[0];
  if (!file || isUploading) return;
  isUploading = true;
  const label = q('.upload-button');
  event.target.disabled = true;
  label.classList.add('is-loading');
  label.setAttribute('aria-disabled', 'true');
  setMessage(`正在上传 ${file.name}…`, 'is-loading');
  const body = new FormData(); body.append('file', file);
  try { await api(`/api/upload?path=${enc(current)}`, {method:'POST', body}); setMessage(`“${file.name}”上传成功`, 'is-success'); event.target.value = ''; await load(current); }
  catch (error) { setMessage(error.message.includes('already exists') ? '上传失败：同名文件已存在，请更换文件名。' : `上传失败：${error.message}`, 'is-error'); }
  finally { isUploading = false; event.target.disabled = false; label.classList.remove('is-loading'); label.removeAttribute('aria-disabled'); }
});
q('#logout').addEventListener('click', async () => { q('#logout').disabled = true; try { await api('/api/logout', {method:'POST'}); location = '/login.html'; } catch (error) { setMessage(error.message, 'is-error'); q('#logout').disabled = false; } });
load();
