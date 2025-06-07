package repository

import (
	"context"
	"database/sql"
	"errors"
	"stellarfrp/internal/utils"
	"time"

	"github.com/jmoiron/sqlx"
)

// Node FRP节点模型
type Node struct {
	ID           int64          `db:"id"`
	NodeName     string         `db:"node_name"`
	FrpsPort     int            `db:"frps_port"`
	URL          string         `db:"url"`
	Token        string         `db:"token"`
	User         string         `db:"user"`
	Description  sql.NullString `db:"description"`
	Permission   string         `db:"permission"`    // JSON格式的字符串，如["1","2"]表示权限组IDs
	AllowedTypes string         `db:"allowed_types"` // JSON格式的字符串，如["TCP","UDP"]
	Host         sql.NullString `db:"host"`
	PortRange    string         `db:"port_range"`
	IP           string         `db:"ip"`
	Status       int            `db:"status"`
	OwnerID      sql.NullInt64  `db:"owner_id"` // 节点所属的用户ID，系统节点为null
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

// NodeRepository 节点仓库接口
type NodeRepository interface {
	Create(ctx context.Context, node *Node) error
	GetByID(ctx context.Context, id int64) (*Node, error)
	GetByNodeName(ctx context.Context, nodeName string) (*Node, error)
	GetByUser(ctx context.Context, user string) ([]*Node, error)
	GetByPermission(ctx context.Context, permission int64) ([]*Node, error)
	GetByOwnerID(ctx context.Context, ownerID int64) ([]*Node, error)
	Update(ctx context.Context, node *Node) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]*Node, error)
}

// nodeRepository 节点仓库实现
type nodeRepository struct {
	db *sqlx.DB
}

// NewNodeRepository 创建节点仓库实例
func NewNodeRepository(db *sqlx.DB) NodeRepository {
	return &nodeRepository{db: db}
}

// Create 创建节点
func (r *nodeRepository) Create(ctx context.Context, node *Node) error {
	query := `INSERT INTO nodes (node_name, frps_port, url, token, user, description, permission, allowed_types, host, port_range, ip, status, owner_id, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	result, err := r.db.ExecContext(ctx, query,
		node.NodeName, node.FrpsPort, node.URL, node.Token, node.User,
		node.Description, node.Permission, node.AllowedTypes, node.Host,
		node.PortRange, node.IP, node.Status, node.OwnerID)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	node.ID = id
	return nil
}

// GetByID 根据ID获取节点
func (r *nodeRepository) GetByID(ctx context.Context, id int64) (*Node, error) {
	node := &Node{}
	query := `SELECT * FROM nodes WHERE id = ?`
	err := r.db.GetContext(ctx, node, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("node not found")
		}
		return nil, err
	}
	return node, nil
}

// GetByNodeName 根据节点名称获取节点
func (r *nodeRepository) GetByNodeName(ctx context.Context, nodeName string) (*Node, error) {
	node := &Node{}
	query := `SELECT * FROM nodes WHERE node_name = ?`
	err := r.db.GetContext(ctx, node, query, nodeName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("node not found")
		}
		return nil, err
	}
	return node, nil
}

// GetByUser 根据用户获取节点列表
func (r *nodeRepository) GetByUser(ctx context.Context, user string) ([]*Node, error) {
	nodes := []*Node{}
	query := `SELECT * FROM nodes WHERE user = ?`
	err := r.db.SelectContext(ctx, &nodes, query, user)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetByPermission 根据权限获取节点列表
// permission参数表示用户组ID，返回权限为空数组(公共节点)或权限数组中包含用户组ID的所有节点
func (r *nodeRepository) GetByPermission(ctx context.Context, permission int64) ([]*Node, error) {
	nodes := []*Node{}
	// 首先获取所有节点
	query := `SELECT * FROM nodes`
	err := r.db.SelectContext(ctx, &nodes, query)
	if err != nil {
		return nil, err
	}

	// 过滤出权限为空数组(公共节点)或权限数组中包含用户组ID的节点
	var filteredNodes []*Node

	for _, node := range nodes {
		// 使用工具函数判断用户组是否有权限访问节点
		hasAccess, err := utils.IsGroupInPermission(permission, node.Permission)
		if err != nil {
			continue // 如果解析出错，跳过该节点
		}

		if hasAccess {
			filteredNodes = append(filteredNodes, node)
		}
	}

	return filteredNodes, nil
}

// GetByOwnerID 根据所属用户ID获取节点列表
func (r *nodeRepository) GetByOwnerID(ctx context.Context, ownerID int64) ([]*Node, error) {
	nodes := []*Node{}
	query := `SELECT * FROM nodes WHERE owner_id = ?`
	err := r.db.SelectContext(ctx, &nodes, query, ownerID)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// Update 更新节点信息
func (r *nodeRepository) Update(ctx context.Context, node *Node) error {
	query := `UPDATE nodes SET node_name = ?, frps_port = ?, url = ?, token = ?, user = ?, 
		description = ?, permission = ?, allowed_types = ?, host = ?, port_range = ?, ip = ?, status = ?, owner_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query,
		node.NodeName, node.FrpsPort, node.URL, node.Token, node.User,
		node.Description, node.Permission, node.AllowedTypes, node.Host,
		node.PortRange, node.IP, node.Status, node.OwnerID, node.ID)
	return err
}

// Delete 删除节点
func (r *nodeRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM nodes WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List 获取节点列表
func (r *nodeRepository) List(ctx context.Context, offset, limit int) ([]*Node, error) {
	nodes := []*Node{}
	query := `SELECT * FROM nodes LIMIT ? OFFSET ?`
	err := r.db.SelectContext(ctx, &nodes, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}
