# Nginx 反向代理配置

如果您需要通过域名和 HTTPS 访问白虎面板，推荐使用 Nginx 作为反向代理并配置 WebSocket 负载均衡。

## Nginx 反向代理配置示例

### 1. 配置映射
首先，在 `nginx.conf` 的 `http` 块中添加 WebSocket 升级映射：
```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    '' close;
}
```

---

### 2. 服务器配置
将 `example.com` 替换为您的域名，并指定宿主机监听端口：
```nginx
server {
    listen 443 ssl http2;
    server_name example.com;
    
    ssl_certificate     /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;
    
    access_log /var/log/nginx/example.access.log;
    error_log  /var/log/nginx/example.error.log warn;
    
    location / {
        proxy_pass http://127.0.0.1:8052; # 指定白虎面板宿主机 IP 和端口
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket 支持（在线控制台必需）
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        
        proxy_buffering off;
        proxy_read_timeout 60s;
    }
}

# 自动 HTTP 跳转 HTTPS (可选)
server {
    listen 80;
    server_name example.com;
    return 301 https://$server_name$request_uri;
}
```

---

### 3. 子路径部署场景

如果您是通过 `BH_SERVER_URL_PREFIX=/baihu` 进行子路径托管，请修改 `location` 参数：
```nginx
location /baihu/ {
    proxy_pass http://127.0.0.1:8052/baihu/;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

---

## 验证与发布

在保存配置文件后，请执行以下命令确保 Nginx 配置正确并重启：
```bash
# 检查语法
nginx -t
# 重启服务
nginx -s reload
```
