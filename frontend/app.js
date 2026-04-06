const API_BASE = '/api';
let autoRefreshTimer = null;

// --- File List ---
async function loadFiles() {
    const fileList = document.getElementById('fileList');
    try {
        const res = await fetch(`${API_BASE}/files`);
        const files = await res.json();

        if (!files || files.length === 0) {
            fileList.innerHTML = '<p class="empty-message">尚無檔案</p>';
            return;
        }

        fileList.innerHTML = files.map(file => {
            const statusClass = file.uploading ? 'status-uploading' : 'status-ready';
            const statusText = file.uploading ? '⏳ 上傳中...' : '✅ 可下載';
            const downloadBtn = file.uploading
                ? `<button class="btn btn-download btn-disabled" disabled title="檔案上傳中，請稍候">⬇ 上傳中</button>`
                : `<a class="btn btn-download" href="${API_BASE}/download/${encodeURIComponent(file.name)}" download>⬇ 下載</a>`;
            const metaParts = [formatSize(file.size), formatTime(file.modTime)];
            if (file.uploaderIp) metaParts.push(`上傳者: ${escapeHtml(file.uploaderIp)}`);
            if (file.downloadCount > 0) metaParts.push(`下載: ${file.downloadCount} 次`);

            return `
            <div class="file-item ${file.uploading ? 'file-uploading' : ''}">
                <div class="file-info">
                    <div class="file-name-row">
                        <span class="file-name" title="${escapeHtml(file.name)}">${escapeHtml(file.name)}</span>
                        <span class="file-status ${statusClass}">${statusText}</span>
                    </div>
                    <span class="file-meta">${metaParts.join(' · ')}</span>
                </div>
                <div class="file-actions">
                    ${downloadBtn}
                    <button class="btn btn-delete" onclick="deleteFile('${escapeJs(file.name)}')" ${file.uploading ? 'disabled' : ''}>🗑 刪除</button>
                </div>
            </div>`;
        }).join('');
    } catch (err) {
        fileList.innerHTML = '<p class="empty-message">無法載入檔案列表</p>';
        console.error('Load files error:', err);
    }
}

// Auto-refresh every 5 seconds
function startAutoRefresh() {
    stopAutoRefresh();
    autoRefreshTimer = setInterval(loadFiles, 5000);
}

function stopAutoRefresh() {
    if (autoRefreshTimer) {
        clearInterval(autoRefreshTimer);
        autoRefreshTimer = null;
    }
}

// --- Upload ---
function uploadFiles(files) {
    if (!files || files.length === 0) return;

    const formData = new FormData();
    for (const file of files) {
        formData.append('files', file);
    }

    const xhr = new XMLHttpRequest();
    const progressContainer = document.getElementById('uploadProgress');
    const progressFill = document.getElementById('progressFill');
    const uploadPercent = document.getElementById('uploadPercent');
    const uploadFileName = document.getElementById('uploadFileName');
    const uploadSpeed = document.getElementById('uploadSpeed');
    const uploadSize = document.getElementById('uploadSize');

    const totalSize = Array.from(files).reduce((sum, f) => sum + f.size, 0);
    const names = Array.from(files).map(f => f.name).join(', ');
    uploadFileName.textContent = files.length === 1 ? names : `${files.length} 個檔案`;

    progressContainer.hidden = false;
    progressFill.style.width = '0%';

    let startTime = Date.now();

    xhr.upload.onprogress = (e) => {
        if (!e.lengthComputable) return;
        const percent = Math.round((e.loaded / e.total) * 100);
        progressFill.style.width = percent + '%';
        uploadPercent.textContent = percent + '%';

        const elapsed = (Date.now() - startTime) / 1000;
        const speed = e.loaded / elapsed;
        uploadSpeed.textContent = formatSize(speed) + '/s';
        uploadSize.textContent = `${formatSize(e.loaded)} / ${formatSize(e.total)}`;
    };

    xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
            progressFill.style.width = '100%';
            uploadPercent.textContent = '完成!';
            loadFiles();
            setTimeout(() => {
                progressContainer.hidden = true;
            }, 2000);
        } else {
            uploadPercent.textContent = '上傳失敗';
            console.error('Upload failed:', xhr.statusText);
        }
    };

    xhr.onerror = () => {
        uploadPercent.textContent = '上傳錯誤';
        console.error('Upload error');
    };

    xhr.open('POST', `${API_BASE}/upload`);
    xhr.send(formData);
}

// --- Delete ---
async function deleteFile(filename) {
    if (!confirm(`確定要刪除 "${filename}"？`)) return;

    try {
        const res = await fetch(`${API_BASE}/files/${encodeURIComponent(filename)}`, {
            method: 'DELETE'
        });
        if (res.ok) {
            loadFiles();
        } else {
            alert('刪除失敗');
        }
    } catch (err) {
        alert('刪除錯誤');
        console.error('Delete error:', err);
    }
}

// --- Helpers ---
function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i];
}

function formatTime(isoString) {
    const d = new Date(isoString);
    return d.toLocaleString('zh-TW');
}

function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

function escapeJs(str) {
    return str.replace(/\\/g, '\\\\').replace(/'/g, "\\'");
}

// --- Drag & Drop ---
const dropZone = document.getElementById('dropZone');
const fileInput = document.getElementById('fileInput');

dropZone.addEventListener('click', () => fileInput.click());

fileInput.addEventListener('change', (e) => {
    uploadFiles(e.target.files);
    fileInput.value = '';
});

dropZone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropZone.classList.add('dragover');
});

dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('dragover');
});

dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropZone.classList.remove('dragover');
    uploadFiles(e.dataTransfer.files);
});

// --- Init ---
loadFiles();
startAutoRefresh();
