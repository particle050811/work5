package handler

import (
	"context"

	userv1 "example.com/fanone/gen-rpc/kitex_gen/user/v1"
	"example.com/fanone/services/user/internal/repository"
	"example.com/fanone/services/user/internal/repository/model"
	"example.com/fanone/services/user/internal/service"
)

type RPCHandler struct {
	store  *repository.Store
	syncer service.UserReplicaSyncer
}

func NewRPCHandler(store *repository.Store, syncer service.UserReplicaSyncer) *RPCHandler {
	return &RPCHandler{store: store, syncer: syncer}
}

func (h *RPCHandler) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	svc := service.NewUserService(h.store, h.syncer)
	result, err := svc.Register(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	return &userv1.RegisterResponse{User: modelToUserProfile(result.User)}, nil
}

func (h *RPCHandler) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	svc := service.NewUserService(h.store, h.syncer)
	result, err := svc.Login(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	return &userv1.LoginResponse{
		User:         modelToUserProfile(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (h *RPCHandler) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.RefreshTokenResponse, error) {
	svc := service.NewUserService(h.store, h.syncer)
	result, err := svc.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, err
	}
	return &userv1.RefreshTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (h *RPCHandler) GetUserInfo(ctx context.Context, req *userv1.GetUserInfoRequest) (*userv1.GetUserInfoResponse, error) {
	svc := service.NewUserService(h.store, h.syncer)
	user, err := svc.GetUserByID(ctx, uint(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	return &userv1.GetUserInfoResponse{User: modelToUserProfile(user)}, nil
}

func (h *RPCHandler) UpdateAvatar(ctx context.Context, req *userv1.UpdateAvatarRequest) (*userv1.UpdateAvatarResponse, error) {
	svc := service.NewUserService(h.store, h.syncer)
	user, err := svc.UpdateAvatar(ctx, uint(req.GetUserId()), req.GetAvatarUrl())
	if err != nil {
		return nil, err
	}
	return &userv1.UpdateAvatarResponse{User: modelToUserProfile(user)}, nil
}

func (h *RPCHandler) SyncUser(ctx context.Context, req *userv1.SyncUserRequest) (*userv1.SyncUserResponse, error) {
	user := req.GetUser()
	svc := service.NewUserService(h.store, h.syncer)
	if err := svc.SyncUser(ctx, &model.User{
		ID:        uint(user.GetId()),
		Username:  user.GetUsername(),
		AvatarURL: user.GetAvatarUrl(),
	}); err != nil {
		return nil, err
	}
	return &userv1.SyncUserResponse{}, nil
}

func modelToUserProfile(user *model.User) *userv1.UserProfile {
	if user == nil {
		return nil
	}
	out := &userv1.UserProfile{
		Id:        uint64(user.ID),
		Username:  user.Username,
		AvatarUrl: user.AvatarURL,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if user.DeletedAt.Valid {
		out.DeletedAt = user.DeletedAt.Time.Format("2006-01-02 15:04:05")
	}
	return out
}
