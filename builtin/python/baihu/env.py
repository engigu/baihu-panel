import os
import json
import urllib.request
import urllib.error

def _get_headers():
    token = os.environ.get("BHPKG_OPENAPI_TOKEN") or os.environ.get("OPENAPI_TOKEN") or os.environ.get("BHPKG_NOTIFY_TOKEN")
    if not token:
        raise RuntimeError("没有正确配置或缺少 BHPKG_OPENAPI_TOKEN 环境变量以使用 env 函数。请在白虎面板的任务设置中配置这些 Key。")
    return {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {token}"
    }

def _get_envs_url():
    url = os.environ.get("BHPKG_OPENAPI_URL") or os.environ.get("OPENAPI_URL")
    if url:
        return url
    
    notify_url = os.environ.get("BHPKG_NOTIFY_URL", "http://localhost:8052/api/v1/notify/send")
    for target in ["/api/v1/notify/send/", "/api/v1/notify/send", "/api/v1/notify/", "/api/v1/notify"]:
        if target in notify_url:
            return notify_url.replace(target, "/open2api/v1/env")
            
    return "http://localhost:8052/open2api/v1/env"

def _request(url, method="GET", data=None):
    headers = _get_headers()
    payload = None
    if data is not None:
        payload = json.dumps(data).encode("utf-8")
    
    req = urllib.request.Request(url, data=payload, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as resp:
            body = resp.read().decode("utf-8")
            if not body:
                return {}
            parsed = json.loads(body)
            if isinstance(parsed, dict) and parsed.get("code") is not None and parsed.get("code") != 200:
                msg = parsed.get("msg") or parsed.get("message") or "未知错误"
                raise RuntimeError(f"请求失败 [{parsed.get('code')}]: {msg}")
            return parsed
    except urllib.error.HTTPError as e:
        err_body = e.read().decode("utf-8")
        try:
            err_json = json.loads(err_body)
            msg = err_json.get("msg") or err_json.get("message") or err_body
        except Exception:
            msg = err_body
        raise RuntimeError(f"请求失败 [{e.code}]: {msg}")
    except Exception as e:
        raise RuntimeError(f"请求发生异常: {e}")

def get_envs():
    """
    获取所有的环境变量列表。
    """
    url = f"{_get_envs_url()}/all"
    res = _request(url, "GET")
    return res.get("data", [])

def get_env(name):
    """
    根据变量名获取环境变量。如果不存在则返回 None。
    """
    envs = get_envs()
    for env in envs:
        if env.get("name") == name:
            return env
    return None

def add_envs(envs_list):
    """
    批量添加环境变量。
    envs_list: 包含环境变量字典的列表，如 [{"name": "KEY", "value": "VAL", "remark": "备注"}]
    """
    url = _get_envs_url()
    added_envs = []
    for env in envs_list:
        if "name" not in env or "value" not in env:
            raise ValueError("环境变量必须包含 'name' 和 'value'")
        res = _request(url, "POST", env)
        if "data" in res:
            added_envs.append(res["data"])
    return added_envs

def add_env(name, value, remark="", type="normal", hidden=True, enabled=True):
    """
    添加单个环境变量。
    """
    url = _get_envs_url()
    payload = {
        "name": name,
        "value": value,
        "remark": remark,
        "type": type,
        "hidden": hidden,
        "enabled": enabled
    }
    res = _request(url, "POST", payload)
    return res.get("data")

def update_env(id, name, value, remark=None, type=None, hidden=None, enabled=None):
    """
    根据 ID 更新环境变量。
    """
    url = f"{_get_envs_url()}/{id}"
    payload = {}
    if name is not None: payload["name"] = name
    if value is not None: payload["value"] = value
    if remark is not None: payload["remark"] = remark
    if type is not None: payload["type"] = type
    if hidden is not None: payload["hidden"] = hidden
    if enabled is not None: payload["enabled"] = enabled

    res = _request(url, "PUT", payload)
    return res.get("data")

def delete_envs(ids):
    """
    批量删除环境变量。
    """
    for fid in ids:
        delete_env(fid)

def delete_env(id):
    """
    根据 ID 删除指定的环境变量。
    """
    url = f"{_get_envs_url()}/{id}"
    _request(url, "DELETE")
    return True
