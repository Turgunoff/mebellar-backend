package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"mebellar-backend/models"
)

// ============================================
// GET SELLER ORDERS
// ============================================

// GetSellerOrders godoc
// @Summary      Sotuvchi buyurtmalarini olish
// @Description  Joriy do'konning barcha buyurtmalarini olish
// @Tags         seller-orders
// @Accept       json
// @Produce      json
// @Param        X-Shop-ID header string true "Do'kon ID"
// @Param        status query string false "Status filter (new, confirmed, shipping, completed, cancelled)"
// @Param        page query int false "Sahifa raqami (default: 1)"
// @Param        limit query int false "Har sahifadagi buyurtmalar soni (default: 20)"
// @Success      200  {object}  models.OrdersResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /seller/orders [get]
func GetSellerOrders(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Get shop ID from header
		shopID := r.Header.Get("X-Shop-ID")
		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "X-Shop-ID header kerak",
			})
			return
		}

		// Parse query params
		pageStr := r.URL.Query().Get("page")
		limitStr := r.URL.Query().Get("limit")
		statusFilter := r.URL.Query().Get("status")

		page := 1
		limit := 20

		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}

		offset := (page - 1) * limit

		// Build query
		countQuery := `SELECT COUNT(*) FROM orders WHERE shop_id = $1`
		dataQuery := `
			SELECT 
				id, shop_id, client_name, client_phone, client_address,
				total_amount, delivery_price, status,
				client_note, seller_note,
				created_at, updated_at, completed_at
			FROM orders 
			WHERE shop_id = $1
		`

		args := []interface{}{shopID}
		argIndex := 2

		// Filter by status
		if statusFilter != "" {
			// Support multiple statuses (comma-separated)
			statuses := strings.Split(statusFilter, ",")
			if len(statuses) == 1 {
				countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
				dataQuery += fmt.Sprintf(" AND status = $%d", argIndex)
				args = append(args, statusFilter)
				argIndex++
			} else {
				placeholders := make([]string, len(statuses))
				for i, s := range statuses {
					placeholders[i] = fmt.Sprintf("$%d", argIndex)
					args = append(args, strings.TrimSpace(s))
					argIndex++
				}
				statusCondition := fmt.Sprintf(" AND status IN (%s)", strings.Join(placeholders, ","))
				countQuery += statusCondition
				dataQuery += statusCondition
			}
		}

		dataQuery += " ORDER BY created_at DESC"
		dataQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

		// Get total count
		var total int
		countArgs := args[:len(args)]
		err := db.QueryRow(countQuery, countArgs...).Scan(&total)
		if err != nil {
			log.Printf("❌ Orders count xatosi: %v", err)
			total = 0
		}

		// Get orders
		args = append(args, limit, offset)
		rows, err := db.Query(dataQuery, args...)
		if err != nil {
			log.Printf("❌ Seller orders query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Buyurtmalarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		orders := []models.Order{}
		orderIDs := []string{}

		for rows.Next() {
			var o models.Order
			var clientAddress, clientNote, sellerNote sql.NullString
			var completedAt sql.NullTime

			err := rows.Scan(
				&o.ID, &o.ShopID, &o.ClientName, &o.ClientPhone, &clientAddress,
				&o.TotalAmount, &o.DeliveryPrice, &o.Status,
				&clientNote, &sellerNote,
				&o.CreatedAt, &o.UpdatedAt, &completedAt,
			)
			if err != nil {
				log.Printf("Order scan xatosi: %v", err)
				continue
			}

			if clientAddress.Valid {
				o.ClientAddress = clientAddress.String
			}
			if clientNote.Valid {
				o.ClientNote = clientNote.String
			}
			if sellerNote.Valid {
				o.SellerNote = sellerNote.String
			}
			if completedAt.Valid {
				o.CompletedAt = &completedAt.Time
			}

			orders = append(orders, o)
			orderIDs = append(orderIDs, o.ID)
		}

		// Fetch order items for all orders
		if len(orderIDs) > 0 {
			itemsMap := make(map[string][]models.OrderItem)

			// Build placeholders for IN clause
			placeholders := make([]string, len(orderIDs))
			itemArgs := make([]interface{}, len(orderIDs))
			for i, id := range orderIDs {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
				itemArgs[i] = id
			}

			itemsQuery := fmt.Sprintf(`
				SELECT id, order_id, product_id, product_name, product_image, quantity, price
				FROM order_items 
				WHERE order_id IN (%s)
				ORDER BY created_at ASC
			`, strings.Join(placeholders, ","))

			itemRows, err := db.Query(itemsQuery, itemArgs...)
			if err != nil {
				log.Printf("⚠️ Order items query xatosi: %v", err)
			} else {
				defer itemRows.Close()

				for itemRows.Next() {
					var item models.OrderItem
					var productID sql.NullString
					var productImage sql.NullString

					err := itemRows.Scan(
						&item.ID, &item.OrderID, &productID, &item.ProductName, &productImage,
						&item.Quantity, &item.Price,
					)
					if err != nil {
						log.Printf("Order item scan xatosi: %v", err)
						continue
					}

					if productID.Valid {
						item.ProductID = &productID.String
					}
					if productImage.Valid {
						item.ProductImage = productImage.String
					}

					itemsMap[item.OrderID] = append(itemsMap[item.OrderID], item)
				}
			}

			// Attach items to orders
			for i := range orders {
				if items, ok := itemsMap[orders[i].ID]; ok {
					orders[i].Items = items
					orders[i].ItemsCount = len(items)
				}
			}
		}

		log.Printf("✅ Seller %s: %d ta buyurtma topildi (sahifa %d, status=%s)",
			shopID, len(orders), page, statusFilter)

		writeJSON(w, http.StatusOK, models.OrdersResponse{
			Success: true,
			Orders:  orders,
			Total:   total,
			Page:    page,
			Limit:   limit,
		})
	}
}

// ============================================
// GET ORDER STATS
// ============================================

// GetOrderStats godoc
// @Summary      Buyurtmalar statistikasi
// @Description  Joriy do'konning buyurtmalar statistikasini olish
// @Tags         seller-orders
// @Accept       json
// @Produce      json
// @Param        X-Shop-ID header string true "Do'kon ID"
// @Success      200  {object}  models.OrderStatsResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /seller/orders/stats [get]
func GetOrderStats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		shopID := r.Header.Get("X-Shop-ID")
		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "X-Shop-ID header kerak",
			})
			return
		}

		var stats models.OrderStats

		// Get counts by status
		query := `
			SELECT 
				COALESCE(SUM(CASE WHEN status = 'new' THEN 1 ELSE 0 END), 0) as new_count,
				COALESCE(SUM(CASE WHEN status = 'confirmed' THEN 1 ELSE 0 END), 0) as confirmed_count,
				COALESCE(SUM(CASE WHEN status = 'shipping' THEN 1 ELSE 0 END), 0) as shipping_count,
				COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) as completed_count,
				COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0) as cancelled_count,
				COALESCE(SUM(CASE WHEN status = 'completed' THEN total_amount ELSE 0 END), 0) as total_revenue
			FROM orders 
			WHERE shop_id = $1
		`

		err := db.QueryRow(query, shopID).Scan(
			&stats.NewCount,
			&stats.ConfirmedCount,
			&stats.ShippingCount,
			&stats.CompletedCount,
			&stats.CancelledCount,
			&stats.TotalRevenue,
		)
		if err != nil {
			log.Printf("❌ Order stats xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Statistikani olishda xatolik",
			})
			return
		}

		writeJSON(w, http.StatusOK, models.OrderStatsResponse{
			Success: true,
			Stats:   stats,
		})
	}
}

// ============================================
// SEED ORDERS (DEBUG)
// ============================================

// SeedOrders godoc
// @Summary      Test buyurtmalarini yaratish (Debug)
// @Description  Joriy do'kon uchun test buyurtmalarini yaratish
// @Tags         debug
// @Accept       json
// @Produce      json
// @Param        X-Shop-ID header string true "Do'kon ID"
// @Param        count query int false "Yaratilishi kerak bo'lgan buyurtmalar soni (default: 10)"
// @Success      200  {object}  models.AuthResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /debug/seed-orders [post]
func SeedOrders(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat POST metodi qo'llab-quvvatlanadi",
			})
			return
		}

		shopID := r.Header.Get("X-Shop-ID")
		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "X-Shop-ID header kerak",
			})
			return
		}

		// Parse count param
		countStr := r.URL.Query().Get("count")
		count := 10
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 && c <= 50 {
			count = c
		}

		// Get products from this shop to use in orders
		type SimpleProduct struct {
			ID    string
			Name  string
			Image string
			Price float64
		}

		productsQuery := `
			SELECT id, name, COALESCE(images[1], ''), price 
			FROM products 
			WHERE shop_id = $1 AND is_active = true
			LIMIT 20
		`
		rows, err := db.Query(productsQuery, shopID)
		if err != nil {
			log.Printf("❌ Products query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Mahsulotlarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		var products []SimpleProduct
		for rows.Next() {
			var p SimpleProduct
			if err := rows.Scan(&p.ID, &p.Name, &p.Image, &p.Price); err != nil {
				continue
			}
			products = append(products, p)
		}

		if len(products) == 0 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Bu do'konda mahsulot yo'q. Avval mahsulot qo'shing.",
			})
			return
		}

		// Random data for seeding
		clientNames := []string{
			"Azizbek Karimov", "Dilshod Rahimov", "Shoxrux To'xtayev",
			"Malika Qodirova", "Gulnora Saidova", "Jamshid Aliyev",
			"Sardor Xolmatov", "Nilufar Ergasheva", "Bobur Mirzayev",
			"Zarina Usmonova", "Akmal Toshpulatov", "Dilfuza Qosimova",
		}

		clientPhones := []string{
			"+998901234567", "+998912345678", "+998933456789",
			"+998944567890", "+998905678901", "+998916789012",
			"+998937890123", "+998948901234", "+998909012345",
		}

		addresses := []string{
			"Toshkent sh., Chilonzor tumani, 7-mavze, 15-uy",
			"Toshkent sh., Yunusobod tumani, 4-mavze, 8-uy",
			"Toshkent sh., Mirzo Ulug'bek tumani, Buyuk Ipak yo'li 45",
			"Samarqand vil., Samarqand sh., Registon ko'chasi 12",
			"Farg'ona vil., Farg'ona sh., Mustaqillik ko'chasi 78",
			"Andijon vil., Andijon sh., A. Navoiy ko'chasi 34",
			"Buxoro vil., Buxoro sh., Olmazor mahallasi, 5-uy",
		}

		statuses := []string{
			models.OrderStatusNew,
			models.OrderStatusNew,
			models.OrderStatusConfirmed,
			models.OrderStatusConfirmed,
			models.OrderStatusShipping,
			models.OrderStatusCompleted,
			models.OrderStatusCompleted,
			models.OrderStatusCancelled,
		}

		rand.Seed(time.Now().UnixNano())
		createdCount := 0

		for i := 0; i < count; i++ {
			// Random client info
			clientName := clientNames[rand.Intn(len(clientNames))]
			clientPhone := clientPhones[rand.Intn(len(clientPhones))]
			clientAddress := addresses[rand.Intn(len(addresses))]
			status := statuses[rand.Intn(len(statuses))]

			// Random items (1-4 products)
			itemCount := rand.Intn(4) + 1
			var totalAmount float64
			var items []struct {
				ProductID    string
				ProductName  string
				ProductImage string
				Quantity     int
				Price        float64
			}

			usedProducts := make(map[int]bool)
			for j := 0; j < itemCount && j < len(products); j++ {
				// Pick a random product (avoid duplicates)
				var productIdx int
				for {
					productIdx = rand.Intn(len(products))
					if !usedProducts[productIdx] {
						usedProducts[productIdx] = true
						break
					}
					if len(usedProducts) >= len(products) {
						break
					}
				}

				product := products[productIdx]
				quantity := rand.Intn(3) + 1
				itemTotal := product.Price * float64(quantity)
				totalAmount += itemTotal

				items = append(items, struct {
					ProductID    string
					ProductName  string
					ProductImage string
					Quantity     int
					Price        float64
				}{
					ProductID:    product.ID,
					ProductName:  product.Name,
					ProductImage: product.Image,
					Quantity:     quantity,
					Price:        product.Price,
				})
			}

			// Random delivery price
			deliveryPrice := float64(rand.Intn(5)+1) * 50000 // 50k - 250k

			// Random created_at (last 30 days)
			daysAgo := rand.Intn(30)
			createdAt := time.Now().AddDate(0, 0, -daysAgo)
			
			// Set completed_at for completed orders
			var completedAt *time.Time
			if status == models.OrderStatusCompleted {
				completedTime := createdAt.Add(time.Duration(rand.Intn(7)+1) * 24 * time.Hour)
				completedAt = &completedTime
			}

			// Create order
			orderID := uuid.New().String()
			orderQuery := `
				INSERT INTO orders (id, shop_id, client_name, client_phone, client_address, 
					total_amount, delivery_price, status, created_at, completed_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			`
			_, err := db.Exec(orderQuery,
				orderID, shopID, clientName, clientPhone, clientAddress,
				totalAmount, deliveryPrice, status, createdAt, completedAt,
			)
			if err != nil {
				log.Printf("⚠️ Order insert xatosi: %v", err)
				continue
			}

			// Create order items
			for _, item := range items {
				itemID := uuid.New().String()
				itemQuery := `
					INSERT INTO order_items (id, order_id, product_id, product_name, product_image, quantity, price)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
				`
				_, err := db.Exec(itemQuery,
					itemID, orderID, item.ProductID, item.ProductName, item.ProductImage,
					item.Quantity, item.Price,
				)
				if err != nil {
					log.Printf("⚠️ Order item insert xatosi: %v", err)
				}
			}

			createdCount++
		}

		log.Printf("✅ %d ta test buyurtma yaratildi (shop: %s)", createdCount, shopID)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: fmt.Sprintf("%d ta test buyurtma muvaffaqiyatli yaratildi", createdCount),
		})
	}
}

// ============================================
// UPDATE ORDER STATUS
// ============================================

// UpdateStatusRequest - status o'zgartirish so'rovi
type UpdateStatusRequest struct {
	Status string `json:"status"` // 'confirmed', 'shipping', 'completed', 'cancelled'
	Reason string `json:"reason"` // Required if status is 'cancelled'
	Note   string `json:"note"`   // Optional custom note
}

// Note: Cancellation reasons are now dynamic from DB (cancellation_reasons table)
// The frontend fetches reasons from GET /api/common/cancellation-reasons
// and sends the reason_text string directly

// UpdateOrderStatus godoc
// @Summary      Buyurtma statusini yangilash
// @Description  Buyurtma statusini o'zgartirish (bekor qilishda sabab ko'rsatish shart)
// @Tags         seller-orders
// @Accept       json
// @Produce      json
// @Param        X-Shop-ID header string true "Do'kon ID"
// @Param        id path string true "Buyurtma ID"
// @Param        body body UpdateStatusRequest true "Status va sabab"
// @Success      200  {object}  models.OrderResponse
// @Failure      400  {object}  models.AuthResponse
// @Failure      403  {object}  models.AuthResponse
// @Failure      404  {object}  models.AuthResponse
// @Failure      500  {object}  models.AuthResponse
// @Security     BearerAuth
// @Router       /seller/orders/{id}/status [put]
func UpdateOrderStatus(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat PUT metodi qo'llab-quvvatlanadi",
			})
			return
		}

		// Get order ID from URL path
		path := r.URL.Path
		parts := strings.Split(path, "/")
		if len(parts) < 5 {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Buyurtma ID kerak",
			})
			return
		}
		orderID := parts[len(parts)-2] // /seller/orders/{id}/status

		shopID := r.Header.Get("X-Shop-ID")
		if shopID == "" {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "X-Shop-ID header kerak",
			})
			return
		}

		// Parse JSON body
		var req UpdateStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Fallback to query param for backward compatibility
			req.Status = r.URL.Query().Get("status")
		}

		newStatus := req.Status
		if !models.IsValidOrderStatus(newStatus) {
			writeJSON(w, http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "Noto'g'ri status: " + newStatus,
			})
			return
		}

		// Validate cancellation reason (must be provided, value comes from DB)
		if newStatus == models.OrderStatusCancelled {
			if req.Reason == "" {
				writeJSON(w, http.StatusBadRequest, models.AuthResponse{
					Success: false,
					Message: "Bekor qilish sababi ko'rsatilishi shart",
				})
				return
			}
			// Note: Reason value is now dynamic from DB, no hardcoded validation
		}

		// Check order belongs to shop
		var existingShopID string
		err := db.QueryRow("SELECT shop_id FROM orders WHERE id = $1", orderID).Scan(&existingShopID)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, models.AuthResponse{
				Success: false,
				Message: "Buyurtma topilmadi",
			})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Buyurtmani tekshirishda xatolik",
			})
			return
		}

		if existingShopID != shopID {
			writeJSON(w, http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "Bu buyurtma sizga tegishli emas",
			})
			return
		}

		// Prepare timestamps based on status
		var completedAt, confirmedAt *time.Time
		now := time.Now()

		switch newStatus {
		case models.OrderStatusConfirmed:
			confirmedAt = &now
		case models.OrderStatusCompleted:
			completedAt = &now
		}

		// Update status with reason if cancelled
		var query string
		var execErr error

		if newStatus == models.OrderStatusCancelled {
			query = `UPDATE orders SET 
				status = $1, 
				cancellation_reason = $2, 
				rejection_note = $3,
				completed_at = $4
			WHERE id = $5`
			_, execErr = db.Exec(query, newStatus, req.Reason, req.Note, &now, orderID)
			log.Printf("📋 Buyurtma bekor qilindi: %s, Sabab: %s", orderID, req.Reason)
		} else if newStatus == models.OrderStatusConfirmed {
			query = `UPDATE orders SET status = $1, confirmed_at = $2 WHERE id = $3`
			_, execErr = db.Exec(query, newStatus, confirmedAt, orderID)
		} else {
			query = `UPDATE orders SET status = $1, completed_at = $2 WHERE id = $3`
			_, execErr = db.Exec(query, newStatus, completedAt, orderID)
		}

		if execErr != nil {
			log.Printf("❌ Order status update xatosi: %v", execErr)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Statusni yangilashda xatolik",
			})
			return
		}

		log.Printf("✅ Buyurtma %s statusi yangilandi: %s", orderID, newStatus)

		writeJSON(w, http.StatusOK, models.AuthResponse{
			Success: true,
			Message: fmt.Sprintf("Status '%s' ga o'zgartirildi", models.GetStatusLabel(newStatus)),
		})
	}
}

// ============================================
// HANDLER ROUTERS
// ============================================

// SellerOrdersHandler - GET uchun handler
func SellerOrdersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetSellerOrders(db)(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
		}
	}
}

// ============================================
// GET CANCELLATION REASONS
// ============================================

// GetCancellationReasons godoc
// @Summary      Bekor qilish sabablarini olish
// @Description  Dinamik bekor qilish sabablari ro'yxati
// @Tags         common
// @Produce      json
// @Success      200  {object}  models.CancellationReasonsResponse
// @Failure      500  {object}  models.AuthResponse
// @Router       /common/cancellation-reasons [get]
func GetCancellationReasons(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{
				Success: false,
				Message: "Faqat GET metodi qo'llab-quvvatlanadi",
			})
			return
		}

		query := `
			SELECT reason_text 
			FROM cancellation_reasons 
			WHERE is_active = true 
			ORDER BY sort_order ASC
		`

		rows, err := db.Query(query)
		if err != nil {
			log.Printf("❌ Cancellation reasons query xatosi: %v", err)
			writeJSON(w, http.StatusInternalServerError, models.AuthResponse{
				Success: false,
				Message: "Sabablarni olishda xatolik",
			})
			return
		}
		defer rows.Close()

		var reasons []string
		for rows.Next() {
			var reason string
			if err := rows.Scan(&reason); err != nil {
				continue
			}
			reasons = append(reasons, reason)
		}

		// Return empty array instead of null
		if reasons == nil {
			reasons = []string{}
		}

		log.Printf("✅ %d ta bekor qilish sababi qaytarildi", len(reasons))

		writeJSON(w, http.StatusOK, models.CancellationReasonsResponse{
			Success: true,
			Reasons: reasons,
		})
	}
}
