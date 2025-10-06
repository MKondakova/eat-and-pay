package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"eats-backend/internal/models"
	"eats-backend/tests/integration"
)

type ProductSuite struct {
	integration.SuiteWithRequests
}

func TestProductEndpoints(t *testing.T) {
	suite.Run(t, &ProductSuite{})
}

func (s *ProductSuite) TestDeleteProduct() {
	id := "c5268b2c-0501-4784-b4de-72760d819baf"

	res, code := s.DeleteAPI("http://localhost:8080", "/products/"+id, nil, nil)
	s.Equal(http.StatusNoContent, code)

	s.Empty(res)

	res, code = s.DeleteAPI("http://localhost:8080", "/products/"+id, nil, nil)
	s.Equal(http.StatusNotFound, code)

	s.Equal("GetProductByID: not found: product c5268b2c-0501-4784-b4de-72760d819baf not found", string(res))
}

func (s *ProductSuite) TestAddProduct() {

	res, code := s.PostAPI("http://localhost:8080", "/products/generate", nil, nil, nil)
	s.Equal(http.StatusOK, code)

	fmt.Println(string(res))

	var newProduct models.ProductPreview
	err := json.Unmarshal(res, &newProduct)
	s.NoError(err)

	s.NotEmpty(newProduct.ID)

	newID := newProduct.ID

	res, code = s.GetAPI("http://localhost:8080", "/products/"+newID, nil, nil)
	s.Equal(http.StatusOK, code)

	fmt.Println(string(res))

	s.NotEmpty(res)
	var newProductFullInfo models.ProductPageInfo
	err = json.Unmarshal(res, &newProductFullInfo)
	s.NoError(err)
}
