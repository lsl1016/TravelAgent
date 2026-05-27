// Package service 实现用户行程历史业务逻辑。
package service

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"travelagent/backend/internal/cache"
	"travelagent/backend/internal/model"
	"travelagent/backend/internal/repository"
)

// 负责用户行程历史的 CRUD。
type TripService struct {
	store *repository.Store
	cache *cache.Cache
}

// 创建行程服务。
func NewTripService(store *repository.Store, cache *cache.Cache) *TripService {
	return &TripService{store: store, cache: cache}
}

// 创建或覆盖一条用户行程，并失效长期摘要缓存。
func (s *TripService) Create(ctx context.Context, userID string, req model.TripRequest) (*model.TripDTO, error) {
	if userID == "" {
		return nil, InvalidArgument("user_id is required")
	}
	start, err := parseDate(req.StartDate)
	if err != nil {
		return nil, NewError(http.StatusBadRequest, CodeInvalidArgument, "start_date must be YYYY-MM-DD", err)
	}
	end, err := parseDate(req.EndDate)
	if err != nil {
		return nil, NewError(http.StatusBadRequest, CodeInvalidArgument, "end_date must be YYYY-MM-DD", err)
	}
	tripID := req.TripID
	if tripID == "" {
		tripID = "trip_" + uuid.NewString()
	}
	trip := model.Trip{
		TripID:        tripID,
		UserID:        userID,
		SessionID:     req.SessionID,
		Origin:        req.Origin,
		Destination:   req.Destination,
		StartDate:     start,
		EndDate:       end,
		Purpose:       req.Purpose,
		ItineraryJSON: rawToJSON(req.ItineraryJSON),
		RawJSON:       rawToJSON(req.RawJSON),
	}
	saved, err := s.store.UpsertTrip(ctx, trip)
	if err != nil {
		return nil, MapError(err)
	}
	_ = s.cache.InvalidateSummary(ctx, userID)
	dto := tripDTO(*saved)
	return &dto, nil
}

// List 分页查询用户历史行程。
func (s *TripService) List(ctx context.Context, userID string, limit, offset int) ([]model.TripDTO, error) {
	if userID == "" {
		return nil, InvalidArgument("user_id is required")
	}
	trips, err := s.store.ListTrips(ctx, userID, limit, offset)
	if err != nil {
		return nil, MapError(err)
	}
	dtos := make([]model.TripDTO, 0, len(trips))
	for _, trip := range trips {
		dtos = append(dtos, tripDTO(trip))
	}
	return dtos, nil
}

// 查询用户指定行程。
func (s *TripService) Get(ctx context.Context, userID, tripID string) (*model.TripDTO, error) {
	if userID == "" || tripID == "" {
		return nil, InvalidArgument("user_id and trip_id are required")
	}
	trip, err := s.store.GetTrip(ctx, userID, tripID)
	if err != nil {
		return nil, MapError(err)
	}
	dto := tripDTO(*trip)
	return &dto, nil
}

// 删除用户指定行程，并失效长期摘要缓存。
func (s *TripService) Delete(ctx context.Context, userID, tripID string) error {
	if userID == "" || tripID == "" {
		return InvalidArgument("user_id and trip_id are required")
	}
	if err := s.store.DeleteTrip(ctx, userID, tripID); err != nil {
		return MapError(err)
	}
	_ = s.cache.InvalidateSummary(ctx, userID)
	return nil
}
