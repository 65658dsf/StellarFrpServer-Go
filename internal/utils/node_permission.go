package utils

import (
	"encoding/json"
	"fmt"
)

// ParseNodePermission 解析节点权限组字符串为字符串数组
func ParseNodePermission(permission string) ([]string, error) {
	if permission == "" || permission == "[]" {
		return []string{}, nil
	}

	var permGroups []string
	err := json.Unmarshal([]byte(permission), &permGroups)
	if err != nil {
		return nil, fmt.Errorf("解析节点权限组失败: %v", err)
	}

	return permGroups, nil
}

// FormatNodePermission 将权限组ID数组格式化为JSON字符串
func FormatNodePermission(permissionGroups []string) (string, error) {
	// 如果数组为空，直接返回空数组JSON表示
	if len(permissionGroups) == 0 {
		return "[]", nil
	}

	// 序列化为JSON字符串
	permBytes, err := json.Marshal(permissionGroups)
	if err != nil {
		return "", fmt.Errorf("格式化权限组失败: %v", err)
	}

	return string(permBytes), nil
}

// IsGroupInPermission 检查用户组ID是否在节点权限组中
// groupID: 用户组ID，permission: 节点权限组JSON字符串
func IsGroupInPermission(groupID int64, permission string) (bool, error) {
	// 空权限或空数组表示公共节点，所有用户组都有权限
	if permission == "" || permission == "[]" {
		return true, nil
	}

	// 解析权限组
	permGroups, err := ParseNodePermission(permission)
	if err != nil {
		return false, err
	}

	// 检查用户组ID是否在权限组中
	groupIDStr := fmt.Sprintf("%d", groupID)
	for _, permGroup := range permGroups {
		if permGroup == groupIDStr {
			return true, nil
		}
	}

	return false, nil
}
