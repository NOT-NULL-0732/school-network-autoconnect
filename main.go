package main

import (
	"context"
	"log"
	"os/exec"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/spf13/viper"
)

func init() {
	// 设置默认值
	viper.SetDefault("login_url", "http://10.0.0.1")
	viper.SetDefault("account", "")
	viper.SetDefault("password", "")
	viper.SetDefault("user_agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1")
	viper.SetDefault("ping_target", "1.1.1.1")
	viper.SetDefault("ping_timeout_seconds", 4)
	viper.SetDefault("check_period", "10s")
	viper.SetDefault("chrome_timeout", "30s")

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("⚠️ 读取配置文件失败: %v", err)
		}
	}

	viper.SetEnvPrefix("NETGUARD")
	viper.AutomaticEnv()
}

func isOnline() bool {
	target := viper.GetString("ping_target")
	timeout := viper.GetInt("ping_timeout_seconds")
	cmd := exec.Command("ping", "-c", "1", "-W", strconv.Itoa(timeout), target)
	err := cmd.Run()
	return err == nil
}

func doLogin(account, password string) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.IgnoreCertErrors,
		chromedp.UserAgent(viper.GetString("user_agent")),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, viper.GetDuration("chrome_timeout"))
	defer cancel()

	loginURL := viper.GetString("login_url")
	log.Printf(">>> 正在访问 %s 等待网关自动重定向...", loginURL)

	return chromedp.Run(ctx,
		chromedp.Navigate(loginURL),

		chromedp.WaitVisible(`input[name="DDDDD"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`input[name="C1"]`, chromedp.ByQuery),

		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Println("[Debug] 正在强制勾选协议...")
			return nil
		}),
		chromedp.Evaluate(`
        var cb = document.querySelector('input[name="C1"]');
        if (cb) {
            cb.checked = true;
            cb.dispatchEvent(new Event('input', { bubbles: true }));
            cb.dispatchEvent(new Event('change', { bubbles: true }));
        }
    `, nil),

		chromedp.SendKeys(`input[name="DDDDD"]`, account, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="upass"]`, password, chromedp.ByQuery),

		chromedp.Click(`input[name="0MKKey"]`, chromedp.ByQuery),

		chromedp.Sleep(3*time.Second),
	)
}

func main() {
	account := viper.GetString("account")
	password := viper.GetString("password")
	if account == "" || password == "" {
		log.Fatal("❌ 账号或密码未配置，请通过环境变量 NETGUARD_ACCOUNT / NETGUARD_PASSWORD 或配置文件提供")
	}

	checkPeriod := viper.GetDuration("check_period")
	log.Println("🛠️ 校园网守护进程已启动 (Ping 检测 + " + viper.GetString("login_url") + " 入口)")

	for {
		if !isOnline() {
			log.Println("⚠️ 网络不可达，尝试自动登录...")
			if err := doLogin(account, password); err != nil {
				log.Printf("❌ 尝试失败: %v", err)
			} else {
				log.Println("✅ 登录指令发送完成")
			}
		} else {
			log.Println("🌐 网络状态良好")
		}

		time.Sleep(checkPeriod)
	}
}
