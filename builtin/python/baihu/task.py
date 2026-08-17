import os
import json
import urllib.request
import urllib.error

def _get_headers():
    token = os.environ.get("BHPKG_OPENAPI_TOKEN") or os.environ.get("OPENAPI_TOKEN") or os.environ.get("BHPKG_NOTIFY_TOKEN")
    if not token:
        raise RuntimeError("没有正确配置或缺少 BHPKG_OPENAPI_TOKEN 环境变量以使用 task 函数。请在白虎面板的任务设置中配置这些 Key。")
    return {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {token}"
    }

def _get_base_url():
    url = os.environ.get("BHPKG_OPENAPI_URL") or os.environ.get("OPENAPI_URL")
    if url:
        # If openapi_url ends with /env, replace it with nothing or use base
        if url.endswith("/env"):
            return url[:-4]
        elif url.endswith("/env/"):
            return url[:-5]
        return url
    
    notify_url = os.environ.get("BHPKG_NOTIFY_URL", "http://localhost:8052/api/v1/notify/send")
    for target in ["/api/v1/notify/send/", "/api/v1/notify/send", "/api/v1/notify/", "/api/v1/notify"]:
        if target in notify_url:
            return notify_url.replace(target, "/open2api/v1")
            
    return "http://localhost:8052/open2api/v1"

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

def get_tasks():
    """
    获取全部任务列表。
    """
    url = f"{_get_base_url()}/tasks"
    res = _request(url, "GET")
    return res.get("data", [])

def get_task(id):
    """
    根据 ID 获取单个任务的详细信息。
    """
    url = f"{_get_base_url()}/tasks/{id}"
    res = _request(url, "GET")
    return res.get("data")

def update_task(id, name=None, command=None, remark=None, pin_type=None, trigger_type=None, schedule=None, timeout=None, work_dir=None, retry_count=None, retry_interval=None, random_range=None, enabled=None):
    """
    根据 ID 更新任务。
    """
    url = f"{_get_base_url()}/tasks/{id}"
    payload = {}
    if name is not None: payload["name"] = name
    if command is not None: payload["command"] = command
    if remark is not None: payload["remark"] = remark
    if pin_type is not None: payload["pin_type"] = pin_type
    if trigger_type is not None: payload["trigger_type"] = trigger_type
    if schedule is not None: payload["schedule"] = schedule
    if timeout is not None: payload["timeout"] = timeout
    if work_dir is not None: payload["work_dir"] = work_dir
    if retry_count is not None: payload["retry_count"] = retry_count
    if retry_interval is not None: payload["retry_interval"] = retry_interval
    if random_range is not None: payload["random_range"] = random_range
    if enabled is not None: payload["enabled"] = enabled

    res = _request(url, "PUT", payload)
    return res.get("data")

def delete_task(id):
    """
    根据 ID 删除指定任务。
    """
    url = f"{_get_base_url()}/tasks/{id}"
    _request(url, "DELETE")
    return True

def execute_task(id):
    """
    触发执行特定任务。
    """
    url = f"{_get_base_url()}/execute/task/{id}"
    res = _request(url, "POST")
    return res.get("data")

def stop_task(log_id):
    """
    根据日志 ID 停止正在运行的任务。
    """
    url = f"{_get_base_url()}/tasks/stop/{log_id}"
    res = _request(url, "POST")
    return res.get("data")

def get_last_results():
    """
    获取最近的执行结果列表。
    """
    url = f"{_get_base_url()}/execute/results"
    res = _request(url, "GET")
    return res.get("data", [])
