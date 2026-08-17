const baihu = require('baihu');

/**
 * 内建工具环境变量管理测试示例 (Node.js)
 * 
 * 使用说明：
 * 1. 确保已在该 Node.js 环境下安装过内建包。
 * 2. 运行时系统需要注入有效的 OpenAPI 凭证 (BHPKG_OPENAPI_TOKEN 或 OPENAPI_TOKEN)。
 */

async function main() {
    console.log("====== 开始运行 Node.js 环境变量管理示例 ======");

    try {
        // 获取全部环境变量
        const envs = await baihu.getEnvs();
        console.log(`当前共有 ${envs.length} 个环境变量`);

        // 新增一个临时环境变量
        const newEnvName = "BHPKG_TEST_KEY_JS";
        const newEnvVal = "HelloBaihuJS";
        console.log(`正在创建环境变量: ${newEnvName}...`);
        const createdEnv = await baihu.addEnv(
            newEnvName,
            newEnvVal,
            "Node.js SDK 测试自动创建"
        );
        console.log(`创建成功: ID=${createdEnv.id}, Name=${createdEnv.name}`);

        // 查询该环境变量
        const checkedEnv = await baihu.getEnv(newEnvName);
        if (checkedEnv) {
            console.log(`成功查询到变量: ${checkedEnv.name} = ${checkedEnv.value}`);

            // 修改该环境变量的值
            const updatedVal = "HelloBaihuJS_Updated";
            console.log(`正在修改环境变量的值为: ${updatedVal}...`);
            const updatedEnv = await baihu.updateEnv(
                checkedEnv.id,
                newEnvName,
                updatedVal,
                "Node.js SDK 测试自动更新"
            );
            console.log(`更新成功: Value=${updatedEnv.value}`);

            // 删除该临时环境变量
            console.log(`正在删除临时环境变量: ID=${checkedEnv.id}...`);
            await baihu.deleteEnv(checkedEnv.id);
            console.log("删除成功！");
        }

    } catch (e) {
        console.error(`环境变量操作失败: ${e.message}`);
        console.log("提示: 请确保在面板任务设置中正确注入了 OpenAPI Token。");
    }
}

main();
