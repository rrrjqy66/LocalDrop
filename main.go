package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/skip2/go-qrcode" // 第三方库：生成二维码
)

// 它的作用是：在数据发给用户之前，先被我们拦截一下，统计发送了多少字节，算进度
type ProgressWriter struct {
	http.ResponseWriter

	Total       int64 // 文件总大小 (字节)
	Written     int64 // 本次连接已经发送的字节数
	StartOffset int64 // 断点续传的起始位置 (比如用户下载了一半暂停了，下次从这里开始)
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {

	// 调用原始 ResponseWriter 的 Write 方法把数据 p 发给浏览器
	// n 是成功发送的字节数
	n, err := pw.ResponseWriter.Write(p)

	pw.Written += int64(n)

	// 计算当前总共下载了多少 (起始偏移量 + 本次已传)
	totalWritten := pw.StartOffset + pw.Written

	percent := float64(totalWritten) / float64(pw.Total) * 100

	fmt.Printf("\r %.1f%% (%s/%s)   ",
		percent,
		formatSize(totalWritten), // 调用下面的辅助函数转成 MB/GB
		formatSize(pw.Total))

	// 返回发送的字节数和错误信息 (这是 Write 接口的标准要求)
	return n, err
}

func main() {

	// flag.String 返回的是一个 指针 (*string)
	// 如果用户没传参数，filePath 的值就是无
	filePath := flag.String("file", " ", "要传输的文件路径")
	port := flag.String("port", "8989", "端口")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("请输入文件名，例如: go run main.go -file video.mp4")
		return
	}

	//检查文件是否存在
	fileStat, err := os.Stat(*filePath) //获取文件信息 (大小、修改时间等)
	if os.IsNotExist(err) {
		// log.Fatal 会打印错误并直接退出程序
		log.Fatal("文件找不到，请检查路径")
	}

	fileName := filepath.Base(*filePath) //拿文件名

	//取本机局域网IP
	localIP := getLocalIP()

	// 拼接成 http://192.168.1.5:8989/video.mp4 这种格式
	url := fmt.Sprintf("http://%s:%s/%s", localIP, *port, fileName)

	//打印界面信息
	fmt.Println("\n========================================")
	fmt.Printf("文件: %s (%s)\n", fileName, formatSize(fileStat.Size()))
	fmt.Println("链接:", url)
	fmt.Println("========================================")

	// qrcode.New 生成二维码对象，qrcode.Medium 是二维码的纠错等级(奇怪的知识又增加了)
	qr, err := qrcode.New(url, qrcode.Medium)
	if err == nil {
		// ToSmallString(false) 将二维码转成控制台可见的 ASCII 字符打印出来
		fmt.Println(qr.ToSmallString(false))
	}

	fmt.Println("服务已启动！(按 Ctrl+C 停止服务)")
	fmt.Println("========================================")

	//请求先全进来然后筛选
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// 忽略浏览器自动请求的小图标
		if strings.Contains(r.URL.Path, "favicon") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// 微信扫码给个提示
		if strings.Contains(r.UserAgent(), "MicroMessenger") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8") //是text/html类型
			fmt.Fprint(w, "<h1>请点击右上角 -> 在浏览器打开</h1>")
			return
		}

		// 处理断点续传逻辑
		// 浏览器下载大文件时，如果不小心断了，下次请求头里会带上 "Range: bytes=1024-"
		rangeHeader := r.Header.Get("Range")
		startOffset := int64(0)

		if rangeHeader != "" {
			fmt.Printf("\n断点续传: %s (从 %s 继续)\n", r.RemoteAddr, rangeHeader) //用户ip地址

			var start int64
			// 这里的 &start 取地址修改 start 变量
			if n, _ := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); n == 1 {
				startOffset = start
			}
		} else {
			//就真断了是新链接
			fmt.Printf("\n新连接接入: %s\n", r.RemoteAddr)
		}
		fmt.Printf("开始传输: %s (%s)\n", fileName, formatSize(fileStat.Size()))

		// 应该“下载”而不是直接在网页里播放
		w.Header().Set("Content-Disposition", "attachment; filename="+fileName)

		// 初始化 ProgressWriter 结构体
		pw := &ProgressWriter{
			ResponseWriter: w,
			Total:          fileStat.Size(),
			StartOffset:    startOffset,
		}

		//实现断点续传
		http.ServeFile(pw, r, *filePath)

		totalWritten := pw.StartOffset + pw.Written
		if totalWritten == pw.Total {
			fmt.Printf("\n传输完成！共 %s\n", formatSize(pw.Total))
			fmt.Println("========================================")
		} else {
			fmt.Printf("\n传输中断 (已传输 %s)\n", formatSize(totalWritten))
		}
	})

	//阻塞在这里，一直监听请求，直到报错
	err = http.ListenAndServe(":"+*port, nil)
	if err != nil {
		log.Fatal("启动失败: ", err)
	}
}

// 获取本机的局域网 IP
func getLocalIP() string {

	conn, err := net.Dial("udp", "baidu.com:80") //udp协议连接baidu.com:80
	if err != nil {
		fmt.Println("获取局域网IP失败:", err)
		return "127.0.0.1" //挂了但还是动吧
	}
	//关闭udp链接
	defer conn.Close()

	// 获取本地地址并转成字符串,强制转换为 *net.UDPAddr 类型
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// 格式化文件大小 (将字节转为 MB, GB)
func formatSize(size int64) string {
	const unit = 1024

	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	//div(除数)exp(指数)
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
