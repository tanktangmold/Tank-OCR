// Tank-OCR UI translations (zh / en)
window.TankOcrI18n = (function () {
    const STORAGE_KEY = 'tank-ocr-lang';

    const strings = {
        zh: {
            pageTitle: 'Tank-OCR - 独立本地文字提取器',
            checkingEngine: '正在检查引擎状态...',
            versionLabel: '版本:',
            maxDimLabel: '最大尺寸:',
            history: '历史记录',
            clear: '清空',
            emptyHistory: '暂无历史记录',
            footerPrivacy: '本地离线运行 • 100% 隐私安全',
            pdfPagesLabel: '页码范围 (PDF):',
            pdfPagesPlaceholder: '例如: 1-3, 5',
            pdfPagesTitle: '留空表示识别全部页面',
            makeSearchable: '生成双层可搜索 PDF',
            makeSearchableTitle: '生成双层可搜索可复制的PDF文件并下载',
            lightMode: '极速模式 (低占用)',
            lightModeTitle: '使用轻量级模型进行OCR，识别速度更快，精度略有降低',
            dropTitle: '拖放文件到这里，或点击上传',
            dropHint: '支持图片 (PNG, JPG, WebP, BMP, GIF) 和 PDF 文件',
            loadingTitle: '正在识别中，请稍候...',
            loadingDefault: '启动本地 OCR 神经网络进行图像解析...',
            previewTitle: '图像预览与文字层',
            zoomIn: '放大',
            zoomOut: '缩小',
            fit: '适应',
            pdfRecognized: 'PDF 文件已成功识别',
            pdfPagesDefault: '已成功提取全部页面的文本数据',
            resultTitle: '识别结果 (纯文本)',
            resultPlaceholder: 'OCR文字提取结果将在这里显示...',
            downloadJson: '下载 JSON',
            copyText: '复制文字',
            copied: '已复制！',
            reupload: '重新上传',
            langZh: '中文',
            langEn: 'EN',
            engineReady: '引擎已就绪 ({engine})',
            engineFailed: '引擎初始化失败',
            engineOffline: '无法连接到后端服务器',
            unsupportedFormat: '不支持的文件格式！请上传常见图片或 PDF 文件。',
            fileTooLarge: '文件过大，{kind}最大允许 {max} MB。',
            kindImage: '图片',
            kindPdf: 'PDF',
            loadingPdfSearchable: '正在通过本地 OCR 模块解析 PDF 页面... (正在生成可搜索 PDF)',
            loadingPdfText: '正在通过本地 OCR 模块解析 PDF 页面... (正在提取文字)',
            loadingImage: '正在启动本地 OCR 神经网络进行图像解析...',
            ocrFailedHttp: '识别失败 (HTTP {status})',
            searchablePdfDone: '可搜索 PDF 生成成功，已自动启动下载！',
            ocrError: 'OCR 识别出错: {message}',
            previewWithFile: '{filename} - 图像预览与文字层',
            pdfPagesDone: '已成功识别全部 {count} 页的文本数据，右侧可直接编辑、复制。',
            confidence: '置信度: {pct}%',
            historyTypeImage: '图片',
            historyTypePdf: 'PDF',
            historyCacheMissing: '无法加载该记录的详细结果，可能由于本地缓存已清除。',
            clearHistoryConfirm: '确定要清除所有 OCR 历史记录吗？这还将清空本地结果缓存。',
        },
        en: {
            pageTitle: 'Tank-OCR - Local Offline Text Extractor',
            checkingEngine: 'Checking engine status...',
            versionLabel: 'Version:',
            maxDimLabel: 'Max size:',
            history: 'History',
            clear: 'Clear',
            emptyHistory: 'No history yet',
            footerPrivacy: 'Runs offline locally • 100% private',
            pdfPagesLabel: 'PDF page range:',
            pdfPagesPlaceholder: 'e.g. 1-3, 5',
            pdfPagesTitle: 'Leave empty to process all pages',
            makeSearchable: 'Make searchable PDF',
            makeSearchableTitle: 'Generate a dual-layer searchable/copyable PDF and download it',
            lightMode: 'Fast mode (low usage)',
            lightModeTitle: 'Use the lightweight OCR model — faster, slightly lower accuracy',
            dropTitle: 'Drop a file here, or click to upload',
            dropHint: 'Images (PNG, JPG, WebP, BMP, GIF) and PDF files',
            loadingTitle: 'Recognizing, please wait...',
            loadingDefault: 'Starting local OCR neural network...',
            previewTitle: 'Image preview & text layer',
            zoomIn: 'Zoom in',
            zoomOut: 'Zoom out',
            fit: 'Fit',
            pdfRecognized: 'PDF recognized successfully',
            pdfPagesDefault: 'Text extracted from all pages',
            resultTitle: 'Result (plain text)',
            resultPlaceholder: 'OCR text will appear here...',
            downloadJson: 'Download JSON',
            copyText: 'Copy text',
            copied: 'Copied!',
            reupload: 'Upload again',
            langZh: '中文',
            langEn: 'EN',
            engineReady: 'Engine ready ({engine})',
            engineFailed: 'Engine initialization failed',
            engineOffline: 'Cannot reach the backend server',
            unsupportedFormat: 'Unsupported format. Please upload a common image or PDF.',
            fileTooLarge: 'File too large. Max {max} MB for {kind}.',
            kindImage: 'images',
            kindPdf: 'PDFs',
            loadingPdfSearchable: 'Parsing PDF pages with local OCR... (building searchable PDF)',
            loadingPdfText: 'Parsing PDF pages with local OCR... (extracting text)',
            loadingImage: 'Starting local OCR neural network...',
            ocrFailedHttp: 'OCR failed (HTTP {status})',
            searchablePdfDone: 'Searchable PDF created — download started.',
            ocrError: 'OCR error: {message}',
            previewWithFile: '{filename} — image preview & text layer',
            pdfPagesDone: 'Extracted text from {count} page(s). Edit or copy on the right.',
            confidence: 'Confidence: {pct}%',
            historyTypeImage: 'Image',
            historyTypePdf: 'PDF',
            historyCacheMissing: 'Cannot load this history item — local cache may have been cleared.',
            clearHistoryConfirm: 'Clear all OCR history? This also clears the local result cache.',
        },
    };

    function detectLang() {
        try {
            const saved = localStorage.getItem(STORAGE_KEY);
            if (saved === 'zh' || saved === 'en') return saved;
        } catch (_) { /* ignore */ }
        const nav = (navigator.language || navigator.userLanguage || 'en').toLowerCase();
        return nav.startsWith('zh') ? 'zh' : 'en';
    }

    let current = detectLang();

    function t(key, vars) {
        const table = strings[current] || strings.en;
        let text = table[key] ?? strings.en[key] ?? key;
        if (vars) {
            Object.keys(vars).forEach((name) => {
                text = text.replace(new RegExp('\\{' + name + '\\}', 'g'), String(vars[name]));
            });
        }
        return text;
    }

    function setLang(lang) {
        if (lang !== 'zh' && lang !== 'en') return;
        current = lang;
        try {
            localStorage.setItem(STORAGE_KEY, lang);
        } catch (_) { /* ignore */ }
        apply();
        document.dispatchEvent(new CustomEvent('tank-ocr:langchange', { detail: { lang } }));
    }

    function getLang() {
        return current;
    }

    function apply() {
        document.documentElement.lang = current === 'zh' ? 'zh-CN' : 'en';
        document.title = t('pageTitle');

        document.querySelectorAll('[data-i18n]').forEach((el) => {
            const key = el.getAttribute('data-i18n');
            if (!key) return;
            el.textContent = t(key);
        });

        document.querySelectorAll('[data-i18n-html]').forEach((el) => {
            const key = el.getAttribute('data-i18n-html');
            if (!key) return;
            el.innerHTML = t(key);
        });

        document.querySelectorAll('[data-i18n-placeholder]').forEach((el) => {
            const key = el.getAttribute('data-i18n-placeholder');
            if (!key) return;
            el.setAttribute('placeholder', t(key));
        });

        document.querySelectorAll('[data-i18n-title]').forEach((el) => {
            const key = el.getAttribute('data-i18n-title');
            if (!key) return;
            el.setAttribute('title', t(key));
        });

        document.querySelectorAll('.lang-btn').forEach((btn) => {
            const lang = btn.getAttribute('data-lang');
            btn.classList.toggle('active', lang === current);
            btn.setAttribute('aria-pressed', lang === current ? 'true' : 'false');
        });
    }

    return { t, setLang, getLang, apply, strings };
})();
