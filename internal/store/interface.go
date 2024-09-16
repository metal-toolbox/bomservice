package store

import (
	"context"

	"github.com/metal-toolbox/bomservice/internal/app"
	fleetdbapi "github.com/metal-toolbox/fleetdb/pkg/api/v1"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NOTE: when updating this interface, run make gen-store-mock to make sure the mocks are updated.
type Repository interface {
	// GetBomInfoByAOCMacAddr gets bom object by AOCMacAddr.
	GetBomInfoByAOCMacAddr(ctx context.Context, macAddr string) (*fleetdbapi.Bom, *fleetdbapi.ServerResponse, error)

	// GetBomInfoByBMCMacAddr gets bom object by BMCMacAddr.
	GetBomInfoByBMCMacAddr(ctx context.Context, macAddr string) (*fleetdbapi.Bom, *fleetdbapi.ServerResponse, error)

	// BillOfMaterialsBatchUpload creates a bom on a server.
	BillOfMaterialsBatchUpload(ctx context.Context, boms []fleetdbapi.Bom) (*fleetdbapi.ServerResponse, error)
}

var (
	ErrRepository = errors.New("storage repository error")
)

func NewStore(ctx context.Context, config *app.Configuration, logger *logrus.Logger) (Repository, error) {
	return newServerserviceStore(ctx, &config.ServerserviceOptions, logger)
}
