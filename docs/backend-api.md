# 后端接口文档

本文档基于当前 Go/Gin 后端实现整理，用于前端适配 API。

## 基本信息

- 默认服务地址：`http://localhost:8181`
- 默认 WebSocket 地址：`ws://localhost:8181/ws`
- 默认前端 CORS 白名单：`http://localhost:5173`
- HTTP 请求体格式：`application/json`
- 鉴权方式：JWT Bearer Token
- Token 有效期：24 小时

除公开接口外，所有 `/api/*` 接口都需要携带请求头：

```http
Authorization: Bearer <token>
```

通用错误响应：

```json
{
  "error": "错误原因"
}
```

部分旧接口也可能返回：

```json
{
  "msg": "提示信息"
}
```

前端建议同时兼容 `error` 和 `msg`。

## 公共接口

### 健康检查

```http
GET /ping
```

成功响应：

```json
{
  "message": "Hello, world!"
}
```

## 用户接口

### 注册

```http
POST /register
```

是否鉴权：否

请求体：

```json
{
  "username": "alice",
  "password": "123456"
}
```

成功响应：

```json
{
  "msg": "注册成功"
}
```

失败示例：

```json
{
  "error": "username already exists"
}
```

### 登录

```http
POST /login
```

是否鉴权：否

请求体：

```json
{
  "username": "alice",
  "password": "123456"
}
```

成功响应：

```json
{
  "token": "<jwt-token>",
  "user_id": 1,
  "username": "alice"
}
```

失败示例：

```json
{
  "error": "user not found"
}
```

```json
{
  "error": "wrong password"
}
```

### 当前用户信息

```http
GET /api/userinfo
```

是否鉴权：是

成功响应：

```json
{
  "id": 1,
  "username": "alice",
  "created": "2026-06-13T14:30:00+08:00"
}
```

### 退出登录

```http
POST /api/logout
```

是否鉴权：是

说明：后端会把当前 JWT 加入黑名单。Token 过期前，再用该 token 调用 HTTP 接口或连接 WebSocket 都会被拒绝。

- Redis 可用时：黑名单跨进程生效，并按 token 剩余有效期自动过期。
- Redis 不可用时：使用当前进程内存兜底，服务重启后失效。

成功响应：

```json
{
  "msg": "logged out"
}
```

前端仍需要自行关闭 WebSocket，并清理本地 `token`、`user_id`、`username`。

### 用户列表和在线状态

```http
GET /api/users
```

是否鉴权：是

说明：返回除当前用户以外的用户列表。

成功响应：

```json
[
  {
    "id": 2,
    "username": "bob",
    "online": true
  },
  {
    "id": 3,
    "username": "carol",
    "online": false
  }
]
```

## 消息接口

### 获取聊天历史

```http
GET /api/messages/history?target_id=2&is_group=false&before_id=0&limit=20
```

是否鉴权：是

查询参数：

| 参数 | 类型 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- | --- |
| target_id | number | 是 | 无 | 单聊时为对方用户 ID；群聊时为群 ID |
| is_group | boolean | 是 | 无 | `false` 单聊，`true` 群聊 |
| before_id | number | 否 | `0` | 分页游标，只返回 `id < before_id` 的消息 |
| limit | number | 否 | `20` | 每页条数，最大 `100` |

成功响应：

```json
[
  {
    "id": 10,
    "sender_id": 1,
    "receiver_id": 2,
    "content": "hello",
    "is_group": false,
    "is_read": false,
    "created_at": "2026-06-13T14:30:00+08:00"
  }
]
```

排序说明：后端按 `id DESC` 查询后反转，因此响应数组按消息 ID 从小到大排列。加载更早消息时，传当前列表最小消息 ID 作为 `before_id`。

失败示例：

```json
{
  "error": "invalid target_id"
}
```

### 获取单聊未读数量

```http
GET /api/messages/unread
```

是否鉴权：是

说明：旧版兼容接口，只返回单聊未读数量。对象 key 是发送者用户 ID，JSON 中会表现为字符串。

成功响应：

```json
{
  "2": 3,
  "5": 1
}
```

### 获取未读数量汇总

```http
GET /api/messages/unread/summary
```

是否鉴权：是

说明：结构化返回单聊和群聊未读数，建议新前端优先使用这个接口。

成功响应：

```json
{
  "users": {
    "2": 3,
    "5": 1
  },
  "groups": {
    "1": 8,
    "3": 2
  }
}
```

注意：群未读数当前基于 Redis 维护；Redis 不可用时，`groups` 可能为空对象。

### 标记会话已读

```http
POST /api/messages/read
```

是否鉴权：是

请求体：

```json
{
  "target_id": 2,
  "is_group": false
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| target_id | number | 是 | 单聊时为对方用户 ID；群聊时为群 ID |
| is_group | boolean | 是 | 是否群聊 |

成功响应：

```json
{
  "msg": "marked as read"
}
```

行为说明：

- 单聊：把 `sender_id = target_id` 且 `receiver_id = 当前用户` 的未读消息标记为已读，并清理 Redis 单聊未读计数。
- 群聊：清理 Redis 中当前用户对该群的未读计数。

### 撤回消息

```http
POST /api/messages/revoke?message_id=10
```

是否鉴权：是

说明：只有消息发送者本人可以撤回。当前实现为数据库硬删除，并通过 WebSocket 广播撤回事件。

成功响应：

```json
{
  "msg": "message revoked"
}
```

失败示例：

```json
{
  "error": "only sender can revoke message"
}
```

WebSocket 撤回事件：

```json
{
  "type": "revoke",
  "message_id": 10,
  "sender_id": 1,
  "receiver_id": 2,
  "is_group": false
}
```

## 群聊接口

### 创建群

```http
POST /api/groups
```

是否鉴权：是

请求体：

```json
{
  "name": "项目讨论组"
}
```

成功响应：

```json
{
  "id": 1,
  "name": "项目讨论组",
  "created_by": 1,
  "created_at": "2026-06-13T14:30:00+08:00"
}
```

说明：创建者会自动加入该群。

### 获取我的群列表

```http
GET /api/groups
```

是否鉴权：是

成功响应：

```json
[
  {
    "id": 1,
    "name": "项目讨论组",
    "created_by": 1,
    "member_count": 3
  }
]
```

### 加入群

```http
POST /api/groups/join
```

是否鉴权：是

请求体：

```json
{
  "group_id": 1
}
```

成功响应：

```json
{
  "id": 1,
  "name": "项目讨论组",
  "created_by": 1,
  "created_at": "2026-06-13T14:30:00+08:00"
}
```

说明：重复加入同一个群不会创建重复成员记录。

### 退出群

```http
POST /api/groups/leave
```

是否鉴权：是

请求体：

```json
{
  "group_id": 1
}
```

成功响应：

```json
{
  "msg": "left group"
}
```

失败示例：

```json
{
  "error": "group membership not found"
}
```

### 获取群成员

```http
GET /api/groups/1/members
```

是否鉴权：是

成功响应：

```json
[
  {
    "id": 1,
    "username": "alice",
    "online": true
  },
  {
    "id": 2,
    "username": "bob",
    "online": false
  }
]
```

## WebSocket 接口

### 建立连接

```http
GET /ws?token=<jwt-token>
```

是否鉴权：是，通过查询参数 `token` 鉴权。

前端示例：

```js
const ws = new WebSocket(`ws://localhost:8181/ws?token=${encodeURIComponent(token)}`)
```

连接失败：

| 状态码 | 响应文本 | 说明 |
| --- | --- | --- |
| 401 | `missing token` | 未传 token |
| 401 | `invalid token` | token 无效、过期或已退出登录 |

### 发送单聊消息

前端发送：

```json
{
  "receiver_id": 2,
  "content": "hello",
  "is_group": false
}
```

说明：前端发送的 `sender_id` 会被后端忽略，后端以 token 中的用户 ID 为准。

接收方收到：

```json
{
  "id": 10,
  "sender_id": 1,
  "receiver_id": 2,
  "content": "hello",
  "is_group": false,
  "is_read": false,
  "created_at": "2026-06-13T14:30:00+08:00"
}
```

### 发送群聊消息

前端发送：

```json
{
  "receiver_id": 1,
  "content": "hello group",
  "is_group": true
}
```

说明：群聊时 `receiver_id` 表示群 ID。服务端会向群成员推送消息，不会回推给发送者本人。

群成员收到：

```json
{
  "id": 11,
  "sender_id": 1,
  "receiver_id": 1,
  "content": "hello group",
  "is_group": true,
  "is_read": false,
  "created_at": "2026-06-13T14:31:00+08:00"
}
```

### 在线状态广播

用户上线或下线时，服务端会向所有在线 WebSocket 客户端广播：

```json
{
  "type": "online",
  "onlineUser": {
    "1": "alice",
    "2": "bob"
  }
}
```

### 离线消息推送

WebSocket 建立连接后，后端会查询当前用户数据库中的未读单聊消息并推送给客户端：

```json
{
  "id": 10,
  "sender_id": 1,
  "receiver_id": 2,
  "content": "offline message",
  "is_group": false,
  "is_read": false,
  "created_at": "2026-06-13T14:30:00+08:00"
}
```

### 撤回事件

消息撤回后，相关客户端会收到：

```json
{
  "type": "revoke",
  "message_id": 10,
  "sender_id": 1,
  "receiver_id": 2,
  "is_group": false
}
```

### 心跳

服务端每 50 秒发送 WebSocket `ping` 帧；客户端需要正常响应 `pong`，否则连接可能被关闭。

## 数据模型字段

### Message

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | number | 消息 ID |
| sender_id | number | 发送者用户 ID |
| receiver_id | number | 单聊为接收者用户 ID；群聊为群 ID |
| content | string | 消息内容 |
| is_group | boolean | 是否群聊消息 |
| is_read | boolean | 是否已读 |
| created_at | string | 创建时间 |

### Group

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | number | 群 ID |
| name | string | 群名称 |
| created_by | number | 创建者用户 ID |
| created_at | string | 创建时间 |

## 数据迁移

启动时会自动迁移：

- `users`
- `messages`
- `groups`
- `group_members`

## 前端适配建议

- 登录成功后保存 `token`、`user_id`、`username`。
- REST 请求统一添加 `Authorization: Bearer <token>`。
- WebSocket 使用 `/ws?token=<token>` 连接。
- 新前端建议使用 `/api/messages/unread/summary`，旧的 `/api/messages/unread` 只适合单聊。
- 消息列表建议以 `id` 作为主键。
- 在线状态可以先使用 `/api/users` 的 `online` 字段初始化，再用 WebSocket `online` 广播实时更新。
- 调用 `/api/logout` 后，前端仍要关闭 WebSocket 并清理本地登录态。
