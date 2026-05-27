package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go-backend/internal/http/response"
	"go-backend/internal/middleware"
	"go-backend/internal/store/model"
)

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteJSON(w, response.ErrDefault("请求失败"))
		return
	}

	_, roleID, err := userRoleFromRequest(r)
	if err != nil {
		response.WriteJSON(w, response.Err(-2, "用户信息错误"))
		return
	}

	onlyActive := roleID != 0
	items, err := h.repo.ListProducts(onlyActive)
	if err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}
	if items == nil {
		items = []*model.Product{}
	}
	response.WriteJSON(w, response.OK(items))
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	tier, _ := middleware.GetLicenseTier()
	if tier == middleware.TierBlocked {
		response.WriteJSON(w, response.Err(403, "授权无效，无法操作"))
		return
	}

	var req map[string]interface{}
	if err := decodeJSON(r.Body, &req); err != nil {
		response.WriteJSON(w, response.ErrDefault("请求参数错误"))
		return
	}

	name := asString(req["name"])
	if name == "" {
		response.WriteJSON(w, response.ErrDefault("商品名称不能为空"))
		return
	}

	sortOrder := asInt(req["sort_order"], asInt(req["sortOrder"], 0))

	product, err := h.repo.CreateProduct(
		name,
		asString(req["description"]),
		asString(req["type"]),
		asInt64(req["price"], 0),
		asInt64(req["value"], 0),
		sortOrder,
		asInt(req["status"], 1),
	)
	if err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}

	response.WriteJSON(w, response.OK(product))
}

func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request) {
	tier, _ := middleware.GetLicenseTier()
	if tier == middleware.TierBlocked {
		response.WriteJSON(w, response.Err(403, "授权无效，无法操作"))
		return
	}

	var req map[string]interface{}
	if err := decodeJSON(r.Body, &req); err != nil {
		response.WriteJSON(w, response.ErrDefault("请求参数错误"))
		return
	}

	id := asInt64(req["id"], 0)
	if id <= 0 {
		response.WriteJSON(w, response.ErrDefault("商品ID不能为空"))
		return
	}

	name := asString(req["name"])
	if name == "" {
		response.WriteJSON(w, response.ErrDefault("商品名称不能为空"))
		return
	}

	productType := asString(req["type"])
	sortOrder := asInt(req["sort_order"], asInt(req["sortOrder"], 0))

	if err := h.repo.UpdateProduct(
		id, name, asString(req["description"]),
		productType,
		asInt64(req["price"], 0), asInt64(req["value"], 0),
		sortOrder, asInt(req["status"], 1),
	); err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}

	response.WriteJSON(w, response.OKEmpty())
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	tier, _ := middleware.GetLicenseTier()
	if tier == middleware.TierBlocked {
		response.WriteJSON(w, response.Err(403, "授权无效，无法操作"))
		return
	}

	id := idFromBody(r, w)
	if id <= 0 {
		return
	}

	if err := h.repo.DeleteProduct(id); err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}

	response.WriteJSON(w, response.OKEmpty())
}

func (h *Handler) updateProductOrder(w http.ResponseWriter, r *http.Request) {
	tier, _ := middleware.GetLicenseTier()
	if tier == middleware.TierBlocked {
		response.WriteJSON(w, response.Err(403, "授权无效，无法操作"))
		return
	}

	var req map[string]interface{}
	if err := decodeJSON(r.Body, &req); err != nil {
		response.WriteJSON(w, response.ErrDefault("请求参数错误"))
		return
	}

	ids := asInt64Slice(req["ids"])
	if len(ids) == 0 {
		response.WriteJSON(w, response.ErrDefault("排序数据不能为空"))
		return
	}

	if err := h.repo.UpdateProductOrder(ids); err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}
	response.WriteJSON(w, response.OKEmpty())
}

func (h *Handler) listPackages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteJSON(w, response.ErrDefault("请求失败"))
		return
	}
	_, roleID, err := userRoleFromRequest(r)
	if err != nil {
		response.WriteJSON(w, response.Err(-2, "用户信息错误"))
		return
	}
	items, err := h.repo.ListPackages(roleID != 0)
	if err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}
	if items == nil {
		items = []*model.SubscriptionPackage{}
	}
	response.WriteJSON(w, response.OK(items))
}

func (h *Handler) createPackage(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	var req struct {
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		PriceYuan      float64 `json:"priceYuan"`
		ValidityDays   int     `json:"validityDays"`
		TrafficLimit   int64   `json:"trafficLimit"`
		PortCount      int     `json:"portCount"`
		SpeedLimit     int     `json:"speedLimit"`
		MaxRules       int     `json:"maxRules"`
		MaxConnections int     `json:"maxConnections"`
		MaxIPAccess    int     `json:"maxIPAccess"`
		AutoRenew      int     `json:"autoRenew"`
		SortOrder      int     `json:"sortOrder"`
		Enabled        int     `json:"enabled"`
		ShopVisible    int     `json:"shopVisible"`
		TunnelGroupIDs []int64 `json:"tunnelGroupIds"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		response.WriteJSON(w, response.ErrDefault("请求参数错误"))
		return
	}
	if req.Name == "" {
		response.WriteJSON(w, response.ErrDefault("套餐名称不能为空"))
		return
	}
	pkg := &model.SubscriptionPackage{
		Name:           req.Name,
		Description:    req.Description,
		Price:          int64(req.PriceYuan * 100),
		ValidityDays:   req.ValidityDays,
		TrafficLimit:   req.TrafficLimit,
		PortCount:      req.PortCount,
		SpeedLimit:     req.SpeedLimit,
		MaxRules:       req.MaxRules,
		MaxConnections: req.MaxConnections,
		MaxIPAccess:    req.MaxIPAccess,
		AutoRenew:      req.AutoRenew,
		SortOrder:      req.SortOrder,
		Enabled:        req.Enabled,
		ShopVisible:    req.ShopVisible,
	}
	if err := h.repo.CreatePackage(pkg, req.TunnelGroupIDs); err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}
	response.WriteJSON(w, response.OK(pkg))
}

func (h *Handler) updatePackage(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	var req struct {
		ID             int64   `json:"id"`
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		PriceYuan      float64 `json:"priceYuan"`
		ValidityDays   int     `json:"validityDays"`
		TrafficLimit   int64   `json:"trafficLimit"`
		PortCount      int     `json:"portCount"`
		SpeedLimit     int     `json:"speedLimit"`
		MaxRules       int     `json:"maxRules"`
		MaxConnections int     `json:"maxConnections"`
		MaxIPAccess    int     `json:"maxIPAccess"`
		AutoRenew      int     `json:"autoRenew"`
		SortOrder      int     `json:"sortOrder"`
		Enabled        int     `json:"enabled"`
		ShopVisible    int     `json:"shopVisible"`
		TunnelGroupIDs []int64 `json:"tunnelGroupIds"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		response.WriteJSON(w, response.ErrDefault("请求参数错误"))
		return
	}
	if req.ID <= 0 || req.Name == "" {
		response.WriteJSON(w, response.ErrDefault("参数错误"))
		return
	}
	pkg := &model.SubscriptionPackage{
		ID:             req.ID,
		Name:           req.Name,
		Description:    req.Description,
		Price:          int64(req.PriceYuan * 100),
		ValidityDays:   req.ValidityDays,
		TrafficLimit:   req.TrafficLimit,
		PortCount:      req.PortCount,
		SpeedLimit:     req.SpeedLimit,
		MaxRules:       req.MaxRules,
		MaxConnections: req.MaxConnections,
		MaxIPAccess:    req.MaxIPAccess,
		AutoRenew:      req.AutoRenew,
		SortOrder:      req.SortOrder,
		Enabled:        req.Enabled,
		ShopVisible:    req.ShopVisible,
	}
	if err := h.repo.UpdatePackage(pkg, req.TunnelGroupIDs); err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}
	response.WriteJSON(w, response.OKEmpty())
}

func (h *Handler) deletePackage(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	id := idFromBody(r, w)
	if id <= 0 {
		return
	}
	if err := h.repo.DeletePackage(id); err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}
	response.WriteJSON(w, response.OKEmpty())
}

func (h *Handler) getPackageDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteJSON(w, response.ErrDefault("请求失败"))
		return
	}
	id := idFromBody(r, w)
	if id <= 0 {
		return
	}
	pkg, err := h.repo.GetPackageByID(id)
	if err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}
	tunnelGroupIDs, err := h.repo.GetPackageTunnelGroupIDs(id)
	if err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}
	response.WriteJSON(w, response.OK(map[string]interface{}{
		"package":         pkg,
		"tunnelGroupIds": tunnelGroupIDs,
	}))
}

func (h *Handler) createPackageOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteJSON(w, response.ErrDefault("请求失败"))
		return
	}
	var req map[string]interface{}
	if err := decodeJSON(r.Body, &req); err != nil {
		response.WriteJSON(w, response.ErrDefault("请求参数错误"))
		return
	}
	userID, userName := h.userNameFromRequest(r)
	if userID <= 0 {
		response.WriteJSON(w, response.Err(-2, "用户信息错误"))
		return
	}
	packageID := asInt64(req["package_id"], 0)
	if packageID <= 0 {
		response.WriteJSON(w, response.ErrDefault("套餐ID不能为空"))
		return
	}
	currency := strings.ToUpper(asString(req["pay_currency"]))
	if currency == "" {
		currency = "BALANCE"
	}
	if currency != "BALANCE" && currency != "USDT" && currency != "YIPAY" {
		response.WriteJSON(w, response.ErrDefault("不支持的支付方式"))
		return
	}
	pkg, err := h.repo.GetPackageByID(packageID)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault("套餐不存在"))
		return
	}
	if pkg.Enabled != 1 {
		response.WriteJSON(w, response.ErrDefault("套餐已停用"))
		return
	}
	meta, _ := json.Marshal(pkg)
	tunnelGroupIDs, err := h.repo.GetPackageTunnelGroupIDs(packageID)
	if err != nil {
		tunnelGroupIDs = nil
	}

	order := &model.Order{
		OrderNo:     fmt.Sprintf("PKG%d%d", time.Now().UnixMilli(), time.Now().Nanosecond()%10000),
		UserID:      userID,
		UserName:    userName,
		ProductID:   pkg.ID,
		ProductName: pkg.Name,
		ProductType: "package",
		ProductMeta: string(meta),
		Amount:      pkg.Price,
		PayCurrency: currency,
		Status:      0,
	}

	if currency == "BALANCE" {
		ok, err := h.repo.DeductUserBalance(userID, order.Amount)
		if err != nil {
			response.WriteJSON(w, response.Err(-2, err.Error()))
			return
		}
		if !ok {
			response.WriteJSON(w, response.Err(1001, "余额不足"))
			return
		}
		order.Status = 1
		order.PayTime = time.Now().Unix()

		if err := h.repo.CreateOrder(order); err != nil {
			response.WriteJSON(w, response.Err(-2, err.Error()))
			return
		}

		user, _ := h.repo.GetUserByID(userID)
		if user != nil {
			_ = h.repo.CreateBalanceLog(userID, userName, -order.Amount,
				user.Balance+order.Amount, user.Balance,
				time.Now().Unix(), "余额购买套餐")
		}

		if err := h.repo.DeliverPackageToUser(userID, pkg, order.ID, tunnelGroupIDs); err != nil {
			response.WriteJSON(w, response.Err(-2, "套餐交付失败: "+err.Error()))
			return
		}

		response.WriteJSON(w, response.OK(map[string]interface{}{
			"order_id": order.ID,
			"order_no": order.OrderNo,
			"status":   order.Status,
			"amount":   order.Amount,
		}))
		return
	}

	if err := h.repo.CreateOrder(order); err != nil {
		response.WriteJSON(w, response.Err(-2, err.Error()))
		return
	}

	response.WriteJSON(w, response.OK(map[string]interface{}{
		"order_id": order.ID,
		"order_no": order.OrderNo,
		"status":   order.Status,
		"amount":   order.Amount,
	}))
}
