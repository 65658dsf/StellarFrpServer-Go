# StellarFrp 权限组管理API文档

## 基本信息

- 基础路径: `/api/v1/admin/groups`
- 认证方式: 管理员认证（需要管理员权限）
- 响应格式: JSON

## 通用响应格式

所有API响应都遵循以下格式：

```json
{
  "code": 200,       // 状态码，200表示成功，其他值表示失败
  "msg": "成功",      // 响应消息
  "data": { ... }    // 响应数据，可能是对象或数组
}
```

## API列表

### 1. 获取权限组列表

获取所有权限组的列表。

- **路径**: `GET /api/v1/admin/groups`
- **方法**: GET
- **参数**: 无

**请求示例**:
```
GET /api/v1/admin/groups
```

**响应示例**:
```json
{
  "code": 200,
  "msg": "获取成功",
  "groups": [
    {
      "id": 1,
      "name": "未实名",
      "tunnel_limit": 2,
      "bandwidth_limit": 1,
      "traffic_quota": 10737418240,
      "checkin_min_traffic": 104857600,
      "checkin_max_traffic": 524288000,
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z"
    },
    {
      "id": 2,
      "name": "免费用户",
      "tunnel_limit": 5,
      "bandwidth_limit": 2,
      "traffic_quota": 32212254720,
      "checkin_min_traffic": 104857600,
      "checkin_max_traffic": 524288000,
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z"
    }
    // 更多权限组...
  ]
}
```

### 2. 获取单个权限组

根据ID获取单个权限组的详细信息。

- **路径**: `GET /api/v1/admin/groups/:id`
- **方法**: GET
- **参数**:
  - `:id`: 权限组ID (路径参数)

**请求示例**:
```
GET /api/v1/admin/groups/1
```

**响应示例**:
```json
{
  "code": 200,
  "msg": "获取成功",
  "data": {
    "id": 1,
    "name": "未实名",
    "tunnel_limit": 2,
    "bandwidth_limit": 1,
    "traffic_quota": 10737418240,
    "checkin_min_traffic": 104857600,
    "checkin_max_traffic": 524288000,
    "created_at": "2023-01-01T12:00:00Z",
    "updated_at": "2023-01-01T12:00:00Z"
  }
}
```

### 3. 创建权限组

创建新的权限组。

- **路径**: `POST /api/v1/admin/groups/create`
- **方法**: POST
- **Content-Type**: application/json
- **请求体参数**:

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|-----|------|
| name | string | 是 | 权限组名称 |
| tunnel_limit | int | 是 | 可创建隧道数量 |
| bandwidth_limit | int | 是 | 带宽限制(Mbps) |
| traffic_quota | int64 | 是 | 流量配额(字节) |
| checkin_min_traffic | int64 | 否 | 签到最小流量(字节) |
| checkin_max_traffic | int64 | 否 | 签到最大流量(字节) |

**请求示例**:
```json
{
  "name": "高级会员Plus",
  "tunnel_limit": 20,
  "bandwidth_limit": 10,
  "traffic_quota": 107374182400,
  "checkin_min_traffic": 209715200,
  "checkin_max_traffic": 1048576000
}
```

**响应示例**:
```json
{
  "code": 200,
  "msg": "创建成功",
  "data": {
    "id": 7,
    "name": "高级会员Plus",
    "tunnel_limit": 20,
    "bandwidth_limit": 10,
    "traffic_quota": 107374182400,
    "checkin_min_traffic": 209715200,
    "checkin_max_traffic": 1048576000,
    "created_at": "2023-05-15T08:30:00Z",
    "updated_at": "2023-05-15T08:30:00Z"
  }
}
```

### 4. 更新权限组

更新现有权限组的信息。

- **路径**: `POST /api/v1/admin/groups/update`
- **方法**: POST
- **Content-Type**: application/json
- **请求体参数**:

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|-----|------|
| id | int64 | 是 | 权限组ID |
| name | string | 否 | 权限组名称 |
| tunnel_limit | int | 否 | 可创建隧道数量 |
| bandwidth_limit | int | 否 | 带宽限制(Mbps) |
| traffic_quota | int64 | 否 | 流量配额(字节) |
| checkin_min_traffic | int64 | 否 | 签到最小流量(字节) |
| checkin_max_traffic | int64 | 否 | 签到最大流量(字节) |

**请求示例**:
```json
{
  "id": 7,
  "name": "高级会员Plus-更新",
  "bandwidth_limit": 15,
  "traffic_quota": 214748364800
}
```

**响应示例**:
```json
{
  "code": 200,
  "msg": "更新成功",
  "data": {
    "id": 7,
    "name": "高级会员Plus-更新",
    "tunnel_limit": 20,
    "bandwidth_limit": 15,
    "traffic_quota": 214748364800,
    "checkin_min_traffic": 209715200,
    "checkin_max_traffic": 1048576000,
    "created_at": "2023-05-15T08:30:00Z",
    "updated_at": "2023-05-15T09:15:00Z"
  }
}
```

### 5. 删除权限组

删除指定的权限组。

- **路径**: `POST /api/v1/admin/groups/delete`
- **方法**: POST
- **Content-Type**: application/json
- **请求体参数**:

| 参数名 | 类型 | 必填 | 描述 |
|-------|------|-----|------|
| id | int64 | 是 | 要删除的权限组ID |

**请求示例**:
```json
{
  "id": 7
}
```

**响应示例**:
```json
{
  "code": 200,
  "msg": "删除成功"
}
```

### 6. 搜索权限组

根据关键词搜索权限组。

- **路径**: `GET /api/v1/admin/groups/search`
- **方法**: GET
- **参数**:
  - `keyword`: 搜索关键词，会匹配权限组名称

**请求示例**:
```
GET /api/v1/admin/groups/search?keyword=会员
```

**响应示例**:
```json
{
  "code": 200,
  "msg": "搜索成功",
  "groups": [
    {
      "id": 3,
      "name": "普通会员",
      "tunnel_limit": 10,
      "bandwidth_limit": 4,
      "traffic_quota": 53687091200,
      "checkin_min_traffic": 104857600,
      "checkin_max_traffic": 524288000,
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z"
    },
    {
      "id": 4,
      "name": "高级会员",
      "tunnel_limit": 15,
      "bandwidth_limit": 6,
      "traffic_quota": 107374182400,
      "checkin_min_traffic": 104857600,
      "checkin_max_traffic": 524288000,
      "created_at": "2023-01-01T12:00:00Z",
      "updated_at": "2023-01-01T12:00:00Z"
    }
  ]
}
```

## 错误码说明

| 错误码 | 描述 |
|-------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## 注意事项

1. 所有API都需要管理员权限，请确保在请求头中包含有效的管理员认证信息。
2. 创建权限组时，请确保`name`在系统中唯一，否则会返回错误。
3. 更新权限组时，只需要提供需要更新的字段，不需要提供所有字段。
4. 删除权限组前，请确保该权限组下没有关联的用户，否则会返回错误。
5. 流量配额(traffic_quota)、签到最小流量(checkin_min_traffic)、签到最大流量(checkin_max_traffic)的单位都是字节(B)。例如，1GB = 1073741824字节。 