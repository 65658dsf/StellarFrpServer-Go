# 管理员商品与订单API文档

本文档详细描述了StellarFrp管理员商品与订单管理系统的API接口，包括商品管理和订单管理功能。

## 目录

- [管理员商品与订单API文档](#管理员商品与订单api文档)
  - [目录](#目录)
  - [API概览](#api概览)
  - [商品管理](#商品管理)
    - [获取商品列表](#获取商品列表)
    - [获取单个商品信息](#获取单个商品信息)
    - [创建商品](#创建商品)
    - [更新商品](#更新商品)
    - [删除商品](#删除商品)
  - [订单管理](#订单管理)
    - [获取订单列表](#获取订单列表)
    - [获取单个订单信息](#获取单个订单信息)
    - [更新订单状态](#更新订单状态)
    - [删除订单](#删除订单)

## API概览

| 接口 | 方法 | 路径 | 描述 |
|-----|-----|-----|-----|
| 获取商品列表 | GET | /api/v1/admin/products | 获取所有商品 |
| 获取单个商品信息 | GET | /api/v1/admin/products/:id | 获取指定ID的商品信息 |
| 创建商品 | POST | /api/v1/admin/products/create | 创建新商品 |
| 更新商品 | POST | /api/v1/admin/products/update | 更新商品信息 |
| 删除商品 | POST | /api/v1/admin/products/delete | 删除指定ID的商品 |
| 获取订单列表 | GET | /api/v1/admin/orders | 获取所有订单 |
| 获取单个订单信息 | GET | /api/v1/admin/orders/:order_no | 获取指定订单号的订单信息 |
| 更新订单状态 | POST | /api/v1/admin/orders/update | 更新订单状态 |
| 删除订单 | POST | /api/v1/admin/orders/delete | 删除指定订单号的订单 |

## 商品管理

### 获取商品列表

获取所有商品列表，支持分页。

**请求**

```
GET /api/v1/admin/products?page=1&page_size=10
```

**参数**

| 参数 | 类型 | 必须 | 描述 |
|-----|-----|-----|-----|
| page | int | 否 | 页码，默认为1 |
| page_size | int | 否 | 每页记录数，默认为10，最大为50 |

**响应**

```json
{
  "code": 200,
  "msg": "获取商品列表成功",
  "products": [
    {
      "id": 1,
      "sku_id": "088f8490bf5711efaa1652540025c377",
      "name": "实名认证次数",
      "description": "购买后可增加1次实名认证机会",
      "price": 1.00,
      "plan_id": "0886a92ebf5711efa5c552540025c377",
      "is_active": true,
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z",
      "reward_action": "ADD_VERIFY_COUNT",
      "reward_value": "1"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 10,
    "pages": 1,
    "total": 1
  }
}
```

### 获取单个商品信息

获取指定ID的商品信息。

**请求**

```
GET /api/v1/admin/products/:id
```

**响应**

```json
{
  "code": 200,
  "msg": "获取商品信息成功",
  "product": {
    "id": 1,
    "sku_id": "088f8490bf5711efaa1652540025c377",
    "name": "实名认证次数",
    "description": "购买后可增加1次实名认证机会",
    "price": 1.00,
    "plan_id": "0886a92ebf5711efa5c552540025c377",
    "is_active": true,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z",
    "reward_action": "ADD_VERIFY_COUNT",
    "reward_value": "1"
  }
}
```

### 创建商品

创建新商品。

**请求**

```
POST /api/v1/admin/products/create
Content-Type: application/json
```

**请求体**

```json
{
  "sku_id": "088f8490bf5711efaa1652540025c377",
  "name": "实名认证次数",
  "description": "购买后可增加1次实名认证机会",
  "price": 1.00,
  "plan_id": "0886a92ebf5711efa5c552540025c377",
  "is_active": true,
  "reward_action": "ADD_VERIFY_COUNT",
  "reward_value": "1"
}
```

**响应**

```json
{
  "code": 200,
  "msg": "创建商品成功",
  "product": {
    "id": 1,
    "sku_id": "088f8490bf5711efaa1652540025c377",
    "name": "实名认证次数",
    "description": "购买后可增加1次实名认证机会",
    "price": 1.00,
    "plan_id": "0886a92ebf5711efa5c552540025c377",
    "is_active": true,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z",
    "reward_action": "ADD_VERIFY_COUNT",
    "reward_value": "1"
  }
}
```

### 更新商品

更新商品信息。

**请求**

```
POST /api/v1/admin/products/update
Content-Type: application/json
```

**请求体**

```json
{
  "id": 1,
  "sku_id": "088f8490bf5711efaa1652540025c377",
  "name": "实名认证次数（更新）",
  "description": "购买后可增加1次实名认证机会",
  "price": 1.50,
  "plan_id": "0886a92ebf5711efa5c552540025c377",
  "is_active": true,
  "reward_action": "ADD_VERIFY_COUNT",
  "reward_value": "1"
}
```

**响应**

```json
{
  "code": 200,
  "msg": "更新商品成功",
  "product": {
    "id": 1,
    "sku_id": "088f8490bf5711efaa1652540025c377",
    "name": "实名认证次数（更新）",
    "description": "购买后可增加1次实名认证机会",
    "price": 1.50,
    "plan_id": "0886a92ebf5711efa5c552540025c377",
    "is_active": true,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:10:00Z",
    "reward_action": "ADD_VERIFY_COUNT",
    "reward_value": "1"
  }
}
```

### 删除商品

删除指定ID的商品。

**请求**

```
POST /api/v1/admin/products/delete
Content-Type: application/json
```

**请求体**

```json
{
  "id": 1
}
```

**响应**

```json
{
  "code": 200,
  "msg": "删除商品成功"
}
```

## 订单管理

### 获取订单列表

获取所有订单列表，支持分页和过滤。

**请求**

```
GET /api/v1/admin/orders?page=1&page_size=10&user_id=123&status=0&order_no=SFP
```

**参数**

| 参数 | 类型 | 必须 | 描述 |
|-----|-----|-----|-----|
| page | int | 否 | 页码，默认为1 |
| page_size | int | 否 | 每页记录数，默认为10 |
| user_id | int | 否 | 用户ID过滤 |
| status | int | 否 | 订单状态过滤 |
| order_no | string | 否 | 订单号模糊搜索 |

**响应**

```json
{
  "code": 200,
  "msg": "获取订单列表成功",
  "orders": [
    {
      "id": 1,
      "order_no": "SFP20230101000000xxxxxxxx",
      "user_id": 123,
      "product_id": 1,
      "product_sku_id": "088f8490bf5711efaa1652540025c377",
      "product_name": "实名认证次数",
      "amount": 1.00,
      "status": 0,
      "remark": "username|SFP20230101000000xxxxxxxx",
      "afdian_trade_no": "",
      "reward_action": "ADD_VERIFY_COUNT",
      "reward_value": "1",
      "reward_executed": false,
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z",
      "paid_at": null
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 10,
    "pages": 1,
    "total": 1
  }
}
```

### 获取单个订单信息

获取指定订单号的订单信息。

**请求**

```
GET /api/v1/admin/orders/:order_no
```

**响应**

```json
{
  "code": 200,
  "msg": "获取订单信息成功",
  "order": {
    "id": 1,
    "order_no": "SFP20230101000000xxxxxxxx",
    "user_id": 123,
    "product_id": 1,
    "product_sku_id": "088f8490bf5711efaa1652540025c377",
    "product_name": "实名认证次数",
    "amount": 1.00,
    "status": 0,
    "remark": "username|SFP20230101000000xxxxxxxx",
    "afdian_trade_no": "",
    "reward_action": "ADD_VERIFY_COUNT",
    "reward_value": "1",
    "reward_executed": false,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z",
    "paid_at": null
  }
}
```

### 更新订单状态

更新订单状态，如果将状态更新为已支付(1)，系统会自动执行奖励。

**请求**

```
POST /api/v1/admin/orders/update
Content-Type: application/json
```

**请求体**

```json
{
  "order_no": "SFP20230101000000xxxxxxxx",
  "status": 1
}
```

**响应**

```json
{
  "code": 200,
  "msg": "更新订单状态成功"
}
```

**订单状态说明**

| 状态码 | 描述 |
|-----|-----|
| 0 | 待支付 |
| 1 | 已支付 |
| 2 | 已取消 |
| 3 | 已退款 |

### 删除订单

删除指定订单号的订单。

**请求**

```
POST /api/v1/admin/orders/delete
Content-Type: application/json
```

**请求体**

```json
{
  "order_no": "SFP20230101000000xxxxxxxx"
}
```

**响应**

```json
{
  "code": 200,
  "msg": "删除订单成功"
}
``` 