import os
import json
import urllib.request

_VALID_FORMATS = ('text', 'markdown', 'html')

def notify(title, content, options=None, **kwargs):
    """
    发送内建通知。

    :param title: 通知标题
    :param content: 通知正文
    :param options: dict 选项 {"format": "markdown", "channel_id": "ch-xxx"} 或格式/渠道字符串简写
    :param kwargs: 支持关键字传参 format='markdown', channel_id='ch-xxx', text='...'
    """
    format_val = ''
    channel_id = ''

    # 1. 解析 options 字典或字符串简写
    opts = {}
    if isinstance(options, dict):
        opts.update(options)
    elif isinstance(options, str):
        if options in _VALID_FORMATS:
            format_val = options
        else:
            channel_id = options

    opts.update(kwargs)

    # 2. 提取字段并兼容旧版 text
    if 'text' in opts and not content:
        content = opts['text']
    format_val = opts.get('format', format_val)
    channel_id = opts.get('channel_id', opts.get('channelId', channel_id))

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
        "format": format_val
    }
    
    data = json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(url, data=data, method='POST')
    req.add_header('Content-Type', 'application/json')
    req.add_header('notify-token', token)
    
    with urllib.request.urlopen(req) as resp:
        return resp.read().decode('utf-8')

