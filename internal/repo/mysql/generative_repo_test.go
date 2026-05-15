package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/modeling/generative"
)

func TestGenerativeModelPackageRepository_Save(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()
	repo := NewGenerativeModelPackageRepository(db)
	pkg := &generative.GenerativeModelPackage{PackageID: uuid.New(), UserID: uuid.New(), Domain: generative.DomainPhysics, Question: "平抛", Status: "success", Confidence: 0.9, CreatedAt: time.Now(), LearningModel: generative.LearningModelSpec{LearningGoal: "理解平抛"}}

	mock.ExpectExec("INSERT INTO `generative_model_packages`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Save(context.Background(), pkg))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerativeModelPackageRepository_GetByID(t *testing.T) {
	db, mock, cleanup := newMockGorm(t)
	defer cleanup()
	repo := NewGenerativeModelPackageRepository(db)
	pkgID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	packageJSON := `{"package_id":"` + pkgID.String() + `","domain":"physics","question":"平抛","learning_model":{"domain":"physics","grade_band":"high_school","topic":"projectile","learning_goal":"理解平抛"},"confidence":0.9,"created_at":"` + now.Format(time.RFC3339Nano) + `"}`

	rows := sqlmock.NewRows([]string{"id", "user_id", "session_id", "domain", "question", "package_json", "model_name", "status", "confidence", "validation_json", "fallback_reason", "created_at"}).
		AddRow(pkgID, userID, nil, "physics", "平抛", packageJSON, "m1", "success", 0.9, `{}`, "", now)
	mock.ExpectQuery("SELECT \\* FROM `generative_model_packages` WHERE id = \\? ORDER BY `generative_model_packages`.`id` LIMIT \\?").
		WithArgs(pkgID, 1).
		WillReturnRows(rows)

	pkg, err := repo.GetByID(context.Background(), pkgID)
	require.NoError(t, err)
	require.Equal(t, pkgID, pkg.PackageID)
	require.Equal(t, "平抛", pkg.Question)
	require.NoError(t, mock.ExpectationsWereMet())
}
