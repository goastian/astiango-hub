package mongo

import "context"

type ClientOption func(options *ClientOptions)

type ClientOptions struct {
	Context                 context.Context
	Uri                     string
	Host                    string
	Port                    int
	Db                      string
	Hosts                   []string
	Username                string
	Password                string
	AuthSource              string
	AuthMechanism           string
	AuthMechanismProperties map[string]string
}
