package mapper

import (
	"mebellar-backend/models"
	"mebellar-backend/pkg/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToPBOrderStatus maps domain status string to proto enum.
func ToPBOrderStatus(status string) pb.OrderStatus {
	switch status {
	case models.OrderStatusNew:
		return pb.OrderStatus_ORDER_STATUS_NEW
	case models.OrderStatusConfirmed:
		return pb.OrderStatus_ORDER_STATUS_CONFIRMED
	case models.OrderStatusShipping:
		return pb.OrderStatus_ORDER_STATUS_SHIPPING
	case models.OrderStatusCompleted:
		return pb.OrderStatus_ORDER_STATUS_COMPLETED
	case models.OrderStatusCancelled:
		return pb.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return pb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

// ToModelOrderStatus maps proto enum to domain string.
func ToModelOrderStatus(status pb.OrderStatus) string {
	switch status {
	case pb.OrderStatus_ORDER_STATUS_NEW:
		return models.OrderStatusNew
	case pb.OrderStatus_ORDER_STATUS_CONFIRMED:
		return models.OrderStatusConfirmed
	case pb.OrderStatus_ORDER_STATUS_SHIPPING:
		return models.OrderStatusShipping
	case pb.OrderStatus_ORDER_STATUS_COMPLETED:
		return models.OrderStatusCompleted
	case pb.OrderStatus_ORDER_STATUS_CANCELLED:
		return models.OrderStatusCancelled
	default:
		return models.OrderStatusNew
	}
}

func ToPBOrderItem(item models.OrderItem) *pb.OrderItem {
	pbItem := &pb.OrderItem{
		Id:          item.ID,
		OrderId:     item.OrderID,
		ProductName: item.ProductName,
		ProductImage: item.ProductImage,
		Quantity:    int32(item.Quantity),
		Price:       item.Price,
		CreatedAt:   timestamppb.New(item.CreatedAt),
	}
	if item.ProductID != nil {
		pbItem.ProductId = *item.ProductID
	}
	return pbItem
}

func ToPBOrder(order models.Order) *pb.Order {
	pbOrder := &pb.Order{
		Id:            order.ID,
		ShopId:        order.ShopID,
		ShopName:      order.ShopName,
		ClientName:    order.ClientName,
		ClientPhone:   order.ClientPhone,
		ClientAddress: order.ClientAddress,
		TotalAmount:   order.TotalAmount,
		DeliveryPrice: order.DeliveryPrice,
		Status:        ToPBOrderStatus(order.Status),
		ClientNote:    order.ClientNote,
		SellerNote:    order.SellerNote,
		ItemsCount:    int32(order.ItemsCount),
		CreatedAt:     timestamppb.New(order.CreatedAt),
		UpdatedAt:     timestamppb.New(order.UpdatedAt),
	}
	if order.CompletedAt != nil {
		pbOrder.CompletedAt = timestamppb.New(*order.CompletedAt)
	}
	for _, item := range order.Items {
		pbOrder.Items = append(pbOrder.Items, ToPBOrderItem(item))
	}
	return pbOrder
}
