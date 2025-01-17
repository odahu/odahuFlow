//
//    Copyright 2019 EPAM Systems
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package packaging

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/odahu/odahu-flow/packages/operator/api/v1alpha1"
	"github.com/odahu/odahu-flow/packages/operator/pkg/apis/packaging"
	"github.com/odahu/odahu-flow/packages/operator/pkg/apiserver/routes"
	"github.com/odahu/odahu-flow/packages/operator/pkg/errors"
	kube_client "github.com/odahu/odahu-flow/packages/operator/pkg/kubeclient/packagingclient"
	mp_repository "github.com/odahu/odahu-flow/packages/operator/pkg/repository/packaging"
	mp_service "github.com/odahu/odahu-flow/packages/operator/pkg/service/packaging"
	"github.com/odahu/odahu-flow/packages/operator/pkg/utils/filter"
	httputil "github.com/odahu/odahu-flow/packages/operator/pkg/utils/httputil"
	"net/http"
	"reflect"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strconv"
)

var logMP = logf.Log.WithName("MP-controller")

const (
	GetModelPackagingURL        = "/model/packaging/:id"
	GetModelPackagingLogsURL    = "/model/packaging/:id/log"
	GetAllModelPackagingURL     = "/model/packaging"
	CreateModelPackagingURL     = "/model/packaging"
	UpdateModelPackagingURL     = "/model/packaging"
	SaveModelPackagingResultURL = "/model/packaging/:id/result"
	DeleteModelPackagingURL     = "/model/packaging/:id"
	IDMpURLParam                = "id"
	FollowURLParam              = "follow"
)

var (
	fieldsCache = map[string]int{}
)

func init() {
	elem := reflect.TypeOf(&mp_repository.MPFilter{}).Elem()
	for i := 0; i < elem.NumField(); i++ {
		tagName := elem.Field(i).Tag.Get(mp_repository.TagKey)

		fieldsCache[tagName] = i
	}
}

type ModelPackagingController struct {
	kubeClient  kube_client.Client
	packService mp_service.Service
	validator   *MpValidator
}

// @Summary Get a Model Packaging
// @Description Get a Model Packaging by id
// @Tags Packaging
// @Name id
// @Accept  json
// @Produce  json
// @Param id path string true "Model Packaging id"
// @Success 200 {object} packaging.ModelPackaging
// @Failure 404 {object} httputil.HTTPResult
// @Failure 400 {object} httputil.HTTPResult
// @Router /api/v1/model/packaging/{id} [get]
func (mpc *ModelPackagingController) getMP(c *gin.Context) {
	mpID := c.Param(IDMpURLParam)

	mp, err := mpc.packService.GetModelPackaging(c.Request.Context(), mpID)
	if err != nil {
		logMP.Error(err, fmt.Sprintf("Retrieving %s model packaging", mpID))
		c.AbortWithStatusJSON(errors.CalculateHTTPStatusCode(err), httputil.HTTPResult{Message: err.Error()})

		return
	}

	c.JSON(http.StatusOK, mp)
}

// @Summary Get list of Model Packagings
// @Description Get list of Model Packagings
// @Tags Packaging
// @Accept  json
// @Produce  json
// @Param size path int false "Number of entities in a response"
// @Param page path int false "Number of a page"
// @Success 200 {array} packaging.ModelPackaging
// @Failure 400 {object} httputil.HTTPResult
// @Router /api/v1/model/packaging [get]
func (mpc *ModelPackagingController) getAllMPs(c *gin.Context) {
	f := &mp_repository.MPFilter{}
	size, page, err := routes.URLParamsToFilter(c, f, fieldsCache)
	if err != nil {
		logMP.Error(err, "Malformed url parameters of model packaging request")
		c.AbortWithStatusJSON(http.StatusBadRequest, httputil.HTTPResult{Message: err.Error()})

		return
	}

	mpList, err := mpc.packService.GetModelPackagingList(
		c.Request.Context(),
		filter.ListFilter(f),
		filter.Size(size),
		filter.Page(page),
	)
	if err != nil {
		logMP.Error(err, "Retrieving list of model packagings")
		c.AbortWithStatusJSON(errors.CalculateHTTPStatusCode(err), httputil.HTTPResult{Message: err.Error()})

		return
	}

	c.JSON(http.StatusOK, mpList)
}

// @Summary Create a Model Packaging
// @Description Create a Model Packaging. Results is created Model Packaging.
// @Param MP body packaging.ModelPackaging true "Create a Model Packaging"
// @Tags Packaging
// @Accept  json
// @Produce  json
// @Success 201 {object} packaging.ModelPackaging
// @Failure 400 {object} httputil.HTTPResult
// @Router /api/v1/model/packaging [post]
func (mpc *ModelPackagingController) createMP(c *gin.Context) {
	var mp packaging.ModelPackaging

	if err := c.ShouldBindJSON(&mp); err != nil {
		logMP.Error(err, "JSON binding of the model packaging is failed")
		c.AbortWithStatusJSON(http.StatusBadRequest, httputil.HTTPResult{Message: err.Error()})

		return
	}

	if err := mpc.validator.ValidateAndSetDefaults(&mp); err != nil {
		logMP.Error(err, fmt.Sprintf("Validation of the model packaging is failed: %v", mp))
		c.AbortWithStatusJSON(http.StatusBadRequest, httputil.HTTPResult{Message: err.Error()})

		return
	}

	if err := mpc.packService.CreateModelPackaging(c.Request.Context(), &mp); err != nil {
		logMP.Error(err, fmt.Sprintf("Creation of the model packaging: %+v", mp))
		c.AbortWithStatusJSON(errors.CalculateHTTPStatusCode(err), httputil.HTTPResult{Message: err.Error()})

		return
	}

	c.JSON(http.StatusCreated, mp)
}

// @Summary Update a Model Packaging
// @Description Update a Model Packaging. Results is updated Model Packaging.
// @Param MP body packaging.ModelPackaging true "Update a Model Packaging"
// @Tags Packaging
// @Accept  json
// @Produce  json
// @Success 200 {object} packaging.ModelPackaging
// @Failure 404 {object} httputil.HTTPResult
// @Failure 400 {object} httputil.HTTPResult
// @Router /api/v1/model/packaging [put]
func (mpc *ModelPackagingController) updateMP(c *gin.Context) {
	var mp packaging.ModelPackaging

	if err := c.ShouldBindJSON(&mp); err != nil {
		logMP.Error(err, "JSON binding of the model packaging is failed")
		c.AbortWithStatusJSON(http.StatusBadRequest, httputil.HTTPResult{Message: err.Error()})

		return
	}

	if err := mpc.validator.ValidateAndSetDefaults(&mp); err != nil {
		logMP.Error(err, fmt.Sprintf("Validation of the model packaging is failed: %v", mp))
		c.AbortWithStatusJSON(http.StatusBadRequest, httputil.HTTPResult{Message: err.Error()})

		return
	}

	if err := mpc.packService.UpdateModelPackaging(c.Request.Context(), &mp); err != nil {
		logMP.Error(err, fmt.Sprintf("Update of the model packaging: %+v", mp))
		c.AbortWithStatusJSON(errors.CalculateHTTPStatusCode(err), httputil.HTTPResult{Message: err.Error()})

		return
	}

	c.JSON(http.StatusOK, mp)
}

// @Summary Save a Model Packaging result
// @Description Save a Model Packaging by id
// @Tags Packaging
// @Name id
// @Accept  json
// @Produce  json
// @Param MP body v1alpha1.ModelPackagingResult true "Model Packaging result"
// @Param id path string true "Model Packaging id"
// @Success 200 {array} v1alpha1.ModelPackagingResult
// @Failure 404 {object} httputil.HTTPResult
// @Failure 400 {object} httputil.HTTPResult
// @Router /api/v1/model/packaging/{id}/result [put]
func (mpc *ModelPackagingController) saveMPResults(c *gin.Context) {
	mpID := c.Param(IDMpURLParam)
	mpResult := make([]v1alpha1.ModelPackagingResult, 0)

	if err := c.ShouldBindJSON(&mpResult); err != nil {
		logMP.Error(err, "JSON binding of the model packaging result is failed")
		c.AbortWithStatusJSON(http.StatusBadRequest, httputil.HTTPResult{Message: err.Error()})

		return
	}

	if err := mpc.kubeClient.SaveModelPackagingResult(mpID, mpResult); err != nil {
		logMP.Error(err, fmt.Sprintf("Save the result of the model packaging: %+v", mpResult))
		c.AbortWithStatusJSON(errors.CalculateHTTPStatusCode(err), httputil.HTTPResult{Message: err.Error()})

		return
	}

	c.JSON(http.StatusOK, mpResult)
}

// @Summary Delete a Model Packaging
// @Description Delete a Model Packaging by id
// @Tags Packaging
// @Name id
// @Accept  json
// @Produce  json
// @Param id path string true "Model Packaging id"
// @Success 200 {object} httputil.HTTPResult
// @Failure 404 {object} httputil.HTTPResult
// @Failure 400 {object} httputil.HTTPResult
// @Router /api/v1/model/packaging/{id} [delete]
func (mpc *ModelPackagingController) deleteMP(c *gin.Context) {
	mpID := c.Param(IDMpURLParam)

	if err := mpc.packService.DeleteModelPackaging(c.Request.Context(), mpID); err != nil {
		logMP.Error(err, fmt.Sprintf("Deletion of %s model packaging is failed", mpID))
		c.AbortWithStatusJSON(errors.CalculateHTTPStatusCode(err), httputil.HTTPResult{Message: err.Error()})

		return
	}

	c.JSON(http.StatusOK, httputil.HTTPResult{Message: fmt.Sprintf("Model packaging %s was deleted", mpID)})
}

// @Summary Stream logs from model packaging pod
// @Description Stream logs from model packaging pod
// @Tags Packaging
// @Name id
// @Produce  plain
// @Accept  plain
// @Param follow query bool false "follow logs"
// @Param id path string true "Model Packaging id"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Router /api/v1/model/packaging/{id}/log [get]
func (mpc *ModelPackagingController) getModelPackagingLog(c *gin.Context) {
	mpID := c.Param(IDMpURLParam)
	follow := false
	var err error

	urlParameters := c.Request.URL.Query()
	followParam := urlParameters.Get(FollowURLParam)

	if len(followParam) != 0 {
		follow, err = strconv.ParseBool(followParam)
		if err != nil {
			errMessage := fmt.Sprintf("Convert %s to bool", followParam)
			logMP.Error(err, errMessage)
			c.AbortWithStatusJSON(http.StatusBadRequest, httputil.HTTPResult{Message: errMessage})

			return
		}
	}

	if err := mpc.kubeClient.GetModelPackagingLogs(mpID, c.Writer, follow); err != nil {
		logMP.Error(err, fmt.Sprintf("Getting %s model packaging logs is failed", mpID))
		c.AbortWithStatusJSON(errors.CalculateHTTPStatusCode(err), httputil.HTTPResult{Message: err.Error()})

		return
	}
}
