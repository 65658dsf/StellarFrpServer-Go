#!/usr/bin/env python
# -*- coding: utf-8 -*-

import requests
import json
import time
import hashlib
import urllib.parse

# 配置信息
API_BASE_URL = "http://localhost:8080/api/v1"  # 替换为实际的API地址
USER_TOKEN = "gznt4gwp6qdw49f82xgt4k6b56zvjwvv"  # 替换为实际的用户Token
AFDIAN_USER_ID = "164fe432651811eb845452540025c377"  # 替换为爱发电用户ID
AFDIAN_TOKEN = "MvbcJwGVC3eh47SkWDqpKyHUur6fTj9B"  # 替换为爱发电Token

# 1. 获取商品列表
def get_products():
    url = f"{API_BASE_URL}/shop/products"
    response = requests.get(url)
    if response.status_code == 200:
        data = response.json()
        if data.get("code") == 200:
            print("商品列表获取成功:")
            for product in data.get("data", []):
                print(f"ID: {product['id']}, 名称: {product['name']}, 价格: {product['price']}, SKU: {product['sku_id']}")
            return data.get("data", [])
        else:
            print(f"获取商品列表失败: {data.get('message')}")
    else:
        print(f"请求失败: {response.status_code}")
    return []

# 2. 创建实名认证订单链接
def create_order_link(product_id):
    url = f"{API_BASE_URL}/shop/order/create"
    headers = {
        "Authorization": f"{USER_TOKEN}",
        "Content-Type": "application/json"
    }
    payload = {
        "product_id": product_id,
        "remark": ""  # 可以添加自定义备注
    }
    
    response = requests.post(url, headers=headers, json=payload)
    if response.status_code == 200:
        data = response.json()
        if data.get("code") == 200:
            order_link = data.get("data", {}).get("order_link")
            order_no = data.get("data", {}).get("order_no")
            print(f"订单创建成功:")
            print(f"订单号: {order_no}")
            print(f"支付链接: {order_link}")
            return order_no, order_link
        else:
            print(f"创建订单失败: {data.get('message')}")
    else:
        print(f"请求失败: {response.status_code}")
    return None, None

# 3. 查询订单状态
def check_order_status(order_no):
    url = f"{API_BASE_URL}/shop/order/status"
    headers = {
        "Authorization": f"Bearer {USER_TOKEN}"
    }
    params = {
        "order_no": order_no
    }
    
    response = requests.get(url, headers=headers, params=params)
    if response.status_code == 200:
        data = response.json()
        if data.get("code") == 200:
            order_data = data.get("data", {})
            status_map = {
                0: "待支付",
                1: "已支付",
                2: "已取消",
                3: "已退款"
            }
            status = status_map.get(order_data.get("status"), "未知状态")
            print(f"订单状态: {status}")
            print(f"订单详情: {order_data}")
            return order_data
        else:
            print(f"查询订单失败: {data.get('message')}")
    else:
        print(f"请求失败: {response.status_code}")
    return None

# 主函数
def main():
    # 1. 获取商品列表
    products = get_products()
    if not products:
        print("没有可用的商品")
        return
    
    # 找到实名认证商品
    verify_product = None
    for product in products:
        if product.get("reward_action") == "ADD_VERIFY_COUNT":
            verify_product = product
            break
    
    if not verify_product:
        print("未找到实名认证商品")
        return
    
    # 2. 创建订单链接
    order_no, order_link = create_order_link(verify_product["id"])
    if not order_no:
        print("创建订单失败")
        return
    
    print("\n请访问上面的链接完成支付，或者按回车键模拟支付完成...")
    input()
    
    # 3. 检查订单状态（此时应该是待支付）
    order_data = check_order_status(order_no)
    
    # 4. 模拟爱发电回调（假设用户已支付）
    print("\n模拟爱发电支付回调...")
    # 从订单数据中获取remark
    remark = f"user|{order_no}"  # 这里简化处理，实际应该从订单中获取
    success = simulate_afdian_webhook(order_no, remark)
    
    if success:
        # 5. 再次检查订单状态（应该变为已支付）
        print("\n支付完成后的订单状态:")
        order_data = check_order_status(order_no)
        
        if order_data and order_data.get("status") == 1:
            print("\n实名认证次数应该已经增加，可以登录系统查看")

if __name__ == "__main__":
    main()
