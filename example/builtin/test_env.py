import baihu

# 内建工具环境变量管理测试示例 (Python)
# 
# 前提条件：
# 1. 已经在环境中安装了 baihu 包（例如通过 `baihu builtininstall`）
# 2. 环境中已注入有效环境变量：
#    - BHPKG_OPENAPI_TOKEN (必填，用于管理接口鉴权)

def main():
    print("====== 开始运行 Python 环境变量管理示例 ======")

    try:
        # 获取全部环境变量
        envs = baihu.get_envs()
        print(f"当前共有 {len(envs)} 个环境变量")
        
        # 新增一个临时环境变量
        new_env_name = "BHPKG_TEST_KEY"
        new_env_val = "HelloBaihu"
        print(f"正在创建环境变量: {new_env_name}...")
        created_env = baihu.add_env(
            name=new_env_name,
            value=new_env_val,
            remark="Python SDK 测试自动创建"
        )
        print(f"创建成功: ID={created_env.get('id')}, Name={created_env.get('name')}")

        # 查询刚才创建的环境变量详情
        checked_env = baihu.get_env(new_env_name)
        if checked_env:
            print(f"成功查询到变量: {checked_env.get('name')} = {checked_env.get('value')}")

            # 修改该环境变量的值
            updated_val = "HelloBaihu_Updated"
            print(f"正在修改环境变量的值为: {updated_val}...")
            updated_env = baihu.update_env(
                id=checked_env.get("id"),
                name=new_env_name,
                value=updated_val,
                remark="Python SDK 测试自动更新"
            )
            print(f"更新成功: Value={updated_env.get('value')}")

            # 删除该临时环境变量
            print(f"正在删除临时环境变量: ID={checked_env.get('id')}...")
            baihu.delete_env(checked_env.get("id"))
            print("删除成功！")
            
    except Exception as e:
        print(f"环境变量操作失败: {e}")
        print("提示: 请确保在面板任务设置中正确注入了 OpenAPI Token。")

if __name__ == "__main__":
    main()
