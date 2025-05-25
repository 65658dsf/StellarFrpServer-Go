# StellarFrp管理员用户管理API使用教程

## 基本信息

所有管理员API均需要在请求头中添加`Authorization`字段，格式为`Bearer {token}`，其中`{token}`为管理员用户的token。

所有API的基础URL为：`/api/v1/admin`

所有API返回的数据格式为：
```json
{
  "code": 200,  // 状态码，200表示成功
  "msg": "操作成功",  // 状态信息
  "data": {}  // 返回的数据，可能是对象或数组
}
```

## 用户管理API

### 1. 获取用户列表

**接口说明**：获取系统中的用户列表，支持分页

**请求方式**：GET

**URL**：`/api/v1/admin/users`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|-------|-----|------|------|
| page | int | 否 | 页码，默认为1 |
| page_size | int | 否 | 每页数量，默认为10 |

**请求示例**：
```
GET /api/v1/admin/users?page=1&page_size=10
```

**响应示例**：
```json
{
  "code": 200,
  "msg": "获取成功",
  "pagination": {
    "page": 1,
    "page_size": 10,
    "pages": 2,
    "total": 17
  },
  "users": [
    {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "register_time": "2023-01-01T00:00:00Z",
      "group_id": 4,
      "group_name": "管理员",
      "is_verified": 0,
      "verify_info": "",
      "verify_count": 0,
      "status": 1,
      "group_time": null,
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z",
      "tunnel_count": 10,
      "bandwidth": 100,
      "traffic_quota": 1073741824,
      "total_tunnel_limit": 20,
      "total_bandwidth": 200,
      "last_checkin": null,
      "checkin_count": 0,
      "continuity_checkin": 0
    }
  ]
}
```

### 2. 获取单个用户信息

**接口说明**：根据用户ID获取用户详细信息

**请求方式**：GET

**URL**：`/api/v1/admin/users/:id`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|-------|-----|------|------|
| id | int | 是 | 用户ID，路径参数 |

**请求示例**：
```
GET /api/v1/admin/users/1
```

**响应示例**：
```json
{
  "code": 200,
  "msg": "获取成功",
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "register_time": "2023-01-01T00:00:00Z",
    "group_id": 4,
    "group_name": "管理员",
    "is_verified": 0,
    "verify_info": "",
    "verify_count": 0,
    "status": 1,
    "group_time": null,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z",
    "tunnel_count": 10,
    "bandwidth": 100,
    "traffic_quota": 1073741824,
    "total_tunnel_limit": 20,
    "total_bandwidth": 200,
    "last_checkin": null,
    "checkin_count": 0,
    "continuity_checkin": 0
  }
}
```

### 3. 创建用户

**接口说明**：创建新用户

**请求方式**：POST

**URL**：`/api/v1/admin/users`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|-------|-----|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |
| email | string | 是 | 邮箱 |
| group_id | int | 是 | 用户组ID |
| status | int | 是 | 用户状态，1为正常，0为禁用 |
| tunnel_count | int | 否 | 隧道数量限制 |
| bandwidth | int | 否 | 带宽限制（Mbps） |
| traffic_quota | int64 | 否 | 流量配额（字节） |
| group_time | string | 否 | 用户组到期时间 |

**请求示例**：
```json
POST /api/v1/admin/users
Content-Type: application/json

{
  "username": "newuser",
  "password": "Password123",
  "email": "newuser@example.com",
  "group_id": 1,
  "status": 1,
  "tunnel_count": 5,
  "bandwidth": 50,
  "traffic_quota": 5368709120
}
```

**响应示例**：
```json
{
  "code": 200,
  "msg": "创建成功",
  "data": {
    "id": 10
  }
}
```

### 4. 更新用户

**接口说明**：更新用户信息

**请求方式**：PUT

**URL**：`/api/v1/admin/users/:id`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|-------|-----|------|------|
| id | int | 是 | 用户ID，路径参数 |
| username | string | 否 | 用户名 |
| password | string | 否 | 密码 |
| email | string | 否 | 邮箱 |
| group_id | int | 否 | 用户组ID |
| status | int | 否 | 用户状态，1为正常，0为禁用 |
| tunnel_count | int | 否 | 隧道数量限制 |
| bandwidth | int | 否 | 带宽限制（Mbps） |
| traffic_quota | int64 | 否 | 流量配额（字节） |
| group_time | string | 否 | 用户组到期时间 |

**请求示例**：
```json
PUT /api/v1/admin/users/10
Content-Type: application/json

{
  "status": 0,
  "group_id": 2,
  "tunnel_count": 10,
  "traffic_quota": 10737418240
}
```

**响应示例**：
```json
{
  "code": 200,
  "msg": "更新成功"
}
```

### 5. 删除用户

**接口说明**：删除用户

**请求方式**：DELETE

**URL**：`/api/v1/admin/users/:id`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|-------|-----|------|------|
| id | int | 是 | 用户ID，路径参数 |

**请求示例**：
```
DELETE /api/v1/admin/users/10
```

**响应示例**：
```json
{
  "code": 200,
  "msg": "删除成功"
}
```

### 6. 搜索用户

**接口说明**：根据关键字搜索用户

**请求方式**：GET

**URL**：`/api/v1/admin/users/search`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|-------|-----|------|------|
| keyword | string | 是 | 搜索关键字 |

**请求示例**：
```
GET /api/v1/admin/users/search?keyword=admin
```

**响应示例**：
```json
{
  "code": 200,
  "msg": "搜索成功",
  "data": [
    {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "register_time": "2023-01-01T00:00:00Z",
      "group_id": 4,
      "is_verified": 0,
      "verify_info": "",
      "verify_count": 0,
      "status": 1
    }
  ]
}
```

### 7. 重置用户Token

**接口说明**：重置用户的访问Token

**请求方式**：POST

**URL**：`/api/v1/admin/users/:id/reset-token`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|-------|-----|------|------|
| id | int | 是 | 用户ID，路径参数 |

**请求示例**：
```
POST /api/v1/admin/users/1/reset-token
```

**响应示例**：
```json
{
  "code": 200,
  "msg": "重置成功",
  "data": {
    "token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
  }
}
```

## 状态码说明

| 状态码 | 说明 |
|-------|------|
| 200 | 操作成功 |
| 400 | 参数错误 |
| 401 | 未授权或Token无效 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 409 | 资源冲突（如用户名或邮箱已存在） |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |

## 注意事项

1. 所有API都需要管理员权限（GroupID为4的用户）
2. 创建和更新用户时，密码会自动加密存储
3. 获取用户信息时，敏感信息如密码和Token会被过滤掉
4. 删除用户操作不可恢复，请谨慎操作
5. 重置Token会导致用户需要重新登录 