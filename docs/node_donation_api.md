# 节点捐赠API文档

本文档描述了StellarFrp中与节点捐赠相关的API接口。

## 用户接口

### 捐赠节点

用户可以通过此接口捐赠自己的节点给StellarFrp使用。节点将处于待审核状态，直到管理员批准。

- **URL**: `/api/v1/nodes/donate`
- **方法**: `POST`
- **需要认证**: 是
- **请求头**:
  - `Authorization`: 用户的授权令牌

**请求参数**:

```json
{
  "node_name": "节点名称",
  "description": "节点描述",
  "bandwidth": "节点带宽，如100Mbps",
  "ip": "节点IP地址",
  "server_port": 7000,
  "panel_url": "面板地址，如http://example.com:7500",
  "port_range": "端口范围，如10000-20000",
  "allowed_types": ["TCP", "UDP"],
  "permission": ["1", "2"],
  "token": "frps_token",
  "user": "frps_user"
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 描述 |
| --- | --- | --- | --- |
| node_name | 字符串 | 是 | 节点名称 |
| description | 字符串 | 是 | 节点描述 |
| bandwidth | 字符串 | 是 | 节点带宽，如100Mbps |
| ip | 字符串 | 是 | 节点IP地址 |
| server_port | 整数 | 是 | 服务端口，默认7000 |
| panel_url | 字符串 | 是 | 面板地址，如http://example.com:7500 |
| port_range | 字符串 | 是 | 端口范围，如10000-20000 |
| allowed_types | 字符串数组 | 是 | 允许的协议类型，如["TCP", "UDP"] |
| permission | 字符串数组 | 是 | 允许访问的权限组ID |
| token | 字符串 | 是 | frps的Token |
| user | 字符串 | 是 | frps的用户名 |

**成功响应**:

```json
{
  "code": 200,
  "msg": "节点捐赠成功，请等待管理员审核",
  "data": {
    "node_id": 123,
    "node_name": "节点名称",
    "status": "待审核"
  }
}
```

**错误响应**:

```json
{
  "code": 400,
  "msg": "参数错误: ..."
}
```

或

```json
{
  "code": 401,
  "msg": "未授权，请先登录"
}
```

### 获取用户自己的节点

用户可以通过此接口获取自己捐赠的所有节点及其状态。

- **URL**: `/api/v1/nodes/my`
- **方法**: `GET`
- **需要认证**: 是
- **请求头**:
  - `Authorization`: 用户的授权令牌
- **查询参数**:
  - `page`: 页码，默认为1
  - `page_size`: 每页数量，默认为10

**成功响应**:

```json
{
  "code": 200,
  "msg": "获取成功",
  "pagination": {
    "page": 1,
    "page_size": 10,
    "pages": 2,
    "total": 15
  },
  "nodes": [
    {
      "id": 123,
      "node_name": "节点名称",
      "allowed_types": ["TCP", "UDP"],
      "port_range": "10000-20000",
      "description": {
        "bandwidth": "100Mbps",
        "donated_by": "username"
      },
      "status": 2,
      "status_desc": "待审核",
      "created_at": "2023-01-01 12:00:00"
    }
  ]
}
```

**错误响应**:

```json
{
  "code": 400,
  "msg": "无效的页码"
}
```

或

```json
{
  "code": 401,
  "msg": "未授权，请先登录"
}
```

或

```json
{
  "code": 500,
  "msg": "服务器内部错误"
}
```

## 管理员接口

### 获取待审核的捐赠节点列表

管理员可以通过此接口获取所有待审核的捐赠节点。

- **URL**: `/api/v1/admin/nodes/donated`
- **方法**: `GET`
- **需要认证**: 是 (管理员)
- **请求头**:
  - `Authorization`: 管理员的授权令牌

**成功响应**:

```json
{
  "code": 200,
  "msg": "获取成功",
  "nodes": [
    {
      "id": 123,
      "node_name": "节点名称",
      "frps_port": 7000,
      "url": "http://example.com:7500",
      "token": "frps_token",
      "user": "frps_user",
      "description": "{\"bandwidth\":\"100Mbps\",\"donated_by\":\"username\"}",
      "permission": "[\"1\",\"2\"]",
      "allowed_types": "[\"TCP\",\"UDP\"]",
      "host": "节点描述",
      "port_range": "10000-20000",
      "ip": "1.2.3.4",
      "status": 2,
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z"
    }
  ]
}
```

### 审核捐赠节点

管理员可以通过此接口审核捐赠节点，可以选择批准或拒绝。

- **URL**: `/api/v1/admin/nodes/review`
- **方法**: `POST`
- **需要认证**: 是 (管理员)
- **请求头**:
  - `Authorization`: 管理员的授权令牌

**请求参数**:

```json
{
  "id": 123,
  "action": "approve"
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 描述 |
| --- | --- | --- | --- |
| id | 整数 | 是 | 节点ID |
| action | 字符串 | 是 | 操作类型，`approve`(批准)或`reject`(拒绝) |

**批准节点的成功响应**:

```json
{
  "code": 200,
  "msg": "节点审核通过",
  "data": {
    "id": 123,
    "node_name": "节点名称",
    "frps_port": 7000,
    "url": "http://example.com:7500",
    "token": "frps_token",
    "user": "frps_user",
    "description": "{\"bandwidth\":\"100Mbps\",\"donated_by\":\"username\"}",
    "permission": "[\"1\",\"2\"]",
    "allowed_types": "[\"TCP\",\"UDP\"]",
    "host": "节点描述",
    "port_range": "10000-20000",
    "ip": "1.2.3.4",
    "status": 1,
    "created_at": "2023-01-01T12:00:00Z",
    "updated_at": "2023-01-01T12:00:00Z"
  }
}
```

**拒绝节点的成功响应**:

```json
{
  "code": 200,
  "msg": "已拒绝并删除该捐赠节点"
}
```

**错误响应**:

```json
{
  "code": 400,
  "msg": "参数错误: ..."
}
```

或

```json
{
  "code": 401,
  "msg": "未授权，请先登录"
}
```

或

```json
{
  "code": 404,
  "msg": "节点不存在"
}
```

## 节点状态说明

| 状态码 | 说明 |
| --- | --- |
| 0 | 异常 |
| 1 | 启用 |
| 2 | 待审核 |
| 3 | 禁用 | 