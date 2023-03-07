// This file is part of MinIO Operator
// Copyright (c) 2020 MinIO, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package statefulsets

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	miniov2 "github.com/minio/operator/pkg/apis/minio.min.io/v2"
)

func TestGetContainerArgs(t *testing.T) {
	type args struct {
		t             *miniov2.Tenant
		hostsTemplate string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Empty Tenant",
			args: args{
				t:             &miniov2.Tenant{},
				hostsTemplate: "",
			},
			want: nil,
		},
		{
			name: "One Pool Tenant",
			args: args{
				t: &miniov2.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "minio",
					},
					Spec: miniov2.TenantSpec{
						Pools: []miniov2.Pool{
							{
								Servers:             4,
								VolumesPerServer:    4,
								VolumeClaimTemplate: nil,
							},
						},
					},
				},
				hostsTemplate: "",
			},
			want: []string{
				"https://minio-ss-0-{0...3}.minio-hl..svc.cluster.local/export{0...3}",
			},
		},
		{
			name: "One Pool Tenant With Named Pool",
			args: args{
				t: &miniov2.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Name: "minio",
					},
					Spec: miniov2.TenantSpec{
						Pools: []miniov2.Pool{
							{
								Name:                "pool-0",
								Servers:             4,
								VolumesPerServer:    4,
								VolumeClaimTemplate: nil,
							},
						},
					},
				},
				hostsTemplate: "",
			},
			want: []string{
				"https://minio-pool-0-{0...3}.minio-hl..svc.cluster.local/export{0...3}",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ensure defaults
			tt.args.t.EnsureDefaults()

			if got := GetContainerArgs(tt.args.t, tt.args.hostsTemplate); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetContainerArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMinioTenantSecurityContextForNewPool(t *testing.T) {
	type expectedSecurityCtxs struct {
		podSecurityCtx       *corev1.PodSecurityContext
		containerSecurityCtx *corev1.SecurityContext
	}

	pool := createPool()
	tests := []struct {
		name string
		args *NewPoolArgs
		want *expectedSecurityCtxs
	}{
		{
			name: "Should be provided default security context for k8s platform",
			args: &NewPoolArgs{
				Pool:         &miniov2.Pool{},
				Tenant:       createTenant(pool),
				PoolStatus:   &miniov2.PoolStatus{LegacySecurityContext: false},
				IsOpenshift4: false,
			},
			want: &expectedSecurityCtxs{
				podSecurityCtx:       createDefaultPodSecurityContext(),
				containerSecurityCtx: createDefaultContainerSecurityContext(),
			},
		},
		{
			name: "Should be provided default security context for openshift platform",
			args: &NewPoolArgs{
				Pool:         &miniov2.Pool{},
				Tenant:       createTenant(pool),
				PoolStatus:   &miniov2.PoolStatus{LegacySecurityContext: false},
				IsOpenshift4: true,
			},
			want: &expectedSecurityCtxs{
				podSecurityCtx:       &corev1.PodSecurityContext{},
				containerSecurityCtx: &corev1.SecurityContext{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.args)
			podSecurityCtx := pool.Spec.Template.Spec.SecurityContext
			if !reflect.DeepEqual(podSecurityCtx, tt.want.podSecurityCtx) {
				t.Errorf("NewPool() podSecurityContext %v, want: %v", podSecurityCtx, tt.want)
			}
			for _, container := range pool.Spec.Template.Spec.Containers {
				containerSecurityCtx := container.SecurityContext
				if !reflect.DeepEqual(containerSecurityCtx, tt.want.containerSecurityCtx) && container.Name == "minio" {
					t.Errorf("NewPool() SecurityContext %v for container %s, want %v", containerSecurityCtx, container.Name, tt.want.containerSecurityCtx)
				}
			}
		})
	}
}

func createTenant(pool miniov2.Pool) *miniov2.Tenant {
	return &miniov2.Tenant{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "storage", Namespace: "minio-tenant"},
		Scheduler:  miniov2.TenantScheduler{},
		Spec: miniov2.TenantSpec{
			ExposeServices:      &miniov2.ExposeServices{MinIO: true, Console: false},
			Features:            &miniov2.Features{BucketDNS: false, Domains: &miniov2.TenantDomains{}},
			CertConfig:          &miniov2.CertificateConfig{},
			PodManagementPolicy: v1.ParallelPodManagement,
			Configuration:       &corev1.LocalObjectReference{Name: "minio-storage-configuration"},
			Env:                 []corev1.EnvVar{},
			Pools: []miniov2.Pool{
				pool,
			},
		},
		Status: miniov2.TenantStatus{},
	}
}

func createPool() miniov2.Pool {
	return miniov2.Pool{
		Servers:          1,
		Name:             "pool-0",
		VolumesPerServer: 2,
	}
}

func createDefaultPodSecurityContext() *corev1.PodSecurityContext {
	runAsNonRoot := true
	var runAsUser int64 = 1000
	var runAsGroup int64 = 1000
	var fsGroup int64 = 1000
	fsGroupChangePolicy := corev1.FSGroupChangeOnRootMismatch

	return &corev1.PodSecurityContext{
		RunAsUser:           &runAsUser,
		RunAsGroup:          &runAsGroup,
		RunAsNonRoot:        &runAsNonRoot,
		FSGroup:             &fsGroup,
		FSGroupChangePolicy: &fsGroupChangePolicy,
	}
}

func createDefaultContainerSecurityContext() *corev1.SecurityContext {
	runAsNonRoot := true
	var runAsUser int64 = 1000
	var runAsGroup int64 = 1000

	return &corev1.SecurityContext{
		RunAsUser:    &runAsUser,
		RunAsGroup:   &runAsGroup,
		RunAsNonRoot: &runAsNonRoot,
	}
}
