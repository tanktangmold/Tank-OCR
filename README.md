# Tank-OCR

本机离线的文字识别服务。启动后监听 `127.0.0.1:8000`，
把图片或 PDF 传进去，返回带坐标的结构化文字。

**不联网，不上传，不需要 API Key。** 识别在本机完成，图片不出这台电脑。

> A local, offline OCR HTTP service for Windows/macOS/Linux. It drives the
> Chrome Screen AI runtime through FFI — the same on-device engine Chrome uses
> for PDF text extraction. No network calls, no API keys, no cloud.

---

## 为什么会有这个东西

需要 OCR 的时候，常见选择是调云端 API，或者装一套 Python + PaddleOCR。
前者要把票据传给别人，后者装一次几个 G。

但如果你机器上装了 Chrome，识别引擎其实已经在硬盘里了——
Chrome 用它来识别扫描版 PDF 里的文字。这个项目就是把那个引擎接出来，
包成一个 6 MB 的 HTTP 服务。

实测识别一张 900×420 的发票图约 **0.4 秒**，中文、数字、标点全对，
包括 20 位的发票号码。

---

## 重要：识别引擎不在这个仓库里

Tank-OCR 自己不做识别，它调用 **Chrome Screen AI** 运行时。
那个运行时是 Google 的组件（约 110 MB），由 Chrome 自己下载安装，
**本项目不分发它**——它有自己的许可条款，详见 [NOTICE](NOTICE)。

服务启动时按这个顺序找运行时：

1. 环境变量 `TANK_OCR_ENGINE_DIR` 指定的目录
2. 程序同目录下的 `engine-runtime\`
3. **浏览器的组件目录**（Chrome / Edge / Chromium，自动选版本最高的）

绝大多数情况下第 3 条就够了：装了 Chrome，就能直接跑。

---

## 快速开始

### 一、拿到运行时

打开 Chrome，访问 `chrome://components`，
找到 **Screen AI**（有的版本显示为「光学字符识别」），点「检查更新」，
等它出现版本号即可。装过较新版 Chrome 的机器通常已经有了。

确认一下（Windows）：

```bash
dir "%LOCALAPPDATA%\Google\Chrome\User Data\screen_ai"
```

看到 `148.12` 之类的版本目录就说明有了。

### 二、编译

需要 [Go 1.22+](https://go.dev/dl/)。

```bash
git clone https://github.com/tanktangmold/Tank-OCR.git
```

```bash
cd Tank-OCR && go build -ldflags "-s -w" -o tank-ocr.exe .
```

Windows 上也可以直接双击 `build.bat`。
用便携版 Go 的话，先 `set GO_EXE=D:\go\bin\go.exe`。

### 三、运行

双击 `run_ocr.bat`，或者：

```bash
./tank-ocr.exe
```

浏览器打开 <http://127.0.0.1:8000/>，把图片拖进去就能识别。

启动日志会写明用的是哪个运行时：

```
Using Screen AI runtime from browser profile: ...\Chrome\User Data\screen_ai\148.12
Screen AI OCR engine successfully initialized.
```

### 离线机器怎么办

目标机器没装 Chrome，或者你想锁定运行时版本，
在**有 Chrome 的机器**上运行：

```bash
powershell -ExecutionPolicy Bypass -File scripts/fetch-engine.ps1
```

它会把运行时复制到 `engine-runtime\`。
把这个目录连同 `tank-ocr.exe`、`config.json`、`static\` 一起拷过去就能用。

`engine-runtime\` 已被 `.gitignore` 排除——**不要提交，也不要对外分发**。

---

## HTTP 接口

### `GET /api/status`

```json
{ "status": "ready", "engine": "ScreenAI", "version": "148.12", "max_dimension": 2048 }
```

`status` 为 `ready` 表示引擎已就绪。找不到运行时时服务仍会启动，此处会说明原因。

### `POST /api/ocr`

`multipart/form-data`，字段名 `file`。

```bash
curl -X POST http://127.0.0.1:8000/api/ocr -F "file=@invoice.png"
```

```json
{
  "text": "增值税普通发票\n\n开票日期:2026年06月18日\n...",
  "duration_seconds": 0.42,
  "result": { "pages": [ { "blocks": [ { "lines": [ { "text": "...", "words": [...] } ] } ] } ] }
}
```

`text` 是拼好的纯文本，`result` 是带 `bounding_box` 坐标的完整结构，
需要定位「这个金额在图上哪个位置」时用它。

超过 `max_image_dimension` 的图会自动等比缩小后再识别。

### `POST /api/ocr-pdf`

同样是 `file` 字段。逐页处理，**优先提取 PDF 自带的文字层**——
电子发票这类本来就有文字的 PDF 直接读出来，又快又准，只有扫描件才走 OCR。

### `GET /api/history`

最近的识别记录，条数由 `history_limit` 控制。

---

## 跨源调用

服务会回显请求的 `Origin`，因此**用 `file://` 直接打开的本地网页也能调用**——
这类页面的 Origin 是字符串 `null`，用通配 `*` 是匹配不上的。

监听地址固定在 `127.0.0.1`，只有本机的程序能连上，不对局域网开放。

---

## 配置

`config.json`：

| 字段 | 默认值 | 说明 |
|---|---|---|
| `active_engine` | `screenai` | 识别引擎，另有 `localfallback` |
| `local_fallback_url` | `http://127.0.0.1:5000/api/ocr` | 兜底引擎地址 |
| `bind_address` | `127.0.0.1:8000` | 监听地址 |
| `max_image_dimension` | `2048` | 超过则先缩放 |
| `history_limit` | `50` | 保留的历史条数 |

`fallback/` 下有一个基于 PaddleOCR 的 Python 兜底服务，
给没装 Chromium 系浏览器的机器用。把 `active_engine` 改成 `localfallback` 即可。

---

## 代码结构

```
main.go              配置加载、路由、CORS
engine/
  screenai.go        Screen AI 运行时的 FFI 绑定与路径发现
  localfallback.go   HTTP 兜底引擎
  engine.go          引擎接口与结果结构
handler/
  ocr.go             图片识别
  pdf.go             PDF 逐页处理与文字层提取
  status.go          状态
  history.go         历史记录
static/              自带的网页界面
scripts/
  fetch-engine.ps1   从本机 Chrome 取运行时
fallback/            PaddleOCR 兜底服务（可选）
```

引擎是接口化的（`engine.OcrEngine`），加一个新引擎实现这个接口即可。

---

## 已知限制

- **运行时依赖 Chrome 的更新节奏**：Chrome 升级组件后版本目录会变，
  服务重启时自动选最新的。想固定版本就用 `engine-runtime\`。
- **手写体识别一般**，这个引擎是为印刷体设计的。
- **单次识别是串行的**：运行时不是线程安全的，服务内部加了锁，
  并发请求会排队。

---

## 许可

代码采用 MIT，见 [LICENSE](LICENSE)。

**但请注意**：识别运行时是 Google 的组件，不适用 MIT，也不随本仓库分发。
Fork 或再分发时不要打包它——用户通过自己的 Chrome 获取。
细节见 [NOTICE](NOTICE)。

FFI 调用序列参考了 [clv-locro](https://github.com/sergiocorreia/clv-locro)（MIT，Sergio Correia）
的 Python 实现，本仓库未复制其源码。
