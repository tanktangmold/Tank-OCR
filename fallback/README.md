# 兜底组件（可选）

这个目录里的东西**平时用不到**。主引擎（Chrome Screen AI）和 PDF 渲染
（Poppler 的 `pdftoppm`）都正常时，Go 服务不会碰这里。

## `pdf_render.py` / `pdf_overlay.py`

PDF 转图片的兜底。Go 服务优先调用 `pdftoppm`，
找不到它时才转而调用这两个脚本（走 PyMuPDF）。

服务会按顺序找 Python：程序目录的 `.venv\` → 工作目录的 `.venv\` → `PATH`。

```bash
pip install PyMuPDF
```

装了 [Poppler](https://poppler.freedesktop.org/) 并把 `pdftoppm` 加进 PATH，
就完全不需要 Python。

## `paddle_server.py`

给**没有 Chromium 系浏览器**的机器用的独立识别引擎。
它把 PaddleOCR 包成和主服务相同形状的 HTTP 接口。

```bash
pip install -r requirements.txt
python paddle_server.py            # 监听 127.0.0.1:5000
```

然后把主服务的 `config.json` 改成：

```json
{ "active_engine": "localfallback",
  "local_fallback_url": "http://127.0.0.1:5000/api/ocr" }
```

注意：PaddleOCR 首次运行会自行下载模型（需要联网），
整套装完约几百 MB——这正是主引擎存在的理由。

PaddleOCR 由其作者按 Apache-2.0 授权，不随本仓库分发。
