package v1

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/sysadminsmedia/homebox/backend/internal/core/permissions"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
	"github.com/sysadminsmedia/homebox/backend/pkgs/labelmaker"
)

func generateOrPrint(ctrl *V1Controller, w http.ResponseWriter, r *http.Request, title string, description string, url string) error {
	lm := ctrl.runtime().LabelMaker
	params := labelmaker.NewGenerateParams(int(lm.Width), int(lm.Height), int(lm.Margin), int(lm.Padding), lm.FontSize, title, description, url, lm.DynamicLength, lm.AdditionalInformation)

	// The labelmaker package takes the whole config but only reads the
	// LabelMaker section; hand it a copy carrying the runtime values.
	cfg := *ctrl.config
	cfg.LabelMaker = lm

	print := queryBool(r.URL.Query().Get("print"))

	if print {
		err := labelmaker.PrintLabel(&cfg, &params)
		if err != nil {
			return err
		}

		_, err = w.Write([]byte("Printed!"))
		return err
	} else {
		return labelmaker.GenerateLabel(w, &params, &cfg)
	}
}

// HandleGetLocationLabel godoc
//
//	@Summary	Get Location label
//	@Tags		Locations
//	@Produce	json
//	@Param		id		path		string	true	"Location ID"
//	@Param		print	query		bool	false	"Print this label, defaults to false"
//	@Success	200		{string}	string	"image/png"
//	@Router		/v1/labelmaker/location/{id} [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleGetLocationLabel() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		ID, err := adapters.RouteUUID(r, "id")
		if err != nil {
			return err
		}

		auth := services.NewContext(r.Context())
		location, err := ctrl.repo.Entities.GetContainerByGroup(auth, auth.GID, ID)
		if err != nil {
			return err
		}

		hbURL := ctrl.hbURL(r)
		return generateOrPrint(ctrl, w, r, location.Name, ctrl.appName(r.Context())+" Location", fmt.Sprintf("%s/location/%s", hbURL, location.ID))
	}
}

// HandleGetItemLabel godoc
//
//	@Summary	Get Item label
//	@Tags		Items
//	@Produce	json
//	@Param		id		path		string	true	"Item ID"
//	@Param		print	query		bool	false	"Print this label, defaults to false"
//	@Success	200		{string}	string	"image/png"
//	@Router		/v1/labelmaker/item/{id} [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleGetItemLabel() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		ID, err := adapters.RouteUUID(r, "id")
		if err != nil {
			return err
		}

		auth := services.NewContext(r.Context())
		// Shared by /labelmaker/entity and /labelmaker/item: enforce the
		// precise section for the entity's kind.
		if err := ctrl.checkEntityPermission(r, ID, permissions.ActionView); err != nil {
			return err
		}
		item, err := ctrl.repo.Entities.GetOneByGroup(auth, auth.GID, ID)
		if err != nil {
			return err
		}

		description := ""

		if item.Parent != nil {
			description += fmt.Sprintf("\nLocation: %s", item.Parent.Name)
		}

		hbURL := ctrl.hbURL(r)
		return generateOrPrint(ctrl, w, r, item.Name, description, fmt.Sprintf("%s/item/%s", hbURL, item.ID))
	}
}

// HandleGetAssetLabel godoc
//
//	@Summary	Get Asset label
//	@Tags		Items
//	@Produce	json
//	@Param		id		path		string	true	"Asset ID"
//	@Param		print	query		bool	false	"Print this label, defaults to false"
//	@Success	200		{string}	string	"image/png"
//	@Router		/v1/labelmaker/asset/{id} [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleGetAssetLabel() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		assetIDParam := chi.URLParam(r, "id")
		assetIDParam = strings.ReplaceAll(assetIDParam, "-", "")
		assetID, err := strconv.ParseInt(assetIDParam, 10, 64)
		if err != nil {
			return err
		}

		auth := services.NewContext(r.Context())
		item, err := ctrl.repo.Entities.QueryByAssetID(auth, auth.GID, repo.AssetID(assetID), 0, 1)
		if err != nil {
			return err
		}

		if len(item.Items) == 0 {
			return validate.NewRequestError(fmt.Errorf("failed to find asset id"), http.StatusNotFound)
		}

		description := item.Items[0].Name

		if item.Items[0].Parent != nil {
			description += fmt.Sprintf("\nLocation: %s", item.Items[0].Parent.Name)
		}

		hbURL := ctrl.hbURL(r)
		return generateOrPrint(ctrl, w, r, item.Items[0].AssetID.String(), description, fmt.Sprintf("%s/a/%s", hbURL, item.Items[0].AssetID.String()))
	}
}
