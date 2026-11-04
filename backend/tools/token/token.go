package token

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/GouTouZhao/PaperCompare/tools/errs"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/metadata"
)

const (
	AccessTokenDuration   = 24 * time.Hour
	RefreshTokenDuration  = 7 * 24 * time.Hour
	RedisRefreshKeyPrefix = "papercompare:refresh_token:"

	MetaKeyUserID = "x-user-id"
	MetaKeyEmail  = "x-user-email"
)

type CustomClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "c635cac023beed54bbd19f9e989dbc42b80474de167e944ef419c5468f80b2fa"
	}
	return []byte(secret)
}

func GenerateTokens(ctx context.Context, rdb *redis.Client, userID int64, email string) (accessToken string, refreshToken string, err error) {
	claims := CustomClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "papercompare-system",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err = t.SignedString(getJWTSecret())
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	refreshToken = hex.EncodeToString(b)

	if rdb != nil {
		redisKey := RedisRefreshKeyPrefix + refreshToken
		redisVal := fmt.Sprintf("%d:%s", userID, email)
		if err := rdb.Set(ctx, redisKey, redisVal, RefreshTokenDuration).Err(); err != nil {
			return "", "", fmt.Errorf("failed to save refresh token to redis: %w", err)
		}
	}

	return accessToken, refreshToken, nil
}

func ParseAccessToken(tokenString string) (*CustomClaims, error) {
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		return nil, errors.New("empty token")
	}
	t, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := t.Claims.(*CustomClaims); ok && t.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func RefreshTokens(ctx context.Context, rdb *redis.Client, oldRefreshToken string) (newAccessToken string, newRefreshToken string, userID int64, email string, err error) {
	if rdb == nil {
		return "", "", 0, "", errors.New("redis not initialized")
	}
	redisKey := RedisRefreshKeyPrefix + oldRefreshToken
	val, err := rdb.Get(ctx, redisKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", "", 0, "", errors.New("refresh token expired or invalid")
		}
		return "", "", 0, "", err
	}
	rdb.Del(ctx, redisKey)

	parts := strings.SplitN(val, ":", 2)
	if len(parts) != 2 {
		return "", "", 0, "", errors.New("invalid data format in redis")
	}
	if _, err := fmt.Sscanf(parts[0], "%d", &userID); err != nil {
		return "", "", 0, "", errors.New("failed to parse user id")
	}
	email = parts[1]

	newAccessToken, newRefreshToken, err = GenerateTokens(ctx, rdb, userID, email)
	return newAccessToken, newRefreshToken, userID, email, err
}

func RevokeRefreshToken(ctx context.Context, rdb *redis.Client, refreshToken string) {
	if rdb != nil && refreshToken != "" {
		rdb.Del(ctx, RedisRefreshKeyPrefix+refreshToken)
	}
}

func InjectUserMetadata(ctx context.Context, userID int64, email string) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		MetaKeyUserID, strconv.FormatInt(userID, 10),
		MetaKeyEmail, email,
	)
}

func ExtractUserFromContext(ctx context.Context) (int64, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, "", errs.ErrToken.WrapMsg("missing metadata")
	}
	uids := md.Get(MetaKeyUserID)
	if len(uids) == 0 || uids[0] == "" {
		return 0, "", errs.ErrToken.WrapMsg("unauthorized: user_id missing")
	}
	uid, err := strconv.ParseInt(uids[0], 10, 64)
	if err != nil || uid <= 0 {
		return 0, "", errs.ErrToken.WrapMsg("unauthorized: invalid user_id")
	}
	email := ""
	if emails := md.Get(MetaKeyEmail); len(emails) > 0 {
		email = emails[0]
	}
	return uid, email, nil
}

func OptionalUserFromContext(ctx context.Context) (int64, string) {
	uid, email, err := ExtractUserFromContext(ctx)
	if err != nil {
		return 0, ""
	}
	return uid, email
}

