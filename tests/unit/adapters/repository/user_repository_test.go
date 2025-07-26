package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/growthfolio/go-priceguard-api/internal/adapters/repository"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type UserRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo repositories.UserRepository
	ctx  context.Context
}

func (suite *UserRepositoryTestSuite) SetupSuite() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	suite.Require().NoError(err)
	err = db.AutoMigrate(&entities.User{})
	suite.Require().NoError(err)
	suite.db = db
	suite.repo = repository.NewUserRepository(db)
	suite.ctx = context.Background()
}

func (suite *UserRepositoryTestSuite) TearDownTest() {
	suite.db.Exec("DELETE FROM users")
}

func (suite *UserRepositoryTestSuite) TestCreateAndGet() {
	user := &entities.User{GoogleID: "gid", Email: "test@example.com", Name: "Test"}
	err := suite.repo.Create(suite.ctx, user)
	suite.Require().NoError(err)
	assert.NotEqual(suite.T(), uuid.Nil, user.ID)

	fetched, err := suite.repo.GetByID(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Email, fetched.Email)

	byEmail, err := suite.repo.GetByEmail(suite.ctx, user.Email)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, byEmail.ID)

	byGoogle, err := suite.repo.GetByGoogleID(suite.ctx, user.GoogleID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.ID, byGoogle.ID)
}

func (suite *UserRepositoryTestSuite) TestUpdateAndDelete() {
	user := &entities.User{GoogleID: "gid2", Email: "foo@example.com", Name: "Foo"}
	err := suite.repo.Create(suite.ctx, user)
	suite.Require().NoError(err)

	user.Name = "Bar"
	err = suite.repo.Update(suite.ctx, user)
	suite.Require().NoError(err)

	fetched, err := suite.repo.GetByID(suite.ctx, user.ID)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "Bar", fetched.Name)

	err = suite.repo.Delete(suite.ctx, user.ID)
	suite.Require().NoError(err)

	none, err := suite.repo.GetByID(suite.ctx, user.ID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), none)
}

func TestUserRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}
