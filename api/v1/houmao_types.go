/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	//autoscaling "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta2"
	//"k8s.io/kubernetes"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HoumaoSpec defines the desired state of Houmao
type HoumaoSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//兼容k8s原生hpa
	ScaleTargetRef autoscaling.CrossVersionObjectReference	`json:"scaleTargetRef,omitempty"`
	Metrics autoscaling.MetricSpec							`json:"metrics,omitempty"`
	Behavor autoscaling.HorizontalPodAutoscalerBehavior		`json:"behavor"`
	MinReplicas *int										`json:"minReplicas"`
	MaxReplicas int											`json:"maxReplicas"`

	//造化需要的资源配置，即毫毛变猴需要的仙气量
	XianQiLiang	corev1.ResourceRequirements		`json:"xianQiLiang"`

	//要关注的service，关注对应的endpoints变化或服务质量等
	ServiceName []string	`json:"serviceName"`
}

//仙气信息，为每根毫毛都会备好一缕仙气，随时准备变化
type XianQiInfo struct {
	Name	string		`json:"name"`
	Namespace	string	`json:"namespace"`
}

type ZaohuaResult struct {
	Timestamp	time.Time	`json:"timestamp"`
	CurrentReplicas	int		`json:"currentReplicas"`
	DesiredReplicas int		`json:"desiredReplicas"`
}

// HoumaoStatus defines the observed state of Houmao
type HoumaoStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//原始实例数
	OriginReplicas	int	`json:originReplicas`

	//各造化名称，与猴毛名称一样
	XianQiInfo	XianQiInfo	`json:xianQiInfo`

	//当前造化结果
	CurrentZaohuaResult	ZaohuaResult	`json:"currentZaohuaResult"`

	//记录过去5次造化结果，不包含当前的造化结果
	Last5thZaohuaResult	[5]ZaohuaResult		`json:"last5thZaohuaResult"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Houmao is the Schema for the houmaoes API
type Houmao struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HoumaoSpec   `json:"spec,omitempty"`
	Status HoumaoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HoumaoList contains a list of Houmao
type HoumaoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Houmao `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Houmao{}, &HoumaoList{})
}
