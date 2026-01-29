package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"mebellar-backend/internal/grpc/mapper"
	"mebellar-backend/internal/grpc/middleware"
	"mebellar-backend/models"
	"mebellar-backend/pkg/pb"
	"mebellar-backend/pkg/websocket"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderServiceServer struct {
	pb.UnimplementedOrderServiceServer
	db          *sql.DB
	broadcaster *orderBroadcaster
}

func NewOrderServiceServer(db *sql.DB) *OrderServiceServer {
	return &OrderServiceServer{
		db:          db,
		broadcaster: newOrderBroadcaster(),
	}
}

func (s *OrderServiceServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	shopID := strings.TrimSpace(req.GetShopId())
	if shopID == "" && auth != nil {
		shopID = auth.ShopID
	}
	if shopID == "" {
		return nil, status.Error(codes.InvalidArgument, "shop_id is required")
	}
	if len(req.GetItems()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one item is required")
	}

	orderID := uuid.NewString()
	now := time.Now()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "tx begin error: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, shop_id, client_name, client_phone, client_address, total_amount, delivery_price, status, client_note, seller_note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, '', $10, $11)
	`, orderID, shopID, req.GetClientName(), req.GetClientPhone(), req.GetClientAddress(),
		req.GetTotalAmount(), req.GetDeliveryPrice(), models.OrderStatusNew, req.GetClientNote(), now, now)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "insert order error: %v", err)
	}

	for _, item := range req.GetItems() {
		itemID := uuid.NewString()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO order_items (id, order_id, product_id, product_name, product_image, quantity, price, created_at)
			VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8)
		`, itemID, orderID, item.GetProductId(), item.GetProductName(), item.GetProductImage(), item.GetQuantity(), item.GetPrice(), now)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "insert item error: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit error: %v", err)
	}

	order, err := s.fetchOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Broadcast to gRPC stream subscribers
	s.broadcaster.Publish(shopID, &pb.OrderEvent{
		Type:  pb.OrderEventType_ORDER_EVENT_TYPE_CREATED,
		Order: mapper.ToPBOrder(order),
	})

	// Also broadcast to WebSocket hub for backward compatibility
	if len(order.Items) > 0 {
		websocket.BroadcastNewOrder(shopID, websocket.NewOrderPayload{
			OrderID:      order.ID,
			ClientName:   order.ClientName,
			ClientPhone:  order.ClientPhone,
			TotalAmount:  order.TotalAmount,
			ProductCount: len(order.Items),
			ProductName:  order.Items[0].ProductName,
			ProductImage: order.Items[0].ProductImage,
			CreatedAt:    order.CreatedAt.Format("02.01.2006 15:04"),
		})
	}

	return &pb.OrderResponse{Order: mapper.ToPBOrder(order)}, nil
}

func (s *OrderServiceServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	order, err := s.fetchOrder(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &pb.OrderResponse{Order: mapper.ToPBOrder(order)}, nil
}

func (s *OrderServiceServer) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.OrderResponse, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	newStatus := mapper.ToModelOrderStatus(req.GetStatus())
	if !models.IsValidOrderStatus(newStatus) {
		return nil, status.Error(codes.InvalidArgument, "invalid status")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE orders SET status = $1, seller_note = COALESCE($2, seller_note), updated_at = NOW()
		WHERE id = $3
	`, newStatus, req.GetSellerNote(), req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update error: %v", err)
	}

	order, err := s.fetchOrder(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	// Broadcast to gRPC stream subscribers
	s.broadcaster.Publish(order.ShopID, &pb.OrderEvent{
		Type:  pb.OrderEventType_ORDER_EVENT_TYPE_STATUS_CHANGED,
		Order: mapper.ToPBOrder(order),
	})

	// Also broadcast to WebSocket hub for backward compatibility
	websocket.BroadcastOrderUpdate(order.ShopID, websocket.OrderUpdatePayload{
		OrderID:   order.ID,
		OldStatus: "", // We don't track old status in this context
		NewStatus: order.Status,
	})

	return &pb.OrderResponse{Order: mapper.ToPBOrder(order)}, nil
}

func (s *OrderServiceServer) DeleteOrder(ctx context.Context, req *pb.DeleteOrderRequest) (*pb.Empty, error) {
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// fetch shop for publishing
	order, err := s.fetchOrder(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM order_items WHERE order_id = $1`, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete items error: %v", err)
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM orders WHERE id = $1`, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete order error: %v", err)
	}

	// Broadcast to gRPC stream subscribers
	s.broadcaster.Publish(order.ShopID, &pb.OrderEvent{
		Type:  pb.OrderEventType_ORDER_EVENT_TYPE_DELETED,
		Order: mapper.ToPBOrder(order),
	})

	// Also broadcast to WebSocket hub for backward compatibility
	if websocket.GlobalHub != nil {
		websocket.GlobalHub.BroadcastToShop(order.ShopID, "order_deleted", map[string]interface{}{
			"order_id": order.ID,
		})
	}

	return &pb.Empty{}, nil
}

func (s *OrderServiceServer) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	auth := middleware.GetAuthContext(ctx)
	shopID := strings.TrimSpace(req.GetShopId())
	if shopID == "" && auth != nil {
		shopID = auth.ShopID
	}
	if shopID == "" {
		return nil, status.Error(codes.InvalidArgument, "shop_id is required")
	}

	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	limit := req.GetLimit()
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	args := []interface{}{shopID}
	countQuery := `SELECT COUNT(*) FROM orders WHERE shop_id = $1`
	dataQuery := `
		SELECT id, shop_id, client_name, client_phone, COALESCE(client_address, ''), total_amount, delivery_price, status, COALESCE(client_note, ''), COALESCE(seller_note, ''), created_at, updated_at, completed_at
		FROM orders
		WHERE shop_id = $1
	`
	argIndex := 2

	if len(req.GetStatuses()) > 0 {
		placeholders := make([]string, len(req.GetStatuses()))
		for i, st := range req.GetStatuses() {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, mapper.ToModelOrderStatus(st))
			argIndex++
		}
		condition := " AND status IN (" + strings.Join(placeholders, ",") + ")"
		countQuery += condition
		dataQuery += condition
	}

	dataQuery += " ORDER BY created_at DESC"
	dataQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)

	var total int32
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, status.Errorf(codes.Internal, "count error: %v", err)
	}

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	defer rows.Close()

	var orders []models.Order
	var orderIDs []string
	for rows.Next() {
		var o models.Order
		var completedAt sql.NullTime
		if err := rows.Scan(
			&o.ID, &o.ShopID, &o.ClientName, &o.ClientPhone, &o.ClientAddress,
			&o.TotalAmount, &o.DeliveryPrice, &o.Status, &o.ClientNote, &o.SellerNote,
			&o.CreatedAt, &o.UpdatedAt, &completedAt,
		); err != nil {
			log.Printf("order scan error: %v", err)
			continue
		}
		if completedAt.Valid {
			o.CompletedAt = &completedAt.Time
		}
		orders = append(orders, o)
		orderIDs = append(orderIDs, o.ID)
	}

	if len(orderIDs) > 0 {
		itemsByOrder, err := s.fetchItemsForOrders(ctx, orderIDs)
		if err == nil {
			for i := range orders {
				if items, ok := itemsByOrder[orders[i].ID]; ok {
					orders[i].Items = items
					orders[i].ItemsCount = len(items)
				}
			}
		}
	}

	resp := &pb.ListOrdersResponse{
		Total: total,
		Page:  page,
		Limit: limit,
	}
	for _, o := range orders {
		resp.Orders = append(resp.Orders, mapper.ToPBOrder(o))
	}
	return resp, nil
}

func (s *OrderServiceServer) StreamOrders(req *pb.StreamOrdersRequest, stream pb.OrderService_StreamOrdersServer) error {
	auth := middleware.GetAuthContext(stream.Context())
	shopID := strings.TrimSpace(req.GetShopId())
	if shopID == "" && auth != nil {
		shopID = auth.ShopID
	}
	if shopID == "" {
		return status.Error(codes.InvalidArgument, "shop_id is required")
	}

	var statusFilter map[pb.OrderStatus]struct{}
	if len(req.GetStatuses()) > 0 {
		statusFilter = make(map[pb.OrderStatus]struct{})
		for _, st := range req.GetStatuses() {
			statusFilter[st] = struct{}{}
		}
	}

	events, cancel := s.broadcaster.Subscribe(shopID)
	defer cancel()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case evt := <-events:
			if evt == nil || evt.Order == nil {
				continue
			}
			if statusFilter != nil {
				if _, ok := statusFilter[evt.Order.Status]; !ok {
					continue
				}
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}

// fetchOrder loads order with items.
func (s *OrderServiceServer) fetchOrder(ctx context.Context, orderID string) (models.Order, error) {
	var o models.Order
	var completedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, shop_id, client_name, client_phone, COALESCE(client_address, ''), total_amount, delivery_price, status, COALESCE(client_note, ''), COALESCE(seller_note, ''), created_at, updated_at, completed_at
		FROM orders
		WHERE id = $1
	`, orderID).Scan(
		&o.ID, &o.ShopID, &o.ClientName, &o.ClientPhone, &o.ClientAddress,
		&o.TotalAmount, &o.DeliveryPrice, &o.Status, &o.ClientNote, &o.SellerNote,
		&o.CreatedAt, &o.UpdatedAt, &completedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return o, status.Error(codes.NotFound, "order not found")
		}
		return o, status.Errorf(codes.Internal, "query error: %v", err)
	}
	if completedAt.Valid {
		o.CompletedAt = &completedAt.Time
	}

	items, err := s.fetchItemsForOrders(ctx, []string{orderID})
	if err == nil {
		o.Items = items[orderID]
		o.ItemsCount = len(o.Items)
	}
	return o, nil
}

func (s *OrderServiceServer) fetchItemsForOrders(ctx context.Context, orderIDs []string) (map[string][]models.OrderItem, error) {
	placeholders := make([]string, len(orderIDs))
	args := make([]interface{}, len(orderIDs))
	for i, id := range orderIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, order_id, product_id, product_name, product_image, quantity, price, created_at
		FROM order_items
		WHERE order_id IN (%s)
		ORDER BY created_at ASC
	`, strings.Join(placeholders, ",")), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]models.OrderItem)
	for rows.Next() {
		var item models.OrderItem
		var productID sql.NullString
		if err := rows.Scan(
			&item.ID, &item.OrderID, &productID, &item.ProductName, &item.ProductImage, &item.Quantity, &item.Price, &item.CreatedAt,
		); err != nil {
			continue
		}
		if productID.Valid {
			item.ProductID = &productID.String
		}
		result[item.OrderID] = append(result[item.OrderID], item)
	}
	return result, nil
}

// orderBroadcaster is a lightweight pubsub used to replace the WebSocket hub for streaming orders.
type orderBroadcaster struct {
	mu           sync.RWMutex
	subscribers  map[string]map[chan *pb.OrderEvent]struct{}
	bufferLength int
}

func newOrderBroadcaster() *orderBroadcaster {
	return &orderBroadcaster{
		subscribers:  make(map[string]map[chan *pb.OrderEvent]struct{}),
		bufferLength: 32,
	}
}

func (b *orderBroadcaster) Subscribe(shopID string) (<-chan *pb.OrderEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan *pb.OrderEvent, b.bufferLength)
	if _, ok := b.subscribers[shopID]; !ok {
		b.subscribers[shopID] = make(map[chan *pb.OrderEvent]struct{})
	}
	b.subscribers[shopID][ch] = struct{}{}

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if subs, ok := b.subscribers[shopID]; ok {
			delete(subs, ch)
			close(ch)
			if len(subs) == 0 {
				delete(b.subscribers, shopID)
			}
		}
	}
	return ch, cancel
}

func (b *orderBroadcaster) Publish(shopID string, evt *pb.OrderEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers[shopID] {
		select {
		case ch <- evt:
		default:
			// drop if buffer is full to avoid blocking
		}
	}
}
