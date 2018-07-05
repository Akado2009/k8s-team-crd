package handler

import (
	log "github.com/Sirupsen/logrus"
	team_v1 "github.com/cmoulliard/k8s-team-crd/pkg/apis/team/v1"
)

// Handler interface contains the methods that are required
type HandlerInterface interface {
	Init() error
	ObjectCreated(obj interface{})
	ObjectDeleted(obj interface{})
	ObjectUpdated(objOld, objNew interface{})
}

type TeamHandler struct{}

// Init handles any handler initialization
func (t *TeamHandler) Init() error {
	log.Info("TeamHandler.Init")
	return nil
}

// ObjectCreated is called when an object is created
func (t *TeamHandler) ObjectCreated(obj interface{}) {
	log.Info("TeamHandler.ObjectCreated")

	team := obj.(*team_v1.Team)
	log.Infof("    ResourceVersion: %s", team.ObjectMeta.ResourceVersion)
	log.Infof("    Team name: %s", team.Spec.Name)
	log.Infof("    Team description: %s", team.Spec.Description)
	log.Infof("    Team size: %s", team.Spec.Size)
}

// ObjectDeleted is called when an object is deleted
func (t *TeamHandler) ObjectDeleted(obj interface{}) {
	log.Info("TeamHandler.ObjectDeleted")
}

// ObjectUpdated is called when an object is updated
func (t *TeamHandler) ObjectUpdated(objOld, objNew interface{}) {
	log.Info("TeamHandler.ObjectUpdated")
}