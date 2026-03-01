package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/term"
)

type User struct {
	StudentID int
	Name      string
	Grade     int
}

var (
	UserStore = make(map[int]*User)
)

const loginURL = "https://account.ccnu.edu.cn/cas/login"
const libURL = "http://kjyy.ccnu.edu.cn/clientweb/xcus/ic2/Default.aspx"

func main() {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("获取程序路径失败喵:", err)
	}
	exeDir := filepath.Dir(exePath)

	userDataDir := filepath.Join(exeDir, "chromedp-data")

	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		log.Fatal("创建用户数据目录失败喵:", err)
	}

	fmt.Printf("chromedp 用户数据目录: %s\n", userDataDir)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(userDataDir),
		// chromedp.NoFirstRun,
		// chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		// chromedp.DisableGPU,
	)

	var username, pswd string
	fmt.Println("请输入用户名:")
	fmt.Scanln(&username)
	fmt.Println("请输入密码:")

	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal("读取密码失败:", err)
	}
	pswd = string(bytePassword)
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	// taskCtx, taskCancel := chromedp.NewContext(ctx)
	// defer taskCancel()

	err = chromedp.Run(taskCtx,
		chromedp.Navigate(loginURL),
		chromedp.WaitVisible(`#username`, chromedp.ByID),
	)
	if err != nil {
		fmt.Printf("failed at directing: %q", err)
		return
	}

	err = chromedp.Run(taskCtx,
		chromedp.Navigate(loginURL),
		chromedp.SendKeys(`#username`, username, chromedp.ByID),
		chromedp.SendKeys(`#password`, pswd, chromedp.ByID),
		chromedp.Click(`#fm1 > section.row.btn-row > input.btn-submit`, chromedp.ByQuery),
	)
	if err != nil {
		fmt.Printf("发送表单失败: %q", err)
		return
	}

	err = chromedp.Run(taskCtx,
		chromedp.WaitNotPresent(`#userame`, chromedp.ByID),
		chromedp.WaitNotPresent(`#password`, chromedp.ByID),
	)
	if err != nil {
		fmt.Printf("登录失败: %q", err)
		return
	}
	// body > div.ui-dialog.ui-widget.ui-widget-content.ui-corner-all.ui-front

	err = chromedp.Run(taskCtx,
		chromedp.Navigate(libURL),
		chromedp.Click(`#item_list > ul > li:nth-child(2) > ul > li.it.activity`, chromedp.ByID),
		chromedp.WaitVisible(`#detail_con > div > div:nth-child(2) > h1`, chromedp.ByID),
	)
	if err != nil {
		fmt.Printf("侧边栏点击失败: %q", err)
		return
	}

	// err = chromedp.Run(taskCtx,
	// 	chromedp.WaitVisible(`body > div.ui-dialog.ui-widget.ui-widget-content.ui-corner-all.ui-front`, chromedp.ByID),
	// )
	// if err != nil {
	// 	fmt.Printf("输入框显示失败: %q", err)
	// 	return
	// }

	// err = chromedp.Run(taskCtx,
	// 	chromedp.SendKeys(`#dlg_resv_panel_default_638996826004808973 > form > div:nth-child(1) > table > tbody:nth-child(1) > tr.md_group > td.dlg_mb_panel > div > div > input`, username, chromedp.ByID),
	// 	chromedp.WaitVisible(`#ui-id-1 > li`, chromedp.ByID),
	// )
	// if err != nil {
	// 	fmt.Printf("输入学号失败: %q", err)
	// 	return
	// }

	// var rawString string
	// err = chromedp.Run(taskCtx,
	// 	chromedp.Text(`#ui-id-1 li a`, &rawString, chromedp.ByQuery),
	// )
	// if err != nil {
	// 	fmt.Printf("获取用户信息失败: %q", err)
	// }
	// fmt.Printf("rawString: %s", rawString)
}
