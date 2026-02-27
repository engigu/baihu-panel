package resetpwd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/engigu/baihu-panel/internal/bootstrap"
	"github.com/engigu/baihu-panel/internal/services"
	"github.com/engigu/baihu-panel/internal/utils"
)

func Run(args []string) {
	fmt.Print("此操作将重置 admin 用户的密码，是否继续? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println("操作已取消。")
		return
	}

	// 必须初始化环境与数据库才能修改密码
	bootstrap.InitBasic()

	userService := services.NewUserService()
	adminUser := userService.GetUserByUsername("admin")
	if adminUser == nil {
		fmt.Println("找不到 admin 用户。")
		return
	}

	fmt.Print("请输入 admin 用户的新密码 (留空则自动随机生成): ")
	inputPwd, _ := reader.ReadString('\n')
	newPassword := strings.TrimSpace(inputPwd)
	if newPassword == "" {
		newPassword = utils.RandomString(12)
		fmt.Println("未输入密码，系统已自动生成。")
	}

	err := userService.UpdatePassword(adminUser.ID, newPassword)
	if err != nil {
		fmt.Printf("重置密码失败: %v\n", err)
		return
	}

	fmt.Println("--------------------------------------------------")
	fmt.Println("admin 用户密码已重置成功:")
	fmt.Printf("新密码: %s\n", newPassword)
	fmt.Println("请妥善保管您的新密码，并登录后及时修改。")
	fmt.Println("--------------------------------------------------")
}
