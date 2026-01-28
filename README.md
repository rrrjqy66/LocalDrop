# LocalTran

局域网文件快传工具，扫码即传。

## 功能

- 扫码下载 - 自动生成二维码
- 断点续传 - 下载中断可继续
- 实时进度 - 显示传输进度

## 使用

```bash
go run main.go -file video.mp4
```

**参数说明**

- `-file` 文件路径（必填）
- `-port` 端口号（默认 8989）

**使用流程**

1. 运行命令，终端显示二维码和链接
2. 手机扫码或浏览器访问链接
3. 点击下载

**示例**

```bash
# 传输视频
go run main.go -file movie.mp4

# 指定端口
go run main.go -file document.pdf -port 8080
```

## 注意

- 需在同一局域网
- 微信扫码后点击"在浏览器打开"
- 路径有空格需加引号

## 编译

```bash
go build
```
