package models

import (
	"time"
)

// Order statuses
const (
	OrderStatusNew       = "new"
	OrderStatusConfirmed = "confirmed"
	OrderStatusShipping  = "shipping"
	OrderStatusCompleted = "completed"
	OrderStatusCancelled = "cancelled"
)

// OrderItem - buyurtma mahsuloti
// @Description Buyurtmadagi bitta mahsulot
type OrderItem struct {
	ID           string    `json:"id"`
	OrderID      string    `json:"order_id,omitempty"`
	ProductID    *string   `json:"product_id,omitempty"`
	ProductName  string    `json:"product_name"`
	ProductImage string    `json:"product_image,omitempty"`
	Quantity     int       `json:"quantity"`
	Price        float64   `json:"price"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

// Order - buyurtma modeli
// @Description Buyurtma ma'lumotlari
type Order struct {
	ID            string      `json:"id"`
	ShopID        string      `json:"shop_id"`
	ClientName    string      `json:"client_name"`
	ClientPhone   string      `json:"client_phone"`
	ClientAddress string      `json:"client_address,omitempty"`
	TotalAmount   float64     `json:"total_amount"`
	DeliveryPrice float64     `json:"delivery_price,omitempty"`
	Status        string      `json:"status"`
	ClientNote    string      `json:"client_note,omitempty"`
	SellerNote    string      `json:"seller_note,omitempty"`
	Items         []OrderItem `json:"items,omitempty"`
	ItemsCount    int         `json:"items_count,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at,omitempty"`
	CompletedAt   *time.Time  `json:"completed_at,omitempty"`
}

// OrderResponse - bitta buyurtma javobi
type OrderResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Order   *Order `json:"order,omitempty"`
}

// OrdersResponse - buyurtmalar ro'yxati javobi
type OrdersResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message,omitempty"`
	Orders  []Order `json:"orders"`
	Total   int     `json:"total"`
	Page    int     `json:"page"`
	Limit   int     `json:"limit"`
}

// OrderStats - buyurtmalar statistikasi
type OrderStats struct {
	NewCount       int     `json:"new_count"`
	ConfirmedCount int     `json:"confirmed_count"`
	ShippingCount  int     `json:"shipping_count"`
	CompletedCount int     `json:"completed_count"`
	CancelledCount int     `json:"cancelled_count"`
	TotalRevenue   float64 `json:"total_revenue"`
}

// OrderStatsResponse - statistika javobi
type OrderStatsResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message,omitempty"`
	Stats   OrderStats `json:"stats"`
}

// ValidOrderStatuses - ruxsat etilgan statuslar
var ValidOrderStatuses = []string{
	OrderStatusNew,
	OrderStatusConfirmed,
	OrderStatusShipping,
	OrderStatusCompleted,
	OrderStatusCancelled,
}

// IsValidOrderStatus - statusni tekshirish
func IsValidOrderStatus(status string) bool {
	for _, s := range ValidOrderStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// GetStatusLabel - status uchun o'zbekcha nom
func GetStatusLabel(status string) string {
	switch status {
	case OrderStatusNew:
		return "Yangi"
	case OrderStatusConfirmed:
		return "Tasdiqlangan"
	case OrderStatusShipping:
		return "Yetkazilmoqda"
	case OrderStatusCompleted:
		return "Yakunlangan"
	case OrderStatusCancelled:
		return "Bekor qilingan"
	default:
		return status
	}
}

// CancellationReason - bekor qilish sababi
// @Description Dinamik bekor qilish sabablari
type CancellationReason struct {
	ID         int    `json:"id"`
	ReasonText string `json:"reason_text"`
	SortOrder  int    `json:"sort_order"`
}

// CancellationReasonsResponse - sabablar ro'yxati javobi
type CancellationReasonsResponse struct {
	Success bool     `json:"success"`
	Reasons []string `json:"reasons"`
}

// CancellationBreakdown - bekor qilish sababi statistikasi
type CancellationBreakdown struct {
	Reason     string  `json:"reason"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// CancellationStats - bekor qilish statistikasi
type CancellationStats struct {
	TotalCancelled int                     `json:"total_cancelled"`
	Breakdown      []CancellationBreakdown `json:"breakdown"`
}

// CancellationStatsResponse - bekor qilish statistikasi javobi
type CancellationStatsResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message,omitempty"`
	Stats   CancellationStats `json:"stats"`
}
