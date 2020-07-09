/*
	Copyright NetFoundry, Inc.

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

	https://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package routes

import (
	"encoding/json"
	"fmt"
	"github.com/go-openapi/runtime/middleware"
	"github.com/michaelquigley/pfxlog"
	"github.com/openziti/edge/build"
	"github.com/openziti/edge/controller/env"
	"github.com/openziti/edge/controller/internal/permissions"
	"github.com/openziti/edge/controller/response"
	"github.com/openziti/edge/rest_model"
	"github.com/openziti/edge/rest_server"
	"github.com/openziti/edge/rest_server/operations/informational"
	"github.com/openziti/fabric/controller/models"
	"time"
)

const EntityNameSpecs = "specs"

var SpecLinkFactory = NewSpecLinkFactory()

type SpecLinkFactoryImpl struct {
	BasicLinkFactory
}

func NewSpecLinkFactory() *SpecLinkFactoryImpl {
	return &SpecLinkFactoryImpl{
		BasicLinkFactory{entityName: EntityNameSpecs},
	}
}

func (factory *SpecLinkFactoryImpl) Links(entity models.Entity) rest_model.Links {
	links := factory.BasicLinkFactory.Links(entity)
	links["spec"] = factory.NewNestedLink(entity, "spec")

	return links
}

type Spec struct {
	models.BaseEntity
	name string
	body map[string]interface{}
}

var swaggerSpec *Spec
var specs []*Spec

func init() {
	info := build.GetBuildInfo()
	date := time.Now()
	if info.GetBuildDate() != "unknown" {
		var err error
		date, err = time.Parse("", info.GetBuildDate())
		if err != nil {
			pfxlog.Logger().WithError(err).Warn("could not parse build info date for swagger spec")
		}
	}
	swaggerSpec = &Spec{
		BaseEntity: models.BaseEntity{
			Id:        "swagger",
			CreatedAt: date,
			UpdatedAt: date,
			Tags:      map[string]interface{}{},
		},
		name: "swagger",
	}

	err := json.Unmarshal(rest_server.SwaggerJSON, &swaggerSpec.body)
	if err != nil {
		pfxlog.Logger().WithError(err).Panic("could not parse rest server JSON spec")
	}

	specs = append(specs, swaggerSpec)

	r := NewSpecRouter()
	env.AddRouter(r)
}

type SpecRouter struct {
	BasePath string
}

func NewSpecRouter() *SpecRouter {
	return &SpecRouter{
		BasePath: "/specs",
	}
}

func (r *SpecRouter) Register(ae *env.AppEnv) {
	ae.Api.InformationalListSpecsHandler = informational.ListSpecsHandlerFunc(func(params informational.ListSpecsParams) middleware.Responder {
		return ae.IsAllowed(r.List, params.HTTPRequest, "", "", permissions.Always())
	})

	ae.Api.InformationalDetailSpecHandler = informational.DetailSpecHandlerFunc(func(params informational.DetailSpecParams) middleware.Responder {
		return ae.IsAllowed(r.Detail, params.HTTPRequest, params.ID, "", permissions.Always())
	})

	ae.Api.InformationalDetailSpecBodyHandler = informational.DetailSpecBodyHandlerFunc(func(params informational.DetailSpecBodyParams) middleware.Responder {
		return ae.IsAllowed(r.DetailBody, params.HTTPRequest, params.ID, "", permissions.Always())
	})
}

func (r *SpecRouter) List(_ *env.AppEnv, rc *response.RequestContext) {
	data := rest_model.SpecList{
		mapSpecToRestModel(swaggerSpec),
	}

	rc.RespondWithOk(data, &rest_model.Meta{})
}

func (r *SpecRouter) Detail(_ *env.AppEnv, rc *response.RequestContext) {
	id, err := rc.GetEntityId()
	if err != nil {
		rc.RespondWithError(fmt.Errorf("entity id not set"))
	}
	for _, spec := range specs {
		if spec.GetId() == id {
			rc.RespondWithOk(mapSpecToRestModel(spec), &rest_model.Meta{})
			return
		}
	}

	rc.RespondWithNotFound()
}

func (r *SpecRouter) DetailBody(_ *env.AppEnv, rc *response.RequestContext) {
	id, err := rc.GetEntityId()
	if err != nil {
		rc.RespondWithError(fmt.Errorf("entity id not set"))
	}
	for _, spec := range specs {
		if spec.GetId() == id {
			_ = rc.GetProducer().Produce(rc.ResponseWriter, spec.body)
			return
		}
	}

	rc.RespondWithNotFound()
}

func mapSpecToRestModel(spec *Spec) *rest_model.SpecDetail {
	return &rest_model.SpecDetail{
		BaseEntity: BaseEntityToRestModel(spec, SpecLinkFactory),
		Name:       &spec.name,
	}
}
