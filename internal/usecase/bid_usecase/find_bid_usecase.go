package bid_usecase

import (
	"context"
	"fullcycle-auction_go/internal/internal_error"
)

func (bu *BidUseCase) FindBidByAuctionId(
	ctx context.Context, auctionId string) ([]BidOutputDTO, *internal_error.InternalError) {
	bidEntities, err := bu.BidRepository.FindBidByAuctionId(ctx, auctionId)
	if err != nil {
		return nil, err
	}

	var bidOutputDTOs []BidOutputDTO
	for _, bid := range bidEntities {
		bidOutputDTOs = append(bidOutputDTOs, BidOutputDTO{
			Id:        bid.Id,
			UserId:    bid.UserId,
			AuctionId: bid.AuctionId,
			Amount:    bid.Amount,
			Timestamp: bid.Timestamp,
		})
	}

	return bidOutputDTOs, nil
}

func (bu *BidUseCase) FindWinningBidByAuctionId(
	ctx context.Context, auctionId string) (*BidOutputDTO, *internal_error.InternalError) {
	bidEntity, err := bu.BidRepository.FindWinningBidByAuctionId(ctx, auctionId)
	if err != nil {
		return nil, err
	}

	return &BidOutputDTO{
		Id:        bidEntity.Id,
		UserId:    bidEntity.UserId,
		AuctionId: bidEntity.AuctionId,
		Amount:    bidEntity.Amount,
		Timestamp: bidEntity.Timestamp,
	}, nil
}
