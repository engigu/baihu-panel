import os
import json
import urllib.request

def notify(title, text, channel_id=None):
    """
    发送内建通知。
    """
    token = os.environ.get("BHPKG_NOTIFY_TOKEN")
    url = os.environ.get("BHPKG_NOTIFY_URL", "http://localhost:8052/api/v1/notify/send")
    default_channel = os.environ.get("BHPKG_NOTIFY_CHANNEL")
    
    cid = channel_id or default_channel
    
    if not url or not token or not cid:
        return
    
    payload = {
        "channel_id": cid,
        "title": title,
        "text": text
    }
    
    data = json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(url, data=data, method='POST')
    req.add_header('Content-Type', 'application/json')
    req.add_header('notify-token', token)
    
    
    with urllib.request.urlopen(req) as resp:
        return resp.read().decode('utf-8')

