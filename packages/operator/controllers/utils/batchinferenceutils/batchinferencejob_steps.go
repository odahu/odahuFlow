/*
 *
 *     Copyright 2021 EPAM Systems
 *
 *     Licensed under the Apache License, Version 2.0 (the "License");
 *     you may not use this file except in compliance with the License.
 *     You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 *     Unless required by applicable law or agreed to in writing, software
 *     distributed under the License is distributed on an "AS IS" BASIS,
 *     WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *     See the License for the specific language governing permissions and
 *     limitations under the License.
 */

package batchinferenceutils

import (
	"fmt"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"path"
)

const (
	pathToOdahuToolsBin   = "/opt/odahu-flow/odahu-tools"
	toolsConfigVolume     = "config"
	workspacePath         = "/workspace"
	odahuModelPathEnvName = "ODAHU_MODEL_PATH"
)

// Step names
const (
	StepSyncData       = "sync-data"
	StepSyncModel      = "sync-model"
	StepValidateInput  = "validate-input"
	StepLogInput       = "log-input"
	StepUserContainer  = "user-container"
	StepValidateOutput = "validate-output"
	StepLogOutput      = "log-output"
	StepSyncOutput     = "sync-output"
)

var (
	toolsConfigVM = corev1.VolumeMount{
		Name:      toolsConfigVolume,
		ReadOnly:  true,
		MountPath: path.Join(XDGConfigHome, "odahu", ".odahu-tools.yaml"),
		SubPath:   ".odahu-tools.yaml",
	}
)

// paths
var (
	XDGConfigHome = path.Join(workspacePath, "config")
	// Dir where raw data should be synced
	rawInputPath = path.Join(workspacePath, "odahu-ws-raw-input")
	// Dir where raw data should be copied after validation
	odahuInputPath = path.Join(workspacePath, "odahu-ws-input")
	// Dir where raw model should be copied after validation and unzip
	odahuModelPath = path.Join(workspacePath, "odahu-ws-model")
	// Dir where raw user output is expected
	odahuRawOutputPath = path.Join(workspacePath, "odahu-ws-raw-output")
	// Dir where raw user output is copied after validation
	outputPath = path.Join(workspacePath, "odahu-ws-output")
)

// ENVs
var (
	XDGConfigHomeEnv = corev1.EnvVar{
		Name:  "XDG_CONFIG_HOME",
		Value: XDGConfigHome,
	}
	ToolsConfigPathEnv = corev1.EnvVar{
		Name:  "ODAHU_TOOLS_CONFIG",
		Value: toolsConfigVM.MountPath,
	}
	// Where we copy model in case of remote model registry (if embedded then path is provided by user)
	DefaultOdahuModelPathEnv = corev1.EnvVar{
		Name:  odahuModelPathEnvName,
		Value: odahuModelPath,
	}
	OdahuInputPathEnv = corev1.EnvVar{
		Name:  "ODAHU_INPUT_PATH",
		Value: odahuInputPath,
	}
	OdahuOutputPathEnv = corev1.EnvVar{
		Name:  "ODAHU_OUTPUT_PATH",
		Value: odahuRawOutputPath,
	}
)

// GetConfigureRCloneStep return step that
// configures environment (rclone config) for syncing data and model
// using ODAHU connections
func GetConfigureRCloneStep(image string, res corev1.ResourceRequirements, conns ...string) tektonv1beta1.Step {

	var args = []string{"auth", "configure-rclone"}
	for _, c := range conns {
		args = append(args, "--conn", fmt.Sprintf("%s:%s", c, c))
	}
	return tektonv1beta1.Step{
		Container: corev1.Container{
			Name:         "configure-rclone",
			Image:        image,
			Command:      []string{pathToOdahuToolsBin},
			Args:         args,
			Env:          []corev1.EnvVar{XDGConfigHomeEnv, ToolsConfigPathEnv},
			VolumeMounts: []corev1.VolumeMount{toolsConfigVM},
			Resources:    res,
		},
	}
}

// GetSyncDataStep return step that
// syncs input data to pre-stage directory inside Pod
// where input will be validated and copied to user container's input directory
func GetSyncDataStep(
	rcloneImage string,
	rcloneConfigName string,
	bucketName string,
	inputPath string,
	res corev1.ResourceRequirements,
) tektonv1beta1.Step {
	sourcePrefix := fmt.Sprintf("%s:%s", rcloneConfigName, bucketName)
	source := path.Join(sourcePrefix, inputPath)
	return tektonv1beta1.Step{
		Container: corev1.Container{
			Name:      StepSyncData,
			Image:     rcloneImage,
			Command:   []string{"rclone"},
			Args:      []string{"sync", "-P", source, rawInputPath},
			Env:       []corev1.EnvVar{XDGConfigHomeEnv},
			Resources: res,
		},
	}
}

// GetSyncModelStep return step that
// syncs model from s3/gcs/azureblob bucket to workspace
func GetObjectStorageModelSyncStep(image string, connName string,
	modelPath string, res corev1.ResourceRequirements) tektonv1beta1.Step {
	return tektonv1beta1.Step{
		Container: corev1.Container{
			Name:    StepSyncModel,
			Image:   image,
			Command: []string{pathToOdahuToolsBin},
			Args: []string{
				"registry", "object-storage", "sync",
				"--conn", connName,
				"--remotePath", modelPath,
				"--localPath", odahuModelPath},
			VolumeMounts: []corev1.VolumeMount{toolsConfigVM},
			Env:          []corev1.EnvVar{ToolsConfigPathEnv},
			Resources:    res,
		},
	}
}

// GetValidateInputStep return step that
// validates raw input according kubeflow prediction api (version 2) InferenceRequest object.
// Only validated files are copied to user container for inference.
func GetValidateInputStep(image string, res corev1.ResourceRequirements) tektonv1beta1.Step {
	return tektonv1beta1.Step{
		Container: corev1.Container{
			Name:         StepValidateInput,
			Image:        image,
			Command:      []string{pathToOdahuToolsBin},
			Args:         []string{"batch", "validate", "input", "-s", rawInputPath, "-d", odahuInputPath},
			VolumeMounts: []corev1.VolumeMount{toolsConfigVM},
			Env:          []corev1.EnvVar{ToolsConfigPathEnv},
			Resources:    res,
		},
	}
}

// GetLogInputStep return step that
// logs model input to feedback storage (catch requests)
func GetLogInputStep(image string, requestID string, res corev1.ResourceRequirements,
	modelName string, modelVersion string) tektonv1beta1.Step {
	return tektonv1beta1.Step{
		Container: corev1.Container{
			Name:    StepLogInput,
			Image:   image,
			Command: []string{pathToOdahuToolsBin},
			Args: []string{"batch", "log", "input", odahuInputPath,
				"-m", modelName, "--version", modelVersion, "-r", requestID},
			VolumeMounts: []corev1.VolumeMount{toolsConfigVM},
			Env:          []corev1.EnvVar{ToolsConfigPathEnv},
			Resources:    res,
		},
	}
}

// GetUserContainer return step that
// execute user defined container for inference
func GetUserContainer(
	image string, command []string, args []string,
	res corev1.ResourceRequirements, odahuModelPathEnv corev1.EnvVar) tektonv1beta1.Step {
	return tektonv1beta1.Step{
		Container: corev1.Container{
			Name:         StepUserContainer,
			Image:        image,
			Command:      command,
			Args:         args,
			VolumeMounts: []corev1.VolumeMount{toolsConfigVM},
			Env:          []corev1.EnvVar{OdahuInputPathEnv, OdahuOutputPathEnv, odahuModelPathEnv},
			Resources:    res,
		},
	}
}

// GetValidateOutputStep return step that
// validates raw output according kubeflow prediction api (version 2) InferenceResponse object.
// Only validated files are copied from user container results to destination
func GetValidateOutputStep(image string, res corev1.ResourceRequirements) tektonv1beta1.Step {
	return tektonv1beta1.Step{
		Container: corev1.Container{
			Name:         StepValidateOutput,
			Image:        image,
			Command:      []string{pathToOdahuToolsBin},
			Args:         []string{"batch", "validate", "output", "-s", odahuRawOutputPath, "-d", outputPath},
			VolumeMounts: []corev1.VolumeMount{toolsConfigVM},
			Env:          []corev1.EnvVar{ToolsConfigPathEnv},
			Resources:    res,
		},
	}
}

// GetLogOutputStep return step that
// logs model output to feedback storage (catch responses)
func GetLogOutputStep(image string, requestID string, res corev1.ResourceRequirements,
	modelName string, modelVersion string) tektonv1beta1.Step {
	return tektonv1beta1.Step{
		Container: corev1.Container{
			Name:    StepLogOutput,
			Image:   image,
			Command: []string{pathToOdahuToolsBin},
			Args: []string{"batch", "log", "output", outputPath,
				"-m", modelName, "--version", modelVersion, "-r", requestID},
			VolumeMounts: []corev1.VolumeMount{toolsConfigVM},
			Env:          []corev1.EnvVar{ToolsConfigPathEnv},
			Resources:    res,
		},
	}
}

// GetSyncOutputStep return step that
// syncs output data to bucket
func GetSyncOutputStep(
	rcloneImage string,
	rcloneConfigName string,
	bucketName string,
	remoteOutputPath string,
	res corev1.ResourceRequirements,
) tektonv1beta1.Step {
	prefix := fmt.Sprintf("%s:%s", rcloneConfigName, bucketName)
	dest := path.Join(prefix, remoteOutputPath)
	return tektonv1beta1.Step{
		Container: corev1.Container{
			Name:      StepSyncOutput,
			Image:     rcloneImage,
			Command:   []string{"rclone"},
			Args:      []string{"copy", "-v", "-v", outputPath, dest},
			Env:       []corev1.EnvVar{XDGConfigHomeEnv},
			Resources: res,
		},
	}
}
