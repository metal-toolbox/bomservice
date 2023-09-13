package routes

import "github.com/pkg/errors"

var (
	ErrStore              = errors.New("store error")
	ErrRoutes             = errors.New("error in routes")
	ErrServerserviceQuery = errors.New("Serverservice query error")
)
