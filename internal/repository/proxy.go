package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// Proxy 隧道模型
type Proxy struct {
	ID                int64  `db:"id" json:"id"`
	Username          string `db:"username" json:"username"`
	ProxyName         string `db:"proxy_name" json:"proxy_name"`
	ProxyType         string `db:"proxy_type" json:"proxy_type"`
	LocalIP           string `db:"local_ip" json:"local_ip"`
	LocalPort         int    `db:"local_port" json:"local_port"`
	UseEncryption     string `db:"use_encryption" json:"use_encryption"`
	UseCompression    string `db:"use_compression" json:"use_compression"`
	Domain            string `db:"domain" json:"domain"`
	HostHeaderRewrite string `db:"host_header_rewrite" json:"host_header_rewrite"`
	RemotePort        string `db:"remote_port" json:"remote_port"`
	HeaderXFromWhere  string `db:"header_X-From-Where" json:"header_x_from_where"`
	Status            string `db:"status" json:"status"`
	LastUpdate        string `db:"lastupdate" json:"lastupdate"`
	Node              int64  `db:"node" json:"node"`
	RunID             string `db:"runID" json:"run_id"`
	TrafficQuota      int64  `db:"traffic_quota" json:"traffic_quota"`
}

// ProxyRepository 隧道仓库接口
type ProxyRepository interface {
	Create(ctx context.Context, proxy *Proxy) (int64, error)
	GetByID(ctx context.Context, id int64) (*Proxy, error)
	GetByUsername(ctx context.Context, username string) ([]*Proxy, error)
	GetByUsernameAndName(ctx context.Context, username, proxyName string) (*Proxy, error)
	Update(ctx context.Context, proxy *Proxy) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*Proxy, error)
	Count(ctx context.Context) (int, error)
	IsRemotePortUsed(ctx context.Context, nodeID int64, proxyType string, remotePort string) (bool, error)
}

// proxyRepository 隧道仓库实现
type proxyRepository struct {
	db *sqlx.DB
}

// NewProxyRepository 创建隧道仓库实例
func NewProxyRepository(db *sqlx.DB) ProxyRepository {
	return &proxyRepository{db: db}
}

// Create 创建隧道
func (r *proxyRepository) Create(ctx context.Context, proxy *Proxy) (int64, error) {
	query := `INSERT INTO proxy 
	(username, proxy_name, proxy_type, local_ip, local_port, use_encryption, use_compression, 
	domain, host_header_rewrite, remote_port, ` + "`header_X-From-Where`" + `, status, lastupdate, node, runID, traffic_quota) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	proxy.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
	proxy.Status = "offline" // 默认为未激活状态

	res, err := r.db.ExecContext(ctx, query,
		proxy.Username, proxy.ProxyName, proxy.ProxyType, proxy.LocalIP, proxy.LocalPort,
		proxy.UseEncryption, proxy.UseCompression, proxy.Domain, proxy.HostHeaderRewrite,
		proxy.RemotePort, proxy.HeaderXFromWhere, proxy.Status, proxy.LastUpdate,
		proxy.Node, proxy.RunID, proxy.TrafficQuota)

	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

// GetByID 根据ID获取隧道
func (r *proxyRepository) GetByID(ctx context.Context, id int64) (*Proxy, error) {
	query := `SELECT * FROM proxy WHERE id = ?`
	var proxy Proxy
	err := r.db.GetContext(ctx, &proxy, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &proxy, nil
}

// GetByUsername 根据用户名获取隧道列表
func (r *proxyRepository) GetByUsername(ctx context.Context, username string) ([]*Proxy, error) {
	query := `SELECT * FROM proxy WHERE username = ?`
	var proxies []*Proxy
	err := r.db.SelectContext(ctx, &proxies, query, username)
	if err != nil {
		return nil, err
	}
	return proxies, nil
}

// GetByUsernameAndName 根据用户名和隧道名称获取隧道
func (r *proxyRepository) GetByUsernameAndName(ctx context.Context, username, proxyName string) (*Proxy, error) {
	query := `SELECT * FROM proxy WHERE username = ? AND proxy_name = ?`
	var proxy Proxy
	err := r.db.GetContext(ctx, &proxy, query, username, proxyName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &proxy, nil
}

// Update 更新隧道
func (r *proxyRepository) Update(ctx context.Context, proxy *Proxy) error {
	query := `UPDATE proxy SET 
	proxy_name = ?, proxy_type = ?, local_ip = ?, local_port = ?, 
	use_encryption = ?, use_compression = ?, domain = ?, host_header_rewrite = ?, 
	remote_port = ?, ` + "`header_X-From-Where`" + ` = ?, status = ?, lastupdate = ?, 
	node = ?, runID = ?, traffic_quota = ? 
	WHERE id = ?`

	proxy.LastUpdate = time.Now().Format("2006-01-02 15:04:05")

	_, err := r.db.ExecContext(ctx, query,
		proxy.ProxyName, proxy.ProxyType, proxy.LocalIP, proxy.LocalPort,
		proxy.UseEncryption, proxy.UseCompression, proxy.Domain, proxy.HostHeaderRewrite,
		proxy.RemotePort, proxy.HeaderXFromWhere, proxy.Status, proxy.LastUpdate,
		proxy.Node, proxy.RunID, proxy.TrafficQuota, proxy.ID)

	return err
}

// Delete 删除隧道
func (r *proxyRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM proxy WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List 获取隧道列表
func (r *proxyRepository) List(ctx context.Context, offset, limit int) ([]*Proxy, error) {
	query := `SELECT * FROM proxy LIMIT ? OFFSET ?`
	var proxies []*Proxy
	err := r.db.SelectContext(ctx, &proxies, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return proxies, nil
}

// Count 获取隧道总数
func (r *proxyRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM proxy`
	var count int
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// IsRemotePortUsed 检查同一节点下相同协议类型的隧道是否已经使用了相同的远程端口
func (r *proxyRepository) IsRemotePortUsed(ctx context.Context, nodeID int64, proxyType string, remotePort string) (bool, error) {
	query := `SELECT COUNT(*) FROM proxy WHERE node = ? AND proxy_type = ? AND remote_port = ?`
	var count int
	err := r.db.GetContext(ctx, &count, query, nodeID, proxyType, remotePort)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
