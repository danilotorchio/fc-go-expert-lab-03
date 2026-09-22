package auction

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCreateAuctionClosesAutomatically(t *testing.T) {
	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		t.Skip("MONGODB_URL not set")
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)

	database := client.Database("auctions_test_" + time.Now().Format("20060102150405"))
	defer database.Drop(ctx)

	interval := time.Second
	t.Setenv("AUCTION_INTERVAL", interval.String())
	repository := NewAuctionRepository(database)

	auction, internalErr := auction_entity.CreateAuction("Notebook", "Eletrônicos", "Notebook usado em bom estado", auction_entity.Used)
	if internalErr != nil {
		t.Fatal(internalErr)
	}
	if internalErr := repository.CreateAuction(ctx, auction); internalErr != nil {
		t.Fatal(internalErr)
	}

	created, internalErr := repository.FindAuctionById(ctx, auction.Id)
	if internalErr != nil {
		t.Fatal(internalErr)
	}
	if created.Status != auction_entity.Active {
		t.Fatalf("status after create = %d, want Active", created.Status)
	}

	time.Sleep(interval)

	deadline := time.Now().Add(5 * time.Second)
	for {
		found, internalErr := repository.FindAuctionById(ctx, auction.Id)
		if internalErr != nil {
			t.Fatal(internalErr)
		}
		if found.Status == auction_entity.Completed {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("status after %s = %d, want Completed", interval, found.Status)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
