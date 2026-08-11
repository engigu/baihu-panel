import os
import json
import urllib.request

_VALID_FORMATS = ('text', 'markdown', 'html', '')

def notify(title, content, format='', channel_id=None, **_kwargs):
    """
    发送内建通知。

    :param title: 通知标题
    :param content: 通知正文
    :param format: 内容格式："text"(默认), "markdown", "html"
    :param channel_id: 渠道ID，不传则使用环境变量 BHPKG_NOTIFY_CHANNEL
    """
    # 兼容旧版: 关键字参数 text= 映射到 content
    if 'text' in _kwargs:
        content = _kwargs['text']
    # 兼容旧版: 位置调用 notify(title, text, channel_id)
    # 若第3个参数不是合法 format 且 channel_id 未显式传入，则将其视为 channel_id
    if format and format not in _VALID_FORMATS and channel_id is None:
        channel_id = format
        format = ''
    token = os.environ.get("BHPKG_NOTIFY_TOKEN")
    url = os.environ.get("BHPKG_NOTIFY_URL", "http://localhost:8052/api/v1/notify/send")
    default_channel = os.environ.get("BHPKG_NOTIFY_CHANNEL")
    
    cid = channel_id or default_channel
    
    if not url or not token or not cid:
        return
    
    payload = {
        "channel_id": cid,
        "title": title,
        "content": content,
        "format": format
    }
    
    data = json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(url, data=data, method='POST')
    req.add_header('Content-Type', 'application/json')
    req.add_header('notify-token', token)
    
    
    with urllib.request.urlopen(req) as resp:
        return resp.read().decode('utf-8')

