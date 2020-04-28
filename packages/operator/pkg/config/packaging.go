//
//    Copyright 2019 EPAM Systems
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package config

import (
	odahuflowv1alpha1 "github.com/odahu/odahu-flow/packages/operator/pkg/apis/odahuflow/v1alpha1"
	"os"
	"time"
)

var (
	defaultPackagingMemoryLimit    = "2028Mi"
	defaultPackagingCPULimit       = "2000m"
	defaultPackagingMemoryRequests = "1024Mi"
	defaultPackagingCPURequests    = "1000m"
)

type ModelPackagingConfig struct {
	// Kubernetes namespace, where model trainings will be deployed
	Namespace                     string `json:"namespace"`
	PackagingIntegrationNamespace string `json:"packagerIntegrationNamespace"`
	// Enable packaging API/operator
	Enabled            bool   `json:"enabled"`
	ServiceAccount     string `json:"serviceAccount"`
	OutputConnectionID string `json:"outputConnectionID"`
	// Kubernetes node selectors for model packaging pods
	NodeSelector map[string]string `json:"nodeSelector"`
	// Kubernetes tolerations for model packaging pods
	Toleration         map[string]string `json:"toleration"`
	ModelPackagerImage string            `json:"modelPackagerImage"`
	// Timeout for full training process
	Timeout time.Duration `json:"timeout"`
	// Default resources for packaging pods
	DefaultResources odahuflowv1alpha1.ResourceRequirements `json:"defaultResources"`
}

func NewDefaultModelPackagingConfig() ModelPackagingConfig {
	return ModelPackagingConfig{
		Namespace:                     "odahu-flow-packaging",
		PackagingIntegrationNamespace: "odahu-flow",
		Enabled:                       true,
		Timeout:                       4 * time.Hour,
		ServiceAccount:                "odahu-flow-model-packager",
		// workaround of the issue https://github.com/spf13/viper/issues/761
		ModelPackagerImage: os.Getenv("PACKAGING_MODEL_PACKAGER_IMAGE"),
		DefaultResources: odahuflowv1alpha1.ResourceRequirements{
			Requests: &odahuflowv1alpha1.ResourceList{
				CPU:    &defaultPackagingCPURequests,
				Memory: &defaultPackagingMemoryRequests,
			},
			Limits: &odahuflowv1alpha1.ResourceList{
				CPU:    &defaultPackagingCPULimit,
				Memory: &defaultPackagingMemoryLimit,
			},
		},
	}
}