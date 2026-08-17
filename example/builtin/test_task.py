import baihu

# 内建工具任务管理与执行控制测试示例 (Python)
# 
# 前提条件：
# 1. 已经在环境中安装了 baihu 包（例如通过 `baihu builtininstall`）
# 2. 环境中已注入有效环境变量：
#    - BHPKG_OPENAPI_TOKEN (必填，用于管理接口鉴权)

def main():
    print("====== 开始运行 Python 任务管理与执行控制示例 ======")

    try:
        # 获取所有任务列表
        tasks = baihu.get_tasks()
        print(f"成功获取到 {len(tasks)} 个定时任务:")
        for task in tasks[:5]:  # 仅打印前5个
            print(f"  - [{task.get('id')}] {task.get('name')} (表达式: {task.get('schedule')}, 备注: {task.get('remark')})")

        # 示例：尝试触发第一个任务的运行
        if tasks:
            target_task = tasks[0]
            print(f"\n尝试手动触发任务运行: [{target_task.get('id')}] {target_task.get('name')}...")
            result = baihu.execute_task(target_task.get("id"))
            print("执行指令发送成功。")

        # 获取最近的执行结果列表
        results = baihu.get_last_results()
        print(f"\n最近共有 {len(results)} 条任务执行记录。")
            
    except Exception as e:
        print(f"任务操作失败: {e}")
        print("提示: 请确保在面板任务设置中正确注入了 OpenAPI Token。")

if __name__ == "__main__":
    main()
