package httphandler

import (
	"github.com/db-mipt/game-client/internal/model"
	"github.com/db-mipt/game-client/internal/repository"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

type HttpHandler struct {
	log  *logrus.Entry
	repo *repository.PlayerRepository
}

func NewHttpHandler(log *logrus.Entry, repo *repository.PlayerRepository) *HttpHandler {
	return &HttpHandler{
		log:  log,
		repo: repo,
	}
}

func (ctrl *HttpHandler) UpdateScore(e echo.Context) error {
	updateScoreReq := new(model.UpdateScoreRequest)
	err := e.Bind(updateScoreReq)

	if err != nil {
		ctrl.log.Errorf("Failed to bind request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid input data")
	}

	err = ctrl.repo.UpdateScore(updateScoreReq.Id, updateScoreReq.Score)
	if err != nil {
		ctrl.log.Errorf("Failed to update score for %v: %v", updateScoreReq, err)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return e.JSON(http.StatusOK, updateScoreReq)
}

func (ctrl *HttpHandler) GetLeaderboard(e echo.Context) error {
	count, err := strconv.Atoi(e.Request().Header.Get("count"))
	if err != nil {
		ctrl.log.Errorf("Failed to parse request: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid input data")
	}

	res, err := ctrl.repo.GetTopNPlayers(int64(count))
	if err != nil {
		ctrl.log.Errorf("Failed to get player Leaderboard: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return e.JSON(http.StatusOK, res)
}

func (ctrl *HttpHandler) GetPlayerRank(e echo.Context) error {
	id := e.Request().Header.Get("id")

	res, err := ctrl.repo.GetPlayerRank(id)
	if err != nil {
		ctrl.log.Errorf("Failed to get player %v rank: %v", id, err)
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	ctrl.log.Infof("Result is %v", res)

	return e.JSON(http.StatusOK, res)
}
