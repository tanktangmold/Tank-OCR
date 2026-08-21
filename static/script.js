// Client-side script for Tank-OCR

function t(key, vars) {
    return window.TankOcrI18n ? window.TankOcrI18n.t(key, vars) : key;
}

// Global states
let currentFile = null;
let currentOcrResult = null; // Store current OCR JSON response
let currentImageSrc = null; // Store base64 DataURL of current image
let currentScale = 1.0;
let resizeListener = null;
let lastEngineStatus = null; // { kind: 'ready'|'failed'|'offline', engine? }
let lastResultMeta = null; // { isPdf, filename, pagesCount }

// Selectors
const dropZone = document.getElementById('drop-zone');
const fileInput = document.getElementById('file-input');
const loadingZone = document.getElementById('loading-zone');
const loadingStatus = document.getElementById('loading-status');
const progressFill = document.getElementById('progress-fill');
const resultViewer = document.getElementById('result-viewer');

const sourceImage = document.getElementById('source-image');
const overlayWrapper = document.getElementById('overlay-wrapper');
const boxOverlay = document.getElementById('box-overlay');
const pdfFileInfo = document.getElementById('pdf-file-info');
const pdfPagesCount = document.getElementById('pdf-pages-count');
const fileTitle = document.getElementById('file-title');

const outputText = document.getElementById('output-text');
const elapsedTime = document.getElementById('elapsed-time');
const copyBtn = document.getElementById('copy-btn');
const downloadJsonBtn = document.getElementById('download-json-btn');
const closeBtn = document.getElementById('close-btn');

const zoomIn = document.getElementById('zoom-in');
const zoomOut = document.getElementById('zoom-out');
const resetZoom = document.getElementById('reset-zoom');

const pdfPagesInput = document.getElementById('pdf-pages');
const makeSearchableCheckbox = document.getElementById('make-searchable');
const lightModeCheckbox = document.getElementById('light-mode');

const engineStatus = document.getElementById('engine-status');
const historyContainer = document.getElementById('history-container');
const clearHistoryBtn = document.getElementById('clear-history-btn');
const ocrTooltip = document.getElementById('ocr-tooltip');

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    if (window.TankOcrI18n) {
        window.TankOcrI18n.apply();
    }
    checkEngineStatus();
    loadHistory();
    setupEventListeners();
});

document.addEventListener('tank-ocr:langchange', () => {
    refreshDynamicLabels();
    loadHistory();
});

function refreshDynamicLabels() {
    applyEngineStatusText();
    if (lastResultMeta && !resultViewer.classList.contains('hidden')) {
        fileTitle.textContent = t('previewWithFile', { filename: lastResultMeta.filename });
        if (lastResultMeta.isPdf) {
            pdfPagesCount.textContent = t('pdfPagesDone', { count: lastResultMeta.pagesCount });
        }
    }
    const copyLabel = copyBtn.querySelector('.btn-text');
    if (copyLabel && copyLabel.dataset.i18n === 'copyText') {
        copyLabel.textContent = t('copyText');
    }
}

function applyEngineStatusText() {
    if (!lastEngineStatus) return;
    const text = engineStatus.querySelector('.status-text');
    if (!text) return;
    if (lastEngineStatus.kind === 'ready') {
        text.textContent = t('engineReady', { engine: lastEngineStatus.engine });
    } else if (lastEngineStatus.kind === 'failed') {
        text.textContent = t('engineFailed');
    } else {
        text.textContent = t('engineOffline');
    }
}

// Check Engine Status
async function checkEngineStatus() {
    try {
        const res = await fetch('/api/status');
        const data = await res.json();
        
        const dot = engineStatus.querySelector('.status-dot');
        const details = engineStatus.querySelector('.status-details');
        
        dot.className = 'status-dot';
        
        if (data.status === 'ready') {
            dot.classList.add('success');
            lastEngineStatus = { kind: 'ready', engine: data.engine };
            applyEngineStatusText();
            document.getElementById('engine-version').textContent = data.version;
            document.getElementById('engine-max-dim').textContent = data.max_dimension;
            details.classList.remove('hidden');
        } else {
            dot.classList.add('error');
            lastEngineStatus = { kind: 'failed' };
            applyEngineStatusText();
            console.error('OCR Engine error:', data.detail);
        }
    } catch (e) {
        console.error('Failed to get status:', e);
        const dot = engineStatus.querySelector('.status-dot');
        dot.className = 'status-dot error';
        lastEngineStatus = { kind: 'offline' };
        applyEngineStatusText();
    }
}

// Setup Event Listeners
function setupEventListeners() {
    document.querySelectorAll('.lang-btn').forEach((btn) => {
        btn.addEventListener('click', () => {
            const lang = btn.getAttribute('data-lang');
            if (window.TankOcrI18n) {
                window.TankOcrI18n.setLang(lang);
            }
        });
    });

    // Click dropzone triggers file picker
    dropZone.addEventListener('click', () => fileInput.click());
    fileInput.addEventListener('change', (e) => {
        if (e.target.files.length > 0) {
            handleFileSelection(e.target.files[0]);
        }
    });

    // Drag-and-drop events
    ['dragenter', 'dragover'].forEach(eventName => {
        dropZone.addEventListener(eventName, (e) => {
            e.preventDefault();
            e.stopPropagation();
            dropZone.classList.add('dragover');
        }, false);
    });

    ['dragleave', 'drop'].forEach(eventName => {
        dropZone.addEventListener(eventName, (e) => {
            e.preventDefault();
            e.stopPropagation();
            dropZone.classList.remove('dragover');
        }, false);
    });

    dropZone.addEventListener('drop', (e) => {
        const dt = e.dataTransfer;
        const files = dt.files;
        if (files.length > 0) {
            handleFileSelection(files[0]);
        }
    }, false);

    // Zoom Controls
    zoomIn.addEventListener('click', () => adjustZoom(0.15));
    zoomOut.addEventListener('click', () => adjustZoom(-0.15));
    resetZoom.addEventListener('click', () => resetZoomFactor());

    // Action Buttons
    copyBtn.addEventListener('click', copyTextToClipboard);
    downloadJsonBtn.addEventListener('click', downloadOcrJson);
    closeBtn.addEventListener('click', resetWorkspace);
    clearHistoryBtn.addEventListener('click', clearAllHistory);

    // Tooltip mouse movement tracking
    document.addEventListener('mousemove', (e) => {
        if (!ocrTooltip.classList.contains('hidden')) {
            // Keep tooltip close to cursor
            ocrTooltip.style.left = (e.clientX + 12) + 'px';
            ocrTooltip.style.top = (e.clientY + 12) + 'px';
        }
    });
}

// Handle File Selection
function handleFileSelection(file) {
    currentFile = file;
    
    // Check file type
    const isPdf = file.type === 'application/pdf' || file.name.toLowerCase().endsWith('.pdf');
    const isImage = file.type.startsWith('image/') || /\.(png|jpe?g|webp|bmp|gif|tiff?)$/i.test(file.name);
    
    if (!isPdf && !isImage) {
        alert(t('unsupportedFormat'));
        return;
    }
    const maxBytes = isPdf ? 50 * 1024 * 1024 : 32 * 1024 * 1024;
    if (file.size > maxBytes) {
        alert(t('fileTooLarge', {
            kind: isPdf ? t('kindPdf') : t('kindImage'),
            max: isPdf ? 50 : 32,
        }));
        return;
    }

    if (isImage) {
        // Read image file locally to display preview
        const reader = new FileReader();
        reader.onload = (e) => {
            currentImageSrc = e.target.result;
            startOcrProcess(file, isPdf);
        };
        reader.readAsDataURL(file);
    } else {
        currentImageSrc = null;
        startOcrProcess(file, isPdf);
    }
}

// Start OCR Process
async function startOcrProcess(file, isPdf) {
    // UI Transitions
    dropZone.classList.remove('active');
    loadingZone.classList.remove('hidden');
    resultViewer.classList.add('hidden');
    
    // Reset loader progress fill animation
    progressFill.style.animation = 'progressDummy 10s infinite linear';
    
    const pages = pdfPagesInput.value;
    const makeSearchable = makeSearchableCheckbox.checked;
    const lightMode = lightModeCheckbox.checked;
    
    loadingStatus.textContent = isPdf
        ? (makeSearchable ? t('loadingPdfSearchable') : t('loadingPdfText'))
        : t('loadingImage');

    // Prepare multipart data
    const formData = new FormData();
    formData.append('file', file);
    formData.append('light_mode', lightMode);
    
    let url = '/api/ocr';
    if (isPdf) {
        url = '/api/ocr-pdf';
        formData.append('pages', pages);
        formData.append('make_searchable', makeSearchable);
    }

    try {
        const response = await fetch(url, {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            const contentType = response.headers.get('content-type') || '';
            let message = t('ocrFailedHttp', { status: response.status });
            if (contentType.includes('application/json')) {
                const errData = await response.json();
                message = errData.detail || message;
            } else {
                const errorText = await response.text();
                if (errorText.trim()) message = errorText.trim();
            }
            throw new Error(message);
        }

        // If user requested searchable PDF, it will return a blob file download
        if (isPdf && makeSearchable) {
            const blob = await response.blob();
            const downloadUrl = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = downloadUrl;
            
            // Get filename from header or use default
            const contentDisp = response.headers.get('Content-Disposition');
            let downloadName = `${file.name.replace(/\.[^/.]+$/, '')}_searchable.pdf`;
            if (contentDisp && contentDisp.includes('filename=')) {
                const match = contentDisp.match(/filename="?([^"]+)"?/);
                if (match && match[1]) downloadName = match[1];
            }
            
            a.download = downloadName;
            document.body.appendChild(a);
            a.click();
            a.remove();
            window.URL.revokeObjectURL(downloadUrl);
            
            // Re-render dropzone
            resetWorkspace();
            loadHistory();
            alert(t('searchablePdfDone'));
            return;
        }

        // Else we parse JSON
        const data = await response.json();
        currentOcrResult = data;
        
        // Show results
        renderOcrResults(data, isPdf);
        
        // Cache result in localStorage for history details reloading
        // Note: The actual entry will be refreshed from history API which returns timestamps
        // But we map by filename + timestamp / temp cache. Let's load history from server to get correct ID.
        await loadHistory();
        
        // Get the newest entry ID we just added in backend
        const histRes = await fetch('/api/history');
        const historyList = await histRes.json();
        if (historyList.length > 0) {
            const newestId = historyList[0].id;
            try {
                localStorage.setItem(`ocr_result_${newestId}`, JSON.stringify(data));
                if (currentImageSrc) {
                    localStorage.setItem(`ocr_image_${newestId}`, currentImageSrc);
                }
            } catch (cacheError) {
                console.warn('OCR succeeded, but browser history cache is full:', cacheError);
                localStorage.removeItem(`ocr_image_${newestId}`);
            }
        }
        
    } catch (error) {
        alert(t('ocrError', { message: error.message }));
        resetWorkspace();
    }
}

// Render Results
function renderOcrResults(data, isPdf) {
    loadingZone.classList.add('hidden');
    resultViewer.classList.remove('hidden');
    
    const pagesCount = data.result && data.result.pages ? data.result.pages.length : 0;
    lastResultMeta = { isPdf, filename: data.filename, pagesCount };

    // Set headers
    fileTitle.textContent = t('previewWithFile', { filename: data.filename });
    elapsedTime.textContent = `${data.duration_seconds.toFixed(2)}s`;
    
    // Set output text
    outputText.value = data.text;
    
    // Reset zoom
    resetZoomFactor();

    if (isPdf) {
        sourceImage.classList.add('hidden');
        pdfFileInfo.classList.remove('hidden');
        overlayWrapper.classList.add('hidden');
        
        pdfPagesCount.textContent = t('pdfPagesDone', { count: pagesCount });
    } else {
        sourceImage.classList.remove('hidden');
        pdfFileInfo.classList.add('hidden');
        overlayWrapper.classList.remove('hidden');
        
        // Render image
        sourceImage.src = currentImageSrc;
        
        // Render bounding boxes when image finishes loading
        sourceImage.onload = () => {
            drawBoxes(data.result);
        };
        
        // Handle window resizing dynamically to keep boxes aligned
        if (resizeListener) {
            window.removeEventListener('resize', resizeListener);
        }
        resizeListener = () => {
            drawBoxes(data.result);
        };
        window.addEventListener('resize', resizeListener);
    }
}

// Draw Overlay Bounding Boxes
function drawBoxes(result) {
    // Clear old boxes
    boxOverlay.innerHTML = '';
    
    if (!result || !result.pages || result.pages.length === 0) return;
    
    const page = result.pages[0];
    const ocrWidth = page.width;
    const ocrHeight = page.height;
    
    // Get actual displayed dimensions of the preview image
    const dispWidth = sourceImage.clientWidth;
    const dispHeight = sourceImage.clientHeight;
    
    if (dispWidth === 0 || dispHeight === 0) {
        // Wait a tiny bit and retry if layout is not fully painted yet
        setTimeout(() => drawBoxes(result), 100);
        return;
    }

    const scaleX = dispWidth / ocrWidth;
    const scaleY = dispHeight / ocrHeight;
    
    // Recursively extract all words
    const words = [];
    page.blocks.forEach(block => {
        block.lines.forEach(line => {
            line.words.forEach(word => {
                if (word.bounding_box) {
                    words.push(word);
                }
            });
        });
    });

    // Draw divs
    words.forEach(word => {
        const box = word.bounding_box;
        const boxDiv = document.createElement('div');
        boxDiv.className = 'ocr-box';
        
        // Layout positioning (percentages are safer, but pixel coords relative to container work since container is styled display:inline-block)
        boxDiv.style.left = (box.x * scaleX) + 'px';
        boxDiv.style.top = (box.y * scaleY) + 'px';
        boxDiv.style.width = (box.width * scaleX) + 'px';
        boxDiv.style.height = (box.height * scaleY) + 'px';
        
        if (box.angle && Math.abs(box.angle) > 0.1) {
            boxDiv.style.transform = `rotate(${box.angle}deg)`;
        }
        
        // Setup tooltip hover events
        boxDiv.addEventListener('mouseenter', () => {
            const conf = word.confidence !== null && word.confidence !== undefined
                ? t('confidence', { pct: Math.round(word.confidence * 100) })
                : '';
            ocrTooltip.innerHTML = `<strong>${escapeHtml(word.text)}</strong>${conf ? `<span class="confidence-score">${conf}</span>` : ''}`;
            ocrTooltip.classList.remove('hidden');
        });
        
        boxDiv.addEventListener('mouseleave', () => {
            ocrTooltip.classList.add('hidden');
        });
        
        boxOverlay.appendChild(boxDiv);
    });
}

// Adjust Zoom
function adjustZoom(amount) {
    currentScale += amount;
    // Limit bounds
    currentScale = Math.max(0.5, Math.min(3.0, currentScale));
    applyZoom();
}

function resetZoomFactor() {
    currentScale = 1.0;
    applyZoom();
}

function applyZoom() {
    overlayWrapper.style.transform = `scale(${currentScale})`;
}

// Actions
function copyTextToClipboard() {
    const text = outputText.value;
    if (!text.trim()) return;
    
    navigator.clipboard.writeText(text).then(() => {
        const label = copyBtn.querySelector('.btn-text');
        label.textContent = t('copied');
        copyBtn.style.background = 'linear-gradient(135deg, #10b981, #059669)'; // Success green
        
        setTimeout(() => {
            label.textContent = t('copyText');
            copyBtn.style.background = ''; // Revert to stylesheet default
        }, 1500);
    }).catch(err => {
        console.error('Failed to copy text:', err);
    });
}

function downloadOcrJson() {
    if (!currentOcrResult) return;
    const jsonStr = JSON.stringify(currentOcrResult.result, null, 2);
    const blob = new Blob([jsonStr], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${currentOcrResult.filename}_ocr.json`;
    a.click();
    URL.revokeObjectURL(url);
}

function resetWorkspace() {
    currentFile = null;
    currentOcrResult = null;
    currentImageSrc = null;
    lastResultMeta = null;
    boxOverlay.innerHTML = '';
    outputText.value = '';
    
    if (resizeListener) {
        window.removeEventListener('resize', resizeListener);
        resizeListener = null;
    }
    
    resultViewer.classList.add('hidden');
    loadingZone.classList.add('hidden');
    dropZone.classList.add('active');
}

// History Handling
async function loadHistory() {
    try {
        const res = await fetch('/api/history');
        const history = await res.json();
        
        historyContainer.innerHTML = '';
        
        if (history.length === 0) {
            historyContainer.innerHTML = `<div class="empty-state-small">${escapeHtml(t('emptyHistory'))}</div>`;
            return;
        }

        history.forEach(item => {
            const itemDiv = document.createElement('div');
            itemDiv.className = 'history-item';
            itemDiv.dataset.id = item.id;
            
            const isPdf = item.type.startsWith('pdf');
            const sizeMb = (item.size / (1024 * 1024)).toFixed(2);
            const typeLabel = isPdf ? t('historyTypePdf') : t('historyTypeImage');
            
            itemDiv.innerHTML = `
                <div class="history-item-top">
                    <span class="history-item-name" title="${escapeHtml(item.filename)}">${escapeHtml(item.filename)}</span>
                    <span class="history-item-type ${isPdf ? 'pdf' : ''}">${escapeHtml(typeLabel)}</span>
                </div>
                <div class="history-item-meta">
                    <span>${item.timestamp}</span>
                    <span>${sizeMb} MB | ${item.duration}s</span>
                </div>
            `;
            
            itemDiv.addEventListener('click', () => loadHistoryDetail(item.id, isPdf));
            historyContainer.appendChild(itemDiv);
        });
    } catch (e) {
        console.error('Failed to load history:', e);
    }
}

function loadHistoryDetail(id, isPdf) {
    // Find item
    const cachedOcr = localStorage.getItem(`ocr_result_${id}`);
    const cachedImage = localStorage.getItem(`ocr_image_${id}`);
    
    if (!cachedOcr) {
        alert(t('historyCacheMissing'));
        return;
    }

    // Update active state in UI
    document.querySelectorAll('.history-item').forEach(el => {
        el.classList.toggle('active', el.dataset.id === id);
    });

    currentOcrResult = JSON.parse(cachedOcr);
    currentImageSrc = cachedImage;
    
    // Switch view
    dropZone.classList.remove('active');
    loadingZone.classList.add('hidden');
    
    renderOcrResults(currentOcrResult, isPdf);
}

async function clearAllHistory() {
    if (!confirm(t('clearHistoryConfirm'))) {
        return;
    }
    try {
        await fetch('/api/history', { method: 'DELETE' });
        // Clear matching localStorage keys
        for (let i = localStorage.length - 1; i >= 0; i--) {
            const key = localStorage.key(i);
            if (key.startsWith('ocr_result_') || key.startsWith('ocr_image_')) {
                localStorage.removeItem(key);
            }
        }
        resetWorkspace();
        loadHistory();
    } catch (e) {
        console.error('Failed to clear history:', e);
    }
}

// Helpers
function escapeHtml(text) {
    const map = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return String(text).replace(/[&<>"']/g, function(m) { return map[m]; });
}
