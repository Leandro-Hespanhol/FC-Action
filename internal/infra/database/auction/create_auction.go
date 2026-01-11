package auction

import (
	"context"
	"os"
	"strconv"
	"time"

	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	Active = iota
	Finished
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}

type AuctionRepository struct {
	Collection *mongo.Collection
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection: database.Collection("auctions"),
	}
}

func getAuctionDuration() time.Duration {
	v := os.Getenv("AUCTION_DURATION_SECONDS")
	if v == "" {
		return time.Duration(600) * time.Second
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs <= 0 {
		return time.Duration(600) * time.Second
	}
	return time.Duration(secs) * time.Second
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {

	if auctionEntity.Timestamp.IsZero() {
		auctionEntity.Timestamp = time.Now()
	}

	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}

	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	duration := getAuctionDuration()
	createdAt := time.Unix(auctionEntityMongo.Timestamp, 0)
	elapsed := time.Since(createdAt)
	var remaining time.Duration
	if elapsed >= duration {
		remaining = 0
	} else {
		remaining = duration - elapsed
	}

	go func(auctionID string, wait time.Duration) {
		if wait > 0 {
			timer := time.NewTimer(wait)
			<-timer.C
		}

		updateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		filter := bson.M{"_id": auctionID, "status": Active}
		update := bson.M{
			"$set": bson.M{
				"status": auction_entity.AuctionStatus(Finished),
			},
		}

		opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

		var updated AuctionEntityMongo
		err := ar.Collection.FindOneAndUpdate(updateCtx, filter, update, opts).Decode(&updated)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				logger.Info("Auction already closed or not found, skipping auto-close")
				return
			}

			logger.Error("Error auto-closing auction", err)
			return
		}

		logger.Info("Auction auto-closed")
	}(auctionEntityMongo.Id, remaining)

	return nil
}
