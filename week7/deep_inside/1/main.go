package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"
)

const userFile = "/run/container_user"
const observePath = "/tmp/2/main"

var baseDelay = 60 * time.Millisecond

// typeWriter 模拟打字机效果
func typeWriter(text string, baseDelay time.Duration) {
	for _, c := range text {
		fmt.Printf("%c", c)
		os.Stdout.Sync()
		d := baseDelay + time.Duration(rand.Intn(20)-10)*time.Millisecond
		if d < 0 {
			d = 0
		}
		time.Sleep(d)
	}
	fmt.Println()
	os.Stdout.Sync()
}

func startOtherProcess() {
	// 检查是否存在 observe
	if _, err := os.Stat(observePath); err != nil {
		fmt.Println("⚠️ 未找到观察模块，剧情无法继续，联系管理员检查镜像完整性！")
		return
	}

	cmd := exec.Command(observePath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	// 后台启动
	cmd.Start()

	typeWriter("\n\n你感到有什么东西在看着你,可能是你的错觉???", baseDelay)
	time.Sleep(1 * time.Second)
	typeWriter("\n\nTips: 1.多尝试查看各种文件，2.减少删除和写入操作，以及不要尝试重启容器，避免死档。3.如果遇到一些奇怪的现象可能是为了氛围感特意营造的，不要感到恐慌", baseDelay)

}

func main() {
	// 必须 sudo 运行，否则拒绝
	if os.Getenv("SUDO_USER") == "" {
		typeWriter("“请以 sudo 面对它…以招来祂的注视！”", baseDelay)
		os.Exit(2)
	}

	// 若已记录用户则不再重复剧情
	if _, err := os.Stat(userFile); err == nil {
		startOtherProcess()
		return
	}

	fmt.Print("请输入你的姓名： ")
	var username string
	fmt.Scanf("%s", &username)

	// 写入全局
	_ = os.WriteFile(userFile, []byte(username), 0644)
	fmt.Printf("游戏开始：%s，希望能给你带来一场难忘的体验。\n", username)
	time.Sleep(2 * time.Second)

	rand.Seed(time.Now().UnixNano())

	paragraphs := []string{
		"11月29日，夜深了，工位的灯光只剩下幽暗的黄光，像是被时间吞噬的一缕残影。我坐在显示器前，手里攥着前辈留给我的笔记本，心跳莫名地加快。",
		"前辈——那个沉默而执着的人——就在三天前猝然离职。没有预兆，也没有留下任何告别。他离职前唯一在意的，便是那个被他称作“深处”的 Docker 容器。",
		"没人知道那意味着什么，也没有人敢随便去启动它。前辈在最后的留言里写道：",
		"“如果有人要接手这份工作……请小心。它不是普通的容器，它……观察你，也在等待。”",
		"我深吸一口气，手指悬在键盘上，眼前的终端里只有那行冷漠的命令提示符。",
		"这是我的任务——去启动它，去理解它，去面对前辈没能承受的东西。\n",
	}

	for _, p := range paragraphs {
		typeWriter(p, baseDelay)
		time.Sleep(500 * time.Millisecond)
	}

	// 剧情结束后自动启动 observe
	startOtherProcess()
}
