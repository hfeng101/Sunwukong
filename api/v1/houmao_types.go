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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	//autoscaling "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta2"
	//"k8s.io/kubernetes"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// 猴毛可以随便变什么，参考：https://zhidao.baidu.com/question/1310541771961498219.html
// HoumaoSpec defines the desired state of Houmao
type HoumaoSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//兼容k8s原生hpa，更新会使得zaohua在下一轮开始前重新加载
	ScaleTargetRef autoscaling.CrossVersionObjectReference	`json:"scaleTargetRef,omitempty"`
	Metrics []autoscaling.MetricSpec							`json:"metrics,omitempty"`
	Behavor autoscaling.HorizontalPodAutoscalerBehavior		`json:"behavor"`
	MinReplicas *int32									`json:"minReplicas"`
	MaxReplicas int32									`json:"maxReplicas"`

	//要关注的service，关注对应的endpoints变化或服务质量等
	ServiceName []string	`json:"serviceName"`

	//造化需要的资源配置，即毫毛变猴需要仙气的量，一大口还是一小口，更新会重建xianqi deployment
	XianqiLiang	corev1.ResourceRequirements		`json:"xianqiLiang"`

}

//仙气信息，为每根毫毛都会有一类仙气支持，随时准备变化成不同
type XianqiInfo struct {
	Name	string		`json:"name"`
	Namespace	string	`json:"namespace"`
}

//记录当时弹性伸缩决策过程及结果
type MetricStatus struct {
	//Metrics	autoscaling.MetricStatus	`json:"metricStatus"`
	IsScaled bool `json:"isScaled"`
	ScaledReplicas int32 `json:"scaledReplicas"`
	MetricIndex int32	`json:"metricIndex"`
	MetricType	string	`json:"metricTeyp"`
	MetricName	string	`json:"metricName"`
	//ScaledValueStatus autoscaling.MetricValueStatus	`json:"scaledValueStatus"`
}

type ZaohuaResult struct {
	Timestamp	time.Time	`json:"timestamp"`
	CurrentReplicas	int32		`json:"currentReplicas"`
	DesiredReplicas int32		`json:"desiredReplicas"`

	MetricsElection MetricStatus	`json:"metricElection"`
	//autoscaling.MetricSpec
}

type ZaohuaRecord struct {
	//原始实例数，由猴毛controller更新
	OriginReplicas	int32	`json:originReplicas`

	//当前造化结果，由造化结果更新
	CurrentZaohuaResult	ZaohuaResult	`json:"currentZaohuaResult"`

	//记录过去5次造化结果，不包含当前的造化结果
	Last5thZaohuaResult	[5]ZaohuaResult		`json:"last5thZaohuaResult"`
}

// HoumaoStatus defines the observed state of Houmao
type HoumaoStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//施法结果，并记录过程状态
	ShifaPhase string `json:shifaPhase`

	//各造化名称，与猴毛名称一样，由猴毛controller更新
	XianqiInfo	XianqiInfo	`json:xianQiInfo`

	// 造化记录
	ZaohuaRecord ZaohuaRecord `json:zaohuaRecord`

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
