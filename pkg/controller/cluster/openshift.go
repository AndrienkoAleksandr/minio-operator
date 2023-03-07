package cluster

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

const (
	OPENSHIFT_ROUTE_API_GROUP   = "route.openshift.io"
	OPENSHIFT4_CONFIG_API_GROUP = "config.openshift.io"
)

func IsOpenshift4(dc *discovery.DiscoveryClient) (bool, error) {
	apiGroups, err := dc.ServerGroups()
	if err != nil {
		return false, err
	}
	isOpenshift4 := isAPIPresent(apiGroups, OPENSHIFT_ROUTE_API_GROUP) && isAPIPresent(apiGroups, OPENSHIFT4_CONFIG_API_GROUP)
	return isOpenshift4, nil
}

func isAPIPresent(apiGroups *v1.APIGroupList, apiName string) bool {
	for _, apiGroup := range apiGroups.Groups {
		if apiGroup.Name == apiName {
			return true
		}
	}
	return false
}
