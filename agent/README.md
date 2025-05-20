# agent

邮件发送程序

## 健康检查

每`5`秒进行发送一次心跳

`URL`: `/a/h`

`POST`请求: 
```json
{
  "hostname": "主机名"
}
```

`hostname`: 主机名

## 邮件处理

每`5`秒尝试获取一个邮件发送请求

`URL`: `/a/m`

`POST`请求:
```json
{}
```

## 邮件确认

每次处理一封邮件，就给与邮件确认反馈

`URL` : `/a/v`

`POST`请求:
```json
{
  "id": "邮件唯一标识",
  "success": true
}
```

- `id`: "邮件唯一标识"
- `success`: 邮件是否发送成功