package openapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// Validator validates HTTP requests and responses against an OpenAPI specification.
type Validator struct {
	doc    *openapi3.T
	router routers.Router
}

// NewValidator creates a new OpenAPI validator from a specification file.
func NewValidator(specPath string) (*Validator, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec from %s: %w", specPath, err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to create router: %w", err)
	}

	return &Validator{
		doc:    doc,
		router: router,
	}, nil
}

// NewValidatorFromBytes creates a new OpenAPI validator from specification bytes.
func NewValidatorFromBytes(specBytes []byte) (*Validator, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromData(specBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to create router: %w", err)
	}

	return &Validator{
		doc:    doc,
		router: router,
	}, nil
}

// ValidateRequest validates an HTTP request against the OpenAPI specification.
func (v *Validator) ValidateRequest(req *http.Request) error {
	route, pathParams, err := v.router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("failed to find route for %s %s: %w", req.Method, req.URL.Path, err)
	}

	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
		Options: &openapi3filter.Options{
			MultiError: true,
		},
	}

	if err := openapi3filter.ValidateRequest(context.Background(), requestValidationInput); err != nil {
		return fmt.Errorf("request validation failed: %w", err)
	}

	return nil
}

// ValidateResponse validates an HTTP response against the OpenAPI specification.
func (v *Validator) ValidateResponse(req *http.Request, resp *http.Response) error {
	route, pathParams, err := v.router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("failed to find route for %s %s: %w", req.Method, req.URL.Path, err)
	}

	requestValidationInput := &openapi3filter.RequestValidationInput{
		Request:    req,
		PathParams: pathParams,
		Route:      route,
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	responseValidationInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestValidationInput,
		Status:                 resp.StatusCode,
		Header:                 resp.Header,
		Body:                   io.NopCloser(bytes.NewBuffer(bodyBytes)),
		Options: &openapi3filter.Options{
			MultiError:            true,
			IncludeResponseStatus: true,
		},
	}

	if err := openapi3filter.ValidateResponse(context.Background(), responseValidationInput); err != nil {
		return fmt.Errorf("response validation failed: %w", err)
	}

	return nil
}

// ValidateRequestResponse validates both request and response in a single call.
func (v *Validator) ValidateRequestResponse(req *http.Request, resp *http.Response) error {
	if err := v.ValidateRequest(req); err != nil {
		return err
	}
	return v.ValidateResponse(req, resp)
}

// GetOperationID returns the operation ID for a given request.
func (v *Validator) GetOperationID(req *http.Request) (string, error) {
	route, _, err := v.router.FindRoute(req)
	if err != nil {
		return "", fmt.Errorf("failed to find route: %w", err)
	}
	return route.Operation.OperationID, nil
}

// GetDocument returns the parsed OpenAPI document.
func (v *Validator) GetDocument() *openapi3.T {
	return v.doc
}

// GetPaths returns all paths defined in the OpenAPI specification.
func (v *Validator) GetPaths() []string {
	if v.doc.Paths == nil {
		return nil
	}

	paths := make([]string, 0, v.doc.Paths.Len())
	for path := range v.doc.Paths.Map() {
		paths = append(paths, path)
	}
	return paths
}
