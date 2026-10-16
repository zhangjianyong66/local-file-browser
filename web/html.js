const path = new URL(location.href).searchParams.get('path') || '';
const status = document.querySelector('#status');
const page = document.querySelector('#page');
document.querySelector('#title').textContent = path.split('/').pop() || '网页预览';
document.title = `${path.split('/').pop() || 'HTML'} · 网页预览`;
document.querySelector('#download').href = `/api/download?path=${encodeURIComponent(path)}`;
(async () => {
  try {
    const response = await fetch(`/api/html-preview?path=${encodeURIComponent(path)}`);
    const data = await response.json();
    if (response.status === 401) { location.assign('/login.html'); return; }
    if (!response.ok) throw new Error(data.error || '网页加载失败');
    page.src = data.url;
    page.hidden = false;
    status.hidden = true;
  } catch (error) { status.textContent = `网页加载失败：${error.message}`; }
})();
