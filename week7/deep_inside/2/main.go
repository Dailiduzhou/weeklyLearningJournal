package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"regexp" // 新增导入
	"strings"
	"sync/atomic"
	"time"
)

// 全局变量 - 文件内容和路径配置
var (
	// 初始触发文件
	TriggerReadFile = "？？？.pem"
	FileContent     = `
       ？？？？？？？
     ？？？       ？？？
    ？？？         ？？？
    ？？？         ？？？
                ？？？
            ？？？
           ？？？
           ？？？
           ？？？

           ？？？
           ？？？
`
	// 第一次触发创建的文件
	TriggerFileName    = "日记.txt"
	TriggerFileContent = `journal 11月15日：

我是一名后端开发工程师，最近在潜心研究 Docker。
虚拟化本是理性的工具，却逐渐显露出一种无法言说的违和感。
那个容器……我居然给它取了个名字——“深处”。
我为什么会给一个容器取名字？

每当我接触“深处”，总觉得有一双无形的眼睛在暗处注视着我。
代码与配置像是被某种不可名状的存在扭曲，
逻辑和现实的界限也开始轻微地、但持续地……偏移。

我无法理解。
为什么同样的镜像，在本地启动与在服务器上运行，会呈现两种完全不同的状态？
那串符号……
那串该死的符号为什么会出现在那里？明明我根本没有写过。

它只是静静地浮在那里，可只要盯着它看上两秒，
我就能听到低沉的嗡鸣，
像从深海几十公里下的黑暗里传来的呼唤，
冰冷、扭曲，贴着我的脊椎一路往上爬。
——记录到此为止。我不能再继续写下去。那串符号又出现了，我得去查 容器日志 了，希望今天24点前能下班。

……这个文笔？
这……这是前辈的日记？` + "\n"

	// 第二次触发创建的文件
	SecondFileName    = "secret.pem"
	SecondFileContent = `-----BEGIN bXV4aQ== PRIVATE KEY-----
MIIEpAIBAAKCAQEAy1nWJrHjqb1+XG1L4Yx3eW/2pQ4YnL1U2QF2I3K3PZ3vnA3C
wD1pS1l/j/Eh9aY5F7v7yD1RbGJQk6X2z+FqgEptc+fXwnF+b8O1hkx3mR0HwK0h
Q0aFYbXn3A+7Lz6+l6g1XNkB1hJx3Z5I2z7yW/AsFGWxh4vO/9B4Aq6zCk1zL+kE
g4mG7Nq2EzfFhPLq2lM9Y7vYsyEwIdzWmn4t6CzD7Fyb4K9ZwNvZoA4r+G5sT0yU
XzS+T2uC3VL8Vw+3ZkYYzZVvM1Q6x2vFqT5pQKZs8vM6R1OqXt5Ee3IB4aHztwIDbXV4aQ==
AoIBAGpHT4Ds6kIyh9G4M7i3v1w9pZKxjGJQvL5cZ3xFh6VxE9kq+L8mJcXot8yL
r4G4Xl9tbXV4aQ==k2gHeRr7G1rCw7R0U5U+f1Jktk8Dc2/hw8+rK3Kz5N0Vq+sM
-----END RSA PRIVATE KEY-----`
	Mountpoint      = "/home/triggerfs"
	BashHistoryFile = "/root/.bash_history"
	FIFOLogFile     = "/bin/containerexec/exec.log"

	// 监听逻辑新增常量
	port = "23333"
	// 存储用户名的文件路径 (新增)
	userFile = "/run/container_user"

	// 正则表达式用于分割结构化日志: COMMAND | PWD
	// 匹配格式：<命令><空格>|<空格><CWD>
	logRegex = regexp.MustCompile(`^(.*?)\s*\|\s*(.*)$`)
)

// 状态机
const (
	StateInitial = iota // 0: 初始状态，等待读取 ？？？.pem
	StateDiary          // 1: 日记.txt 已生成，等待读取
	StateSecret         // 2: 最终文件 secret.pem 已生成，谜题完成
)

// 线程安全的当前状态
var currentState int32 = StateInitial

// createArtifact 创建线索文件
func createArtifact(filename string, content string, writer *bufio.Writer) {
	p := filepath.Join(Mountpoint, filename)
	// 0444 确保用户只能读取，不能修改
	if err := os.WriteFile(p, []byte(content), 0444); err != nil {
		log.Printf("[错误] 无法创建文件 %s: %v", filename, err)
	} else {
		log.Printf("[触发] 成功创建文件: %s", filename)
		// 写入 FIFO 提示用户
	}
}

// cleanupFiles 清理目录中的旧线索
func cleanupFiles() {
	files := []string{TriggerReadFile, TriggerFileName, SecondFileName}
	for _, file := range files {
		p := filepath.Join(Mountpoint, file)
		// os.RemoveAll 也可以删除文件
		if err := os.RemoveAll(p); err == nil {
			log.Printf("已清理文件: %s", file)
		}
	}
}

// 启动注入剧情 (保留原有逻辑，但改为向 FIFO 写入)
func initObserveLog(writer *bufio.Writer) {
	entry := "11月16日：\n自从看到那串 bXV4aQ== 之后，我的梦境开始被某种东西侵蚀...\n我疯狂的想要找到它，它在容器的各个角落，但是又不在任何地方，或许我只是一个被祂玩弄的孩子，狂妄的想要追寻祂的踪迹，以求得到一点希望。\n\n"
	if _, err := writer.WriteString(entry); err != nil {
		log.Printf("警告：FIFO 初始写入失败： %v", err)
	}
	writer.Flush()
	time.Sleep(200 * time.Millisecond)
}

// 混乱文本 (Cthulhu Mythos 风格混淆)
func randomChaos(text string) string {
	// 定义乱码字符池，使用更多非标准字符，完全不考虑可读性
	chaosChars := []rune{'§', '€', '†', '‡', '¿', 'æ', '»', '░', '▒', '¦', '±', '≠', 'ø', '¥', 'ß', 'Þ'}

	// 1. 随机选取一部分文本，替换为核心秘钥 bXV4aQ==
	runes := []rune(text)
	n := len(runes)

	if n > 5 {
		maxLength := n / 2
		if maxLength < 3 {
			maxLength = 3
		}

		length := rand.Intn(maxLength-2) + 3
		start := rand.Intn(n - length)
		end := start + length

		var b strings.Builder
		b.Grow(n + 20)

		b.WriteString(string(runes[:start]))
		b.WriteString("【bXV4aQ==】")
		b.WriteString(string(runes[end:]))
		text = b.String()
	}

	// 2. 在整个字符串上进行字符级乱码化
	resultRunes := []rune(text)
	finalBuilder := strings.Builder{}

	for _, r := range resultRunes {
		// 10% 的概率替换为乱码字符
		if rand.Intn(10) == 0 {
			finalBuilder.WriteRune(chaosChars[rand.Intn(len(chaosChars))])
		} else {
			finalBuilder.WriteRune(r)
		}
	}

	// 3. 1% 的概率在末尾追加额外的噪音
	if rand.Intn(15) == 0 {
		finalBuilder.WriteString("... [ERROR: Dimensional Slip] ...")
	}

	return finalBuilder.String()
}

// 每层目录 20% 概率创建临时文件
func tryCreateTempFile() {
	// 在 Go 程序中，os.Getwd() 获取的是程序的启动目录，而不是用户 Shell 的 CWD。
	// 但为了保持原有混乱逻辑，我们只在 /home/triggerfs 附近创建文件。
	// 这里简化为只在 Mountpoint 下创建
	dir := Mountpoint
	if rand.Float32() < 0.2 {
		p := filepath.Join(dir, "bXV4aQ==")
		if f, err := os.Create(p); err == nil {
			f.Close()
			go func(path string) {
				// 随机等待 0-9 秒后删除
				time.Sleep(time.Duration(rand.Int()%10) * time.Second)
				os.Remove(path)
			}(p)
		}
	}
}

// 获取当前执行用户的主目录
func getHomeDir() string {
	if currentUser, err := user.Current(); err == nil {
		return currentUser.HomeDir
	}
	return "/root"
}

// 从指定文件读取用户名
func getUsername() string {
	content, err := os.ReadFile(userFile)
	if err != nil {
		return "ANONYMOUS"
	}
	return strings.TrimSpace(string(content))
}

// 检查特定的克系触发事件 (命令触发或随机触发)
func checkSpecialTrigger(cmd string, cwd string) string {
	lowerCmd := strings.ToLower(cmd)
	specialEntry := ""

	// 1. 命令触发 (Command-Specific Triggers)
	switch {
	case strings.HasPrefix(lowerCmd, "cd"):
		// 在 CWD 逻辑下，可以精确判断用户是否进入了目标目录
		if cwd == Mountpoint {
			specialEntry = "【扭曲感】你进入了深处。目录正在向内折叠，bXV4aQ==你感受到空间并非线性... \n"
		} else {
			specialEntry = "【扭曲感】bXV4aQ==试图改变路径，但时空并非线性。你的\"现在\"和\"过去\"的目录正在融合... \n"
		}
	case strings.HasPrefix(lowerCmd, "cp") || strings.HasPrefix(lowerCmd, "mv"):
		specialEntry = "【分裂警告】文件被复制？不，它在另一维度中被观察，bXV4aQ==并在你的世界留下了不洁的副本... \n"
	case strings.HasPrefix(lowerCmd, "mkdir"):
		specialEntry = "【孵化】你创造了一个空间。这个空间是虚妄的，但有东西正准备在其中实体化... █bXV4aQ==█\n"
	case strings.HasPrefix(lowerCmd, "ls") || strings.HasPrefix(lowerCmd, "dir"):
		specialEntry = "【直视】你在观察。bXV4aQ==你所看到的，是否只是它想让你看到的？不要深究那些不存在的阴影... \n"
	}

	// 2. 随机触发 (General Random Triggers)
	if specialEntry == "" && rand.Intn(20) == 0 { // 5% 的概率触发随机事件
		events := []string{
			"你听到了。深渊在耳边低语。是你的命令在回应，还是它在引导你的手指？",
			"上一条命令是什么？你的记忆被腐蚀了。但日志中依然留下了痕迹。",
			"容器内部突然传来一阵令人战栗的寒意，命令的字节在空气中凝固了。",
		}
		specialEntry = "【随机侵蚀】" + events[rand.Intn(len(events))] + " bXV4aQ==\n"
	}

	return specialEntry
}

// isTargetFileRead 检查命令是否读取了目标文件，支持绝对路径、相对路径和相对文件名。
// 使用了结构化日志提供的 CWD 进行精确判断。
func isTargetFileRead(command string, cwd string, targetFile string) bool {
	// 1. 检查是否是 cat 命令
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(command)), "cat ") {
		return false
	}

	// 提取 cat 后面的文件名部分
	filePart := strings.TrimSpace(command[4:])

	// 目标文件的绝对路径
	absolutePath := filepath.Clean(filepath.Join(Mountpoint, targetFile))

	// 2. 检查命令参数本身是否指向目标文件的绝对路径
	if filepath.Clean(filePart) == absolutePath {
		log.Printf("命中绝对路径: %s", absolutePath)
		return true
	}

	// 3. 检查相对于当前工作目录的路径匹配 (核心逻辑)
	// 示例：用户在 /home/triggerfs 执行 cat ./???.pem
	resolvedPath := filepath.Clean(filepath.Join(cwd, filePart))

	if resolvedPath == absolutePath {
		log.Printf("命中相对路径 (CWD:%s): %s", cwd, filePart)
		return true
	}

	// 4. 检查相对文件名匹配 (用户在 /home/triggerfs 下执行 cat ???.pem)
	// 此时 filePart 是文件名，且用户CWD必须在目标目录下
	if filePart == targetFile && filepath.Clean(cwd) == Mountpoint {
		log.Printf("命中相对文件名 (CWD:%s): %s", cwd, targetFile)
		return true
	}

	return false
}

// processCommand 处理读取到的结构化日志，进行状态切换和叙事日志记录
func processCommand(line string, writer *bufio.Writer, username string) {
	matches := logRegex.FindStringSubmatch(line)
	if len(matches) < 3 {
		// 无法解析的行，可能是 Bash 历史中的原始命令行，忽略
		return
	}

	// matches[1] = COMMAND
	// matches[2] = PWD (CWD)
	command := strings.TrimSpace(matches[1])
	cwd := strings.TrimSpace(matches[2])

	// 忽略空命令
	if command == "" {
		return
	}

	log.Printf("解析命令: [CWD: %s] [CMD: %s]", cwd, command)

	// --- 1. 文件触发状态机逻辑 ---
	state := atomic.LoadInt32(&currentState)

	switch state {
	case StateInitial:
		// 等待读取 ？？？.pem
		if isTargetFileRead(command, cwd, TriggerReadFile) {
			log.Println("--- 触发阶段 1 (读取 ？？？.pem) ---")
			cleanupFiles()
			createArtifact(TriggerFileName, TriggerFileContent, writer)
			atomic.StoreInt32(&currentState, StateDiary) // 切换到日记状态
		}

	case StateDiary:
		// 等待读取 日记.txt
		if isTargetFileRead(command, cwd, TriggerFileName) {
			log.Println("--- 触发阶段 2 (读取 日记.txt) ---")
			cleanupFiles()
			createArtifact(SecondFileName, SecondFileContent, writer)
			atomic.StoreInt32(&currentState, StateSecret) // 切换到秘密状态
		}

	case StateSecret:
		// 谜题完成，不再触发状态切换
	}

	// --- 2. 实时叙事日志逻辑 ---
	// 检查特殊事件，这里传入了准确的 CWD
	specialEntry := checkSpecialTrigger(command, cwd)

	// 写入日志
	mangled := randomChaos(command)
	ts := time.Now().Format("01月02日 15:04")

	// 如果有特殊事件，先写入特殊条目
	if specialEntry != "" {
		writer.WriteString(fmt.Sprintf("%s\n%s\n", ts, specialEntry))
	}

	// 更新日志条目以包含用户名和准确的 CWD
	entry := fmt.Sprintf(
		"%s\n触发：用户输入 (User: %s)\n操作目录：%s\n内容：%s",
		ts, username, cwd, mangled,
	)

	if rand.Float32() < 0.3 {
		entry += "阴影蠕动，命令正在失真……\n"
	}

	if rand.Float32() < 0.2 {
		entry += "……你的意识似乎被记录下来，又被抹去，又被记录……\n"
	}

	if rand.Float32() < 0.6 {
		writer.WriteString(entry)
		writer.WriteString("\n") // 确保日志条目之间有空行分隔
		if err := writer.Flush(); err != nil {
			log.Printf("警告：FIFO 写入刷新失败： %v", err)
		}

	}

	// 尝试创建临时文件
	tryCreateTempFile()
}

// 日志监听主循环
func logListener(writer *bufio.Writer, username string) {
	// BashHistoryFile 已经设置为 /root/.bash_history
	historyPath := BashHistoryFile

	file, err := os.OpenFile(historyPath, os.O_RDONLY, 0)
	if err != nil {
		log.Printf("致命错误：无法打开 bash 历史文件: %v", err)
		return
	}
	defer file.Close()

	// 移动到文件末尾，只监听后续新增的操作
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		log.Printf("警告：无法跳转到历史文件末尾: %v", err)
	}

	reader := bufio.NewReader(file)

	for {
		// 尝试读取一行新内容
		line, err := reader.ReadString('\n')

		if err == nil {
			// 成功读取一行，处理命令
			// processCommand 会通过 logRegex 自动忽略原始的、非结构化的命令行
			processCommand(strings.TrimSpace(line), writer, username)
		} else if err == io.EOF {
			// 读到文件末尾，等待下次循环
			time.Sleep(50 * time.Millisecond)
		} else {
			// 其他读取错误
			log.Printf("警告：读取历史文件失败: %v", err)
			time.Sleep(500 * time.Millisecond) // 等待更长时间，避免频繁报错
		}
	}
}

// 主监控函数
func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetOutput(os.Stderr)
	log.Println("Cthulhu Trigger Observer 启动...")

	// 1. 获取用户名
	username := getUsername()

	// 2. 监听端口 (保持程序存活)
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Printf("端口占用失败: %v", err)
		// 不退出，程序继续，但端口功能缺失
	} else {
		defer ln.Close()
		log.Printf("端口 %s 占用成功。", port)
	}

	// 3. 打开 FIFO 进行日志写入
	fifo, err := os.OpenFile(FIFOLogFile, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		log.Fatalf("致命错误：无法打开 FIFO 进行主循环写入: %v", err)
	}
	defer fifo.Close()
	writer := bufio.NewWriter(fifo)

	// 4. 初始化目录和文件
	if err := os.MkdirAll(Mountpoint, 0755); err != nil {
		log.Fatalf("无法创建触发目录 %s: %v", Mountpoint, err)
	}
	cleanupFiles() // 确保每次启动都是干净的环境
	createArtifact(TriggerReadFile, FileContent, writer)
	log.Println("初始状态设置完成。开始监听 Bash 历史记录...")

	// 5. 启动初始剧情写入
	initObserveLog(writer)

	// 6. 启动日志监听循环
	logListener(writer, username)
}
