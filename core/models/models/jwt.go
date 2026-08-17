package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JWTRefreshSession records a single refresh-token identifier. The JWT itself
// is never stored, and the document expires automatically when it is useless.
type JWTRefreshSession struct {
	any       `collection:"jwt_refresh_sessions"`
	BaseModel `bson:",inline"`
	JTI       string             `json:"jti" bson:"jti"`
	UserID    primitive.ObjectID `json:"user_id" bson:"user_id"`
	ExpiresAt time.Time          `json:"expires_at" bson:"expires_at"`
	RevokedAt *time.Time         `json:"revoked_at,omitempty" bson:"revoked_at,omitempty"`
}

// JWTRevocation invalidates an access-token identifier until its natural
// expiration. It enables immediate logout without retaining the raw token.
type JWTRevocation struct {
	any       `collection:"jwt_revocations"`
	BaseModel `bson:",inline"`
	JTI       string    `json:"jti" bson:"jti"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
}
