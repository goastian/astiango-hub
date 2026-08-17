package user

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/goastian/astiango-hub/core/models/models"
	"github.com/goastian/astiango-hub/core/models/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongo2 "go.mongodb.org/mongo-driver/mongo"
)

const (
	accessTokenType  = "access"
	refreshTokenType = "refresh"
)

type jwtKeysetInput struct {
	ActiveKID string            `json:"active_kid"`
	Keys      map[string]string `json:"keys"`
}

type jwtKeyset struct {
	activeKID  string
	keys       map[string][]byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
	leeway     time.Duration
}

type tokenClaims struct {
	Username  string `json:"username"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// TokenPair keeps the access and refresh credentials distinct so callers can
// store the short-lived access token independently from the refresh token.
type TokenPair struct {
	AccessToken      string `json:"token"`
	RefreshToken     string `json:"refresh_token"`
	AccessExpiresIn  int64  `json:"access_expires_in"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
}

func loadJWTKeyset() (*jwtKeyset, error) {
	raw := viper.GetString("jwt.keyset")
	if secretFile := viper.GetString("jwt.keyset_file"); secretFile != "" {
		data, err := os.ReadFile(secretFile)
		if err != nil {
			return nil, fmt.Errorf("read JWT keyset file: %w", err)
		}
		raw = string(data)
	}
	if raw == "" {
		return nil, errors.New("JWT keyset is required; inject ASTIANGO_JWT_KEYSET or ASTIANGO_JWT_KEYSET_FILE from a secret manager")
	}

	var input jwtKeysetInput
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return nil, fmt.Errorf("parse JWT keyset: %w", err)
	}
	if input.ActiveKID == "" || input.Keys == nil {
		return nil, errors.New("JWT keyset requires active_kid and keys")
	}

	keys := make(map[string][]byte, len(input.Keys))
	for kid, encoded := range input.Keys {
		if kid == "" {
			return nil, errors.New("JWT key id cannot be empty")
		}
		secret, err := base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			secret, err = base64.StdEncoding.DecodeString(encoded)
		}
		if err != nil || len(secret) < 32 {
			return nil, fmt.Errorf("JWT key %q must be base64 and at least 256 bits", kid)
		}
		keys[kid] = secret
	}
	if _, ok := keys[input.ActiveKID]; !ok {
		return nil, errors.New("JWT active_kid does not exist in the keyset")
	}

	accessTTL, refreshTTL, leeway := viper.GetDuration("jwt.access_ttl"), viper.GetDuration("jwt.refresh_ttl"), viper.GetDuration("jwt.leeway")
	if accessTTL <= 0 || refreshTTL <= accessTTL || leeway < 0 {
		return nil, errors.New("JWT TTL configuration is invalid")
	}
	issuer, audience := viper.GetString("jwt.issuer"), viper.GetString("jwt.audience")
	if issuer == "" || audience == "" {
		return nil, errors.New("JWT issuer and audience are required")
	}
	return &jwtKeyset{input.ActiveKID, keys, issuer, audience, accessTTL, refreshTTL, leeway}, nil
}

// ValidateJWTConfiguration is called by the backend before starting listeners.
func ValidateJWTConfiguration() error {
	_, err := loadJWTKeyset()
	return err
}

func (svc *Service) issueToken(user *models.User, tokenType string, ttl time.Duration) (string, *tokenClaims, error) {
	now := time.Now().UTC()
	claims := &tokenClaims{
		Username:  user.Username,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    svc.jwt.issuer,
			Subject:   user.Id.Hex(),
			Audience:  jwt.ClaimStrings{svc.jwt.audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = svc.jwt.activeKID
	signed, err := token.SignedString(svc.jwt.keys[svc.jwt.activeKID])
	if err != nil {
		return "", nil, err
	}
	return signed, claims, nil
}

func (svc *Service) issueTokenPair(user *models.User) (*TokenPair, error) {
	access, accessClaims, err := svc.issueToken(user, accessTokenType, svc.jwt.accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, refreshClaims, err := svc.issueToken(user, refreshTokenType, svc.jwt.refreshTTL)
	if err != nil {
		return nil, err
	}
	session := models.JWTRefreshSession{JTI: refreshClaims.ID, UserID: user.Id, ExpiresAt: refreshClaims.ExpiresAt.Time}
	session.SetCreated(user.Id)
	if _, err := service.NewModelService[models.JWTRefreshSession]().InsertOne(session); err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken: access, RefreshToken: refresh,
		AccessExpiresIn:  int64(accessClaims.ExpiresAt.Time.Sub(time.Now().UTC()).Seconds()),
		RefreshExpiresIn: int64(refreshClaims.ExpiresAt.Time.Sub(time.Now().UTC()).Seconds()),
	}, nil
}

func (svc *Service) parseToken(raw, expectedType string) (*tokenClaims, error) {
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected JWT signing algorithm")
		}
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("JWT kid is required")
		}
		key, ok := svc.jwt.keys[kid]
		if !ok {
			return nil, errors.New("unknown JWT kid")
		}
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(svc.jwt.issuer), jwt.WithAudience(svc.jwt.audience), jwt.WithLeeway(svc.jwt.leeway), jwt.WithIssuedAt(), jwt.WithExpirationRequired())
	if err != nil || token == nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != expectedType || claims.Subject == "" || claims.Username == "" || claims.ID == "" || claims.IssuedAt == nil || claims.NotBefore == nil || claims.ExpiresAt == nil {
		return nil, errors.New("token is missing required claims")
	}
	if _, err := primitive.ObjectIDFromHex(claims.Subject); err != nil {
		return nil, errors.New("invalid token subject")
	}
	return claims, nil
}

func (svc *Service) isAccessRevoked(jti string) (bool, error) {
	_, err := service.NewModelService[models.JWTRevocation]().GetOne(bson.M{"jti": jti}, nil)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo2.ErrNoDocuments) {
		return false, nil
	}
	return false, err
}

func (svc *Service) revokeAccessClaims(claims *tokenClaims) error {
	entry := models.JWTRevocation{JTI: claims.ID, ExpiresAt: claims.ExpiresAt.Time}
	_, err := service.NewModelService[models.JWTRevocation]().InsertOne(entry)
	if mongo2.IsDuplicateKeyError(err) {
		return nil
	}
	return err
}

func (svc *Service) Refresh(refreshToken string) (*TokenPair, error) {
	claims, err := svc.parseToken(refreshToken, refreshTokenType)
	if err != nil {
		return nil, err
	}
	userID, _ := primitive.ObjectIDFromHex(claims.Subject)
	now := time.Now().UTC()
	result, err := service.NewModelService[models.JWTRefreshSession]().GetCol().GetCollection().UpdateOne(context.Background(), bson.M{
		"jti": claims.ID, "user_id": userID, "revoked_at": bson.M{"$exists": false}, "expires_at": bson.M{"$gt": now},
	}, bson.M{"$set": bson.M{"revoked_at": now}})
	if err != nil || result.ModifiedCount != 1 {
		return nil, errors.New("refresh token is revoked or expired")
	}
	user, err := service.NewModelService[models.User]().GetById(userID)
	if err != nil || subtle.ConstantTimeCompare([]byte(user.Username), []byte(claims.Username)) != 1 {
		return nil, errors.New("invalid refresh token user")
	}
	return svc.issueTokenPair(user)
}

func (svc *Service) Logout(accessToken, refreshToken string) error {
	if accessToken != "" {
		if claims, err := svc.parseToken(accessToken, accessTokenType); err == nil {
			if err := svc.revokeAccessClaims(claims); err != nil {
				return err
			}
		}
	}
	if refreshToken != "" {
		if claims, err := svc.parseToken(refreshToken, refreshTokenType); err == nil {
			_, err := service.NewModelService[models.JWTRefreshSession]().GetCol().GetCollection().UpdateOne(context.Background(), bson.M{
				"jti": claims.ID, "revoked_at": bson.M{"$exists": false},
			}, bson.M{"$set": bson.M{"revoked_at": time.Now().UTC()}})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
