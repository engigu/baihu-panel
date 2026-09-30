import os
import json
import urllib.request

_VALID_FORMATS = {'text', 'markdown', 'html'}

def _build_from_string(options_str: str) -> dict:
    if options_str in _VALID_FORMATS:
        return {'format': options_str}
    return {'channel_id': options_str}

def _build_from_dict(options_dict: dict) -> dict:
    if not isinstance(options_dict, dict):
        return {}

    result = {}
    for k, v in options_dict.items():
        if k == 'channelId':
            result['channel_id'] = v
        elif k == 'options' and isinstance(v, dict):
            # 展平合并嵌套的 options 字典
            result.update(v)
        else:
            result[k] = v

    return result

def parse_and_build_options(options, kwargs) -> dict:
    opts = {}
    if isinstance(options, str):
        opts.update(_build_from_string(options))
    elif isinstance(options, dict):
        opts.update(_build_from_dict(options))

    if kwargs:
        opts.update(_build_from_dict(kwargs))

    return opts

def notify(title, content, options=None, **kwargs):
    """
    发送内建通知。

    =========================================================================
    支持的 5 种传参格式与转换为后端 Payload 的映射关系:
    1. 2参调用: notify('标题', '正文')
       => {"channel_id": "默认", "title": "标题", "content": "正文", "options": {}}

    2. 3参格式简写: notify('标题', '正文', 'markdown')
       => {"channel_id": "默认", "title": "标题", "content": "正文", "options": {"format": "markdown"}}

    3. 3参渠道简写: notify('标题', '正文', 'ch-123')
       => {"channel_id": "ch-123", "title": "标题", "content": "正文", "options": {}}

    4. 3参标准字典 (推荐): notify('标题', '正文', {'format': 'html', 'channel_id': 'ch-123', 'group': '打卡'})
       => {"channel_id": "ch-123", "title": "标题", "content": "正文", "options": {"format": "html", "group": "打卡"}}

    5. 关键字传参 (Python 优雅推荐): notify('标题', '正文', channel_id='ch-123', options={'format': 'markdown', 'group': '打卡'})
       => {"channel_id": "ch-123", "title": "标题", "content": "正文", "options": {"format": "markdown", "group": "打卡"}}
    =========================================================================
    """
    options_obj = parse_and_build_options(options, kwargs)

    # 提取 channel_id 至顶层并从 options_obj 中移除
    channel_id = options_obj.pop('channel_id', None)

    # 兼容旧版 text 入参
    if 'text' in options_obj and not content:
        content = options_obj.pop('text')

    default_channel = os.environ.get("BHPKG_NOTIFY_CHANNEL")
    if not channel_id and default_channel:
        channel_id = default_channel

    token = os.environ.get("BHPKG_NOTIFY_TOKEN")
    url = os.environ.get("BHPKG_NOTIFY_URL", "http://localhost:8052/api/v1/notify/send")
    
    if not url or not token or not channel_id:
        return
    
    payload = {
        "channel_id": channel_id,
        "title": title or "系统通知",
        "content": content,
        "options": options_obj
    }
    
    data = json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(url, data=data, method='POST')
    req.add_header('Content-Type', 'application/json')
    req.add_header('notify-token', token)
    
    with urllib.request.urlopen(req) as resp:
        return resp.read().decode('utf-8')
