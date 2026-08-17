package models

import (
	"time"
)

type Database struct {
	any               `collection:"databases"`
	BaseModel         `bson:",inline"`
	Name              string    `json:"name" bson:"name" description:"Name"`
	Description       string    `json:"description" bson:"description" description:"Description"`
	DataSource        string    `json:"data_source" bson:"data_source" description:"Data source"`
	Host              string    `json:"host" bson:"host" description:"Host"`
	Port              int       `json:"port" bson:"port" description:"Port"`
	URI               string    `json:"uri,omitempty" bson:"uri,omitempty" description:"URI"`
	Database          string    `json:"database,omitempty" bson:"database,omitempty" description:"Database"`
	Username          string    `json:"username,omitempty" bson:"username,omitempty" description:"Username"`
	Password          string    `json:"password,omitempty" bson:"-" binding:"-"`
	EncryptedPassword string    `json:"-,omitempty" bson:"encrypted_password,omitempty" description:"Encrypted password"`
	Status            string    `json:"status" bson:"status" description:"Status"`
	Error             string    `json:"error" bson:"error" description:"Error"`
	Active            bool      `json:"active" bson:"active" description:"Active"`
	ActiveAt          time.Time `json:"active_ts" bson:"active_ts" description:"Active at"`
	IsDefault         bool      `json:"is_default" bson:"-" binding:"-"`
	UseORM            bool      `json:"use_orm" bson:"use_orm" description:"Use ORM instead of legacy database service (only for MySQL, PostgreSQL, SQL Server)"`

	MongoParams *struct {
		AuthSource    string `json:"auth_source,omitempty" bson:"auth_source,omitempty" description:"Auth source"`
		AuthMechanism string `json:"auth_mechanism,omitempty" bson:"auth_mechanism,omitempty" description:"Auth mechanism"`
	} `json:"mongo_params,omitempty" bson:"mongo_params,omitempty" description:"Mongo params"`
	PostgresParams *struct {
		SSLMode string `json:"ssl_mode,omitempty" bson:"ssl_mode,omitempty" description:"SSL mode"`
	} `json:"postgres_params,omitempty" bson:"postgres_params,omitempty" description:"Postgres params"`
	SnowflakeParams *struct {
		Account   string `json:"account,omitempty" bson:"account,omitempty" description:"Account"`
		Schema    string `json:"schema,omitempty" bson:"schema,omitempty" description:"Schema"`
		Warehouse string `json:"warehouse,omitempty" bson:"warehouse,omitempty" description:"Warehouse"`
		Role      string `json:"role,omitempty" bson:"role,omitempty" description:"Role"`
	} `json:"snowflake_params,omitempty" bson:"snowflake_params,omitempty" description:"Snowflake params"`
	CassandraParams *struct {
		Keyspace string `json:"keyspace,omitempty" bson:"keyspace,omitempty" description:"Keyspace"`
	} `json:"cassandra_params,omitempty" bson:"cassandra_params,omitempty" description:"Cassandra params"`
	HiveParams *struct {
		Auth string `json:"auth,omitempty" bson:"auth,omitempty" description:"Auth"`
	} `json:"hive_params,omitempty" bson:"hive_params,omitempty" description:"Hive params"`
	RedisParams *struct {
		DB int `json:"db,omitempty" bson:"db,omitempty" description:"DB"`
	} `json:"redis_params,omitempty" bson:"redis_params,omitempty" description:"Redis params"`
}

type DatabaseDTO struct {
	Database `json:",inline" bson:",inline"`

	IsDefault bool `json:"is_default" bson:"-"`
}
