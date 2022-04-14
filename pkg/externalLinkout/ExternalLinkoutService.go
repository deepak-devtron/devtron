/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package externalLinkout

import (
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/devtron-labs/devtron/pkg/user"
	"go.uber.org/zap"
	"time"
)

type ExternalLinkoutService interface {
	Create(requests []*ExternalLinkoutRequest) ([]*ExternalLinkoutRequest, error)
	GetAllActiveTools() ([]ExternalLinksMonitoringToolsRequest, error)
	FetchAllActiveLinks(clusterIds int) ([]*ExternalLinkoutRequest, error)
	Update(request *ExternalLinkoutRequest) (*ExternalLinkoutRequest, error)
	DeleteLink(id int) error
}
type ExternalLinkoutServiceImpl struct {
	logger                          *zap.SugaredLogger
	externalLinkoutToolsRepository  ExternalLinkoutToolsRepository
	externalLinksClustersRepository ExternalLinksClustersRepository
	externalLinksRepository         ExternalLinksRepository
	userAuthService                 user.UserAuthService
}
type ExternalLinksMonitoringToolsRequest struct {
	Id   int    `json:"Id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}
type ExternalLinkoutRequest struct {
	Id               int    `json:"id"`
	Name             string `json:"name"`
	Url              string `json:"url"`
	Active           bool   `json:"active"`
	MonitoringToolId int    `json:"monitoringToolId"`
	ClusterIds       []int  `json:"clusterIds"`
	UserId           int    `json:"-"`
}

func NewExternalLinkoutServiceImpl(logger *zap.SugaredLogger, externalLinkoutToolsRepository ExternalLinkoutToolsRepository,
	externalLinksClustersRepository ExternalLinksClustersRepository, externalLinksRepository ExternalLinksRepository, userAuthService user.UserAuthService) *ExternalLinkoutServiceImpl {
	return &ExternalLinkoutServiceImpl{
		logger:                          logger,
		externalLinkoutToolsRepository:  externalLinkoutToolsRepository,
		externalLinksClustersRepository: externalLinksClustersRepository,
		externalLinksRepository:         externalLinksRepository,
		userAuthService:                 userAuthService,
	}
}

func (impl ExternalLinkoutServiceImpl) Create(requests []*ExternalLinkoutRequest) ([]*ExternalLinkoutRequest, error) {
	impl.logger.Debugw("external linkout create request", "req", requests)
	for _, request := range requests {
		t := &ExternalLinks{
			Name:                          request.Name,
			Active:                        true,
			ExternalLinksMonitoringToolId: request.MonitoringToolId,
			Url:                           request.Url,
			AuditLog:                      sql.AuditLog{CreatedOn: time.Now(), UpdatedOn: time.Now()},
		}
		err := impl.externalLinksRepository.Save(t)
		if err != nil {
			impl.logger.Errorw("error in saving link", "data", t, "err", err)
			err = &util.ApiError{
				InternalMessage: "external link failed to create in db",
				UserMessage:     "external link failed to create in db",
			}
			return nil, err
		}

		for _, clusterId := range request.ClusterIds {
			externalLinksMapping := &ExternalLinksClusters{
				ExternalLinksId: t.Id,
				ClusterId:       clusterId,
				AuditLog:        sql.AuditLog{CreatedOn: time.Now(), UpdatedOn: time.Now()},
			}
			err := impl.externalLinksClustersRepository.Save(externalLinksMapping)
			if err != nil {
				impl.logger.Errorw("error in saving cluster id's", "data", t, "err", err)
				err = &util.ApiError{
					InternalMessage: "cluster id failed to create in db",
					UserMessage:     "cluster id failed to create in db",
				}
				return nil, err
			}
		}
	}
	return requests, nil
}

func (impl ExternalLinkoutServiceImpl) GetAllActiveTools() ([]ExternalLinksMonitoringToolsRequest, error) {
	impl.logger.Debug("fetch all links from db")
	tools, err := impl.externalLinkoutToolsRepository.FindAllActive()
	if err != nil {
		impl.logger.Errorw("error in fetch all tools", "err", err)
		return nil, err
	}
	var toolRequests []ExternalLinksMonitoringToolsRequest
	for _, tool := range tools {
		providerRes := ExternalLinksMonitoringToolsRequest{
			Id:   tool.Id,
			Name: tool.Name,
			Icon: tool.Icon,
		}
		toolRequests = append(toolRequests, providerRes)
	}
	return toolRequests, err
}

func (impl ExternalLinkoutServiceImpl) FetchAllActiveLinks(clusterId int) ([]*ExternalLinkoutRequest, error) {
	impl.logger.Debug("fetch all links from db")
	var links []ExternalLinksClusters
	var err error

	var mappedExternalLinksIds []int
	externalLinksMap := make(map[int]int)
	allActiveExternalLinks, err := impl.externalLinksClustersRepository.FindAllActive()
	for _, link := range allActiveExternalLinks {
		externalLinksMap[link.ExternalLinksId] = link.ExternalLinksId
	}
	for _, externalLinksId := range externalLinksMap {
		mappedExternalLinksIds = append(mappedExternalLinksIds, externalLinksId)
	}

	if clusterId == 0 {
		links = allActiveExternalLinks
	} else {
		links, err = impl.externalLinksClustersRepository.FindAllActiveByClusterId(clusterId)
	}
	if err != nil {
		impl.logger.Errorw("error in fetch all links", "err", err)
		return nil, err
	}
	var externalLinkResponse []*ExternalLinkoutRequest
	response := make(map[int]*ExternalLinkoutRequest)
	for _, link := range links {
		if clusterId > 0 {
			providerRes := &ExternalLinkoutRequest{
				Name:             link.ExternalLinks.Name,
				Url:              link.ExternalLinks.Url,
				Active:           link.ExternalLinks.Active,
				MonitoringToolId: link.ExternalLinks.ExternalLinksMonitoringToolId,
			}
			providerRes.ClusterIds = append(providerRes.ClusterIds, link.ClusterId)
			externalLinkResponse = append(externalLinkResponse, providerRes)
		} else {
			if _, ok := response[link.ExternalLinksId]; !ok {
				response[link.ExternalLinksId] = &ExternalLinkoutRequest{
					Name:             link.ExternalLinks.Name,
					Url:              link.ExternalLinks.Url,
					Active:           link.ExternalLinks.Active,
					MonitoringToolId: link.ExternalLinks.ExternalLinksMonitoringToolId,
				}
			}
			response[link.ExternalLinksId].ClusterIds = append(response[link.ExternalLinksId].ClusterIds, link.ClusterId)
		}
	}
	for _, v := range response {
		externalLinkResponse = append(externalLinkResponse, v)
	}

	//now add all the links which are not mapped to any clusters
	additionalExternalLinks, err := impl.externalLinksRepository.FindAllNonMapped(mappedExternalLinksIds)
	if err != nil {
		impl.logger.Errorw("error in fetch all links", "err", err)
		return nil, err
	}
	for _, link := range additionalExternalLinks {
		providerRes := &ExternalLinkoutRequest{
			Name:             link.Name,
			Url:              link.Url,
			Active:           link.Active,
			MonitoringToolId: link.ExternalLinksMonitoringToolId,
			ClusterIds:       []int{},
		}
		externalLinkResponse = append(externalLinkResponse, providerRes)
	}

	return externalLinkResponse, err
}
func (impl ExternalLinkoutServiceImpl) Update(request *ExternalLinkoutRequest) (*ExternalLinkoutRequest, error) {
	impl.logger.Debugw("link update request", "req", request)
	externalLinks, err0 := impl.externalLinksRepository.FindOne(request.Id)
	if err0 != nil {
		impl.logger.Errorw("No matching entry found for update.", "id", request.Id)
		return nil, err0
	}
	externalLinks.Name = request.Name
	externalLinks.Url = request.Url
	externalLinks.Active = request.Active
	externalLinks.ExternalLinksMonitoringToolId = request.MonitoringToolId
	externalLinks.UpdatedBy = int32(request.UserId)
	externalLinks.UpdatedOn = time.Now()
	err := impl.externalLinksRepository.Update(&externalLinks)
	if err != nil {
		impl.logger.Errorw("error in updating link", "data", externalLinks, "err", err)
		return nil, err
	}

	allExternalLinksMapping, err := impl.externalLinksClustersRepository.FindAllClusters(request.Id)
	if err != nil {
		impl.logger.Errorw("error in fetching link", "data", externalLinks, "err", err)
		return nil, err
	}
	for _, clusterId := range allExternalLinksMapping {
		externalLinksMap := &ExternalLinksClusters{
			ExternalLinksId: request.Id,
			ClusterId:       clusterId,
			Active:          false,
			AuditLog:        sql.AuditLog{UpdatedOn: time.Now()},
		}
		err := impl.externalLinksClustersRepository.Update(externalLinksMap)
		if err != nil {
			impl.logger.Errorw("error in updating clusters to false", "data", externalLinksMap, "err", err)
		}
	}
	for _, requestedClusterId := range request.ClusterIds {
		flag := 0
		for _, existingClusterId := range allExternalLinksMapping {
			if requestedClusterId == existingClusterId {
				flag = 1
				break
			}
		}

		x := &ExternalLinksClusters{
			ExternalLinksId: request.Id,
			ClusterId:       requestedClusterId,
			Active:          true,
			AuditLog:        sql.AuditLog{UpdatedOn: time.Now()},
		}
		if flag == 1 {
			err = impl.externalLinksClustersRepository.Update(x)
		} else {
			err = impl.externalLinksClustersRepository.Save(x)
		}
		if err != nil {
			impl.logger.Errorw("error in saving cluster id's", "data", x, "err", err)
			err = &util.ApiError{
				InternalMessage: "cluster id failed to create in db",
				UserMessage:     "cluster id failed to create in db",
			}
			return nil, err
		}
	}
	return request, nil
}
func (impl ExternalLinkoutServiceImpl) DeleteLink(id int) error {
	impl.logger.Debugw("link delete request", "req", id)
	externalLinksMapping, _ := impl.externalLinksClustersRepository.FindAllClusters(id)
	for _, v := range externalLinksMapping {
		x := &ExternalLinksClusters{
			ExternalLinksId: id,
			ClusterId:       v,
			Active:          false,
			AuditLog:        sql.AuditLog{UpdatedOn: time.Now()},
		}
		err := impl.externalLinksClustersRepository.Update(x)
		if err != nil {
			impl.logger.Errorw("error in deleting clusters to false", "data", x, "err", err)
		}
	}
	link := &ExternalLinks{
		Id:       id,
		Active:   false,
		AuditLog: sql.AuditLog{UpdatedOn: time.Now()},
	}
	err := impl.externalLinksRepository.Update(link)
	if err != nil {
		impl.logger.Errorw("error in deleting link", "data", link, "err", err)
		return err
	}
	return nil
}
