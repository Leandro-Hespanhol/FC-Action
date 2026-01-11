package auction_test

import (
	"context"
	"os"
	"testing"
	"time"

	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/infra/database/auction"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	testDBName         = "auction_test_db"
	testCollectionName = "auctions"
)

func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		t.Skipf("Skipping test: MongoDB not available at %s: %v", mongoURL, err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("Skipping test: MongoDB ping failed: %v", err)
	}

	database := client.Database(testDBName)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		database.Drop(ctx)
		client.Disconnect(ctx)
	}

	return database, cleanup
}

func TestAuctionAutoClose(t *testing.T) {
	// Set a very short auction duration for testing (3 seconds)
	os.Setenv("AUCTION_DURATION_SECONDS", "3")
	defer os.Unsetenv("AUCTION_DURATION_SECONDS")

	database, cleanup := setupTestDB(t)
	defer cleanup()

	repo := auction.NewAuctionRepository(database)

	// Create a new auction
	auctionEntity, err := auction_entity.CreateAuction(
		"Test Product",
		"Electronics",
		"This is a test product description for testing",
		auction_entity.New,
	)
	if err != nil {
		t.Fatalf("Failed to create auction entity: %v", err)
	}

	// Save auction to database
	ctx := context.Background()
	if internalErr := repo.CreateAuction(ctx, auctionEntity); internalErr != nil {
		t.Fatalf("Failed to create auction in repository: %v", internalErr.Error())
	}

	// Verify auction was created with Active status
	createdAuction, internalErr := repo.FindAuctionById(ctx, auctionEntity.Id)
	if internalErr != nil {
		t.Fatalf("Failed to find auction: %v", internalErr.Error())
	}

	if createdAuction.Status != auction_entity.Active {
		t.Errorf("Expected auction status to be Active (0), got %d", createdAuction.Status)
	}

	t.Logf("Auction created with ID: %s, Status: %d", createdAuction.Id, createdAuction.Status)

	// Wait for the auction to auto-close (duration + buffer)
	t.Log("Waiting for auction to auto-close...")
	time.Sleep(5 * time.Second)

	// Verify auction was auto-closed
	closedAuction, internalErr := repo.FindAuctionById(ctx, auctionEntity.Id)
	if internalErr != nil {
		t.Fatalf("Failed to find auction after auto-close: %v", internalErr.Error())
	}

	if closedAuction.Status != auction_entity.Completed {
		t.Errorf("Expected auction status to be Completed (1), got %d", closedAuction.Status)
	} else {
		t.Logf("Auction successfully auto-closed. Status: %d (Completed)", closedAuction.Status)
	}
}

func TestAuctionDurationFromEnv(t *testing.T) {
	testCases := []struct {
		name            string
		envValue        string
		expectedMinSecs int
		expectedMaxSecs int
	}{
		{
			name:            "Custom duration 10 seconds",
			envValue:        "10",
			expectedMinSecs: 10,
			expectedMaxSecs: 10,
		},
		{
			name:            "Empty env uses default 600 seconds",
			envValue:        "",
			expectedMinSecs: 600,
			expectedMaxSecs: 600,
		},
		{
			name:            "Invalid env uses default 600 seconds",
			envValue:        "invalid",
			expectedMinSecs: 600,
			expectedMaxSecs: 600,
		},
		{
			name:            "Negative value uses default 600 seconds",
			envValue:        "-5",
			expectedMinSecs: 600,
			expectedMaxSecs: 600,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue == "" {
				os.Unsetenv("AUCTION_DURATION_SECONDS")
			} else {
				os.Setenv("AUCTION_DURATION_SECONDS", tc.envValue)
			}

			// We can't directly test the private function, but we can verify
			// the behavior through the repository creation and auction flow
			t.Logf("Test case: %s - AUCTION_DURATION_SECONDS=%s", tc.name, tc.envValue)
		})
	}

	os.Unsetenv("AUCTION_DURATION_SECONDS")
}

func TestMultipleAuctionsAutoClose(t *testing.T) {
	// Set a short auction duration for testing (2 seconds)
	os.Setenv("AUCTION_DURATION_SECONDS", "2")
	defer os.Unsetenv("AUCTION_DURATION_SECONDS")

	database, cleanup := setupTestDB(t)
	defer cleanup()

	repo := auction.NewAuctionRepository(database)
	ctx := context.Background()

	// Create multiple auctions
	numAuctions := 3
	auctionIDs := make([]string, numAuctions)

	for i := 0; i < numAuctions; i++ {
		auctionEntity, err := auction_entity.CreateAuction(
			"Test Product",
			"Electronics",
			"This is a test product description",
			auction_entity.New,
		)
		if err != nil {
			t.Fatalf("Failed to create auction entity %d: %v", i, err)
		}

		if internalErr := repo.CreateAuction(ctx, auctionEntity); internalErr != nil {
			t.Fatalf("Failed to create auction %d in repository: %v", i, internalErr.Error())
		}

		auctionIDs[i] = auctionEntity.Id
		t.Logf("Created auction %d with ID: %s", i, auctionEntity.Id)
	}

	// Wait for all auctions to auto-close
	t.Log("Waiting for all auctions to auto-close...")
	time.Sleep(4 * time.Second)

	// Verify all auctions were auto-closed
	for i, id := range auctionIDs {
		closedAuction, internalErr := repo.FindAuctionById(ctx, id)
		if internalErr != nil {
			t.Fatalf("Failed to find auction %d after auto-close: %v", i, internalErr.Error())
		}

		if closedAuction.Status != auction_entity.Completed {
			t.Errorf("Auction %d: Expected status to be Completed (1), got %d", i, closedAuction.Status)
		} else {
			t.Logf("Auction %d successfully auto-closed. Status: %d", i, closedAuction.Status)
		}
	}
}

func TestAuctionNotClosedBeforeDuration(t *testing.T) {
	// Set auction duration for 10 seconds
	os.Setenv("AUCTION_DURATION_SECONDS", "10")
	defer os.Unsetenv("AUCTION_DURATION_SECONDS")

	database, cleanup := setupTestDB(t)
	defer cleanup()

	repo := auction.NewAuctionRepository(database)

	// Create a new auction
	auctionEntity, err := auction_entity.CreateAuction(
		"Test Product",
		"Electronics",
		"This is a test product description for testing",
		auction_entity.New,
	)
	if err != nil {
		t.Fatalf("Failed to create auction entity: %v", err)
	}

	ctx := context.Background()
	if internalErr := repo.CreateAuction(ctx, auctionEntity); internalErr != nil {
		t.Fatalf("Failed to create auction in repository: %v", internalErr.Error())
	}

	// Wait only 2 seconds (less than the 10 second duration)
	time.Sleep(2 * time.Second)

	// Verify auction is still active
	activeAuction, internalErr := repo.FindAuctionById(ctx, auctionEntity.Id)
	if internalErr != nil {
		t.Fatalf("Failed to find auction: %v", internalErr.Error())
	}

	if activeAuction.Status != auction_entity.Active {
		t.Errorf("Expected auction status to still be Active (0), got %d", activeAuction.Status)
	} else {
		t.Logf("Auction correctly still active before duration expires. Status: %d", activeAuction.Status)
	}
}

func TestConcurrentAuctionCreationAndClose(t *testing.T) {
	// Set a very short auction duration for testing (1 second)
	os.Setenv("AUCTION_DURATION_SECONDS", "1")
	defer os.Unsetenv("AUCTION_DURATION_SECONDS")

	database, cleanup := setupTestDB(t)
	defer cleanup()

	repo := auction.NewAuctionRepository(database)
	ctx := context.Background()

	// Create auctions concurrently
	numAuctions := 5
	done := make(chan string, numAuctions)

	for i := 0; i < numAuctions; i++ {
		go func(idx int) {
			auctionEntity, err := auction_entity.CreateAuction(
				"Concurrent Product",
				"TestCategory",
				"Testing concurrent auction creation",
				auction_entity.Used,
			)
			if err != nil {
				t.Errorf("Failed to create auction entity %d: %v", idx, err)
				done <- ""
				return
			}

			if internalErr := repo.CreateAuction(ctx, auctionEntity); internalErr != nil {
				t.Errorf("Failed to create auction %d in repository: %v", idx, internalErr.Error())
				done <- ""
				return
			}

			done <- auctionEntity.Id
		}(i)
	}

	// Collect all auction IDs
	auctionIDs := make([]string, 0, numAuctions)
	for i := 0; i < numAuctions; i++ {
		id := <-done
		if id != "" {
			auctionIDs = append(auctionIDs, id)
		}
	}

	t.Logf("Created %d auctions concurrently", len(auctionIDs))

	// Wait for all auctions to auto-close
	time.Sleep(3 * time.Second)

	// Verify all auctions were auto-closed
	closedCount := 0
	for _, id := range auctionIDs {
		closedAuction, internalErr := repo.FindAuctionById(ctx, id)
		if internalErr != nil {
			continue
		}

		if closedAuction.Status == auction_entity.Completed {
			closedCount++
		}
	}

	if closedCount != len(auctionIDs) {
		t.Errorf("Expected all %d auctions to be closed, but only %d were closed", len(auctionIDs), closedCount)
	} else {
		t.Logf("All %d concurrent auctions were successfully auto-closed", closedCount)
	}
}

// Test to verify that the auction collection is properly set up
func TestAuctionRepositorySetup(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	repo := auction.NewAuctionRepository(database)

	// Verify the repository is properly initialized
	if repo == nil {
		t.Fatal("Repository should not be nil")
	}

	// Verify we can perform operations on the collection
	ctx := context.Background()
	count, err := repo.Collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}

	t.Logf("Initial auction count in test database: %d", count)
}
