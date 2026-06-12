package v1

import (
	"net/http"

	"github.com/hay-kot/httpkit/errchain"
	"github.com/hay-kot/httpkit/server"
)

// HandleLabelMakerSettings godoc
//
//	@Summary		Label layout settings
//	@Description	Returns the instance-wide label sheet layout used by the browser label renderer
//	@Tags			Labelmaker
//	@Produce		json
//	@Success		200	{object}	config.LabelMakerConf
//	@Router			/v1/labelmaker/settings [GET]
//	@Security		Bearer
func (ctrl *V1Controller) HandleLabelMakerSettings() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		return server.JSON(w, http.StatusOK, ctrl.runtime().LabelMaker)
	}
}
