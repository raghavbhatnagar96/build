// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// BuildRunDomain is the domain used for all labels and annotations for this resource
	BuildRunDomain = "buildrun.shipwright.io"

	// LabelBuildRun is a label key for BuildRuns to define the name of the BuildRun
	LabelBuildRun = BuildRunDomain + "/name"

	// LabelBuildRunGeneration is a label key for BuildRuns to define the generation
	LabelBuildRunGeneration = BuildRunDomain + "/generation"
)

// BuildRunSpec defines the desired state of BuildRun
type BuildRunSpec struct {

	// BuildRef refers to the Build
	BuildRef *BuildRef `json:"buildRef"`

	// ServiceAccount refers to the kubernetes serviceaccount
	// which is used for resource control.
	// Default serviceaccount will be set if it is empty
	// +optional
	ServiceAccount *ServiceAccount `json:"serviceAccount,omitempty"`

	// Timeout defines the maximum run time of this BuildRun.
	// +optional
	// +kubebuilder:validation:Format=duration
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Output refers to the location where the generated
	// image would be pushed to. It will overwrite the output image in build spec
	// +optional
	Output *Image `json:"output,omitempty"`
}

// BuildRunStatus defines the observed state of BuildRun
type BuildRunStatus struct {
	// Conditions holds the latest available observations of a resource's current state.
	Conditions Conditions `json:"conditions,omitempty"`

	// The Succeeded status of the TaskRun
	//
	// Deprecated: Use Conditions instead. This will be removed in a future release.
	//
	// +optional
	Succeeded corev1.ConditionStatus `json:"succeeded,omitempty"`

	// The Succeeded reason of the TaskRun
	//
	// Deprecated: Use Conditions instead. This will be removed in a future release.
	//
	// +optional
	Reason string `json:"reason,omitempty"`

	// LatestTaskRunRef is the name of the TaskRun responsible for executing this BuildRun.
	//
	// TODO: This should be called something like "TaskRunName"
	//
	// +optional
	LatestTaskRunRef *string `json:"latestTaskRunRef,omitempty"`

	// StartTime is the time the build is actually started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the build completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// BuildSpec is the Build Spec of this BuildRun.
	// +optional
	BuildSpec *BuildSpec `json:"buildSpec,omitempty"`

	// FailedAt points to the resource where the BuildRun failed
	// +optional
	FailedAt *FailedAt `json:"failedAt,omitempty"`
}

// FailedAt describes the location where the failure happened
type FailedAt struct {
	Pod       string `json:"pod,omitempty"`
	Container string `json:"container,omitempty"`
}

// BuildRef can be used to refer to a specific instance of a Build.
type BuildRef struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// ServiceAccount can be used to refer to a specific ServiceAccount.
type ServiceAccount struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	// +optional
	Name *string `json:"name,omitempty"`
	// If generates a new ServiceAccount for the build
	// +optional
	Generate bool `json:"generate,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildRun is the Schema representing an instance of build execution
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=buildruns,scope=Namespaced,shortName=br;brs
// +kubebuilder:printcolumn:name="Succeeded",type="string",JSONPath=".status.conditions[?(@.type==\"Succeeded\")].status",description="The Succeeded status of the BuildRun"
// +kubebuilder:printcolumn:name="Reason",type="string",JSONPath=".status.conditions[?(@.type==\"Succeeded\")].reason",description="The Succeeded reason of the BuildRun"
// +kubebuilder:printcolumn:name="StartTime",type="date",JSONPath=".status.startTime",description="The start time of this BuildRun"
// +kubebuilder:printcolumn:name="CompletionTime",type="date",JSONPath=".status.completionTime",description="The completion time of this BuildRun"
type BuildRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildRunSpec   `json:"spec,omitempty"`
	Status BuildRunStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildRunList contains a list of BuildRun
type BuildRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildRun `json:"items"`
}

// Conditions defines a list of Condition
type Conditions []Condition

// Type used for defining the conditiont Type field flavour
type Type string

const (
	// Succeeded specifies that the resource has finished.
	// For resources that run to completion.
	Succeeded Type = "Succeeded"
)

// Condition defines the required fields for populating
// Build controllers Conditions
type Condition struct {
	// Type of condition
	// +required
	Type Type `json:"type" description:"type of status condition"`

	// Status of the condition, one of True, False, Unknown.
	// +required
	Status corev1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// LastTransitionTime last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" description:"last time the condition transit from one status to another"`

	// The reason for the condition last transition.
	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
}

func init() {
	SchemeBuilder.Register(&BuildRun{}, &BuildRunList{})
}

// GetReason returns the condition Reason, it ensures that by getting the Reason
// the call will not panic if the Condition is not present
func (c *Condition) GetReason() string {
	if c == nil {
		return ""
	}
	return c.Reason
}

// GetMessage returns the condition Message, it ensures that by getting the Message
// the call will not panic if the Condition is not present
func (c *Condition) GetMessage() string {
	if c == nil {
		return ""
	}
	return c.Message
}

// GetStatus returns the condition Status, it ensures that by getting the Status
// the call will not panic if the Condition is not present
func (c *Condition) GetStatus() corev1.ConditionStatus {
	if c == nil {
		return ""
	}
	return c.Status
}

// GetCondition returns a condition based on a type from a list of Conditions
func (brs *BuildRunStatus) GetCondition(t Type) *Condition {
	for _, c := range brs.Conditions {
		if c.Type == t {
			return &c
		}
	}
	return nil
}

// SetCondition updates a list of conditions with the provided condition
func (brs *BuildRunStatus) SetCondition(condition *Condition) {
	var idx = -1
	for i, c := range brs.Conditions {
		if c.Type == condition.Type {
			idx = i
			break
		}
	}

	if idx != -1 {
		brs.Conditions[idx] = *condition
	} else {
		brs.Conditions = append(brs.Conditions, *condition)
	}
}
