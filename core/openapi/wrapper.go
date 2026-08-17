package openapi

import (
	"fmt"
	"github.com/crawlab-team/fizz/openapi"
	"sync"

	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/crawlab-team/fizz"
	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
)

// FizzWrapper wraps an existing Gin Engine to add OpenAPI functionality
type FizzWrapper struct {
	fizz   *fizz.Fizz
	gin    *gin.Engine
	logger interfaces.Logger
}

// newFizzWrapper creates a new wrapper around an existing Gin Engine
// This approach ensures we don't break existing functionality
func newFizzWrapper(engine *gin.Engine) *FizzWrapper {
	// Create a new Fizz instance using the existing Gin engine
	f := fizz.NewFromEngine(engine)

	// Add security middleware
	f.Generator().SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
		"apiToken": {
			SecurityScheme: &openapi.SecurityScheme{
				Type: "apiKey",
				Name: "API Token",
				In:   "header",
			},
		},
	})

	// Sort tags
	f.Generator().SetSortTags(false)

	return &FizzWrapper{
		fizz:   f,
		gin:    engine,
		logger: utils.NewLogger("FizzWrapper"),
	}
}

// GetFizz returns the underlying Fizz instance
func (w *FizzWrapper) GetFizz() *fizz.Fizz {
	return w.fizz
}

// GetGin returns the underlying Gin engine
func (w *FizzWrapper) GetGin() *gin.Engine {
	return w.gin
}

// Response represents an OpenAPI response
type Response struct {
	Description string
	Model       interface{}
}

// RegisterRoute registers a route with OpenAPI documentation
func (w *FizzWrapper) RegisterRoute(method, path string, group *fizz.RouterGroup, handler interface{}, id, summary, description string, responses map[int]Response) {
	// Build operation options for OpenAPI documentation
	opts := w.buildOperationOptions(id, summary, description, responses)

	// Register the route with OpenAPI documentation
	switch method {
	case "GET":
		group.GET(path, opts, tonic.Handler(handler, 200))
	case "POST":
		group.POST(path, opts, tonic.Handler(handler, 200))
	case "PUT":
		group.PUT(path, opts, tonic.Handler(handler, 200))
	case "DELETE":
		group.DELETE(path, opts, tonic.Handler(handler, 200))
	case "PATCH":
		group.PATCH(path, opts, tonic.Handler(handler, 200))
	case "HEAD":
		group.HEAD(path, opts, tonic.Handler(handler, 200))
	case "OPTIONS":
		group.OPTIONS(path, opts, tonic.Handler(handler, 200))
	}
}

// BuildModelResponse builds a standard response model with a specific data type
func (w *FizzWrapper) BuildModelResponse() map[int]Response {
	return map[int]Response{
		400: {
			Description: "Bad Request",
		},
		401: {
			Description: "Unauthorized",
		},
		500: {
			Description: "Internal Server Error",
		},
	}
}

// buildOperationOptions builds the options for a Fizz operation
func (w *FizzWrapper) buildOperationOptions(id, summary, description string, responses map[int]Response) []fizz.OperationOption {
	var opts []fizz.OperationOption

	// Add ID
	if id != "" {
		opts = append(opts, fizz.ID(id))
	}

	// Add summary
	if summary != "" {
		opts = append(opts, fizz.Summary(summary))
	}

	// Add description
	if description != "" {
		opts = append(opts, fizz.Description(description))
	}

	// Add responses
	if responses != nil {
		for status, response := range responses {
			if response.Model != nil {
				opts = append(opts, fizz.Response(fmt.Sprintf("%d", status), response.Description, response.Model, nil, nil))
			} else {
				opts = append(opts, fizz.Response(fmt.Sprintf("%d", status), response.Description, nil, nil, nil))
			}
		}
	}

	return opts
}

var wrapper *FizzWrapper
var wrapperOnce sync.Once

func GetFizzWrapper(app *gin.Engine) *FizzWrapper {
	wrapperOnce.Do(func() {
		wrapper = newFizzWrapper(app)
	})
	return wrapper
}
