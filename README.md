# File Share - 區域網路檔案分享服務

區域網路內的串流檔案上下傳服務。A 上傳檔案，B 即可下載。

## 啟動

```bash
docker compose up -d --build
```

瀏覽器開啟 `http://<你的IP>` 即可使用。

## 停止（自動清理所有上傳檔案）

```bash
docker compose down -v
```

`-v` 會刪除 uploads volume，清除所有上傳的檔案。

若只想停止服務但保留檔案：
```bash
docker compose down
```

## 架構

```
Browser ──▶ Nginx (:80) ──▶ Go Backend (:8080) ──▶ Disk
               │
               └── 靜態前端 (HTML/CSS/JS)
```

- **後端**: Go — 串流 I/O，1GB+ 檔案記憶體消耗極低
- **前端**: 靜態 HTML + 拖曳上傳 + 進度條
- **反向代理**: Nginx — 無上傳大小限制、關閉 buffering

## API

| Method | Path | 說明 |
|--------|------|------|
| POST | `/api/upload` | 上傳檔案（multipart/form-data, field: `files`）|
| GET | `/api/files` | 列出所有檔案 |
| GET | `/api/download/{filename}` | 下載檔案 |
| DELETE | `/api/files/{filename}` | 刪除檔案 |

## 查看區域網路 IP

```bash
# Linux
hostname -I

# macOS
ifconfig | grep "inet " | grep -v 127.0.0.1

# Windows
ipconfig
```

將 IP 分享給同網路的人即可存取。
