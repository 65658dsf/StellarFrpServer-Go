# 商品API文档

本文档详细描述了StellarFrp商品系统的API接口，包括商品列表、订单创建、订单查询和爱发电Webhook回调等功能。

## 目录

- [商品API文档](#商品api文档)
  - [目录](#目录)
  - [API概览](#api概览)
  - [公共接口](#公共接口)
    - [获取商品列表](#获取商品列表)
  - [需要认证的接口](#需要认证的接口)
    - [创建订单链接](#创建订单链接)
    - [获取用户订单列表](#获取用户订单列表)
    - [查询订单状态](#查询订单状态)
  - [爱发电集成](#爱发电集成)
    - [Webhook回调](#webhook回调)
  - [数据结构](#数据结构)
    - [商品(Product)](#商品product)
    - [订单(Order)](#订单order)
  - [奖励机制](#奖励机制)
    - [实名认证次数](#实名认证次数)
    - [会员组升级](#会员组升级)

## API概览

| 接口 | 方法 | 路径 | 认证 | 描述 |
|-----|-----|-----|-----|-----|
| 获取商品列表 | GET | /api/v1/shop/products | 否 | 获取所有上架的商品 |
| 创建订单链接 | POST | /api/v1/shop/order/create | 是 | 创建爱发电订单链接 |
| 获取用户订单列表 | GET | /api/v1/shop/orders | 是 | 获取当前用户的所有订单 |
| 查询订单状态 | GET | /api/v1/shop/order/status | 是 | 查询指定订单的状态 |
| 爱发电Webhook回调 | POST | /api/v1/afdian/webhook | 否 | 接收爱发电的支付通知 |

## 公共接口

### 获取商品列表

获取所有上架销售的商品列表。

**请求**

```
GET /api/v1/shop/products
```

**响应**

```json
{
  "code": 200,
  "message": "获取商品列表成功",
  "data": [
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
    },
    {
      "id": 2,
      "sku_id": "2389400c866711efa10552540025c377",
      "name": "白银会员(1个月)",
      "description": "购买后升级为白银会员，享受更多权益，有效期1个月",
      "price": 15.00,
      "plan_id": "2380fa96866711efa15a52540025c377",
      "is_active": true,
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z",
      "reward_action": "UPGRADE_GROUP",
      "reward_value": "3"
    }
  ]
}
```

## 需要认证的接口

以下接口需要在请求头中携带有效的认证Token：

```
Authorization: your_token
```

### 创建订单链接

为指定商品创建爱发电订单链接。

**请求**

```
POST /api/v1/shop/order/create
Content-Type: application/json
```

**请求体**

```json
{
  "product_id": 1,
  "remark": "可选的订单备注"
}
```

**参数**

| 参数 | 类型 | 必须 | 描述 |
|-----|-----|-----|-----|
| product_id | int | 是 | 商品ID |
| remark | string | 否 | 订单备注，可选 |

**响应**

```json
{
  "code": 200,
  "message": "创建订单链接成功",
  "data": {
    "order_link": "https://ifdian.net/order/create?product_type=1&plan_id=0886a92ebf5711efa5c552540025c377&remark=username%7CSFPxxx&sku=%5B%7B%22sku_id%22%3A%22088f8490bf5711efaa1652540025c377%22%2C%22count%22%3A1%7D%5D&viokrz_ex=0",
    "order_no": "SFP20230101000000xxxxxxxx"
  }
}
```

### 获取用户订单列表

获取当前用户的所有订单。

**请求**

```
GET /api/v1/shop/orders?page=1&page_size=10
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
  "msg": "获取成功",
  "orders": [
    {
      "id": 1,
      "order_no": "SFP20230101000000xxxxxxxx",
      "user_id": 123,
      "product_id": 1,
      "product_sku_id": "088f8490bf5711efaa1652540025c377",
      "product_name": "实名认证次数",
      "amount": 1.00,
      "status": 1,
      "remark": "username|SFP20230101000000xxxxxxxx",
      "afdian_trade_no": "202301010000001234567890",
      "reward_action": "ADD_VERIFY_COUNT",
      "reward_value": "1",
      "reward_executed": true,
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:10:00Z",
      "paid_at": "2023-01-01T00:05:00Z"
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

### 查询订单状态

查询指定订单的状态。

**请求**

```
GET /api/v1/shop/order/status?order_no={order_no}
```

**参数**

| 参数 | 类型 | 必须 | 描述 |
|-----|-----|-----|-----|
| order_no | string | 是 | 订单号 |

**响应**

```json
{
  "code": 200,
  "message": "获取订单状态成功",
  "data": {
    "order_no": "SFP20230101000000xxxxxxxx",
    "status": 1,
    "amount": 1.00,
    "product_name": "实名认证次数",
    "created_at": "2023-01-01T00:00:00Z",
    "paid_at": "2023-01-01T00:05:00Z",
    "reward_executed": true
  }
}
```

**订单状态说明**

| 状态码 | 描述 |
|-----|-----|
| 0 | 待支付 |
| 1 | 已支付 |
| 2 | 已取消 |
| 3 | 已退款 |

## 爱发电集成

### Webhook回调

接收爱发电的支付通知。

**请求**

```
POST /api/v1/afdian/webhook
```

**请求体**

```json
{
  "ec": 200,
  "em": "ok",
  "data": {
    "type": "order",
    "order": {
      "out_trade_no": "202301010000001234567890",
      "custom_order_id": "",
      "user_id": "adf397fe8374811eaacee52540025c377",
      "user_private_id": "33",
      "plan_id": "0886a92ebf5711efa5c552540025c377",
      "month": 1,
      "total_amount": "1.00",
      "show_amount": "1.00",
      "status": 2,
      "remark": "username|SFP20230101000000xxxxxxxx",
      "redeem_id": "",
      "product_type": 0,
      "discount": "0.00",
      "sku_detail": [
        {
          "sku_id": "088f8490bf5711efaa1652540025c377",
          "count": 1,
          "name": "实名认证次数",
          "album_id": "",
          "pic": ""
        }
      ],
      "address_person": "",
      "address_phone": "",
      "address_address": ""
    }
  }
}
```

**爱发电订单状态说明**

| 状态码 | 描述 |
|-----|-----|
| 1 | 未支付 |
| 2 | 已支付 |

**响应**

```json
{
  "ec": 200,
  "em": ""
}
```

## 数据结构

### 商品(Product)

| 字段 | 类型 | 描述 |
|-----|-----|-----|
| id | uint64 | 商品ID |
| sku_id | string | 商品SKU ID，爱发电商品唯一标识 |
| name | string | 商品名称 |
| description | string | 商品描述 |
| price | float64 | 商品价格 |
| plan_id | string | 爱发电套餐ID |
| is_active | bool | 是否上架销售 |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |
| reward_action | string | 奖励动作类型 |
| reward_value | string | 奖励动作的值 |

### 订单(Order)

| 字段 | 类型 | 描述 |
|-----|-----|-----|
| id | uint64 | 订单ID |
| order_no | string | 订单号 |
| user_id | uint64 | 用户ID |
| product_id | uint64 | 商品ID |
| product_sku_id | string | 商品SKU ID |
| product_name | string | 商品名称 |
| amount | float64 | 订单金额 |
| status | int | 订单状态：0待支付，1已支付，2已取消，3已退款 |
| remark | string | 订单备注 |
| afdian_trade_no | string | 爱发电交易号 |
| reward_action | string | 奖励动作 |
| reward_value | string | 奖励值 |
| reward_executed | bool | 奖励是否已执行 |
| created_at | time | 创建时间 |
| updated_at | time | 更新时间 |
| paid_at | time | 支付时间 |

## 奖励机制

系统支持基于"动作"的奖励系统，根据商品的`reward_action`和`reward_value`执行不同的奖励逻辑。

### 实名认证次数

- **动作类型**: `ADD_VERIFY_COUNT`
- **值格式**: 数字，表示增加的实名认证次数
- **示例**: `"1"` 表示增加1次实名认证机会

### 会员组升级

- **动作类型**: `UPGRADE_GROUP`
- **值格式**: 数字，表示用户组ID
- **示例**: `"3"` 表示升级到ID为3的用户组（白银会员）
- **特殊逻辑**: 
  - 如果用户已经是会员且未过期，则在现有会员期限基础上增加一个月
  - 如果用户不是会员或会员已过期，则从当前时间开始计算一个月的会员期限 