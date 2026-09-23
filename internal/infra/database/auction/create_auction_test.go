package auction

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"os"
	"strconv"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const testAuctionInterval = time.Second

func TestCreateAuctionClosesAutomatically(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	auction, internalErr := auction_entity.CreateAuction("Notebook", "Eletrônicos", "Notebook usado em bom estado", auction_entity.Used)
	if internalErr != nil {
		t.Fatal(internalErr)
	}
	if internalErr := repository.CreateAuction(ctx, auction); internalErr != nil {
		t.Fatal(internalErr)
	}

	assertStatus(t, repository, auction.Id, auction_entity.Active)

	time.Sleep(testAuctionInterval)

	waitForStatus(t, repository, auction.Id, auction_entity.Completed)
}

func TestScheduleOpenAuctionsClosing(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	expired := AuctionEntityMongo{Id: "expired", ProductName: "Notebook", Status: auction_entity.Active, Timestamp: time.Now().Add(-time.Hour).Unix()}
	running := AuctionEntityMongo{Id: "running", ProductName: "Celular", Status: auction_entity.Active, Timestamp: time.Now().Unix()}
	if _, err := repository.Collection.InsertMany(ctx, []interface{}{expired, running}); err != nil {
		t.Fatal(err)
	}

	if internalErr := repository.ScheduleOpenAuctionsClosing(ctx); internalErr != nil {
		t.Fatal(internalErr)
	}

	waitForStatus(t, repository, expired.Id, auction_entity.Completed)
	assertStatus(t, repository, running.Id, auction_entity.Active)

	time.Sleep(testAuctionInterval)

	waitForStatus(t, repository, running.Id, auction_entity.Completed)
}

func newTestRepository(t *testing.T) *AuctionRepository {
	t.Helper()

	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		t.Skip("MONGODB_URL not set")
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		t.Fatal(err)
	}

	database := client.Database("auctions_test_" + strconv.FormatInt(time.Now().UnixNano(), 10))
	t.Cleanup(func() {
		database.Drop(ctx)
		client.Disconnect(ctx)
	})

	t.Setenv("AUCTION_INTERVAL", testAuctionInterval.String())
	return NewAuctionRepository(database)
}

func assertStatus(t *testing.T, repository *AuctionRepository, id string, want auction_entity.AuctionStatus) {
	t.Helper()

	found, internalErr := repository.FindAuctionById(context.Background(), id)
	if internalErr != nil {
		t.Fatal(internalErr)
	}
	if found.Status != want {
		t.Fatalf("auction %s status = %d, want %d", id, found.Status, want)
	}
}

func waitForStatus(t *testing.T, repository *AuctionRepository, id string, want auction_entity.AuctionStatus) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		found, internalErr := repository.FindAuctionById(context.Background(), id)
		if internalErr != nil {
			t.Fatal(internalErr)
		}
		if found.Status == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("auction %s status = %d, want %d", id, found.Status, want)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
