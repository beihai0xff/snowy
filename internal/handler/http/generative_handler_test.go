package http

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/beihai0xff/snowy/internal/modeling/generative"
)

type mockGenerativeService struct {
	compileFn func(context.Context, *generative.CompileRequest) (*generative.GenerativeModelPackage, error)
	getFn     func(context.Context, string) (*generative.GenerativeModelPackage, error)
}

func (m *mockGenerativeService) Compile(ctx context.Context, req *generative.CompileRequest) (*generative.GenerativeModelPackage, error) {
	return m.compileFn(ctx, req)
}

func (m *mockGenerativeService) GetPackage(ctx context.Context, id string) (*generative.GenerativeModelPackage, error) {
	return m.getFn(ctx, id)
}

func TestGenerativeHandler_CompileSuccess(t *testing.T) {
	pkgID := uuid.New()
	handler := NewGenerativeHandler(&mockGenerativeService{compileFn: func(_ context.Context, req *generative.CompileRequest) (*generative.GenerativeModelPackage, error) {
		assert.Equal(t, "平抛运动", req.Message)
		assert.Equal(t, generative.DomainPhysics, req.Domain)
		return &generative.GenerativeModelPackage{PackageID: pkgID, Domain: generative.DomainPhysics, Question: req.Message, CreatedAt: time.Now(), Confidence: 0.9}, nil
	}})
	r := gin.New()
	r.POST("/compile", handler.Compile)

	w := postJSON(r, "/compile", map[string]string{"message": "平抛运动", "domain": "physics"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), pkgID.String())
}

func TestGenerativeHandler_CompileBadJSON(t *testing.T) {
	handler := NewGenerativeHandler(&mockGenerativeService{})
	r := gin.New()
	r.POST("/compile", handler.Compile)

	w := postJSON(r, "/compile", map[string]string{"domain": "physics"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGenerativeHandler_GetPackage(t *testing.T) {
	pkgID := uuid.New()
	handler := NewGenerativeHandler(&mockGenerativeService{getFn: func(_ context.Context, id string) (*generative.GenerativeModelPackage, error) {
		assert.Equal(t, pkgID.String(), id)
		return &generative.GenerativeModelPackage{PackageID: pkgID, Domain: generative.DomainBiology, Question: "光合作用", CreatedAt: time.Now()}, nil
	}})
	r := gin.New()
	r.GET("/packages/:id", handler.GetPackage)

	w := getRequest(r, "/packages/"+pkgID.String())

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), pkgID.String())
}

func TestGenerativeHandler_GetPackageNotFound(t *testing.T) {
	handler := NewGenerativeHandler(&mockGenerativeService{getFn: func(context.Context, string) (*generative.GenerativeModelPackage, error) {
		return nil, errors.New("not found")
	}})
	r := gin.New()
	r.GET("/packages/:id", handler.GetPackage)

	w := getRequest(r, "/packages/"+uuid.NewString())

	assert.Equal(t, http.StatusNotFound, w.Code)
}
