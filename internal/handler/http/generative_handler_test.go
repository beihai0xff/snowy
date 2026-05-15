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
	"github.com/beihai0xff/snowy/internal/pkg/common"
)

type mockGenerativeService struct {
	compileFn func(context.Context, *generative.CompileRequest) (*generative.GenerativeModelPackage, error)
	getFn     func(context.Context, string) (*generative.GenerativeModelPackage, error)
	listFn    func(context.Context, uuid.UUID, int, int) ([]*generative.GenerativeModelPackage, int64, error)
}

func (m *mockGenerativeService) Compile(ctx context.Context, req *generative.CompileRequest) (*generative.GenerativeModelPackage, error) {
	if m.compileFn == nil {
		return nil, errors.New("compile not implemented")
	}

	return m.compileFn(ctx, req)
}

func (m *mockGenerativeService) GetPackage(ctx context.Context, id string) (*generative.GenerativeModelPackage, error) {
	if m.getFn == nil {
		return nil, errors.New("get package not implemented")
	}

	return m.getFn(ctx, id)
}

func (m *mockGenerativeService) ListPackages(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*generative.GenerativeModelPackage, int64, error) {
	if m.listFn == nil {
		return nil, 0, nil
	}

	return m.listFn(ctx, userID, offset, limit)
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

func TestGenerativeHandler_ListPackages(t *testing.T) {
	userID := uuid.New()
	pkgID := uuid.New()
	handler := NewGenerativeHandler(&mockGenerativeService{listFn: func(_ context.Context, id uuid.UUID, offset, limit int) ([]*generative.GenerativeModelPackage, int64, error) {
		assert.Equal(t, userID, id)
		assert.Equal(t, 0, offset)
		assert.Equal(t, 20, limit)

		return []*generative.GenerativeModelPackage{
			{PackageID: pkgID, UserID: userID, Domain: generative.DomainPhysics, Question: "平抛运动", CreatedAt: time.Now()},
		}, 1, nil
	}})
	r := gin.New()
	r.GET("/packages", func(c *gin.Context) {
		ctx := common.WithUserID(c.Request.Context(), userID.String())
		c.Request = c.Request.WithContext(ctx)
		handler.ListPackages(c)
	})

	w := getRequest(r, "/packages")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), pkgID.String())
	assert.Contains(t, w.Body.String(), `"total":1`)
}
