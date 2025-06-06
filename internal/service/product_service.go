package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"stellarfrp/internal/model"
	"stellarfrp/internal/repository"
	"strings"
	"time"

	"stellarfrp/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// ProductService 商品服务
type ProductService struct {
	productRepo *repository.ProductRepository
	orderRepo   *repository.OrderRepository
	userService UserService
	redisClient *redis.Client
	logger      *logger.Logger
}

// NewProductService 创建商品服务
func NewProductService(
	productRepo *repository.ProductRepository,
	orderRepo *repository.OrderRepository,
	userService UserService,
	redisClient *redis.Client,
	logger *logger.Logger,
) *ProductService {
	return &ProductService{
		productRepo: productRepo,
		orderRepo:   orderRepo,
		userService: userService,
		redisClient: redisClient,
		logger:      logger,
	}
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	b := make([]byte, length/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GetProducts 获取所有商品
func (s *ProductService) GetProducts() ([]model.Product, error) {
	return s.productRepo.GetProducts()
}

// GetProductByID 根据ID获取商品
func (s *ProductService) GetProductByID(id uint64) (*model.Product, error) {
	return s.productRepo.GetProductByID(id)
}

// GetProductBySkuID 根据SkuID获取商品
func (s *ProductService) GetProductBySkuID(skuID string) (*model.Product, error) {
	return s.productRepo.GetProductBySkuID(skuID)
}

// CreateOrderLink 创建订单链接
func (s *ProductService) CreateOrderLink(userID uint64, productID uint64, customRemark string) (string, string, error) {
	// 获取商品信息
	product, err := s.productRepo.GetProductByID(productID)
	if err != nil {
		return "", "", fmt.Errorf("获取商品信息失败: %w", err)
	}

	// 获取用户信息
	repoUser, err := s.userService.GetByID(context.Background(), int64(userID))
	if err != nil {
		return "", "", fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 生成订单号
	orderNo := fmt.Sprintf("SFP%s%s", time.Now().Format("20060102150405"), generateRandomString(8))

	// 生成备注信息（用户名|订单号）
	remark := fmt.Sprintf("%s|%s", repoUser.Username, orderNo)
	if customRemark != "" {
		remark = customRemark
	}

	// 创建订单记录
	order := &model.Order{
		OrderNo:        orderNo,
		UserID:         userID,
		ProductID:      productID,
		ProductSkuID:   product.SkuID,
		ProductName:    product.Name,
		Amount:         product.Price,
		Status:         0, // 待支付
		Remark:         remark,
		RewardAction:   sql.NullString{String: product.RewardAction, Valid: product.RewardAction != ""},
		RewardValue:    sql.NullString{String: product.RewardValue, Valid: product.RewardValue != ""},
		RewardExecuted: false,
	}

	// 保存订单到数据库
	if err := s.orderRepo.CreateOrder(order); err != nil {
		return "", "", fmt.Errorf("创建订单失败: %w", err)
	}

	// 构建爱发电订单链接参数
	params := url.Values{}
	params.Add("product_type", "1")
	params.Add("plan_id", product.PlanID)
	params.Add("remark", remark)

	// 构建SKU参数
	skuParam := fmt.Sprintf(`[{"sku_id":"%s","count":1}]`, product.SkuID)
	params.Add("sku", skuParam)
	params.Add("viokrz_ex", "0")

	// 生成爱发电订单链接
	orderLink := fmt.Sprintf("https://ifdian.net/order/create?%s", params.Encode())

	return orderLink, orderNo, nil
}

// ProcessAfdianWebhook 处理爱发电Webhook回调
func (s *ProductService) ProcessAfdianWebhook(data map[string]interface{}) (bool, error) {
	// 解析订单数据
	orderData, ok := data["data"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("无效的数据格式")
	}

	orderType, ok := orderData["type"].(string)
	if !ok || orderType != "order" {
		return false, fmt.Errorf("无效的订单类型")
	}

	orderInfo, ok := orderData["order"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("无效的订单信息")
	}

	// 获取爱发电订单号和状态
	outTradeNo, ok := orderInfo["out_trade_no"].(string)
	if !ok {
		return false, fmt.Errorf("无效的订单号")
	}

	status, ok := orderInfo["status"].(float64)
	if !ok {
		return false, fmt.Errorf("无效的订单状态")
	}

	// 只处理支付成功的订单 (状态为2表示已支付，1表示未支付)
	if int(status) != 2 {
		s.logger.Info("订单未支付成功", "status", int(status), "out_trade_no", outTradeNo)
		return false, fmt.Errorf("订单未支付成功，状态: %d", int(status))
	}

	// 获取订单备注
	remark, _ := orderInfo["remark"].(string)

	// 检查是否为测试数据
	if remark == "" {
		s.logger.Info("检测到爱发电测试数据", "out_trade_no", outTradeNo)
		return true, nil
	}

	// 记录完整的remark信息
	s.logger.Info("处理已支付订单", "remark", remark, "out_trade_no", outTradeNo)

	// 尝试从remark中提取订单号
	var orderNo string
	parts := strings.Split(remark, "|")
	if len(parts) >= 2 && strings.HasPrefix(parts[1], "SFP") {
		orderNo = parts[1]
		s.logger.Info("从remark中提取订单号", "order_no", orderNo)

		// 根据订单号查询
		order, err := s.orderRepo.GetOrderByOrderNo(orderNo)
		if err == nil && order != nil {
			// 更新订单状态为已支付
			if err := s.orderRepo.UpdateOrderStatus(order.OrderNo, 1, outTradeNo); err != nil {
				return false, fmt.Errorf("更新订单状态失败: %w", err)
			}

			// 执行奖励
			if err := s.executeReward(order); err != nil {
				s.logger.Error("执行奖励失败", "error", err, "order_no", order.OrderNo)
				// 不返回错误，继续处理
			}

			return true, nil
		}
	}

	// 如果通过订单号查询失败，再尝试通过备注查询
	order, err := s.orderRepo.GetOrderByRemarkAndStatus(remark, 0) // 查询待支付状态的订单
	if err != nil {
		s.logger.Error("查询订单失败", "error", err, "remark", remark)
		// 如果是测试数据或特殊情况，返回成功
		if strings.Contains(err.Error(), "converting NULL to string") {
			s.logger.Info("忽略NULL转换错误", "remark", remark)
			return true, nil
		}
		return false, fmt.Errorf("查询订单失败: %w", err)
	}

	if order == nil {
		s.logger.Warn("订单不存在", "remark", remark)
		// 如果是测试数据，返回成功
		if strings.Contains(remark, "test") || strings.Contains(outTradeNo, "test") {
			return true, nil
		}
		return false, fmt.Errorf("订单不存在")
	}

	// 更新订单状态为已支付
	if err := s.orderRepo.UpdateOrderStatus(order.OrderNo, 1, outTradeNo); err != nil {
		return false, fmt.Errorf("更新订单状态失败: %w", err)
	}

	// 执行奖励
	if err := s.executeReward(order); err != nil {
		s.logger.Error("执行奖励失败", "error", err, "order_no", order.OrderNo)
		// 不返回错误，继续处理
	}

	return true, nil
}

// executeReward 执行奖励
func (s *ProductService) executeReward(order *model.Order) error {
	// 检查奖励是否已执行
	if order.RewardExecuted {
		return nil
	}

	// 检查奖励动作是否有效
	if !order.RewardAction.Valid || order.RewardAction.String == "" {
		s.logger.Info("订单没有奖励动作", "order_no", order.OrderNo)
		return nil
	}

	// 根据奖励类型执行不同的奖励逻辑
	switch order.RewardAction.String {
	case "ADD_VERIFY_COUNT":
		// 增加实名认证次数
		count := 1
		if order.RewardValue.Valid && order.RewardValue.String != "" {
			fmt.Sscanf(order.RewardValue.String, "%d", &count)
		}
		if err := s.userService.AddVerifyCount(order.UserID, count); err != nil {
			return fmt.Errorf("增加实名认证次数失败: %w", err)
		}

	case "ADD_TRAFFIC_GB":
		// 增加流量
		var trafficGB float64
		if order.RewardValue.Valid && order.RewardValue.String != "" {
			fmt.Sscanf(order.RewardValue.String, "%f", &trafficGB)
		}
		if err := s.userService.AddTraffic(order.UserID, trafficGB); err != nil {
			return fmt.Errorf("增加流量失败: %w", err)
		}

	case "UPGRADE_GROUP":
		// 升级用户组
		var groupID int64
		if order.RewardValue.Valid && order.RewardValue.String != "" {
			fmt.Sscanf(order.RewardValue.String, "%d", &groupID)
		}
		// 设置用户组
		if err := s.userService.UpdateUserGroup(context.Background(), int64(order.UserID), groupID); err != nil {
			return fmt.Errorf("升级用户组失败: %w", err)
		}

	// 可以添加更多奖励类型
	default:
		return fmt.Errorf("未知的奖励类型: %s", order.RewardAction.String)
	}

	// 更新奖励执行状态
	if err := s.orderRepo.UpdateRewardExecuted(order.OrderNo, true); err != nil {
		return fmt.Errorf("更新奖励执行状态失败: %w", err)
	}

	return nil
}

// GetOrdersByUserID 获取用户的所有订单
func (s *ProductService) GetOrdersByUserID(userID uint64, page, pageSize int) ([]model.Order, int, error) {
	// 设置默认值
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50 // 限制最大为50条
	}

	return s.orderRepo.GetOrdersByUserID(userID, page, pageSize)
}

// GetOrderByOrderNo 根据订单号获取订单
func (s *ProductService) GetOrderByOrderNo(orderNo string) (*model.Order, error) {
	return s.orderRepo.GetOrderByOrderNo(orderNo)
}

// ProcessUnexecutedRewards 处理未执行的奖励
func (s *ProductService) ProcessUnexecutedRewards() error {
	orders, err := s.orderRepo.GetUnexecutedRewards()
	if err != nil {
		return fmt.Errorf("获取未执行奖励的订单失败: %w", err)
	}

	for _, order := range orders {
		if err := s.executeReward(&order); err != nil {
			s.logger.Error("执行奖励失败", "error", err, "order_no", order.OrderNo)
			// 继续处理其他订单
			continue
		}
	}

	return nil
}
