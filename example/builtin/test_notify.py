import baihu

# 内建通知测试示例 (Python)
# 
# 前提条件：
# 1. 已经在环境中安装了 baihu 包（例如通过 `baihu builtininstall`）
# 2. 环境中已注入有效环境变量：
#    - BHPKG_NOTIFY_TOKEN
#    - BHPKG_NOTIFY_CHANNEL

def main():
    print("正在尝试发送 Python 内建通知...")
    try:
        # 调用内建 notify 函数
        # 内部会自动使用 BHPKG_NOTIFY_TOKEN & BHPKG_NOTIFY_CHANNEL 进行鉴权和投递
        response = baihu.notify(
            title="Python 任务提醒",
            text="这是一条来自 Python 示例脚本的通知消息。调用非常简单！"
        )
        print("发送请求已处理。")
        if response:
            print(f"服务器响应: {response}")
            
    except ImportError as e:
        print(f"错误: 库未正确加载。请确保已执行过环境初始化。详情: {e}")
    except Exception as e:
        print(f"发送过程发生异常: {e}")

if __name__ == "__main__":
    main()
