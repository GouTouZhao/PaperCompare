package user

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	protosdkws "github.com/GouTouZhao/PaperCompare/internal/protocol/sdkws"
	protouser "github.com/GouTouZhao/PaperCompare/internal/protocol/user"
	"github.com/GouTouZhao/PaperCompare/pkg/common/config"
	"github.com/GouTouZhao/PaperCompare/pkg/common/storage/database/model"
	"github.com/GouTouZhao/PaperCompare/pkg/common/storage/redis"
	"github.com/GouTouZhao/PaperCompare/pkg/dbbuild"
	"github.com/GouTouZhao/PaperCompare/tools/discovery"
	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"github.com/GouTouZhao/PaperCompare/tools/snowflake"
	"github.com/GouTouZhao/PaperCompare/tools/token"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type Config struct {
	config.ALLConfig
	ConfigPath config.Path
	Index      config.Index
}

type UserServer struct {
	protouser.UnimplementedUserServer
	config *Config
	db     *gorm.DB
	idGen  *snowflake.Snowflake
}

func Start(ctx context.Context, cfg *Config, client discovery.SvcDiscoveryRegistry, server grpc.ServiceRegistrar) error {
	_ = redis.InitRedis(ctx, &cfg.Redis)

	dbB := dbbuild.NewBuilder(&cfg.Postgres)
	sqlcli, err := dbB.Postgres(ctx)
	if err != nil {
		return err
	}

	if err := model.AutoMigrate(sqlcli); err != nil {
		return err
	}

	idGen, err := snowflake.NewPodIPHashSnowflake()
	if err != nil {
		return err
	}

	u := &UserServer{
		config: cfg,
		db:     sqlcli,
		idGen:  idGen,
	}

	u.seedDefaultUser(ctx)

	protouser.RegisterUserServer(server, u)
	return nil
}

func (u *UserServer) seedDefaultUser(ctx context.Context) {
	var count int64
	u.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", 10001).Count(&count)
	if count > 0 {
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	fieldsJSON, _ := json.Marshal([]string{"Federated Learning", "Knowledge Distillation", "Prototype Learning", "Large Language Models", "Computer Vision"})
	demoUser := &model.User{
		ID:             10001,
		Email:          "demo@papercompare.org",
		PasswordHash:   string(hash),
		Nickname:       "Dr. GouTou",
		Bio:            "分布式机器学习与联邦学习研究者 · PaperCompare 核心节点维护者",
		ResearchFields: string(fieldsJSON),
		Role:           "admin",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	_ = u.db.WithContext(ctx).Create(demoUser).Error
}

func toProtoUser(m *model.User) *protouser.UserInfo {
	var fields []string
	if m.ResearchFields != "" {
		_ = json.Unmarshal([]byte(m.ResearchFields), &fields)
	}
	return &protouser.UserInfo{
		Id:             m.ID,
		Email:          m.Email,
		Nickname:       m.Nickname,
		Bio:            m.Bio,
		ResearchFields: fields,
		Role:           m.Role,
		CreatedAt:      m.CreatedAt.Unix(),
	}
}

func (u *UserServer) Register(ctx context.Context, req *protouser.RegisterReq) (*protouser.AuthResp, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, errs.ErrArgs.WrapMsg("请输入有效的邮箱地址")
	}
	if len(req.Password) < 6 {
		return nil, errs.ErrArgs.WrapMsg("密码长度至少为 6 位")
	}

	var existing model.User
	err := u.db.WithContext(ctx).Where("email = ?", email).First(&existing).Error
	if err == nil {
		return nil, errs.ErrDuplicate.WrapMsg("该邮箱已被注册")
	} else if err != gorm.ErrRecordNotFound {
		return nil, errs.ErrInternalServer.WrapMsg(err.Error())
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errs.ErrInternalServer.WrapMsg("密码加密失败")
	}

	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		parts := strings.Split(email, "@")
		nickname = parts[0]
	}

	id, _ := u.idGen.NextID()
	defaultFields, _ := json.Marshal([]string{"Federated Learning", "Deep Learning"})
	newUser := &model.User{
		ID:             id,
		Email:          email,
		PasswordHash:   string(hashedBytes),
		Nickname:       nickname,
		Bio:            "科研文献探索者",
		ResearchFields: string(defaultFields),
		Role:           "researcher",
	}

	if err := u.db.WithContext(ctx).Create(newUser).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg("创建用户失败: " + err.Error())
	}

	accessToken, refreshToken, err := token.GenerateTokens(ctx, redis.GetRedis(), newUser.ID, newUser.Email)
	if err != nil {
		return nil, errs.ErrInternalServer.WrapMsg("生成登录凭证失败: " + err.Error())
	}

	return &protouser.AuthResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toProtoUser(newUser),
	}, nil
}

func (u *UserServer) Login(ctx context.Context, req *protouser.LoginReq) (*protouser.AuthResp, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		return nil, errs.ErrArgs.WrapMsg("邮箱和密码不能为空")
	}

	var usr model.User
	if err := u.db.WithContext(ctx).Where("email = ?", email).First(&usr).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.ErrArgs.WrapMsg("账号或密码错误")
		}
		return nil, errs.ErrInternalServer.WrapMsg("查询用户失败: " + err.Error())
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usr.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errs.ErrArgs.WrapMsg("账号或密码错误")
	}

	accessToken, refreshToken, err := token.GenerateTokens(ctx, redis.GetRedis(), usr.ID, usr.Email)
	if err != nil {
		return nil, errs.ErrInternalServer.WrapMsg("生成 Token 失败: " + err.Error())
	}

	return &protouser.AuthResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toProtoUser(&usr),
	}, nil
}

func (u *UserServer) Refresh(ctx context.Context, req *protouser.RefreshReq) (*protouser.AuthResp, error) {
	if req.RefreshToken == "" {
		return nil, errs.ErrToken.WrapMsg("refresh_token is empty")
	}
	newAccess, newRefresh, uid, _, err := token.RefreshTokens(ctx, redis.GetRedis(), req.RefreshToken)
	if err != nil {
		return nil, errs.ErrToken.WrapMsg(err.Error())
	}
	var usr model.User
	if err := u.db.WithContext(ctx).Where("id = ?", uid).First(&usr).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("user not found")
	}
	return &protouser.AuthResp{
		AccessToken:  newAccess,
		RefreshToken: newRefresh,
		User:         toProtoUser(&usr),
	}, nil
}

func (u *UserServer) Logout(ctx context.Context, req *protouser.LogoutReq) (*protosdkws.Empty, error) {
	token.RevokeRefreshToken(ctx, redis.GetRedis(), req.RefreshToken)
	return &protosdkws.Empty{}, nil
}

func (u *UserServer) GetMe(ctx context.Context, _ *protosdkws.Empty) (*protouser.UserInfo, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var usr model.User
	if err := u.db.WithContext(ctx).Where("id = ?", uid).First(&usr).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("user not found")
	}
	return toProtoUser(&usr), nil
}

func (u *UserServer) UpdateProfile(ctx context.Context, req *protouser.UpdateProfileReq) (*protouser.UserInfo, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	var usr model.User
	if err := u.db.WithContext(ctx).Where("id = ?", uid).First(&usr).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("user not found")
	}

	if strings.TrimSpace(req.Nickname) != "" {
		usr.Nickname = strings.TrimSpace(req.Nickname)
	}
	usr.Bio = strings.TrimSpace(req.Bio)
	if req.ResearchFields != nil {
		b, _ := json.Marshal(req.ResearchFields)
		usr.ResearchFields = string(b)
	}

	if err := u.db.WithContext(ctx).Save(&usr).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg("更新个人资料失败: " + err.Error())
	}
	return toProtoUser(&usr), nil
}

func (u *UserServer) ChangePassword(ctx context.Context, req *protouser.ChangePasswordReq) (*protosdkws.Empty, error) {
	uid, _, err := token.ExtractUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.NewPassword) < 6 {
		return nil, errs.ErrArgs.WrapMsg("新密码长度至少为 6 位")
	}
	var usr model.User
	if err := u.db.WithContext(ctx).Where("id = ?", uid).First(&usr).Error; err != nil {
		return nil, errs.ErrRecordNotFound.WrapMsg("user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(usr.PasswordHash), []byte(req.OldPassword)); err != nil {
		return nil, errs.ErrArgs.WrapMsg("原密码不正确")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, errs.ErrInternalServer.WrapMsg(err.Error())
	}
	usr.PasswordHash = string(hash)
	if err := u.db.WithContext(ctx).Save(&usr).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg(err.Error())
	}
	return &protosdkws.Empty{}, nil
}

func (u *UserServer) GetUsersInfo(ctx context.Context, req *protouser.GetUsersInfoReq) (*protouser.GetUsersInfoResp, error) {
	res := make(map[int64]string)
	if len(req.UserIds) == 0 {
		return &protouser.GetUsersInfoResp{Nicknames: res}, nil
	}
	var users []model.User
	if err := u.db.WithContext(ctx).Where("id IN ?", req.UserIds).Find(&users).Error; err != nil {
		return nil, errs.ErrInternalServer.WrapMsg(err.Error())
	}
	for _, usr := range users {
		res[usr.ID] = usr.Nickname
	}
	return &protouser.GetUsersInfoResp{Nicknames: res}, nil
}

