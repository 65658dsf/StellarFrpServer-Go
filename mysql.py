import pymysql
from datetime import datetime

# 源数据库连接信息
source_db_config = {
    "host": "47.120.77.71",
    "port": 8974,
    "user": "user",
    "password": "4GbrmrLnL5nBY8GG",
    "database": "user",
    "charset": "utf8mb4"
}

# 目标数据库连接信息
target_db_config = {
    "host": "47.120.77.71",
    "port": 8974,
    "user": "StellarFrp",
    "password": "XY5mjkfAMbn6zfJY",
    "database": "stellarfrp",
    "charset": "utf8mb4"
}

def migrate_users():
    # 连接源数据库
    source_conn = pymysql.connect(**source_db_config)
    source_cursor = source_conn.cursor(pymysql.cursors.DictCursor)
    
    # 连接目标数据库
    target_conn = pymysql.connect(**target_db_config)
    target_cursor = target_conn.cursor()
    
    try:
        # 获取源数据库中的所有用户
        source_cursor.execute("SELECT * FROM users")
        users = source_cursor.fetchall()
        
        # 处理每个用户并迁移到目标数据库
        for user in users:
            # 确定group_id
            group_id = 1  # 默认为1
            
            # 判断是否已实名认证
            is_verified = user.get('is_verified', 0)
            
            # 判断用户类型和VIP状态
            if user.get('type') == 'VIP':
                # 检查VIP是否过期
                vip_time = user.get('VIPTime')
                group_time = None
                
                if vip_time and isinstance(vip_time, datetime) and vip_time > datetime.now():
                    # VIP且未过期
                    group_id = 3
                    group_time = vip_time
                else:
                    # VIP已过期
                    group_id = 2 if is_verified else 1
            elif user.get('type') == 'normal':
                # 普通用户
                group_id = 2 if is_verified else 1
                group_time = None
            
            # 创建默认值
            now = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            
            # 确保 email 和 verify_info 不为 NULL
            email = user.get('email', '')
            if email is None:
                email = ''
                
            verify_info = user.get('encrypted_identity_info', '')
            if verify_info is None:
                verify_info = ''
            
            # 检查目标数据库中是否已存在该用户
            target_cursor.execute("SELECT id, token FROM users WHERE username = %s", (user['username'],))
            existing_user = target_cursor.fetchone()
            
            if existing_user:
                # 用户已存在，更新信息（不更新token）
                query = """
                UPDATE users 
                SET password = %s, email = %s, group_id = %s, 
                    verify_info = %s, is_verified = %s, verify_count = %s,
                    status = 1, updated_at = %s
                WHERE username = %s
                """
                
                target_cursor.execute(query, (
                    user.get('password', ''),
                    email,
                    group_id,
                    verify_info,
                    is_verified,
                    user.get('auth_count', 0),
                    now,
                    user['username']
                ))
                
                # 如果有VIP期限，更新group_time
                if group_time:
                    target_cursor.execute(
                        "UPDATE users SET group_time = %s WHERE username = %s",
                        (group_time, user['username'])
                    )
            else:
                # 用户不存在，插入新记录
                query = """
                INSERT INTO users 
                (username, password, email, register_time, group_id, is_verified, 
                verify_info, verify_count, status, created_at, updated_at, group_time,
                tunnel_count, bandwidth, traffic_quota, checkin_count, continuity_checkin)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """
                
                register_time = user.get('created_at', now)
                
                target_cursor.execute(query, (
                    user.get('username', ''),
                    user.get('password', ''),
                    email,
                    register_time,
                    group_id,
                    is_verified,
                    verify_info,
                    user.get('auth_count', 0),
                    1,  # status默认为1（启用）
                    register_time,
                    now,
                    group_time,
                    0,  # tunnel_count默认为0
                    0,  # bandwidth默认为0
                    0,  # traffic_quota默认为0
                    0,  # checkin_count默认为0
                    0   # continuity_checkin默认为0
                ))
        
        # 清空以"6dH+"开头的token
        clean_specific_tokens(target_conn)
        
        # 提交事务
        target_conn.commit()
        print(f"成功迁移 {len(users)} 个用户")
        
    except Exception as e:
        # 回滚事务
        target_conn.rollback()
        print(f"迁移过程中出错: {e}")
        
    finally:
        # 关闭连接
        source_cursor.close()
        source_conn.close()
        target_cursor.close()
        target_conn.close()

def clean_specific_tokens(conn):
    """清除以'6dH+'开头的token"""
    cursor = conn.cursor()
    try:
        # 查询所有token
        cursor.execute("SELECT id, token FROM users WHERE token IS NOT NULL AND token != ''")
        users = cursor.fetchall()
        
        cleaned_count = 0
        for user in users:
            user_id, token = user
            if token and token.startswith('6dH+'):
                cursor.execute("UPDATE users SET token = '' WHERE id = %s", (user_id,))
                cleaned_count += 1
        
        print(f"已清空 {cleaned_count} 个以'6dH+'开头的token")
    except Exception as e:
        print(f"清空token时出错: {e}")
        raise
    finally:
        cursor.close()

def consolidate_node_traffic():
    """合并节点流量记录，将所有节点的流量累加并移除多余记录"""
    # 连接数据库
    conn = pymysql.connect(**target_db_config)
    cursor = conn.cursor(pymysql.cursors.DictCursor)
    
    try:
        # 获取所有节点名称
        cursor.execute("SELECT DISTINCT node_name FROM node_traffic_log")
        nodes = cursor.fetchall()
        
        consolidated_count = 0
        removed_count = 0
        
        for node in nodes:
            node_name = node['node_name']
            print(f"处理节点: {node_name}")
            
            # 获取该节点的所有流量记录，按日期排序
            cursor.execute("""
                SELECT * FROM node_traffic_log 
                WHERE node_name = %s
                ORDER BY record_date ASC
            """, (node_name,))
            records = cursor.fetchall()
            
            if not records:
                print(f"节点 {node_name} 没有找到流量记录")
                continue
                
            # 获取最新的记录日期
            cursor.execute("""
                SELECT MAX(record_date) as latest_date FROM node_traffic_log 
                WHERE node_name = %s
            """, (node_name,))
            latest_date_result = cursor.fetchone()
            latest_date = latest_date_result['latest_date'] if latest_date_result else None
            
            if not latest_date:
                continue
                
            # 计算累计流量
            total_traffic_in = 0
            total_traffic_out = 0
            
            for record in records:
                total_traffic_in += record['traffic_in']
                total_traffic_out += record['traffic_out']
            
            # 创建或更新最新日期的累计记录
            cursor.execute("""
                SELECT id FROM node_traffic_log 
                WHERE node_name = %s AND record_date = %s
                LIMIT 1
            """, (node_name, latest_date))
            latest_record = cursor.fetchone()
            
            now = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            
            if latest_record:
                # 更新最新记录
                cursor.execute("""
                    UPDATE node_traffic_log 
                    SET traffic_in = %s, traffic_out = %s, record_time = %s, is_increment = 0
                    WHERE id = %s
                """, (total_traffic_in, total_traffic_out, now, latest_record['id']))
            else:
                # 创建新记录
                cursor.execute("""
                    INSERT INTO node_traffic_log 
                    (node_name, traffic_in, traffic_out, online_count, record_time, record_date, is_increment)
                    VALUES (%s, %s, %s, %s, %s, %s, %s)
                """, (node_name, total_traffic_in, total_traffic_out, 0, now, latest_date, 0))
            
            consolidated_count += 1
            
            # 删除除了最新日期外的所有记录
            cursor.execute("""
                DELETE FROM node_traffic_log 
                WHERE node_name = %s AND record_date != %s
            """, (node_name, latest_date))
            removed_count += cursor.rowcount
        
        # 提交事务
        conn.commit()
        print(f"成功处理 {consolidated_count} 个节点的流量记录")
        print(f"已移除 {removed_count} 条多余记录")
        
    except Exception as e:
        # 回滚事务
        conn.rollback()
        print(f"处理流量记录时出错: {e}")
        
    finally:
        # 关闭连接
        cursor.close()
        conn.close()

def update_schema_remove_is_increment():
    """更新数据库结构，移除is_increment字段"""
    conn = pymysql.connect(**target_db_config)
    cursor = conn.cursor()
    
    try:
        # 检查字段是否存在
        cursor.execute("""
            SELECT COUNT(*) 
            FROM information_schema.COLUMNS 
            WHERE TABLE_SCHEMA = %s 
            AND TABLE_NAME = 'node_traffic_log' 
            AND COLUMN_NAME = 'is_increment'
        """, (target_db_config['database'],))
        
        if cursor.fetchone()[0] > 0:
            # 字段存在，移除它
            print("正在移除is_increment字段...")
            cursor.execute("ALTER TABLE node_traffic_log DROP COLUMN is_increment")
            print("is_increment字段已成功移除")
        else:
            print("is_increment字段不存在，无需移除")
        
        conn.commit()
        
    except Exception as e:
        conn.rollback()
        print(f"移除字段时出错: {e}")
        
    finally:
        cursor.close()
        conn.close()

if __name__ == "__main__":
    print("1、迁移用户\n2、迁移隧道\n3、迁移节点\n4、合并节点流量记录\n5、移除is_increment字段")
    msg = input("请输入要迁移的序号:")
    if msg == "1":
        migrate_users()
    elif msg == "2":
        migrate_tunnels()
    elif msg == "3":
        migrate_nodes()
    elif msg == "4":
        consolidate_node_traffic()
    elif msg == "5":
        update_schema_remove_is_increment()
