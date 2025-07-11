package order

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ranjannkumar/go-grpc-grpahql-microservice/order/pb"
	"google.golang.org/grpc"
)

type Client struct {
	conn *grpc.ClientConn
	service pb.OrderServiceClient
}

func NewClient(url string) (*Client, error) {
	var conn *grpc.ClientConn
	var err error

	maxRetries := 5
	retryInterval := 2 * time.Second
	totalTimeout := 15 * time.Second

	ctxRetry, cancelRetry := context.WithTimeout(context.Background(), totalTimeout)
	defer cancelRetry()

	for i := 0; i < maxRetries; i++ {
		log.Printf("Attempting to connect to Order service at %s (Attempt %d/%d)...", url, i+1, maxRetries)
		ctxDial, cancelDial := context.WithTimeout(context.Background(), retryInterval)
		defer cancelDial()

		conn, err = grpc.DialContext(ctxDial, url, grpc.WithInsecure(), grpc.WithBlock())
		if err == nil {
			log.Printf("Successfully connected to Order service at %s", url)
			c := pb.NewOrderServiceClient(conn)
			return &Client{conn, c}, nil
		}

		log.Printf("Failed to connect to Order service: %v. Retrying in %v...", err, retryInterval)
		select {
		case <-time.After(retryInterval):
		case <-ctxRetry.Done():
			return nil, fmt.Errorf("failed to connect to Order service after multiple retries (total timeout reached): %w", ctxRetry.Err())
		}
	}

	return nil, fmt.Errorf("failed to connect to Order service after %d retries: %w", maxRetries, err)
}


func (c *Client)PostOrder(ctx context.Context,accountID string,products []OrderedProduct)(*Order,error){
	protoProducts := []*pb.PostOrderRequest_OrderProduct{}
	for _,p := range products{
		protoProducts = append(protoProducts, &pb.PostOrderRequest_OrderProduct{
			ProductId: p.ID,
			Quantity: p.Quantity,
		})
	}
	r,err := c.service.PostOrder(
		ctx,
		&pb.PostOrderRequest{
			AccountId: accountID,
			Products: protoProducts,
		},
	)
	if err != nil{
		return nil,err
	}
	newOrder := r.Order
	newOrderCreatedAt := time.Time{}
	newOrderCreatedAt.UnmarshalBinary(newOrder.CreatedAt)

	return &Order{
		ID: newOrder.Id,
		CreatedAt: newOrderCreatedAt,
		TotalPrice: newOrder.TotalPrice,
		AccountID: newOrder.AccountId,
		Products: products,
	},nil
}

func(c *Client)GetOrdersForAccount(ctx context.Context,accountID string)([]Order,error){
	r,err := c.service.GetOrdersForAccount(ctx,&pb.GetOrdersForAccountRequest{
		AccountId: accountID,
	})

	if err !=nil{
		log.Println(err)
		return nil,err
	}
	orders := []Order{}
	for _,orderProto := range r.Orders{
		newOrder := Order{
			ID: orderProto.Id,
			TotalPrice: orderProto.TotalPrice,
			AccountID: orderProto.AccountId,
		}
		newOrder.CreatedAt = time.Time{}
		newOrder.CreatedAt.UnmarshalBinary(orderProto.CreatedAt)
		products := []OrderedProduct{}
		for _,p := range orderProto.Products{
			products = append(products, OrderedProduct{
				ID:         p.Id,
				Quantity:   p.Quantity,
				Name:       p.Name,
				Description:p.Description,
				Price:      p.Price,
			})
		}
		newOrder.Products=products
		orders = append(orders, newOrder)
	}

	return orders,nil
}