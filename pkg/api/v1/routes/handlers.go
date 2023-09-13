package routes

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/metal-toolbox/hollow-bomservice/internal/parse"
	"github.com/pkg/errors"
	sservice "go.hollow.sh/serverservice/pkg/api/v1"
)

func (r *Routes) billOfMaterialsBatchUpload(c *gin.Context) (int, *sservice.ServerResponse) {
	if c.Request.ContentLength == -1 {
		return http.StatusBadRequest, &sservice.ServerResponse{Error: "reject the request since the file size unknown"}
	}

	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return http.StatusBadRequest, &sservice.ServerResponse{Error: err.Error()}
	}
	boms, err := parse.ParseXlsxFile(data)
	if err != nil {
		return http.StatusBadRequest, nil
	}

	resp, err := r.repository.BillOfMaterialsBatchUpload(c.Request.Context(), boms)
	if err != nil {
		return http.StatusBadRequest, nil
	}

	return http.StatusOK, resp
}

func (r *Routes) getBomInfoByAOCMacAddr(c *gin.Context) (int, *sservice.ServerResponse) {
	_, resp, err := r.repository.GetBomInfoByAOCMacAddr(c.Request.Context(), c.Param("aoc_mac_address"))
	if err != nil {
		return http.StatusBadRequest, &sservice.ServerResponse{Error: (errors.Wrap(ErrServerServiceQuery, err.Error())).Error()}
	}
	return http.StatusOK, resp
}

func (r *Routes) getBomInfoByBMCMacAddr(c *gin.Context) (int, *sservice.ServerResponse) {
	_, resp, err := r.repository.GetBomInfoByBMCMacAddr(c.Request.Context(), c.Param("bmc_mac_address"))
	if err != nil {
		return http.StatusInternalServerError, &sservice.ServerResponse{Error: (errors.Wrap(ErrServerServiceQuery, err.Error())).Error()}
	}
	return http.StatusOK, resp
}
