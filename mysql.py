import pymysql
from datetime import datetime

# 源数据库连接信息
source_db_config = {
    "host": "103.36.220.124",
    "port": 3306,
    "user": "user",
    "password": "4GbrmrLnL5nBY8GG",
    "database": "user",
    "charset": "utf8mb4"
}

# 目标数据库连接信息
target_db_config = {
    "host": "103.36.220.124",
    "port": 3306,
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
                    user.get('email', ''),
                    group_id,
                    user.get('encrypted_identity_info', ''),
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
                token, tunnel_count, bandwidth, traffic_quota, checkin_count, continuity_checkin)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """
                
                register_time = user.get('created_at', now)
                
                target_cursor.execute(query, (
                    user.get('username', ''),
                    user.get('password', ''),
                    user.get('email', ''),
                    register_time,
                    group_id,
                    is_verified,
                    user.get('encrypted_identity_info', ''),
                    user.get('auth_count', 0),
                    1,  # status默认为1（启用）
                    register_time,
                    now,
                    group_time,
                    user.get('token', ''),
                    0,  # tunnel_count默认为0
                    0,  # bandwidth默认为0
                    0,  # traffic_quota默认为0
                    0,  # checkin_count默认为0
                    0   # continuity_checkin默认为0
                ))
        
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

if __name__ == "__main__":
    migrate_users()
