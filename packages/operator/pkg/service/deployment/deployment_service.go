/*
 * Copyright 2020 EPAM Systems
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
 */

package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/odahu/odahu-flow/packages/operator/api/v1alpha1"
	"github.com/odahu/odahu-flow/packages/operator/pkg/apis/deployment"
	"github.com/odahu/odahu-flow/packages/operator/pkg/apis/event"
	odahu_errors "github.com/odahu/odahu-flow/packages/operator/pkg/errors"
	repo "github.com/odahu/odahu-flow/packages/operator/pkg/repository/deployment"
	mrRepo "github.com/odahu/odahu-flow/packages/operator/pkg/repository/route"
	db_utils "github.com/odahu/odahu-flow/packages/operator/pkg/utils/db"
	"github.com/odahu/odahu-flow/packages/operator/pkg/utils/filter"
	hashutil "github.com/odahu/odahu-flow/packages/operator/pkg/utils/hash"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

var (
	log           = logf.Log.WithName("model-deployment--service")
	defaultWeight = int32(100)
)

type Service interface {
	GetModelDeployment(ctx context.Context, id string) (*deployment.ModelDeployment, error)
	GetModelDeploymentList(ctx context.Context, options ...filter.ListOption) ([]deployment.ModelDeployment, error)
	DeleteModelDeployment(ctx context.Context, id string) error
	SetDeletionMark(ctx context.Context, id string, value bool) error
	UpdateModelDeployment(ctx context.Context, mt *deployment.ModelDeployment) error
	// Try to update status. If spec in storage differs from spec snapshot then update does not happen
	UpdateModelDeploymentStatus(
		ctx context.Context, id string, status v1alpha1.ModelDeploymentStatus, spec v1alpha1.ModelDeploymentSpec) error
	CreateModelDeployment(ctx context.Context, mt *deployment.ModelDeployment) error
	GetDefaultModelRoute(ctx context.Context, mdID string) (*deployment.ModelRoute, error)
}

type EventPublisher interface {
	PublishEvent(ctx context.Context, tx *sql.Tx, event event.Event) (err error)
}

type serviceImpl struct {
	// Repository that has "database/sql" underlying storage
	repo     repo.Repository
	mrRepo   mrRepo.Repository
	eventPub EventPublisher
}

func (s serviceImpl) GetModelDeployment(ctx context.Context, id string) (*deployment.ModelDeployment, error) {
	return s.repo.GetModelDeployment(ctx, nil, id)
}

func (s serviceImpl) GetModelDeploymentList(
	ctx context.Context, options ...filter.ListOption,
) ([]deployment.ModelDeployment, error) {
	return s.repo.GetModelDeploymentList(ctx, nil, options...)
}

func GetDefaultModelRoute(ctx context.Context, tx *sql.Tx, mdID string, repository mrRepo.Repository) (string, error) {

	mrs, err := repository.GetModelRouteList(ctx, tx, filter.ListFilter(&mrRepo.Filter{
		Default: []bool{true},
		MdID:    []string{mdID},
	}))
	if err != nil {
		return "", err
	}
	if len(mrs) > 1 {
		return "", fmt.Errorf("model deployment must have one default route, but have %v", len(mrs))
	}
	if len(mrs) == 0 {
		return "", nil
	}
	return mrs[0].ID, nil
}

func (s serviceImpl) GetDefaultModelRoute(ctx context.Context, mdID string) (*deployment.ModelRoute, error) {
	tx, err := s.mrRepo.BeginTransaction(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		db_utils.FinishTx(tx, err, log)
	}()
	id, err := GetDefaultModelRoute(ctx, tx, mdID, s.mrRepo)
	if err != nil {
		return nil, err
	}
	return s.mrRepo.GetModelRoute(ctx, tx, id)
}

func (s serviceImpl) DeleteModelDeployment(ctx context.Context, id string) (err error) {
	tx, err := s.repo.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer func() { db_utils.FinishTx(tx, err, log) }()

	mrDefaultID, err := GetDefaultModelRoute(ctx, tx, id, s.mrRepo)
	if err != nil {
		return err
	}

	if mrDefaultID != "" {
		if err = s.mrRepo.DeleteModelRoute(ctx, tx, mrDefaultID); err != nil {
			return
		}
		e := event.Event{EntityID: mrDefaultID, EventType: event.ModelRouteDeletedEventType,
			EventGroup: event.ModelRouteEventGroup, Payload: nil}
		if err = s.eventPub.PublishEvent(ctx, tx, e); err != nil {
			return
		}
	}

	err = s.repo.DeleteModelDeployment(ctx, tx, id)
	e := event.Event{EntityID: id, EventType: event.ModelDeploymentDeletedEventType,
		EventGroup: event.ModelDeploymentEventGroup, Payload: nil}
	if err = s.eventPub.PublishEvent(ctx, tx, e); err != nil {
		return
	}
	return
}

func (s serviceImpl) SetDeletionMark(ctx context.Context, id string, value bool) error {

	tx, err := s.repo.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer func() { db_utils.FinishTx(tx, err, log) }()

	e := event.Event{
		EntityID:   id,
		EventType:  event.ModelDeploymentDeletionMarkIsSetEventType,
		EventGroup: event.ModelDeploymentEventGroup,
	}
	if err = s.eventPub.PublishEvent(ctx, tx, e); err != nil {
		return err
	}

	return s.repo.SetDeletionMark(ctx, tx, id, value)
}

func (s serviceImpl) UpdateModelDeployment(ctx context.Context, md *deployment.ModelDeployment) (err error) {

	tx, err := s.repo.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer func() { db_utils.FinishTx(tx, err, log) }()

	md.UpdatedAt = time.Now()
	oldMd, err := s.GetModelDeployment(ctx, md.ID)
	if err != nil {
		return err
	}
	md.CreatedAt = oldMd.CreatedAt
	md.DeletionMark = false
	md.Status = v1alpha1.ModelDeploymentStatus{}

	e := event.Event{
		EntityID:   md.ID,
		EventType:  event.ModelDeploymentUpdatedEventType,
		EventGroup: event.ModelDeploymentEventGroup,
		Payload:    *md,
	}
	if err = s.eventPub.PublishEvent(ctx, tx, e); err != nil {
		return err
	}
	return s.repo.UpdateModelDeployment(ctx, tx, md)
}

func (s serviceImpl) UpdateModelDeploymentStatus(
	ctx context.Context, id string, status v1alpha1.ModelDeploymentStatus, spec v1alpha1.ModelDeploymentSpec,
) (err error) {

	var tx *sql.Tx
	tx, err = s.repo.BeginTransaction(ctx)
	if err != nil {
		return err
	}

	defer func() { db_utils.FinishTx(tx, err, log) }()

	oldDeploy, err := s.repo.GetModelDeployment(ctx, tx, id)
	if err != nil {
		return err
	}

	if !hashutil.Equal(oldDeploy.Spec, spec) {
		err = odahu_errors.SpecWasTouched{Entity: id}
		return err
	}

	if hashutil.Equal(oldDeploy.Status, status) {
		// Status is not changed. Skip updating in database and publishing event
		return nil
	}

	if err = s.repo.UpdateModelDeploymentStatus(ctx, tx, id, status); err != nil {
		return err
	}

	updatedDeploy := *oldDeploy
	updatedDeploy.Status = status

	e := event.Event{
		EntityID:   id,
		EventType:  event.ModelDeploymentStatusUpdatedEventType,
		EventGroup: event.ModelDeploymentEventGroup,
		Payload:    updatedDeploy,
	}
	if err = s.eventPub.PublishEvent(ctx, tx, e); err != nil {
		return err
	}

	return err
}

func constructDefaultRoute(mdID string) deployment.ModelRoute {
	return deployment.ModelRoute{
		ID:           mdID + "-" + uuid.New().String()[:5],
		Default:      true,
		DeletionMark: false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Spec: v1alpha1.ModelRouteSpec{
			URLPrefix: fmt.Sprintf("/model/%s", mdID),
			ModelDeploymentTargets: []v1alpha1.ModelDeploymentTarget{
				{
					Name:   mdID,
					Weight: &defaultWeight,
				},
			},
		},
	}
}

func (s serviceImpl) CreateModelDeployment(ctx context.Context, md *deployment.ModelDeployment) (err error) {

	var tx *sql.Tx

	tx, err = s.repo.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer func() { db_utils.FinishTx(tx, err, log) }()

	md.CreatedAt = time.Now()
	md.UpdatedAt = md.CreatedAt
	md.DeletionMark = false
	md.Status = v1alpha1.ModelDeploymentStatus{}
	err = s.repo.SaveModelDeployment(ctx, tx, md)
	if err != nil {
		return
	}

	exists, err := s.mrRepo.DefaultExists(ctx, md.ID, tx)
	if err != nil || exists {
		return err
	}
	// Every Model deployment must have a default HTTP route that sends 100% of traffic to the model
	defRoute := constructDefaultRoute(md.ID)
	err = s.mrRepo.SaveModelRoute(ctx, tx, &defRoute)
	if err != nil {
		return fmt.Errorf("unable to create default ModelRoute: %v", err)
	}

	e := event.Event{
		EntityID:   md.ID,
		EventType:  event.ModelDeploymentCreatedEventType,
		EventGroup: event.ModelDeploymentEventGroup,
		Payload:    *md,
	}
	if err = s.eventPub.PublishEvent(ctx, tx, e); err != nil {
		return err
	}

	e = event.Event{
		EntityID:   defRoute.ID,
		EventType:  event.ModelRouteCreatedEventType,
		EventGroup: event.ModelRouteEventGroup,
		Payload:    defRoute,
	}
	if err = s.eventPub.PublishEvent(ctx, tx, e); err != nil {
		return err
	}

	return err
}

func NewService(repo repo.Repository, mrRepo mrRepo.Repository, eventPub EventPublisher) Service {
	return &serviceImpl{repo: repo, mrRepo: mrRepo, eventPub: eventPub}
}
