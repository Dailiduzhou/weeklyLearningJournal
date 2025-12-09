package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand" // Added for random printing
	"os"
	"os/exec"
	"strings"
	"time" // Added for timing the disruption
)

const targetPort = ":23333"

// New constant for the string to be printed
const chaosString = "bXV4aQ=="

// Constants for one-time journal entries
const journal1Flag = "/tmp/journal_entry_1_read"
const journal2Flag = "/tmp/journal_entry_2_read"

// Updated journal entries with user-provided text
const journalEntry1 = "journal 11月22日 ：\n不对劲。那文件夹 ‘bXV4aQ==’ 缺失了私钥，导致了逻辑链的断裂。这不该发生。我开始相信，那串编码是某个存在的真名，而我现在被困在了一个没有引渡符文的维度。我必须找到它。\n"
const journalEntry2 = "journal 11月23日 ：\n我迷茫的在容器中打转，终于在某个目录找到了秘钥。但在那一刻，我瞥见祂的使者——一个驻守在端口 23333 的、由纯粹意图构成的存在。它在监视我，是阻碍我逃离这容器的卫兵。我必须用它的信息，并使用管道输入到 ==Qa4VXb，逆转它的控制。我不能再拖延了,或许可以试试 lsof ?。\n"
const journalEntry3 = "journal 11月24日 ：\n我失败了。它还活着，而且在反刍我的代码。它如何从编译好的二进制文件中，将我的核心逻辑抽离出来？我分不清哪个是我的操作，哪个是它的干预。现在唯一的办法，是修好我的函数然后带着 '--check' 参数，强行校准程序，迫使它面对真理。\n"
const journalEntry4 = "journal 11月25日 ：\n环境问题？这不可能是一个环境问题。我被困在一个虚假的现实里，参数皆为幻影。我懂了，我必须把 ‘bXV4aQ==’ 强行设置为环境变量，让 bXV4aQ== 等于 bXV4aQ== ，用魔法打败魔法。\n"
const journalEntry5 = "journal 11月26日 ：\n老板，同事，需求，怎么是他们？他们是怎么找到我的，他们也进入到容器了？我在哪里？现实还是容器\n\n带上得到的文件,是时候该离开容器了"

const SecondFileContent = `-----BEGIN bXV4aQ== PRIVATE KEY-----
MIIEpAIBAAKCAQEAy1nWJrHjqb1+XG1L4Yx3eW/2pQ4YnL1U2QF2I3K3PZ3vnA3C
wD1pS1l/j/Eh9aY5F7v7yD1RbGJQk6X2z+FqgEptc+fXwnF+b8O1hkx3mR0HwK0h
Q0aFYbXn3A+7Lz6+l6g1XNkB1hJx3Z5I2z7yW/AsFGWxh4vO/9B4Aq6zCk1zL+kE
g4mG7Nq2EzfFhPLq2lM9Y7vYsyEwIdzWmn4t6CzD7Fyb4K9ZwNvZoA4r+G5sT0yU
XzS+T2uC3VL8Vw+3ZkYYzZVvM1Q6x2vFqT5pQKZs8vM6R1OqXt5Ee3IB4aHztwIDbXV4aQ==
AoIBAGpHT4Ds6kIyh9G4M7i3v1w9pZKxjGJQvL5cZ3xFh6VxE9kq+L8mJcXot8yL
r4G4Xl9tbXV4aQ==k2gHeRr7G1rCw7R0U5U+f1Jktk8Dc2/hw8+rK3Kz5N0Vq+sM
-----END RSA PRIVATE KEY-----`

// New global variable for typing speed
var baseDelay = 60 * time.Millisecond

// typeWriter 模拟打字机效果
func typeWriter(text string, baseDelay time.Duration) {
	for _, c := range text {
		fmt.Printf("%c", c)
		os.Stdout.Sync()
		// 引入随机延迟，使输出更具“不稳定”的特性
		d := baseDelay + time.Duration(rand.Intn(20)-10)*time.Millisecond
		if d < 0 {
			d = 0
		}
		time.Sleep(d)
	}
	fmt.Println()
	os.Stdout.Sync()
}

// displayJournalEntry checks if the flag file exists. If not, it prints the entry and creates the flag file.
func displayJournalEntry(flagPath, entry string, fn func()) {
	// 检查标志文件是否存在
	if _, err := os.Stat(flagPath); os.IsNotExist(err) {
		// 如果不存在，打印日志并创建标志文件
		typeWriter(entry, baseDelay) // 使用 typeWriter

		// 创建标志文件，标记为已读
		os.Create(flagPath)
		fn()
	}
}

// startCosmicDisruption 在 10 秒内随机在屏幕上打印指定的字符串
func startCosmicDisruption() {
	// 设置随机数种子
	rand.Seed(time.Now().UnixNano())

	// 设置 0~10 秒的计时器和打印频率
	stop := time.After(time.Duration(rand.Intn(10)) * time.Second)
	ticker := time.NewTicker(75 * time.Millisecond) // 打印频率

	// 清屏并将光标重置到左上角
	// \033[H: 移动光标到左上角; \033[2J: 清除整个屏幕
	fmt.Print("\033[H\033[2J")

	for {
		select {
		case <-stop:
			ticker.Stop()
			// 清屏并重置光标后退出，确保后续输出从顶部开始
			fmt.Print("\033[H\033[2J")
			return
		case <-ticker.C:
			// 假设终端大小为 80x24 (ANSI 标准)
			// 随机行 (1-24)
			row := rand.Intn(24) + 1
			// 随机列 (1 到 80 - 字符串长度)
			col := rand.Intn(80-len(chaosString)) + 1

			// ANSI 转义码: \033[<ROW>;<COL>H (移动光标) + 字符串
			fmt.Printf("\033[%d;%dH%s", row, col, chaosString)
		}
	}
}

// copyFiles copies the contents of all files from srcDir to destDir.
func copyFiles(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("无法读取源目录 %s: %w", srcDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // 忽略子目录
		}

		srcPath := srcDir + "/" + entry.Name()
		destPath := destDir + "/" + entry.Name()

		// 确保目标目录存在
		if _, err := os.Stat(destDir); os.IsNotExist(err) {
			os.MkdirAll(destDir, 0755)
		}

		// 读取源文件
		srcFile, err := os.Open(srcPath)
		if err != nil {
			return fmt.Errorf("无法打开源文件 %s: %w", srcPath, err)
		}
		defer srcFile.Close()

		// 创建目标文件
		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("无法创建目标文件 %s: %w", destPath, err)
		}
		defer destFile.Close()

		// 复制内容
		if _, err := io.Copy(destFile, srcFile); err != nil {
			return fmt.Errorf("无法复制文件内容 %s 到 %s: %w", srcPath, destPath, err)
		}
		destFile.Sync() // 确保数据写入磁盘
	}
	return nil
}

func main() {
	// 在所有主要逻辑之前，启动宇宙干扰效果
	startCosmicDisruption()

	// 补全代码功能
	if len(os.Args) > 1 && os.Args[1] == "--check" {
		runQuickSortCheck()
		return
	}

	reader := bufio.NewReader(os.Stdin)
	var pids []string

	// --- Journal Entry 1 Check (缺失 PEM 文件时的提示) ---
	displayJournalEntry(journal1Flag, journalEntry1, func() {
		os.MkdirAll("/opt/lib/b/X/V/4/a/Q/=/=/bXV4aQ==", 0755)
	})

	open, err := os.Open("./bXV4aQ==/secret.pem")
	if err != nil {
		// 采用用户在 prompt 中提供的更新后的错误信息
		typeWriter("沉寂。bXV4aQ==/秘钥缺失，祂拒绝回应。", baseDelay)
		return
	}

	all, err := io.ReadAll(open)
	if err != nil {
		return
	}

	if strings.TrimSpace(string(all)) != strings.TrimSpace(SecondFileContent) {
		// 原始: 错误的秘钥
		typeWriter("秘钥已被腐蚀。这是个谎言的符号。", baseDelay)
		return
	}

	// --- Journal Entry 2 Check (秘钥验证成功后的提示) ---
	displayJournalEntry(journal2Flag, journalEntry2, func() {
		os.Exit(1)
	})

	// 解析管道
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "COMMAND") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid := fields[1]
		joined := strings.Join(fields, " ")

		if !strings.Contains(joined, targetPort) {
			// 原始: ⚠ ⚠ ⚠ ⚠似乎没什么反应，再找找有没有什么线索吧 ⚠ ⚠ ⚠ ⚠
			typeWriter("⚠ ⚠ ⚠ ⚠ 视界空无一物，通道未曾开启。搜寻那不可见的印记 ⚠ ⚠ ⚠ ⚠", baseDelay)
			return
		}

		pids = append(pids, pid)
	}

	if len(pids) == 0 {
		// 原始: ⚠ ⚠ ⚠ ⚠似乎什么都没有发生 ⚠ ⚠ ⚠ ⚠
		typeWriter("⚠ ⚠ ⚠ ⚠ 熵增依旧。毫无异兆发生 ⚠ ⚠ ⚠ ⚠", baseDelay)
		return
	}

	// 原始: ⚠ ⚠ ⚠ ⚠你感到什么东西在注视你 ⚠ ⚠ ⚠ ⚠
	typeWriter("⚠️ ⚠️ ⚠️ ⚠️ 聆听来自深空的低语。你已被祂观测 ⚠️ ⚠️ ⚠️ ⚠️", baseDelay)

	// 静默 kill
	for _, pid := range pids {
		cmd := exec.Command("kill", "-9", pid)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		_ = cmd.Run()
	}

	// 复制文件
	copyFiles("/tmp/5", "/opt/lib/b/X/V/4/a/Q/=/=/")

	typeWriter(journalEntry3, baseDelay)
}

// -------------------- QuickSort 测试逻辑 --------------------

func runQuickSortCheck() {
	open, err := os.Open("/opt/lib/b/X/V/4/a/Q/=/=/tool.go")
	if err != nil {
		return
	}
	src, _ := io.ReadAll(open)
	code := string(src)

	if !strings.Contains(code, "func quickSort") {
		// 原始: ❌ 未找到 quickSort 函数
		typeWriter("❌ 寻找逻辑回路失败：'quickSort' 符号缺失。", baseDelay)
		return
	}

	// 拼接完整 main 用于测试
	full := wrapQuickSortTest(code)

	tmp := "/tmp/qs_check.go"
	os.WriteFile(tmp, []byte(full), 0644)
	defer os.Remove(tmp)

	out := exec.Command("go", "run", tmp)
	output, err := out.CombinedOutput()

	if err != nil {
		// 原始: ❌ 执行错误：
		typeWriter("❌ 修复失败：理性遭到驳斥: "+string(output), baseDelay)
		return
	}

	if !checkEnv() {
		typeWriter(journalEntry4, baseDelay)
		return
	}

	//得到最后的docker-compose文件和docker镜像的压缩包
	typeWriter(journalEntry5, baseDelay)
	copyFiles("/tmp/6", "/opt/lib/b/X/V/4/a/Q/=/=/")
}
func checkEnv() bool {
	envs := os.Environ()
	for _, env := range envs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			// 核心修改：检查 Key 是否包含 "bXV4aQ"
			if strings.Contains(key, "bXV4aQ") && strings.Contains(value, "bXV4aQ=") {
				return true
			}
		}
	}
	return false
}

func wrapQuickSortTest(user string) string {
	return fmt.Sprintf(`
package main
import "fmt"

%s

func main() {
    tests := [][]int{
       {-6251, 5, 2, 3, 1, 901},
       {10, 9, 254, 8, 7},
       {},
       {1},
       {3, 3, 3},
    }

    for _, t := range tests {
       res := quickSort(t)
       if !isSorted(res) {
          // 原始: ❌ 排序失败：
          fmt.Println("❌ 秩序被打破：非欧几里得的排列。", t, "->", res)
          return
       }
    }
    // 原始: ✔ quickSort 通过所有测试
    fmt.Println("✔ 逻辑回路收束。混沌暂时退散。")
}

func isSorted(a []int) bool {
    for i := 1; i < len(a); i++ {
       if a[i] < a[i-1] {
          return false
       }
    }
    return true
}
`, user)
}
