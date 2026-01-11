package auction_controller

import (
	"context"
	"fullcycle-auction_go/configuration/rest_err"
	"fullcycle-auction_go/internal/usecase/auction_usecase"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (u *AuctionController) FindAuctions(c *gin.Context) {
	status := c.Query("status")
	category := c.Query("category")
	productName := c.Query("productName")

	statusNumber, err := strconv.Atoi(status)
	if err != nil {
		statusNumber = 0
	}

	auctions, errInternal := u.auctionUseCase.FindAuctions(
		context.Background(),
		auction_usecase.AuctionStatus(statusNumber),
		category,
		productName)
	if errInternal != nil {
		errRest := rest_err.ConvertError(errInternal)
		c.JSON(errRest.Code, errRest)
		return
	}

	c.JSON(http.StatusOK, auctions)
}

func (u *AuctionController) FindAuctionById(c *gin.Context) {
	auctionId := c.Param("auctionId")

	if err := uuid.Validate(auctionId); err != nil {
		errRest := rest_err.NewBadRequestError("Invalid fields", rest_err.Causes{
			Field:   "auctionId",
			Message: "Invalid UUID value",
		})

		c.JSON(errRest.Code, errRest)
		return
	}

	auctionData, errInternal := u.auctionUseCase.FindAuctionById(context.Background(), auctionId)
	if errInternal != nil {
		errRest := rest_err.ConvertError(errInternal)
		c.JSON(errRest.Code, errRest)
		return
	}

	c.JSON(http.StatusOK, auctionData)
}

func (u *AuctionController) FindWinningBidByAuctionId(c *gin.Context) {
	auctionId := c.Param("auctionId")

	if err := uuid.Validate(auctionId); err != nil {
		errRest := rest_err.NewBadRequestError("Invalid fields", rest_err.Causes{
			Field:   "auctionId",
			Message: "Invalid UUID value",
		})

		c.JSON(errRest.Code, errRest)
		return
	}

	auctionData, errInternal := u.auctionUseCase.FindWinningBidByAuctionId(context.Background(), auctionId)
	if errInternal != nil {
		errRest := rest_err.ConvertError(errInternal)
		c.JSON(errRest.Code, errRest)
		return
	}

	c.JSON(http.StatusOK, auctionData)
}
